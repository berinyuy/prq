.PHONY: setup build test fmt lint-dry lint docker-test

setup:
	go mod tidy

build:
	go build ./cmd/prq

test:
	go test ./...

fmt:
	gofmt -w cmd internal

lint-dry:
	golangci-lint run --out-format=tab --issues-exit-code=0

lint: lint-dry
	golangci-lint run

docker-test:
	docker build -t prq-test -f Dockerfile .
	docker run --rm prq-test
