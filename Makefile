.PHONY: build test lint clean install

BINARY := ops
VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)
GOFLAGS := -ldflags="-s -w -X github.com/blindly/ops/internal/version.Version=$(VERSION)" -trimpath

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -o $(BINARY) ./cmd/ops

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)

install:
	CGO_ENABLED=0 go install $(GOFLAGS) ./cmd/ops