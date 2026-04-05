.PHONY: build run dev clean bgg-login

# Build the binary
build:
	go build -o boardgames .

# Build and run
run: build
	./boardgames

# Development: build and run
dev:
	go run .

# Remove build artifacts and database
clean:
	rm -f boardgames games.db

# Print BGG Cookie header (reads ADMIN_* from .env if present, else environment). Run from repo root.
bgg-login:
	go run ./cmd/bgg-login
