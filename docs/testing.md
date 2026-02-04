# Testing

## Local

```bash
go test ./...
```

## Docker

```bash
docker build -t prq-test -f Dockerfile .
docker run --rm prq-test
```

## Lint

```bash
make lint-dry
make lint
```
