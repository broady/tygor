# Root Makefile for tygor
# Tip: run with -j for parallel execution (e.g., make -j precommit)

# Snippet tool (runs from examples module)
SNIPPET_TOOL := cd examples && go run ./cmd/snippet

# Source files for snippet extraction (relative to repo root)
GO_FILES := $(wildcard *.go) $(wildcard middleware/*.go) $(wildcard tygorgen/*.go)
DOC_FILES := $(wildcard doc/examples/quickstart/*.go) $(wildcard doc/examples/quickstart/*.ts) \
             $(wildcard doc/examples/tygorgen/*.go) $(wildcard doc/examples/tygorgen/*.ts) \
             $(wildcard doc/examples/client/*.ts)

.PHONY: all test test-quiet lint lint-quiet check check-quiet readme lint-readme precommit fmt fmt-check ci-local typecheck-docs typecheck-vite-plugin release help

# Default target
all: test lint

# Run tests (GOWORK=off to test main module only; examples tested separately)
test:
	GOWORK=off go test ./...

# Run tests quietly (output only on failure)
test-quiet:
	@output=$$(GOWORK=off go test ./... 2>&1) || (echo "$$output"; exit 1)

# Run linters (GOWORK=off to lint main module only; examples linted separately)
lint:
	GOWORK=off go vet ./...
	GOWORK=off go run honnef.co/go/tools/cmd/staticcheck@v0.6.1 ./...

# Run linters quietly (output only on failure)
lint-quiet:
	@output=$$(GOWORK=off go vet ./... 2>&1) || (echo "$$output"; exit 1)
	@output=$$(GOWORK=off go run honnef.co/go/tools/cmd/staticcheck@v0.6.1 ./... 2>&1) || (echo "$$output"; exit 1)

# Format code
fmt:
	gofmt -w .

# Check formatting
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)

# Update READMEs with snippets
# Uses -root for scoped snippet names (e.g., doc/examples/quickstart:types)
readme:
	@cd examples && go run ./cmd/snippet -root .. -inject ../README.md -format mdx \
		$(addprefix ../,$(GO_FILES)) \
		$(addprefix ../,$(DOC_FILES)) \
		newsserver/main.go newsserver/api/types.go newsserver/client/index.ts \
		blog/main.go blog/api/types.go
	@cd examples && go run ./cmd/snippet -root .. -inject ../tygorgen/README.md -format mdx \
		$(addprefix ../,$(DOC_FILES))
	@cd examples && go run ./cmd/snippet -root .. -inject ../client/README.md -format mdx \
		$(addprefix ../,$(DOC_FILES))

# Lint all markdown files for large code blocks not covered by snippets
lint-readme:
	@echo "Linting markdown files for unmanaged code blocks..."
	@cd examples && find .. -name '*.md' -not -path '*/node_modules/*' -not -path '*/.git/*' | \
		xargs -I {} sh -c 'go run ./cmd/snippet -lint "{}" || exit 1'
	@echo "All markdown files passed lint."

# Check if generated files and READMEs are up-to-date
check: readme
	@if [ -n "$$(git diff --name-only README.md tygorgen/README.md client/README.md 2>/dev/null)" ]; then \
		echo ""; \
		echo "ERROR: README snippets were out of sync with source code."; \
		echo "The files have been updated. Please commit the changes:"; \
		echo ""; \
		git --no-pager diff --stat README.md tygorgen/README.md client/README.md; \
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
	@output=$$($(SNIPPET_TOOL) -root .. -inject ../tygorgen/README.md -format mdx \
		$(addprefix ../,$(DOC_FILES)) 2>&1) || (echo "$$output"; exit 1)
	@output=$$($(SNIPPET_TOOL) -root .. -inject ../client/README.md -format mdx \
		$(addprefix ../,$(DOC_FILES)) 2>&1) || (echo "$$output"; exit 1)
	@if [ -n "$$(git diff --name-only README.md tygorgen/README.md client/README.md 2>/dev/null)" ]; then \
		echo ""; \
		echo "ERROR: README snippets were out of sync with source code."; \
		echo "The files have been updated. Please commit the changes:"; \
		echo ""; \
		git --no-pager diff --stat README.md tygorgen/README.md client/README.md; \
		echo ""; \
		exit 1; \
	fi

# Type-check documentation examples
typecheck-docs:
	@cd doc/examples && npm install --silent
	@bun x typescript --noEmit --project doc/examples/tsconfig.json

# Type-check vite plugin
typecheck-vite-plugin:
	@cd vite-plugin && bun run --silent typecheck

# Precommit sub-targets (for parallel execution, all depend on fmt-check)
.PHONY: precommit-test precommit-lint precommit-check precommit-examples precommit-typecheck precommit-vite-plugin precommit-devserver
precommit-test: fmt-check ; @$(MAKE) --no-print-directory test-quiet
precommit-lint: fmt-check ; @$(MAKE) --no-print-directory lint-quiet
precommit-check: fmt-check ; @$(MAKE) --no-print-directory check-quiet
precommit-examples: fmt-check ; @$(MAKE) --no-print-directory -C examples precommit
precommit-typecheck: fmt-check ; @$(MAKE) --no-print-directory typecheck-docs
precommit-vite-plugin: fmt-check ; @$(MAKE) --no-print-directory typecheck-vite-plugin
precommit-devserver: fmt-check
	@go run ./cmd/tygor gen -p ./cmd/tygor/internal/dev ./vite-plugin/src/devserver
	@if [ -n "$$(git diff --name-only vite-plugin/src/devserver 2>/dev/null)" ]; then \
		echo ""; \
		echo "ERROR: Generated devserver types out of sync with cmd/tygor/internal/dev."; \
		echo "The files have been updated. Please commit the changes:"; \
		echo ""; \
		git --no-pager diff --stat vite-plugin/src/devserver; \
		echo ""; \
		exit 1; \
	fi

# Run all precommit checks in parallel (fmt-check runs first)
precommit: precommit-test precommit-lint precommit-check precommit-examples precommit-typecheck precommit-vite-plugin precommit-devserver
	@echo "All precommit checks passed."

# Run CI locally using act (https://github.com/nektos/act)
ci-local:
	go run github.com/nektos/act@latest --container-architecture linux/amd64

# Release packages (usage: make release TYPE=patch|minor|major)
release:
ifndef TYPE
	$(error TYPE is required. Usage: make release TYPE=patch)
endif
	./release.bash $(TYPE)

# Help
help:
	@echo "Available targets:"
	@echo "  make all            - Run tests and lint (default)"
	@echo "  make test           - Run tests"
	@echo "  make lint           - Run go vet and staticcheck"
	@echo "  make fmt            - Format Go code"
	@echo "  make readme         - Update READMEs with code snippets"
	@echo "  make lint-readme    - Check all .md files for unmanaged code blocks"
	@echo "  make typecheck-docs - Type-check documentation examples"
	@echo "  make check          - Verify READMEs are up-to-date"
	@echo "  make precommit      - Run all checks (test, lint, check, examples, typecheck)"
	@echo "  make ci-local       - Run GitHub Actions workflow locally via Docker (requires act)"
	@echo "  make release TYPE=  - Release packages (TYPE: patch|minor|major)"
