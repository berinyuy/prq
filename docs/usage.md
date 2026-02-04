# Usage

## Queue

```bash
prq queue
prq queue --limit 50 --owner my-org
prq queue --repo acme/app --label bug --checks failure
prq queue --draft false --sort updated
```

## Review

```bash
prq review OWNER/REPO#123
prq review https://github.com/acme/app/pull/123 --format md
prq review acme/app#42 --max-issues 5
```

## Doctor

```bash
prq doctor
```
