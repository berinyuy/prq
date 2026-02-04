# Configuration

`prq` merges user config from `~/.prq/config.yaml` with repo config from `./prq.yaml`.

## User config

Location: `~/.prq/config.yaml`

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

Field notes:

- `provider.command` and `provider.args` control the local CLI used to run the model.
- `user_rules` are appended to every prompt.
- `queue.default_limit` and `queue.default_sort` apply to `prq queue` and `prq pick` when no flags are provided.
- `redaction.enabled` toggles secret redaction before calling the provider.
- `tui.enabled` toggles the full-screen picker.

## Repo config

Location: `./prq.yaml`

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

Field notes:

- `repo_rules` are appended to the prompt for this repo only.
- `tests.commands` are executed when you pass `--run-tests`. Each command is run via `sh -lc` inside a temporary clone of the PR branch. Output is captured and redacted before inclusion.
- `diff.ignore` excludes files from the diff prompt.
- `diff.max_files` and `diff.max_chunk_chars` limit prompt size.

## Overrides (env)

- `PRQ_MOCK=1` enables fixtures and fake provider.
- `PRQ_MOCK_DIR` fixture directory for GH responses.
- `PRQ_PROVIDER_FIXTURE` JSON review plan fixture.
- `PRQ_PROMPT_PATH` prompt template path.
- `PRQ_SCHEMA_PATH` JSON schema path.
- `PRQ_DB_PATH` SQLite DB path.
- `PRQ_NOW` fixed time (RFC3339) for deterministic output.
