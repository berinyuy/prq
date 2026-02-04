# Ideal Workflow

This workflow keeps you safe by default while still moving quickly.

## 1. Verify setup

```bash
prq doctor
```

This confirms `gh` auth and your provider configuration.

## 2. Build your queue

```bash
prq queue --limit 50
```

Or use the full-screen picker:

```bash
prq pick
```

Use filters if you want to narrow down by repo, owner, label, or CI status.

## 3. Generate a review plan

```bash
prq review OWNER/REPO#123
```

This saves a local draft automatically. If you want a preview only (no review output), use:

```bash
prq draft OWNER/REPO#123
```

Optional test run (uses `prq.yaml`):

```bash
prq review OWNER/REPO#123 --run-tests
```

## 4. Inspect before posting

Use submit in preview mode to see exactly what would be posted:

```bash
prq submit OWNER/REPO#123 --dry-run
```

## 5. Submit to GitHub

```bash
prq submit OWNER/REPO#123
```

You will be asked to confirm. Nothing posts without confirmation.

## 6. Follow up after changes

When new commits land or you want to check open threads:

```bash
prq followup OWNER/REPO#123
```

If the head commit changed, re-run `prq review` or `prq draft`.
