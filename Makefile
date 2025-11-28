# Root Makefile for tygor
# Tip: run with -j for parallel execution (e.g., make -j precommit)

# Snippet tool (runs from examples module)
SNIPPET_TOOL := cd examples && go run ./cmd/snippet

# Source files for snippet extraction (relative to repo root)
GO_FILES := $(wildcard *.go) $(wildcard middleware/*.go) $(wildcard tygorgen/*.go)
DOC_FILES := $(wildcard doc/examples/quickstart/*.go)

.PHONY: all test test-quiet lint lint-quiet check check-quiet readme lint-readme precommit fmt fmt-check ci-local help

# Default target
all: test lint

# Run tests
test:
	go test ./...

# Run tests quietly (output only on failure)
test-quiet:
	@output=$$(go test ./... 2>&1) || (echo "$$output"; exit 1)

# Run linters
lint:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@v0.6.1 ./...

# Run linters quietly (output only on failure)
lint-quiet:
	@output=$$(go vet ./... 2>&1) || (echo "$$output"; exit 1)
	@output=$$(go run honnef.co/go/tools/cmd/staticcheck@v0.6.1 ./... 2>&1) || (echo "$$output"; exit 1)

# Format code
fmt:
	gofmt -w .

# Check formatting
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)

# Update README with snippets
# Uses -root for scoped snippet names (e.g., doc/examples/quickstart:types)
readme:
	@cd examples && go run ./cmd/snippet -root .. -inject ../README.md -format mdx \
		$(addprefix ../,$(GO_FILES)) \
		$(addprefix ../,$(DOC_FILES)) \
		newsserver/main.go newsserver/api/types.go newsserver/client/index.ts \
		blog/main.go blog/api/types.go

# Lint all markdown files for large code blocks not covered by snippets
lint-readme:
	@echo "Linting markdown files for unmanaged code blocks..."
	@cd examples && find .. -name '*.md' -not -path '*/node_modules/*' -not -path '*/.git/*' | \
		xargs -I {} sh -c 'go run ./cmd/snippet -lint "{}" || exit 1'
	@echo "All markdown files passed lint."

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

# Check quietly (for precommit)
check-quiet:
	@output=$$($(SNIPPET_TOOL) -root .. -inject ../README.md -format mdx \
		$(addprefix ../,$(GO_FILES)) \
		$(addprefix ../,$(DOC_FILES)) \
		newsserver/main.go newsserver/api/types.go newsserver/client/index.ts \
		blog/main.go blog/api/types.go 2>&1) || (echo "$$output"; exit 1)
	@if [ -n "$$(git diff --name-only README.md 2>/dev/null)" ]; then \
		echo ""; \
		echo "ERROR: README.md snippets were out of sync with source code."; \
		echo "The file has been updated. Please commit the changes:"; \
		echo ""; \
		git --no-pager diff --stat README.md; \
		echo ""; \
		exit 1; \
	fi

# Precommit sub-targets (for parallel execution, all depend on fmt-check)
.PHONY: precommit-test precommit-lint precommit-check precommit-examples
precommit-test: fmt-check ; @$(MAKE) --no-print-directory test-quiet
precommit-lint: fmt-check ; @$(MAKE) --no-print-directory lint-quiet
precommit-check: fmt-check ; @$(MAKE) --no-print-directory check-quiet
precommit-examples: fmt-check ; @$(MAKE) --no-print-directory -C examples precommit

# Run all precommit checks in parallel (fmt-check runs first)
precommit: precommit-test precommit-lint precommit-check precommit-examples
	@echo "All precommit checks passed."

# Run CI locally using act (https://github.com/nektos/act)
ci-local:
	go run github.com/nektos/act@latest --container-architecture linux/amd64

# Help
help:
	@echo "Available targets:"
	@echo "  make all         - Run tests and lint (default)"
	@echo "  make test        - Run tests"
	@echo "  make lint        - Run go vet and staticcheck"
	@echo "  make fmt         - Format Go code"
	@echo "  make readme      - Update README.md with code snippets"
	@echo "  make lint-readme - Check all .md files for unmanaged code blocks"
	@echo "  make check       - Verify README is up-to-date"
	@echo "  make precommit   - Run all checks (test, lint, check, examples)"
	@echo "  make ci-local    - Run GitHub Actions workflow locally via Docker (requires act)"
