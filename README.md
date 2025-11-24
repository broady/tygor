![tygor](doc/tygor-banner.svg)

# tygor

Type-safe RPC framework for Go with automatic TypeScript client generation.

> [!WARNING]
> tygor is very experimental and the API is rapidly changing.
> Pinning the `@tygor/client` version should prevent any unexpected breakages.

## Features

- Type-safe handlers using Go generics
- Automatic TypeScript generation from Go types
- Lightweight TypeScript client using proxies for minimal bundle size
- Zero reflection at runtime for handler execution
- Request validation with struct tags
- Flexible error handling with structured error codes
- Middleware and interceptors at global, service, and handler levels
- Built-in support for GET and POST methods with appropriate decoding

## Philosophy

**tygor is for teams building tightly-coupled Go and TypeScript applications in monorepos.**

If you're building a fullstack application where the Go backend and TypeScript frontend live together, tygor gives you end-to-end type safety without the ceremony of IDLs like protobuf or OpenAPI specs. You write normal Go functions, and tygor generates TypeScript types that match your actual implementation.

### Who is this for?

**Use tygor if you:**
- Build fullstack apps with Go + TypeScript in a monorepo
- Want type safety without learning protobuf/gRPC/OpenAPI
- Value iteration speed and developer ergonomics
- Want to write idiomatic Go handlers and get TypeScript types automatically
- Are okay with incrementally improving types as your domain evolves

**Don't use tygor if you:**
- Need a public API with strict backward compatibility guarantees
- Require multi-language client support (though OpenAPI generation is planned)
- Need the guarantees of a formal IDL (protobuf, Thrift, etc.)
- Have microservices that need to evolve independently

### The tradeoff

tygor isn't trying to be a perfect code generation tool like protobuf. Instead, it's optimized for the common case: a team iterating on a fullstack app where the backend and frontend are tightly coupled anyway. You can always add handwritten TypeScript definitions to improve type safety for your specific domain. This is often nicer than being forced into the constraints of an IDL.

In the future, tygor may generate OpenAPI specs to enable client generation in other languages, giving you the best of both worlds: ergonomic Go + TypeScript for your core app, with optional compatibility for other ecosystems.

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
news.Register("List", tygor.UnaryGet(ListNews))
news.Register("Create", tygor.Unary(CreateNews)) // POST is default

http.ListenAndServe(":8080", reg.Handler())
```

### 4. Generate TypeScript types

```go
if err := tygorgen.Generate(reg, &tygorgen.Config{
    OutDir: "./client/src/rpc",
}); err != nil {
    log.Fatal(err)
}
```

This generates TypeScript types and a manifest describing all available RPC methods.

### 5. Use the TypeScript client

First, install the client runtime:

```bash
npm install @tygor/client
```

The generated client provides a clean, idiomatic API with full type safety:

```typescript
import { createClient } from '@tygor/client';
import { registry } from './rpc/manifest';

const client = createClient(
  registry,
  {
    baseUrl: 'http://localhost:8080',
    headers: () => ({
      'Authorization': 'Bearer my-token'
    })
    // fetch: customFetch  // Optional: for testing or custom environments
  }
);

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
  if (err instanceof RPCError) {
    console.error(err.code, err.message); // "invalid_argument", "validation failed"
    console.error(err.details);           // Additional error context
  }
}
```

The client uses JavaScript Proxies to provide method access without code generation bloat. Your bundle only includes the types and a small runtime, regardless of how many RPC methods you have.

Example `manifest.ts`:

```typescript
export interface RPCManifest {
  "News.List": {
    req: types.ListNewsParams;
    res: types.News[];
  };
  "News.Create": {
    req: types.CreateNewsParams;
    res: types.News;
  };
}

const metadata = {
  "News.List": { method: "GET", path: "/News/List" },
  "News.Create": { method: "POST", path: "/News/Create" },
} as const;

export const registry: ServiceRegistry<RPCManifest> = {
  manifest: {} as RPCManifest,
  metadata,
};
```

## Request Handling

### GET Requests

For GET requests, parameters are decoded from query strings:

```go
type ListParams struct {
    Limit  *int32 `json:"limit"`
    Offset *int32 `json:"offset"`
}

tygor.UnaryGet(List)
```

Query: `/News/List?limit=10&offset=20`

### POST Requests

For POST requests (the default), the body is decoded as JSON:

```go
type CreateParams struct {
    Title string `json:"title" validate:"required"`
}

tygor.Unary(Create) // POST is default
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
    tygor.Unary(CreateNews).
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

Set cache headers on handlers (typically used with GET):

```go
news.Register("List",
    tygor.UnaryGet(ListNews).
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
tygorgen.Generate(reg, &tygorgen.Config{
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
