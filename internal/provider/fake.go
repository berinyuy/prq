package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type FakeRunner struct {
	FixturePath string
}

func NewFakeRunner(path string) *FakeRunner {
	return &FakeRunner{FixturePath: path}
}

func (f *FakeRunner) RunReview(ctx context.Context, prompt string, schemaPath string) (ReviewPlan, string, error) {
	_ = ctx
	_ = prompt
	_ = schemaPath
	data, err := os.ReadFile(f.FixturePath)
	if err != nil {
		return ReviewPlan{}, "", fmt.Errorf("failed to read provider fixture: %w", err)
	}
	var plan ReviewPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return ReviewPlan{}, string(data), fmt.Errorf("invalid provider fixture: %w", err)
	}
	return plan, string(data), nil
}

func (f *FakeRunner) HealthCheck(ctx context.Context, schemaPath string) error {
	_ = ctx
	_ = schemaPath
	return nil
}
