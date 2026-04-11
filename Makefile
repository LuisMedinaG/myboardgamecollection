.PHONY: build run dev clean bgg-login test test-v cover cover-html

# Build the binary
build:
	go build -o boardgames .

# Build and run
run: build
	./boardgames

# Development: build and run
dev:
	go run .

# Run all tests
test:
	go test ./...

# Run all tests with verbose output
test-v:
	go test ./... -v

# Run tests with coverage report
cover:
	go test ./... -cover

# Run tests and generate HTML coverage report
cover-html:
	go test ./... -coverprofile=/tmp/coverage.out && \
	go tool cover -html=/tmp/coverage.out -o /tmp/coverage.html && \
	echo "Coverage report: /tmp/coverage.html"

# Remove build artifacts and database
clean:
	rm -f boardgames games.db

# Print BGG Cookie header (reads ADMIN_* from .env if present, else environment). Run from repo root.
bgg-login:
	go run ./cmd/bgg-login
