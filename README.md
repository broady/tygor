![tygor banner](doc/banner-generator/tygor-banner.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/broady/tygor.svg)](https://pkg.go.dev/github.com/broady/tygor)

# tygor

Type-safe backend for Go + TypeScript apps.

Write Go functions, call them from TypeScript with full type safety. No IDL required, but works with protobuf-generated types if you prefer schema-first design.

> [!IMPORTANT]
> tygor is pre-release and the API and protocol may still change.
> Pin the `@tygor/client` client package and `github.com/broady/tygor` Go module to the same version to avoid any breakage or incompatability.

## Features

- **End-to-end type safety** - Go structs become TypeScript types automatically
- **Standard HTTP/JSON** - Debuggable, cacheable (see `CacheControl`), works with existing tools
- **Flexible** - No IDL required, but works with protobuf-generated types if you prefer schema-first
- **Tiny client footprint** - Proxy-based client with <3KB bundle impact
- **Go-native** - Idiomatic handlers, works with the standard `net/http` ecosystem
- **Robust and pluggable** - Structured errors and logs, interceptors and telemetry hooks, validation

## Philosophy

**tygor is for teams building fullstack Go + TypeScript applications.**

If your Go backend and TypeScript frontend live together (especially in a monorepo), tygor gives you end-to-end type safety without requiring an IDL. You write normal Go functions, and tygor generates TypeScript types that match your actual implementation. If you already use protobuf or prefer schema-first design, tygor works seamlessly with generated Go types.

### Who is this for?

**Use tygor if you:**
- Build fullstack apps with Go + TypeScript, especially in a monorepo
- Want type safety without the overhead of maintaining separate IDL files
- Value iteration speed and developer ergonomics
- Want to write idiomatic Go handlers and get TypeScript types automatically
- Are okay with incrementally improving types as your domain evolves

**Don't use tygor if you:**
- Need a public API with strict versioning guarantees (tygor can help here, but consider adding OpenAPI generation)
- Require multi-language client support today (OpenAPI generation is planned)

### The tradeoff

tygor optimizes for the common case: a team iterating quickly on a fullstack app where the backend and frontend are tightly coupled anyway. You can always add handwritten TypeScript definitions to improve type safety for your specific domain. It's still nicer than being forced into the constraints of an IDL.

In the future, tygor may generate OpenAPI specs to enable client generation in other languages, giving you the best of both worlds: ergonomic Go + TypeScript for your core app, with optional compatibility for other ecosystems.

If you need backward compatibility guarantees, treat your Go struct definitions like you would a protobuf schema. Renaming a field is a breaking change! OpenAPI generation could help flag wire-level breaking changes in your API.

## Installation

### Go (server-side)

```bash
go get github.com/broady/tygor
```

### TypeScript/JavaScript (client-side)

```bash
npm install @tygor/client
```

Or with your preferred package manager:
```bash
pnpm add @tygor/client
bun add @tygor/client
yarn add @tygor/client
```

## Quick Start

### 1. Define your types

<!-- [snippet:doc/examples/quickstart:types] -->
```go
type News struct {
	ID        int32      `json:"id"`
	Title     string     `json:"title"`
	Body      *string    `json:"body"`
	CreatedAt *time.Time `json:"created_at"`
}

type ListNewsParams struct {
	Limit  *int32 `json:"limit"`
	Offset *int32 `json:"offset"`
}

type CreateNewsParams struct {
	Title string  `json:"title" validate:"required,min=3"`
	Body  *string `json:"body"`
}

```
<!-- [/snippet:doc/examples/quickstart:types] -->

### 2. Implement handlers

<!-- [snippet:doc/examples/quickstart:handlers] -->
```go
func ListNews(ctx context.Context, req *ListNewsParams) ([]*News, error) {
	// Your implementation
	return nil, nil
}

func CreateNews(ctx context.Context, req *CreateNewsParams) (*News, error) {
	// Your implementation
	return nil, nil
}

```
<!-- [/snippet:doc/examples/quickstart:handlers] -->

### 3. Register services

<!-- [snippet:doc/examples/quickstart:registration] -->
```go
app := tygor.NewApp()

news := app.Service("News")
news.Register("List", tygor.Query(ListNews))
news.Register("Create", tygor.Exec(CreateNews))

http.ListenAndServe(":8080", app.Handler())
```
<!-- [/snippet:doc/examples/quickstart:registration] -->

### 4. Generate TypeScript types

<!-- [snippet:doc/examples/quickstart:generation] -->
```go
if err := tygorgen.Generate(app, &tygorgen.Config{
	OutDir: "./client/src/rpc",
}); err != nil {
	log.Fatal(err)
}
```
<!-- [/snippet:doc/examples/quickstart:generation] -->

This generates TypeScript types and a manifest describing all available API methods.

### 5. Use the TypeScript client

First, install the client runtime:

```bash
npm install @tygor/client
```

The generated client provides a clean, idiomatic API with full type safety:

<!-- [snippet:examples/newsserver:client-usage] -->
<!-- [/snippet:examples/newsserver:client-usage] -->

The client uses JavaScript Proxies to provide method access without code generation bloat. Your bundle only includes the types and a small runtime, regardless of how many API methods you have.

See [examples/newsserver/client/src/rpc/manifest.ts](examples/newsserver/client/src/rpc/manifest.ts) for an example of generated output.

## Request Handling

### GET Requests

For GET requests, parameters are decoded from query strings:

<!-- [snippet:examples/newsserver:list-params] -->
<!-- [/snippet:examples/newsserver:list-params] -->

Query: `/News/List?limit=10&offset=20`

### POST Requests

For POST requests (the default), the body is decoded as JSON:

<!-- [snippet:examples/newsserver:create-params] -->
<!-- [/snippet:examples/newsserver:create-params] -->

## Error Handling

Use structured error codes for consistent error responses:

<!-- [snippet:examples/newsserver:error-handling] -->
```go
func CreateNews(ctx context.Context, req *api.CreateNewsParams) (*api.News, error) {
	if req.Title == "error" {
		return nil, tygor.NewError(tygor.CodeInvalidArgument, "simulated error")
	}
	now := time.Now()
	return &api.News{
		ID:        123,
		Title:     req.Title,
		Body:      req.Body,
		Status:    api.NewsStatusDraft,
		CreatedAt: &now,
	}, nil
}

```
<!-- [/snippet:examples/newsserver:error-handling] -->

Available error codes and their HTTP status code mapping (not exhaustive):
- `CodeOK` (200)
- `CodeInvalidArgument` (400)
- `CodeUnauthenticated` (401)
- `CodePermissionDenied` (403)
- `CodeNotFound` (404)
- `CodeAlreadyExists` (409)
- `CodeResourceExhausted` (429)
- `CodeInternal` (500)
- `CodeUnavailable` (503)

### Custom Error Transformers

Map application errors to API errors:

<!-- [snippet:examples/newsserver:error-transformer] -->
```go
app := tygor.NewApp().
	WithErrorTransformer(func(err error) *tygor.Error {
		if err.Error() == "database connection failed" {
			return tygor.NewError(tygor.CodeUnavailable, "service unavailable")
		}
		return nil
	})

```
<!-- [/snippet:examples/newsserver:error-transformer] -->

### Masking Internal Errors

Prevent sensitive error details from leaking in production:

<!-- snippet-ignore -->
```go
app := tygor.NewApp().WithMaskInternalErrors()
```

## Interceptors

Interceptors provide cross-cutting concerns at different levels.

### Global Interceptors

Applied to all handlers:

<!-- [snippet:examples/newsserver:global-interceptor] -->
```go
app = app.WithUnaryInterceptor(middleware.LoggingInterceptor(logger))

```
<!-- [/snippet:examples/newsserver:global-interceptor] -->

### Service Interceptors

Applied to all handlers in a service:

<!-- snippet-ignore -->
```go
news := app.Service("News").
    WithUnaryInterceptor(authInterceptor)
```

### Handler Interceptors

Applied to specific handlers:

<!-- [snippet:examples/newsserver:handler-interceptor] -->
```go
news.Register("Create", tygor.Exec(CreateNews).
	WithUnaryInterceptor(func(ctx tygor.Context, req any, handler tygor.HandlerFunc) (any, error) {
		ctx.HTTPWriter().Header().Set("X-Created-By", "tygor")
		return handler(ctx, req)
	}))

```
<!-- [/snippet:examples/newsserver:handler-interceptor] -->

## Middleware

HTTP middleware wraps the entire app:

<!-- [snippet:examples/newsserver:middleware] -->
```go
app = app.WithMiddleware(middleware.CORS(middleware.DefaultCORSConfig()))

```
<!-- [/snippet:examples/newsserver:middleware] -->

## Validation

### POST Requests

POST request bodies are validated using struct tags via the `validator/v10` package:

<!-- snippet-ignore -->
```go
type CreateParams struct {
    Title string `json:"title" validate:"required,min=3,max=100"`
    Email string `json:"email" validate:"required,email"`
}
```

### GET Requests

GET request query parameters are decoded using `gorilla/schema` and then validated with `validator/v10`:

<!-- snippet-ignore -->
```go
type ListParams struct {
    Limit  int    `schema:"limit" validate:"min=0,max=100"`
    Offset int    `schema:"offset" validate:"min=0"`
    Status string `schema:"status" validate:"omitempty,oneof=draft published"`
}
```

Query: `/News/List?limit=10&offset=0&status=published`

**Note:** `gorilla/schema` uses case-insensitive matching for query parameter names. Without a `schema` tag, the field name is used (e.g., field `Limit` matches query param `limit`, `Limit`, or `LIMIT`). For clarity, always use explicit `schema` tags.

## Caching

Set cache headers on GET handlers using `CacheControl`:

<!-- [snippet:examples/newsserver:cache-control] -->
```go
news.Register("List", tygor.Query(ListNews).
	CacheControl(tygor.CacheConfig{
		MaxAge: 1 * time.Minute,
		Public: true,
	}))

```
<!-- [/snippet:examples/newsserver:cache-control] -->

Common patterns:

<!-- snippet-ignore -->
```go
// Browser-only caching (private)
CacheControl(tygor.CacheConfig{MaxAge: 5 * time.Minute})

// CDN + browser caching (public)
CacheControl(tygor.CacheConfig{MaxAge: 5 * time.Minute, Public: true})

// Stale-while-revalidate for smooth updates
CacheControl(tygor.CacheConfig{
    MaxAge:               1 * time.Minute,
    StaleWhileRevalidate: 5 * time.Minute,
    Public:               true,
})
```

## Context Access

Access request metadata and HTTP primitives via `tygor.FromContext`:

<!-- snippet-ignore -->
```go
func Handler(ctx context.Context, req *Request) (*Response, error) {
    tc, ok := tygor.FromContext(ctx)
    if ok {
        // Get service and method name
        service, method := tc.Service(), tc.Method()

        // Access HTTP request headers
        token := tc.HTTPRequest().Header.Get("Authorization")

        // Set custom response headers
        tc.HTTPWriter().Header().Set("X-Custom", "value")
    }

    return &Response{}, nil
}
```

In interceptors, you receive `tygor.Context` directly:

<!-- snippet-ignore -->
```go
func loggingInterceptor(ctx tygor.Context, req any, handler tygor.HandlerFunc) (any, error) {
    log.Printf("calling %s", ctx.EndpointID())
    return handler(ctx, req)
}
```

## Type Mappings

Customize TypeScript type generation for third-party types:

<!-- snippet-ignore -->
```go
tygorgen.Generate(app, &tygorgen.Config{
    OutDir: "./client/src/rpc",
    TypeMappings: map[string]string{
        "github.com/jackc/pgtype.Timestamptz": "string | null",
        "github.com/jackc/pgtype.UUID":        "string",
    },
})
```

## License

MIT

Tiger image by Yan Liu, licensed under CC-BY (with a few modifications).
