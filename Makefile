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

# Print BGG Cookie header (uses ADMIN_USERNAME and ADMIN_PASSWORD). Example: make bgg-login
bgg-login:
	go run ./cmd/bgg-login
