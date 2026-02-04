package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/brianndofor/prq/internal/config"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

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
	args := append([]string{}, c.args...)
	args = append(args, "-p", prompt, "--output-format", "json", "--json-schema", schemaPath)
	cmd := exec.CommandContext(ctx, c.command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return ReviewPlan{}, "", fmt.Errorf("provider failed: %w\n%s", err, stderr.String())
	}
	raw := stdout.String()
	if err := validateJSON(schemaPath, []byte(raw)); err != nil {
		return ReviewPlan{}, raw, err
	}
	var plan ReviewPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return ReviewPlan{}, raw, fmt.Errorf("failed to parse provider JSON: %w", err)
	}
	return plan, raw, nil
}

func (c *ClaudeRunner) HealthCheck(ctx context.Context, schemaPath string) error {
	minimal := `{"summary":"ok","risk_level":"low","decision":"comment","key_changes":[],"issues":[],"questions":[],"praise":[],"draft_review_body":""}`
	args := append([]string{}, c.args...)
	args = append(args, "-p", "Return JSON matching schema.", "--output-format", "json", "--json-schema", schemaPath)
	cmd := exec.CommandContext(ctx, c.command, args...)
	cmd.Stdin = bytes.NewBufferString("Return JSON only.")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("provider health check failed: %w\n%s", err, stderr.String())
	}
	if len(stdout.Bytes()) == 0 {
		return fmt.Errorf("provider health check failed: empty output")
	}
	if err := validateJSON(schemaPath, []byte(stdout.String())); err != nil {
		_ = minimal
		return err
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
	if err := schema.Validate(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("provider output failed schema validation: %w", err)
	}
	return nil
}
