<!-- snippet-lint-disable -->

# Contributing to tygor

Thanks for your interest in contributing to tygor!

## Development Setup

### Prerequisites

- Go 1.21 or later
- Docker (for local CI testing)
- Node.js 18+ or Bun (for running TypeScript examples and tests)
- npm (comes with Node.js)

### Initial Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/broady/tygor.git
   cd tygor
   ```

2. Install dependencies (sets up workspace):
   ```bash
   npm install
   ```

   This creates symlinks so the examples use your local `@tygor/client` package during development.

3. Run Go tests:
   ```bash
   go test ./...
   ```

4. Run TypeScript runtime tests:
   ```bash
   cd client
   bun test
   ```

## Development Workflow

### Make Targets

The root Makefile provides all the commands you need:

```bash
make precommit   # Run ALL checks (format, test, lint, examples) - run before committing
make ci-local    # Run GitHub Actions workflow locally via Docker
make test        # Run Go tests
make lint        # Run go vet and staticcheck
make fmt         # Format Go code
make fmt-check   # Check formatting without modifying files
make readme      # Update README.md with code snippets
make check       # Verify README snippets are up-to-date
```

### Before Committing

Always run `make precommit` before committing. This runs:
1. Format check (`gofmt`)
2. Go tests
3. Linters (`go vet`, `staticcheck`)
4. README snippet check
5. All example checks

### Testing CI Locally

To test the GitHub Actions workflow locally before pushing:

```bash
make ci-local
```

This uses [act](https://github.com/nektos/act) to run the CI workflow in Docker, exactly as it would run on GitHub.

## Project Structure

```
tygor/
├── .github/workflows/   # CI workflow (runs make precommit)
├── client/              # @tygor/client npm package
│   ├── runtime.ts       # TypeScript client runtime
│   ├── runtime.test.ts  # Runtime tests
│   └── package.json     # Published to npm as @tygor/client
├── examples/            # Example applications (separate Go module)
│   ├── go.mod           # Separate module for heavier dependencies
│   ├── newsserver/      # Simple CRUD example
│   ├── blog/            # Complex auth/authz example
│   └── protobuf/        # Protobuf types example
├── middleware/          # Built-in middleware (CORS, logging)
├── tygorgen/            # Code generator
└── *.go                 # Core framework files
```

### Multi-Module Repository

The `examples/` directory is a separate Go module with its own `go.mod`. This allows examples to have heavier dependencies (like protobuf) without polluting the main module's dependency tree.

```bash
# Main module tests
go test ./...

# Build examples (from examples/)
cd examples && go build ./...
```

## Monorepo Setup

This repo uses npm workspaces to manage the TypeScript client and examples:

- **`/client`** - The `@tygor/client` package published to npm
- **`/examples/*/client`** - Example clients that depend on `@tygor/client`

During development, npm creates symlinks so examples automatically use your local client code. When you make changes to `client/runtime.ts`:

1. Rebuild the client: `cd client && npm run build`
2. Examples will use the updated version via symlink

## Making Changes

### Go Code

1. Make your changes
2. Run tests: `go test ./...`
3. Run examples to verify: `cd examples/newsserver && go run main.go`
4. Check test coverage: `go test -cover ./...`

### TypeScript Client

1. Edit `client/runtime.ts`
2. Run tests: `cd client && bun test`
3. Build: `npm run build`
4. Test with examples:
   ```bash
   cd ../examples/newsserver
   go run main.go &  # Start server
   cd client && bun run index.ts  # Run client
   ```

### Code Generator

The code generator lives in `tygorgen/`. Changes here affect:
- Generated TypeScript types (`types.ts`)
- Generated manifest (`manifest.ts`)

Test by running examples and regenerating their types.

## Testing

### Go Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./middleware
```

### TypeScript Tests

```bash
cd client
bun test          # Run tests
bun test --watch  # Watch mode
```

## Publishing the Client

**Note**: Only maintainers can publish. If you're contributing changes to the client, we'll publish after merging.

To publish a new version of `@tygor/client`:

1. Update version in `client/package.json`
2. Rebuild: `cd client && npm run build`
3. Publish: `npm publish --access public`
4. Commit the version bump
5. Tag the release: `git tag client/v0.1.1 && git push --tags`

## Examples

Examples are in a separate Go module (`examples/go.mod`) with standardized make targets.

### Make Targets

Each example supports:
```bash
make run        # Start server
make gen        # Generate TypeScript
make test       # Run tests
make fmt        # Format code
make check      # Verify generated files (for CI)
make snippets   # Extract code snippets as markdown
```

### Snippet Markers

Examples contain marked code regions for documentation extraction. Add markers around key code:

```go
// [snippet:handler-example]
func MyHandler(ctx context.Context, req *api.Request) (*api.Response, error) {
    // This code can be extracted to docs
}
// [/snippet:handler-example]
```

Extract with `make snippet-go` or `make snippet-ts`.

### Guidelines

1. Keep examples simple and focused on demonstrating specific features
2. Add snippet markers around documentation-worthy code
3. Run `make check` before submitting to ensure generated files are current
4. Test that examples build and run before submitting

## Pull Request Process

1. Create a feature branch from `main`
2. Make your changes with clear, focused commits
3. Add tests for new functionality
4. Run `make precommit` to ensure all checks pass
5. Optionally run `make ci-local` to test the full CI workflow
6. Update documentation if needed
7. Submit PR with a clear description of changes

CI will automatically run `make precommit` on your PR.

## Code Style

### Go
- Run `make fmt` to format code (CI enforces `gofmt`)
- Keep handlers simple and focused
- Document exported types and functions

### TypeScript
- Use TypeScript strict mode
- Prefer functional style
- Keep the runtime small and focused

## Questions?

Open an issue or discussion on GitHub!
