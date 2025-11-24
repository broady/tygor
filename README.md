# tygor

Type-safe RPC framework for Go with automatic TypeScript client generation.

## Features

- **Type-safe handlers** using Go generics
- **Automatic TypeScript generation** from Go types
- **Lightweight TypeScript client** using proxies for minimal bundle size
- **Zero reflection at runtime** for handler execution
- **Request validation** with struct tags
- **Flexible error handling** with structured error codes
- **Middleware and interceptors** at global, service, and handler levels
- **Built-in support for GET and POST** methods with appropriate decoding

## Installation

```bash
go get github.com/broady/tygor
```

## Quick Start

### 1. Define your types

```go
package api

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

### 2. Implement handlers

```go
func ListNews(ctx context.Context, req *ListNewsParams) ([]*News, error) {
    // Your implementation
    return news, nil
}

func CreateNews(ctx context.Context, req *CreateNewsParams) (*News, error) {
    // Your implementation
    return &news, nil
}
```

### 3. Register services

```go
reg := tygor.NewRegistry()

news := reg.Service("News")
news.Register("List", tygor.NewHandler(ListNews).Method("GET"))
news.Register("Create", tygor.NewHandler(CreateNews).Method("POST"))

http.ListenAndServe(":8080", reg.Handler())
```

### 4. Generate TypeScript types

```go
if err := reg.Generate(&tygor.GenConfig{
    OutDir: "./client/src/rpc",
}); err != nil {
    log.Fatal(err)
}
```

This generates TypeScript types and a manifest describing all available RPC methods.

### 5. Use the TypeScript client

The generated client provides a clean, idiomatic API with full type safety:

```typescript
import { createClient } from './rpc/client';
import type { RPCManifest } from './rpc/manifest';

const client = createClient<RPCManifest>({
  baseURL: 'http://localhost:8080'
});

// Type-safe calls with autocomplete
const news = await client.News.List({ limit: 10, offset: 0 });
// news: News[]

const created = await client.News.Create({
  title: "Breaking News",
  body: "Important update"
});
// created: News

// Errors are properly typed
try {
  await client.News.Create({ title: "x" }); // Validation error
} catch (err) {
  console.error(err.code, err.message); // 400, "validation failed"
}
```

The client uses JavaScript Proxies to provide method access without code generation bloat—your bundle only includes the types and a small runtime, regardless of how many RPC methods you have.

Example `manifest.ts`:

```typescript
export interface RPCManifest {
  "News.List": {
    req: types.ListNewsParams;
    res: types.News[];
    method: "GET";
    path: "/News/List";
  };
  "News.Create": {
    req: types.CreateNewsParams;
    res: types.News;
    method: "POST";
    path: "/News/Create";
  };
}

export const RPCMetadata = {
  "News.List": { method: "GET", path: "/News/List" },
  "News.Create": { method: "POST", path: "/News/Create" },
} as const;
```

## Request Handling

### GET Requests

For GET requests, parameters are decoded from query strings:

```go
type ListParams struct {
    Limit  *int32 `json:"limit"`
    Offset *int32 `json:"offset"`
}

tygor.NewHandler(List).Method("GET")
```

Query: `/News/List?limit=10&offset=20`

### POST Requests

For POST requests, the body is decoded as JSON:

```go
type CreateParams struct {
    Title string `json:"title" validate:"required"`
}

tygor.NewHandler(Create).Method("POST")
```

## Error Handling

Use structured error codes for consistent error responses:

```go
func CreateNews(ctx context.Context, req *CreateNewsParams) (*News, error) {
    if req.Title == "invalid" {
        return nil, tygor.NewError(tygor.CodeInvalidArgument, "invalid title")
    }
    return &news, nil
}
```

Available error codes:
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

Map application errors to RPC errors:

```go
reg := tygor.NewRegistry().
    WithErrorTransformer(func(err error) *tygor.Error {
        if errors.Is(err, sql.ErrNoRows) {
            return tygor.NewError(tygor.CodeNotFound, "not found")
        }
        return nil
    })
```

### Masking Internal Errors

Prevent sensitive error details from leaking in production:

```go
reg := tygor.NewRegistry().WithMaskInternalErrors()
```

## Interceptors

Interceptors provide cross-cutting concerns at different levels.

### Global Interceptors

Applied to all handlers:

```go
reg := tygor.NewRegistry().
    WithInterceptor(middleware.LoggingInterceptor(logger))
```

### Service Interceptors

Applied to all handlers in a service:

```go
news := reg.Service("News").
    WithInterceptor(authInterceptor)
```

### Handler Interceptors

Applied to specific handlers:

```go
news.Register("Create",
    tygor.NewHandler(CreateNews).
        WithInterceptor(func(ctx context.Context, req any, info *tygor.RPCInfo, handler tygor.HandlerFunc) (any, error) {
            // Custom logic
            return handler(ctx, req)
        }))
```

## Middleware

HTTP middleware wraps the entire registry:

```go
reg := tygor.NewRegistry().
    WithMiddleware(middleware.CORS(middleware.DefaultCORSConfig()))

http.ListenAndServe(":8080", reg.Handler())
```

## Validation

Request validation uses struct tags via the `validator/v10` package:

```go
type CreateParams struct {
    Title string `json:"title" validate:"required,min=3,max=100"`
    Email string `json:"email" validate:"required,email"`
}
```

## Caching

Set cache headers on handlers:

```go
news.Register("List",
    tygor.NewHandler(ListNews).
        Method("GET").
        Cache(5 * time.Minute))
```

## Context Helpers

Access request metadata and modify responses:

```go
func Handler(ctx context.Context, req *Request) (*Response, error) {
    // Get service and method name
    service, method, _ := tygor.MethodFromContext(ctx)

    // Set custom response headers
    tygor.SetHeader(ctx, "X-Custom", "value")

    return &Response{}, nil
}
```

## Type Mappings

Customize TypeScript type generation for third-party types:

```go
reg.Generate(&tygor.GenConfig{
    OutDir: "./client/src/rpc",
    TypeMappings: map[string]string{
        "github.com/jackc/pgtype.Timestamptz": "string | null",
        "github.com/jackc/pgtype.UUID":        "string",
    },
})
```

## License

MIT
