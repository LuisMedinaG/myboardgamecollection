.PHONY: build run dev clean

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
