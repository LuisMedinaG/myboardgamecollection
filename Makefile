.PHONY: build run dev dev-go dev-all clean \
        bgg-login test test-v cover cover-html vet check \
        react-dev react-build react-install react-lint react-test

GOCACHE ?= /tmp/go-build-cache
GOENV = env GOCACHE=$(GOCACHE)
GO = $(GOENV) go

REACT_DIR = react-app

# ── Go ───────────────────────────────────────────────────────────────

build:
	$(GO) build -o boardgames .

run: build
	./boardgames

dev:
	$(GO) run .

dev-go: dev

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

check: build test vet

# ── Combined ─────────────────────────────────────────────────────────

dev-all:
	@trap 'kill 0' INT TERM; \
	$(GO) run . & \
	cd $(REACT_DIR) && bun dev; \
	wait

# ── React frontend ────────────────────────────────────────────────────

react-dev:
	cd $(REACT_DIR) && bun dev

react-build:
	cd $(REACT_DIR) && bun run build

react-install:
	cd $(REACT_DIR) && bun install

react-lint:
	cd $(REACT_DIR) && bun run lint

react-test:
	cd $(REACT_DIR) && bun run playwright test

# ── Maintenance ───────────────────────────────────────────────────────

clean:
	rm -f boardgames games.db

bgg-login:
	$(GO) run ./cmd/bgg-login
