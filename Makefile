.PHONY: build run dev dev-go dev-all css css-watch clean \
        bgg-login test test-v cover cover-html vet check \
        tailwind-install \
        react-dev react-build react-install react-lint

GOCACHE ?= /tmp/go-build-cache
GOENV = env GOCACHE=$(GOCACHE)
GO = $(GOENV) go

TAILWIND = ./tailwindcss
CSS_IN  = static/input.css
CSS_OUT = static/style.css

# ── CSS ──────────────────────────────────────────────────────────────

# Build CSS once (minified) — run before committing or deploying
css:
	$(TAILWIND) -i $(CSS_IN) -o $(CSS_OUT) --minify

# Watch CSS for changes (dev use)
css-watch:
	$(TAILWIND) -i $(CSS_IN) -o $(CSS_OUT) --watch

# Download Tailwind CLI binary (macOS Apple Silicon)
tailwind-install:
	@echo "Downloading Tailwind CSS CLI v4 (macOS arm64)..."
	curl -sLo tailwindcss \
	  https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-arm64
	chmod +x tailwindcss
	@echo "Done. Run 'make css' to build the stylesheet."

# ── Go ───────────────────────────────────────────────────────────────

# Build the binary
build:
	$(GO) build -o boardgames .

# Build CSS + binary, then run
run: css build
	./boardgames

# Development: run Go app + Tailwind watcher concurrently (Ctrl-C stops both)
dev:
	@trap 'kill 0' INT TERM; \
	$(TAILWIND) -i $(CSS_IN) -o $(CSS_OUT) --watch & \
	$(GO) run .; \
	wait

# Development: run Go app only (use when CSS is already up-to-date)
dev-go:
	$(GO) run .

# ── Tests & quality ──────────────────────────────────────────────────

test:
	$(GO) test ./...

test-v:
	$(GO) test ./... -v

cover:
	$(GO) test ./... -cover

cover-html:
	$(GO) test ./... -coverprofile=/tmp/coverage.out && \
	$(GO) tool cover -html=/tmp/coverage.out -o /tmp/coverage.html && \
	echo "Coverage report: /tmp/coverage.html"

vet:
	$(GO) vet ./...

# Full verification: CSS + build + tests + vet
check: css build test vet

# ── Combined ─────────────────────────────────────────────────────────

# Run Go app + Tailwind watcher + React dev server concurrently (Ctrl-C stops all)
dev-all:
	@trap 'kill 0' INT TERM; \
	$(TAILWIND) -i $(CSS_IN) -o $(CSS_OUT) --watch & \
	$(GO) run . & \
	cd $(REACT_DIR) && bun dev; \
	wait

# ── React frontend ────────────────────────────────────────────────────

REACT_DIR = react-app

# Start Vite dev server at localhost:5173
react-dev:
	cd $(REACT_DIR) && bun dev

# Type-check + production build → react-app/dist/
react-build:
	cd $(REACT_DIR) && bun run build

# Install React dependencies
react-install:
	cd $(REACT_DIR) && bun install

# Lint React source
react-lint:
	cd $(REACT_DIR) && bun run lint

# ── Maintenance ───────────────────────────────────────────────────────

clean:
	rm -f boardgames games.db tailwindcss

# Print BGG Cookie header (reads ADMIN_* from .env if present, else environment)
bgg-login:
	$(GO) run ./cmd/bgg-login
