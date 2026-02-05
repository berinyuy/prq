package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/brianndofor/prq/internal/config"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// escapeForShell escapes single quotes in a string for safe use in shell single-quoted strings
func escapeForShell(s string) string {
	// In bash, to include a single quote in a single-quoted string,
	// you need to end the quote, add an escaped quote, and restart: '\''
	return strings.ReplaceAll(s, "'", "'\\''")
}

// loadSchemaContent reads a JSON schema file and returns its content as a string
func loadSchemaContent(schemaPath string) (string, error) {
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file: %w", err)
	}
	// Compact the JSON to remove whitespace (optional, helps with arg length)
	var buf bytes.Buffer
	if err := json.Compact(&buf, content); err != nil {
		// If compacting fails, use the original content
		return string(content), nil
	}
	return buf.String(), nil
}

// claudeResponse represents the wrapper response from the Claude CLI
// when using --output-format json with --json-schema
type claudeResponse struct {
	Type             string          `json:"type"`
	Subtype          string          `json:"subtype"`
	IsError          bool            `json:"is_error"`
	StructuredOutput json.RawMessage `json:"structured_output"`
}

// extractStructuredOutput parses the Claude CLI JSON response and extracts
// the structured_output field which contains the actual schema-conforming data
func extractStructuredOutput(raw []byte) ([]byte, error) {
	var resp claudeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse claude response wrapper: %w", err)
	}
	if resp.IsError {
		return nil, fmt.Errorf("claude returned an error response: %s", string(raw))
	}
	if len(resp.StructuredOutput) == 0 {
		return nil, fmt.Errorf("claude response missing structured_output field")
	}
	return resp.StructuredOutput, nil
}

type Runner interface {
	RunReview(ctx context.Context, prompt string, schemaPath string) (ReviewPlan, string, error)
	HealthCheck(ctx context.Context, schemaPath string) error
}

type ClaudeRunner struct {
	command string
	args    []string
}

func NewClaudeRunner(cfg config.ProviderConfig) *ClaudeRunner {
	command := cfg.Command
	if command == "" {
		command = "claude"
	}
	return &ClaudeRunner{command: command, args: cfg.Args}
}

func (c *ClaudeRunner) RunReview(ctx context.Context, prompt string, schemaPath string) (ReviewPlan, string, error) {
	// Load schema content - CLI expects JSON string, not file path
	schemaContent, err := loadSchemaContent(schemaPath)
	if err != nil {
		return ReviewPlan{}, "", err
	}

	// Escape single quotes in prompt for shell safety
	escapedPrompt := escapeForShell(prompt)

	// Build the full command to run through shell (handles JSON quoting)
	shellCmd := fmt.Sprintf("%s -p --output-format json --json-schema '%s' '%s'",
		c.command, schemaContent, escapedPrompt)
	cmd := exec.CommandContext(ctx, "bash", "-c", shellCmd)
	// Explicitly set stdin to nil to prevent any TTY inheritance
	cmd.Stdin = nil
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return ReviewPlan{}, "", fmt.Errorf("provider failed: %w\n%s", err, stderr.String())
	}
	raw := stdout.String()
	// Extract the structured_output from Claude's response wrapper
	structuredOutput, err := extractStructuredOutput([]byte(raw))
	if err != nil {
		return ReviewPlan{}, raw, err
	}
	if err := validateJSON(schemaPath, structuredOutput); err != nil {
		return ReviewPlan{}, string(structuredOutput), err
	}
	var plan ReviewPlan
	if err := json.Unmarshal(structuredOutput, &plan); err != nil {
		return ReviewPlan{}, string(structuredOutput), fmt.Errorf("failed to parse provider JSON: %w", err)
	}
	return plan, string(structuredOutput), nil
}

func (c *ClaudeRunner) HealthCheck(ctx context.Context, schemaPath string) error {
	minimal := `{"summary":"ok","risk_level":"low","decision":"comment","key_changes":[],"issues":[],"questions":[],"praise":[],"draft_review_body":""}`
	_ = minimal

	// Load schema content - CLI expects JSON string, not file path
	schemaContent, err := loadSchemaContent(schemaPath)
	if err != nil {
		return err
	}

	// Build the full command to run through shell (handles JSON quoting)
	shellCmd := fmt.Sprintf("%s --print --output-format json --json-schema '%s' 'Return JSON matching schema.'",
		c.command, schemaContent)
	cmd := exec.CommandContext(ctx, "bash", "-c", shellCmd)
	// Explicitly set stdin to nil to prevent any TTY inheritance
	cmd.Stdin = nil
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Check if it's a context timeout/cancellation
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("provider health check timed out\nCommand: %s\nStdout: %s\nStderr: %s", shellCmd, stdout.String(), stderr.String())
		}
		if ctx.Err() == context.Canceled {
			return fmt.Errorf("provider health check was cancelled\nCommand: %s\nStdout: %s\nStderr: %s", shellCmd, stdout.String(), stderr.String())
		}
		// Regular error - include stderr and stdout
		return fmt.Errorf("provider health check failed: %w\nCommand: %s\nStdout: %s\nStderr: %s", err, shellCmd, stdout.String(), stderr.String())
	}
	if len(stdout.Bytes()) == 0 {
		return fmt.Errorf("provider health check failed: empty output\nCommand: %s\nStderr: %s", shellCmd, stderr.String())
	}
	// Extract the structured_output from Claude's response wrapper
	structuredOutput, err := extractStructuredOutput(stdout.Bytes())
	if err != nil {
		_ = minimal
		return fmt.Errorf("provider health check failed to extract structured output: %w\nRaw output: %s", err, stdout.String())
	}
	if err := validateJSON(schemaPath, structuredOutput); err != nil {
		return fmt.Errorf("provider output failed schema validation: %w\nOutput: %s\nStderr: %s", err, string(structuredOutput), stderr.String())
	}
	return nil
}

func validateJSON(schemaPath string, data []byte) error {
	abspath, err := filepath.Abs(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to resolve schema path: %w", err)
	}
	schema, err := jsonschema.Compile("file://" + abspath)
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}
	// Unmarshal JSON into interface{} for validation
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := schema.Validate(v); err != nil {
		return fmt.Errorf("provider output failed schema validation: %w", err)
	}
	return nil
}
