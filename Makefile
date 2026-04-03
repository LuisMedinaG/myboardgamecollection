TEMPL := $(shell go env GOPATH)/bin/templ

.PHONY: setup generate build run dev clean

# First-time setup: install templ CLI, generate code, resolve deps
setup:
	go install github.com/a-h/templ/cmd/templ@latest
	$(TEMPL) generate
	go mod tidy

# Generate Go code from .templ files
generate:
	$(TEMPL) generate

# Build the binary
build: generate
	go build -o boardgames .

# Build and run
run: build
	./boardgames

# Development: generate and run (restart manually on changes)
dev: generate
	go run .

# Remove build artifacts and database
clean:
	rm -f boardgames games.db
	rm -f *_templ.go
