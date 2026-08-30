.PHONY: help deps ui ui-deps ui-dev build build-go test test-race test-coverage fmt fmt-check vet lint lint-install clean qa

BINARY_NAME=sourceant-agent
VERSION?=$(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

help:
	@echo "SourceAnt agent - build commands"
	@echo ""
	@echo "make deps          - Download dependencies, Go and npm"
	@echo "make ui            - Build the local view into what the binary embeds"
	@echo "make ui-dev        - Run the view against a running agent"
	@echo "make build         - Build the view, then the agent binary"
	@echo "make build-go      - Build the binary with whatever view is embedded"
	@echo "make test          - Run unit tests"
	@echo "make test-race     - Run unit tests under the race detector"
	@echo "make test-coverage - Run tests with a coverage report"
	@echo "make fmt           - Format code with gofmt"
	@echo "make fmt-check     - Check gofmt formatting"
	@echo "make vet           - Run go vet"
	@echo "make lint          - Run golangci-lint"
	@echo "make qa            - Run fmt-check, vet, lint, and tests"
	@echo "make clean         - Clean build artifacts"

deps: ui-deps
	go mod download
	go mod tidy

ui-deps:
	cd ui && npm install --no-audit --no-fund

# Vite writes into internal/ui/assets, which is what the binary embeds, so a
# build and the sources it came from cannot disagree.
ui:
	cd ui && npm run build

ui-dev:
	cd ui && npm run dev

build: ui build-go

build-go:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/agent

test:
	go test ./...

test-race:
	go test -race ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

fmt:
	go fmt ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Run gofmt on:" && gofmt -l . && exit 1)

vet:
	go vet ./...

lint:
	@command -v golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || \
	(test -x "$$(go env GOPATH)/bin/golangci-lint" && "$$(go env GOPATH)/bin/golangci-lint" run ./... || \
	(echo "golangci-lint not found. Run 'make lint-install' first." && exit 1))

lint-install:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

qa: fmt-check vet lint test

clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	go clean
