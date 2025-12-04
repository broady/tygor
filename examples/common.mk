# Common Makefile rules for tygor examples
# Include this in each example's Makefile: include ../common.mk

# Snippet tool
SNIPPET_TOOL := go run ../cmd/snippet

# Tygor CLI (use local version for development, can be overridden)
TYGOR ?= go run ../../cmd/tygor

# Default targets (can be overridden in example Makefile)
GO_FILES ?= main.go $(wildcard api/*.go)
TS_FILES ?= $(wildcard client/*.ts client/src/*.ts client/src/rpc/*.ts)
GEN_DIR ?= ./client/src/rpc
PREGEN ?=

.PHONY: all test test-quiet gen run clean check check-quiet fmt snippet-go snippet-ts snippets readme lint-readme help

# Default target
all: gen test

# Run tests
# Copy local @tygor packages to test unpublished changes (avoids symlink issues in Docker)
test:
	go build ./...
	@if [ -d client ] && [ -f client/package.json ]; then \
		(cd client && rm -rf node_modules && bun install) && \
		mkdir -p client/node_modules/@tygor/client client/node_modules/@tygor/testing && \
		cp ../../client/runtime.js ../../client/runtime.d.ts ../../client/package.json client/node_modules/@tygor/client/ && \
		cp ../testing/*.ts ../testing/*.json client/node_modules/@tygor/testing/ && \
		echo "Type-checking TypeScript..." && \
		(cd client && bun run typecheck) && \
		echo "Running integration tests..." && \
		(cd client && bun test); \
	fi

# Run tests quietly (output only on failure)
test-quiet:
	@output=$$(go build ./... 2>&1) || (echo "$$output"; exit 1)
	@if [ -d client ] && [ -f client/package.json ]; then \
		output=$$( \
			(cd client && rm -rf node_modules && bun install) && \
			mkdir -p client/node_modules/@tygor/client client/node_modules/@tygor/testing && \
			cp ../../client/runtime.js ../../client/runtime.d.ts ../../client/package.json client/node_modules/@tygor/client/ && \
			cp ../testing/*.ts ../testing/*.json client/node_modules/@tygor/testing/ && \
			(cd client && bun run typecheck) && \
			(cd client && bun test) 2>&1 \
		) || (echo "$$output"; exit 1); \
	fi

# Generate TypeScript types
gen:
	$(if $(PREGEN),$(PREGEN))
	@mkdir -p $(GEN_DIR)
	$(TYGOR) gen $(GEN_DIR)

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

# Check quietly (output only on failure)
check-quiet:
	$(if $(PREGEN),@output=$$($(PREGEN) 2>&1) || (echo "$$output"; exit 1))
	@mkdir -p $(GEN_DIR)
	@output=$$($(TYGOR) gen $(GEN_DIR) 2>&1) || (echo "$$output"; exit 1)
	@output=$$($(SNIPPET_TOOL) -inject README.md $(GO_FILES) $(TS_FILES) 2>&1) || (echo "$$output"; exit 1)
	@if [ -n "$$(git diff --name-only $(GEN_DIR) README.md 2>/dev/null)" ]; then \
		echo ""; \
		echo "ERROR: Generated files were out of sync with source code."; \
		echo "The files have been updated. Please commit the changes:"; \
		echo ""; \
		git --no-pager diff --stat $(GEN_DIR) README.md; \
		echo ""; \
		exit 1; \
	fi

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

# Lint README for large code blocks not covered by snippets
lint-readme:
	@$(SNIPPET_TOOL) -lint README.md

# Help
help:
	@echo "Available targets:"
	@echo "  make all         - Generate and test (default)"
	@echo "  make test        - Run tests"
	@echo "  make gen         - Generate TypeScript types"
	@echo "  make run         - Start the server"
	@echo "  make clean       - Remove generated files"
	@echo "  make check       - Verify generated files are up-to-date"
	@echo "  make fmt         - Format Go and TypeScript code"
	@echo "  make snippet-go  - Extract Go snippets as markdown"
	@echo "  make snippet-ts  - Extract TypeScript snippets as markdown"
	@echo "  make snippets    - Extract all snippets"
	@echo "  make readme      - Update README.md with code snippets"
	@echo "  make lint-readme - Check for unmanaged code blocks in README"
