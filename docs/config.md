# Configuration

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

## Overrides (env)

- `PRQ_MOCK=1` enables fixtures and fake provider
- `PRQ_MOCK_DIR` fixture directory for GH responses
- `PRQ_PROVIDER_FIXTURE` JSON review plan fixture
- `PRQ_PROMPT_PATH` prompt template path
- `PRQ_SCHEMA_PATH` JSON schema path
- `PRQ_DB_PATH` SQLite DB path
- `PRQ_NOW` fixed time (RFC3339) for deterministic output
