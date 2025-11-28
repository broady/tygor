# Tygor Examples

This directory contains example applications demonstrating tygor features. Each example is a complete, runnable application.

## Examples

| Example | Description |
|---------|-------------|
| [newsserver](./newsserver) | Simple CRUD API with branded types, enums, caching |
| [blog](./blog) | Multi-service app with authentication and authorization |
| [protobuf](./protobuf) | Using protobuf-generated types with tygor |
| [reflection](./reflection) | Generic type instantiation with reflection provider |
| [zod](./zod) | Zod schema generation from Go validate tags |

## Running Examples

Each example supports standardized make targets:

```bash
cd newsserver

make run        # Start the server on :8080
make gen        # Generate TypeScript types
make test       # Run tests
make fmt        # Format code
make clean      # Remove generated files
make check      # Verify generated files are up-to-date (for CI)
```

## Snippet System

Examples contain marked code regions that can be extracted as markdown snippets. Snippets are embedded in each example's README and kept up-to-date automatically.

### Updating READMEs

```bash
make readme       # Update README.md with current code snippets
make precommit    # Run all checks including readme updates
```

### Extracting to Stdout

```bash
make snippet-go   # Extract Go snippets
make snippet-ts   # Extract TypeScript snippets
make snippets     # Extract all snippets
```

### Output Format

Snippets are output in MDX-compatible format with title metadata:

````markdown
```go title="main.go:22-45"
func ListNews(ctx context.Context, req *api.ListNewsParams) ([]*api.News, error) {
    // ...
}
```
````

### Adding Snippet Markers

Mark regions in source files using comment markers:

**Go/TypeScript/Proto (// comments):**
```go
// [snippet:my-example]

func Example() {
    // This code will be extracted
}
// [/snippet:my-example]
```

**Note:** Add a blank line after `[snippet:...]` to prevent the marker from being included in doc comments or generated code.

**Shell/YAML (# comments):**
```yaml
# [snippet:config-example]
key: value
# [/snippet:config-example]
```

### Snippet Tool

The extraction tool is at `cmd/snippet/`. Direct usage:

```bash
go run ./cmd/snippet -help

# Extract all snippets from files
go run ./cmd/snippet main.go api/types.go

# Extract specific snippet
go run ./cmd/snippet -name handlers main.go

# Inject snippets into README
go run ./cmd/snippet -inject README.md main.go api/types.go
```

### README Markers

To embed snippets in a README, use HTML comment markers:

```markdown
<!-- [snippet:handlers] -->
<!-- [/snippet:handlers] -->
```

The content between markers is replaced when running `make readme`.

### Whole-File Snippets

To embed an entire file without adding markers to the source:

```markdown
<!-- [snippet-file:client/src/rpc/types.ts] -->
```

This reads the file and injects it as a code block. Useful for generated files.

### Lint Mode

Check for large code blocks (≥5 lines) not covered by snippets:

```bash
make lint-readme                    # Check current example
go run ./cmd/snippet -lint README.md  # Direct usage
```

To ignore intentional code blocks (like simplified examples with comments):

```markdown
<!-- snippet-ignore -->
```typescript
// This block will not trigger lint warnings
```
```

To disable lint for an entire file (design docs, specs, proposals):

```markdown
<!-- snippet-lint-disable -->
```

Place at the top of files that contain standalone conceptual examples.

## Module Structure

The examples directory is a separate Go module (`github.com/broady/tygor/examples`) to allow heavier dependencies (like protobuf) without affecting the main tygor module.

```
examples/
├── go.mod              # Separate module with replace directive
├── common.mk           # Shared make rules
├── cmd/snippet/        # Snippet extraction tool
├── newsserver/
│   ├── Makefile        # includes ../common.mk
│   ├── main.go
│   ├── api/types.go
│   └── client/
├── blog/
└── protobuf/
```

## Adding a New Example

1. Create a new directory: `mkdir myexample`
2. Create `Makefile` that includes common rules:
   ```makefile
   GO_FILES := main.go api/types.go
   TS_FILES := $(wildcard client/src/rpc/*.ts)

   include ../common.mk
   ```
3. Add snippet markers to key code sections
4. Run `make gen` to generate TypeScript
5. Update this README with the new example

## Conventions

### File Structure

Each example follows this pattern:
```
example/
├── Makefile           # include ../common.mk
├── main.go            # Server setup, handlers, registration
├── api/
│   └── types.go       # Request/response types
├── client/
│   ├── index.ts       # Client usage example (optional)
│   └── src/rpc/       # Generated TypeScript (via make gen)
└── README.md
```

### Snippet Naming

Use descriptive, kebab-case names for snippets:
- `handlers` - Handler function definitions
- `app-setup` - App initialization and middleware
- `service-registration` - Service and route registration
- `auth-interceptor` - Authentication middleware
- `client-setup` - TypeScript client configuration
- `client-calls` - Type-safe RPC calls
- `enum-type`, `request-types`, `response-type` - Type definitions

### What to Mark as Snippets

Prioritize documentation-worthy patterns:
- Handler function signatures
- App initialization with middleware/interceptors
- Service registration with Query/Exec
- Type definitions with json/schema/validate tags
- Authentication/authorization patterns
- Client setup and usage
