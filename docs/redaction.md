# Redaction

prq redacts secrets before sending content to the model.

## Coverage

- Common tokens (GitHub, AWS, JWTs)
- PEM and private keys
- URL query params (token, key, secret)
- High-entropy strings

Redacted values are replaced with `[REDACTED_SECRET]`.

## Limitations

- Redaction is heuristic and may not catch all secrets.
- Always review output before sharing.
