# Tygor Examples

This directory contains example applications demonstrating tygor features. Each example is a complete, runnable application.

## Examples

| Example | Description |
|---------|-------------|
| [newsserver](./newsserver) | Simple CRUD API with branded types, enums, caching |
| [blog](./blog) | Multi-service app with authentication and authorization |
| [protobuf](./protobuf) | Using protobuf-generated types with tygor |

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

Examples contain marked code regions that can be extracted as markdown snippets for documentation.

### Extracting Snippets

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

# Output to file
go run ./cmd/snippet -out snippets.md main.go
```

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
