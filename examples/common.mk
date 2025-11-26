# Common Makefile rules for tygor examples
# Include this in each example's Makefile: include ../common.mk

# Snippet tool
SNIPPET_TOOL := go run ../cmd/snippet

# Default targets (can be overridden in example Makefile)
GO_FILES ?= main.go $(wildcard api/*.go)
TS_FILES ?= $(wildcard client/*.ts client/src/*.ts client/src/rpc/*.ts)
GEN_DIR ?= ./client/src/rpc

.PHONY: all test gen run clean check fmt snippet-go snippet-ts snippets readme help

# Default target
all: gen test

# Run tests
# Copy @tygor/testing to avoid symlink issues in Docker
test:
	go build ./...
	@if [ -d client ] && [ -f client/package.json ]; then \
		TYGOR_ROOT=$$(cd ../.. && pwd) && \
		(cd client && rm -rf node_modules && bun install) && \
		mkdir -p client/node_modules/@tygor/testing && \
		cp "$$TYGOR_ROOT/examples/testing/"*.ts "$$TYGOR_ROOT/examples/testing/"*.json client/node_modules/@tygor/testing/ && \
		echo "Type-checking TypeScript..." && \
		(cd client && bun run typecheck) && \
		echo "Running integration tests..." && \
		(cd client && bun test); \
	fi

# Generate TypeScript types
gen:
	@mkdir -p $(GEN_DIR)
	go run . -gen -out $(GEN_DIR)

# Run the server
run:
	go run .

# Clean generated files
clean:
	rm -rf $(GEN_DIR)

# Check if generated files are up-to-date (only unstaged changes)
check: gen readme
	@if [ -n "$$(git diff --name-only $(GEN_DIR) README.md 2>/dev/null)" ]; then \
		echo ""; \
		echo "ERROR: Generated files were out of sync with source code."; \
		echo "The files have been updated. Please commit the changes:"; \
		echo ""; \
		git --no-pager diff --stat $(GEN_DIR) README.md; \
		echo ""; \
		exit 1; \
	fi
	@echo "Generated files are up-to-date."

# Format code
fmt:
	gofmt -w $(GO_FILES)
	@if [ -d client ] && command -v npx >/dev/null 2>&1; then \
		npx prettier --write "client/**/*.ts" 2>/dev/null || true; \
	fi

# Extract Go snippets
snippet-go:
	@$(SNIPPET_TOOL) $(GO_FILES)

# Extract TypeScript snippets
snippet-ts:
	@if [ -n "$(TS_FILES)" ]; then \
		$(SNIPPET_TOOL) $(TS_FILES); \
	fi

# Extract all snippets
snippets: snippet-go snippet-ts

# Update README with snippets
readme:
	@$(SNIPPET_TOOL) -inject README.md $(GO_FILES) $(TS_FILES)

# Help
help:
	@echo "Available targets:"
	@echo "  make all        - Generate and test (default)"
	@echo "  make test       - Run tests"
	@echo "  make gen        - Generate TypeScript types"
	@echo "  make run        - Start the server"
	@echo "  make clean      - Remove generated files"
	@echo "  make check      - Verify generated files are up-to-date"
	@echo "  make fmt        - Format Go and TypeScript code"
	@echo "  make snippet-go - Extract Go snippets as markdown"
	@echo "  make snippet-ts - Extract TypeScript snippets as markdown"
	@echo "  make snippets   - Extract all snippets"
	@echo "  make readme     - Update README.md with code snippets"
