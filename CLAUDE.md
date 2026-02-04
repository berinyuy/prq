# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`prq` is a Go CLI for managing GitHub PR review queues and generating structured code review plans via the local Claude Code CLI. Safety-first design: all content is redacted before LLM calls, nothing posts to GitHub without explicit confirmation.

## Build & Development Commands

```bash
make build          # go build ./cmd/prq
make test           # go test ./...
make fmt            # gofmt -w cmd internal
make lint           # golangci-lint run (fails on issues)
make lint-dry       # golangci-lint run (report only)
make docker-test    # build + run tests in Docker
```

Run a single test:
```bash
go test ./internal/diff/ -run TestParseUnified
```

Run tests with mock fixtures (no live GitHub/Claude needed):
```bash
PRQ_MOCK=1 PRQ_MOCK_DIR=./testdata/gh PRQ_PROVIDER_FIXTURE=./testdata/provider/review.json PRQ_DB_PATH=/tmp/prq.db go test ./...
```

## Architecture

Entry point: `cmd/prq/main.go` bootstraps cobra commands defined in `internal/cli/`.

### Package Layout (`internal/`)

- **cli/** - Cobra command implementations. `app.go` creates the central App context (config, GitHub client, provider, store) injected via cobra context. Commands: `queue`, `review`, `doctor`, `config`.
- **config/** - Loads user config (`~/.prq/config.yaml`) and repo config (`./prq.yaml`) via Viper. Provides safe defaults.
- **github/** - Wraps `gh` CLI via a `Runner` interface. `RealRunner` executes actual commands; `FixtureRunner` returns test data from `testdata/gh/`. Includes `ParsePR()` for parsing `OWNER/REPO#N` and URL formats.
- **provider/** - LLM abstraction via `Runner` interface. `ClaudeRunner` invokes `claude -p <prompt> --output-format json --json-schema <schema>` and validates output. `FakeRunner` loads fixtures for tests.
- **diff/** - Parses unified diffs into `FileDiff` structs. `BuildChunks()` splits diffs by file count and character limits with glob-based ignore patterns.
- **redact/** - Multi-pass secret detection (regex patterns for AWS keys, GitHub tokens, JWTs, etc. + entropy-based detection). Runs on all content before it reaches the LLM.
- **prompt/** - Template rendering for `prompts/code-reviewer.txt`. Substitutes PR metadata, rules, diffs, and CI status into the system prompt.
- **store/** - SQLite persistence for PR tracking and draft reviews.

### Key Data Flow

`review` command: parse PR ref -> fetch PR via `gh` -> parse diff -> chunk by size limits -> redact secrets -> render prompt template -> invoke Claude CLI with JSON schema -> validate response -> output ReviewPlan.

### Testing Patterns

- **Runner/Provider interfaces** enable testing without live GitHub or Claude access
- **Golden tests** in `internal/cli/golden_test.go` compare command output against `testdata/golden/` baselines. Set `PRQ_NOW` env var for deterministic timestamps.
- **Fixture data** lives in `testdata/` (gh API responses, provider output, golden files, sample diffs)
- Mock mode env vars: `PRQ_MOCK=1`, `PRQ_MOCK_DIR`, `PRQ_PROVIDER_FIXTURE`, `PRQ_DB_PATH`, `PRQ_PROMPT_PATH`, `PRQ_SCHEMA_PATH`, `PRQ_NOW`

### Static Assets

- `prompts/code-reviewer.txt` - System prompt for the review LLM call
- `schemas/review_plan.schema.json` - JSON Schema defining the ReviewPlan output structure (risk_level, decision, issues with severity/category, praise, questions, draft_review_body)
