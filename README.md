# prq

`prq` is a CLI for managing GitHub pull request review queues and generating structured review plans using the local Claude Code CLI. It is designed to be safe by default: nothing is posted unless you confirm, and all content is redacted before model calls.

## Requirements

- GitHub CLI: `gh` (authenticated)
- Claude Code CLI: `claude`

## Install

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
curl -fsSL https://example.com/prq/install.sh | sh
```

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
prq review OWNER/REPO#123
```

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

- No posting without explicit confirmation.
- Redaction runs on all prompt content before calling the provider.

## Troubleshooting

- `gh` not found: install GitHub CLI and run `gh auth login`.
- `claude` not found: install Claude Code CLI and confirm it is in PATH.
- Provider schema errors: run `prq doctor` and verify schema path.

## Docs

- docs/usage.md
- docs/config.md
- docs/redaction.md
- docs/provider-claude.md
