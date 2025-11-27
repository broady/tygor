# Root Makefile for tygor
MAKEFLAGS += -j

# Snippet tool (runs from examples module)
SNIPPET_TOOL := cd examples && go run ./cmd/snippet

# Source files for snippet extraction (relative to repo root)
GO_FILES := $(wildcard *.go) $(wildcard middleware/*.go) $(wildcard tygorgen/*.go)
DOC_FILES := $(wildcard doc/examples/quickstart/*.go)

.PHONY: all test lint check readme lint-readme precommit fmt fmt-check ci-local help

# Default target
all: test lint

# Run tests
test:
	go test ./...

# Run linters
lint:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@v0.6.1 ./...

# Format code
fmt:
	gofmt -w .

# Check formatting
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)

# Update README with snippets
# Uses -root for scoped snippet names (e.g., doc/examples/quickstart:types)
readme:
	@cd examples && go run ./cmd/snippet -root .. -inject ../README.md -format simple \
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

# Precommit sub-targets (for parallel execution, all depend on fmt-check)
.PHONY: precommit-test precommit-lint precommit-check precommit-examples
precommit-test: fmt-check ; @$(MAKE) test
precommit-lint: fmt-check ; @$(MAKE) lint
precommit-check: fmt-check ; @$(MAKE) check
precommit-examples: fmt-check ; @$(MAKE) -C examples precommit

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
