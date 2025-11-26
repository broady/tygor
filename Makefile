# Root Makefile for tygor

# Snippet tool (runs from examples module)
SNIPPET_TOOL := cd examples && go run ./cmd/snippet

# Source files for snippet extraction (relative to repo root)
GO_FILES := $(wildcard *.go) $(wildcard middleware/*.go) $(wildcard tygorgen/*.go)
DOC_FILES := $(wildcard doc/examples/quickstart/*.go)

.PHONY: all test lint check readme precommit fmt help

# Default target
all: test lint

# Run tests
test:
	go test ./...

# Run linters
lint:
	go vet ./...
	staticcheck ./...

# Format code
fmt:
	gofmt -w .

# Update README with snippets
# Uses -root for scoped snippet names (e.g., doc/examples/quickstart:types)
readme:
	@cd examples && go run ./cmd/snippet -root .. -inject ../README.md -format simple \
		$(addprefix ../,$(GO_FILES)) \
		$(addprefix ../,$(DOC_FILES)) \
		newsserver/main.go newsserver/api/types.go \
		blog/main.go blog/api/types.go

# Check if generated files and README are up-to-date
check: readme
	@if [ -n "$$(git diff --name-only README.md 2>/dev/null)" ]; then \
		echo ""; \
		echo "ERROR: README.md snippets were out of sync with source code."; \
		echo "The file has been updated. Please commit the changes:"; \
		echo ""; \
		git --no-pager diff --stat README.md; \
		echo ""; \
		exit 1; \
	fi
	@echo "All files are up-to-date."

# Run all precommit checks (including examples)
precommit: test lint check
	@$(MAKE) -C examples precommit
	@echo "All precommit checks passed."

# Help
help:
	@echo "Available targets:"
	@echo "  make all       - Run tests and lint (default)"
	@echo "  make test      - Run tests"
	@echo "  make lint      - Run go vet and staticcheck"
	@echo "  make fmt       - Format Go code"
	@echo "  make readme    - Update README.md with code snippets"
	@echo "  make check     - Verify README is up-to-date"
	@echo "  make precommit - Run all checks (test, lint, check, examples)"
