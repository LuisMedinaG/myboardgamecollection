.PHONY: build run dev clean bgg-login test test-v cover cover-html vet check

GOCACHE ?= /tmp/go-build-cache
GOENV = env GOCACHE=$(GOCACHE)
GO = $(GOENV) go

# Build the binary
build:
	$(GO) build -o boardgames .

# Build and run
run: build
	./boardgames

# Development: build and run
dev:
	$(GO) run .

# Run all tests
test:
	$(GO) test ./...

# Run all tests with verbose output
test-v:
	$(GO) test ./... -v

# Run tests with coverage report
cover:
	$(GO) test ./... -cover

# Run tests and generate HTML coverage report
cover-html:
	$(GO) test ./... -coverprofile=/tmp/coverage.out && \
	$(GO) tool cover -html=/tmp/coverage.out -o /tmp/coverage.html && \
	echo "Coverage report: /tmp/coverage.html"

# Run static analysis
vet:
	$(GO) vet ./...

# Standard verification suite for local and CI usage
check: build test vet

# Remove build artifacts and database
clean:
	rm -f boardgames games.db

# Print BGG Cookie header (reads ADMIN_* from .env if present, else environment). Run from repo root.
bgg-login:
	$(GO) run ./cmd/bgg-login
