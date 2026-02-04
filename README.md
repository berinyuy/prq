# prq

`prq` is a CLI for managing GitHub pull request review queues and generating structured review plans using the local Claude Code CLI. It is designed to be safe by default: nothing is posted unless you confirm, and all content is redacted before model calls.

## Requirements

- GitHub CLI: `gh` (authenticated)
- Claude Code CLI: `claude`

## Install

Placeholders used below:

- `<tap>`: your Homebrew tap in the form `OWNER/tap` (example: `acme/homebrew-tap`).
- `<install_url>`: URL to your install script for released binaries.


### macOS (Homebrew)

```bash
brew install <tap>/prq
```

### Windows (Scoop)

```powershell
scoop install prq
```

### Linux/macOS (curl)

```bash
curl -fsSL <install_url> | sh
```

### From Source (Go)

Requirements: Go 1.22+ and git.

```bash
git clone https://github.com/berinyuy/prq.git
cd prq
go install ./cmd/prq
```

The binary is installed to `$(go env GOPATH)/bin`. Ensure that directory is on your PATH.

## Mock Mode

For testing without live GitHub/Claude access, set:

```bash
PRQ_MOCK=1 \
PRQ_MOCK_DIR=./testdata/gh \
PRQ_PROVIDER_FIXTURE=./testdata/provider/review.json \
PRQ_DB_PATH=/tmp/prq.db \
prq queue
```

You can also override prompt/schema paths for tests:

```bash
PRQ_PROMPT_PATH=./prompts/code-reviewer.txt \
PRQ_SCHEMA_PATH=./schemas/review_plan.schema.json \
prq review acme/app#42
```

## Quick Start

```bash
prq doctor
prq queue --limit 50
prq pick
prq review OWNER/REPO#123
prq draft OWNER/REPO#123
prq submit OWNER/REPO#123
prq followup OWNER/REPO#123
```

`prq review` also saves a local draft you can submit later with `prq submit`.

## Ideal Workflow

See docs/workflow.md for a recommended end-to-end flow.

## TUI Preview

![PRQ TUI list view](docs/images/tui-list.svg)

![PRQ TUI actions](docs/images/tui-actions.svg)

## Configuration

User config: `~/.prq/config.yaml`

```yaml
provider:
  command: claude
  args: []
user_rules: []
queue:
  default_limit: 200
  default_sort: oldest
redaction:
  enabled: true
tui:
  enabled: true
```

Repo config: `./prq.yaml`

```yaml
repo_rules:
  - "Follow repo style guidelines"
tests:
  commands:
    - "go test ./..."
diff:
  ignore:
    - "**/*.md"
  max_files: 50
  max_chunk_chars: 8000
```

## Safety

- Nothing is posted to GitHub unless you run `prq submit` and confirm.
- `prq submit --dry-run` lets you preview what would be posted without actually posting.
- `prq submit` requires typing `y` to confirm (use `--yes` to skip for automation).
- Redaction runs on all prompt content before calling the provider.
- `--run-tests` runs commands from `prq.yaml` in a temp clone and includes the output in the review prompt.

## Troubleshooting

- `gh` not found: install GitHub CLI and run `gh auth login`.
- `claude` not found: install Claude Code CLI and confirm it is in PATH.
- Provider schema errors: run `prq doctor` and verify schema path.

## Docs

- docs/usage.md
- docs/workflow.md
- docs/config.md
- docs/redaction.md
- docs/provider-claude.md
