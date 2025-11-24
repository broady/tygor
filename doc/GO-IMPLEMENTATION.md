# Tygor Go Implementation Specification

**Version:** 1.0
**Status:** Draft

## 1. Introduction

This document specifies the implementation requirements for the **tygor** Go library (`github.com/broady/tygor`). This library provides a code-first RPC framework that implements the [Tygor Protocol](./PROTOCOL.md).

For wire protocol details and interoperability requirements, see **PROTOCOL.md**.

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in RFC 2119.

---

## 2. Data Layer Integration (`sqlc`)

The Go implementation is designed to work seamlessly with `sqlc` for type-safe database interactions.

### 2.1 Recommended `sqlc` Configuration

The following `sqlc.yaml` configuration is RECOMMENDED to ensure optimal integration:

```yaml
version: "2"
sql:
  - schema: "schema.sql"
    queries: "queries.sql"
    engine: "postgresql"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_result_struct_pointers: true
        emit_params_struct_pointers: true
        query_parameter_limit: 0  # Force parameter structs for all queries
```

**Rationale:**
- **`query_parameter_limit: 0`**: Forces `sqlc` to generate a dedicated parameters struct for every query, making them suitable for use as RPC request types
- **`emit_result_struct_pointers: true`**: Nullable SQL rows map to Go pointers, compatible with tygor's optional field handling
- **`emit_params_struct_pointers: true`**: Input parameters use pointers for optional fields
- **`emit_json_tags: true`**: Generated structs include JSON tags for automatic serialization

### 2.2 Query Naming Convention

Query names SHOULD follow `PascalCase` to align with Go naming conventions:

```sql
-- name: GetUser :one
-- name: ListUsers :many
-- name: CreateUser :one
```

These generated structs can then be used directly as RPC request/response types.

---

## 3. Handler Definition

### 3.1 Handler Function Signature

All RPC handler functions MUST conform to the following signature:

```go
func(ctx context.Context, req Req) (Res, error)
```

Where:
- `ctx` is the request context
- `Req` is the request type (typically a `sqlc`-generated struct or custom type)
- `Res` is the response type
- `error` is returned for any failure

**Example:**
```go
func ListNews(ctx context.Context, req *db.ListNewsParams) ([]*db.News, error) {
    queries := db.New(pool) // assumes pgx pool available
    return queries.ListNews(ctx, req)
}
```

### 3.2 Handler Construction

The library MUST provide a `NewHandler` function that wraps handler functions and returns an `RPCMethod` interface.

**Signature:**
```go
func NewHandler[Req, Res any](fn func(context.Context, Req) (Res, error)) *Handler[Req, Res]
```

**Fluent API:**
The `Handler` type MUST support method chaining for configuration:

```go
type Handler[Req, Res any] struct {
    // internal fields
}

func (h *Handler[Req, Res]) Method(method string) *Handler[Req, Res]
func (h *Handler[Req, Res]) Cache(duration time.Duration) *Handler[Req, Res]
func (h *Handler[Req, Res]) WithInterceptor(i Interceptor) *Handler[Req, Res]
func (h *Handler[Req, Res]) WithSkipValidation() *Handler[Req, Res]
```

**Example Usage:**
```go
h := tygor.NewHandler(ListNews).
    Method("GET").
    Cache(5 * time.Minute)
```

**Defaults:**
- HTTP Method: `"POST"`
- No caching
- Validation enabled (if validator is available)

---

## 4. Registry and Service Registration

### 4.1 Registry

The library MUST provide a `Registry` type that manages services and routes.

```go
type Registry struct {
    // internal fields
}

func NewRegistry() *Registry
```

### 4.2 Service Namespacing

The `Registry.Service(name string)` method MUST return a `Service` instance that provides a namespace for related operations.

```go
type Service struct {
    // internal fields
}

func (r *Registry) Service(name string) *Service
```

### 4.3 Handler Registration

The `Service.Register(method string, handler RPCMethod)` method MUST register a handler for a specific method name.

```go
func (s *Service) Register(method string, handler RPCMethod)
```

**Example:**
```go
reg := tygor.NewRegistry()
news := reg.Service("News")
news.Register("List", tygor.NewHandler(ListNews).Method("GET"))
news.Register("Create", tygor.NewHandler(CreateNews))
```

This registers operations:
- `GET /News/List`
- `POST /News/Create`

### 4.4 HTTP Integration

The `Registry` MUST implement `http.Handler` to integrate with the standard library.

```go
func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request)
```

**Example:**
```go
http.ListenAndServe(":8080", reg)
```

---

## 5. Configuration API

### 5.1 Registry Configuration

The `Registry` MUST support configuration via chaining methods:

```go
func (r *Registry) WithErrorTransformer(fn ErrorTransformer) *Registry
func (r *Registry) WithInterceptor(i Interceptor) *Registry
func (r *Registry) WithLogger(logger Logger) *Registry
func (r *Registry) WithSkipValidation() *Registry
```

**Example:**
```go
reg := tygor.NewRegistry().
    WithErrorTransformer(customErrorHandler).
    WithLogger(slog.Default())
```

### 5.2 Service Configuration

The `Service` type SHOULD support configuration methods:

```go
func (s *Service) WithInterceptor(i Interceptor) *Service
```

---

## 6. Error Handling

### 6.1 Error Structure

The library MUST provide an `Error` type conforming to the protocol:

```go
type Error struct {
    Code    ErrorCode      `json:"code"`
    Message string         `json:"message"`
    Details map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string
```

### 6.2 Error Codes

The library MUST define an `ErrorCode` type with constants for all standard protocol error codes:

```go
type ErrorCode string

const (
    CodeInvalidArgument   ErrorCode = "invalid_argument"
    CodeUnauthenticated   ErrorCode = "unauthenticated"
    CodePermissionDenied  ErrorCode = "permission_denied"
    CodeNotFound          ErrorCode = "not_found"
    CodeMethodNotAllowed  ErrorCode = "method_not_allowed"
    CodeConflict          ErrorCode = "conflict"
    CodeGone              ErrorCode = "gone"
    CodeResourceExhausted ErrorCode = "resource_exhausted"
    CodeCancelled         ErrorCode = "cancelled"
    CodeInternal          ErrorCode = "internal"
    CodeNotImplemented    ErrorCode = "not_implemented"
    CodeUnavailable       ErrorCode = "unavailable"
    CodeDeadlineExceeded  ErrorCode = "deadline_exceeded"
)
```

### 6.3 Error Constructor Functions

The library SHOULD provide constructor functions for common errors:

```go
func InvalidArgument(message string, details ...map[string]any) *Error
func NotFound(message string, details ...map[string]any) *Error
func PermissionDenied(message string, details ...map[string]any) *Error
func Unauthenticated(message string, details ...map[string]any) *Error
func Internal(message string, details ...map[string]any) *Error
// ... etc
```

### 6.4 Error Transformer

The library MUST support custom error transformation via an `ErrorTransformer` function:

```go
type ErrorTransformer func(error) *Error
```

**Default Transformation Behavior:**
The default transformer MUST implement the following mappings:

```go
func DefaultErrorTransformer(err error) *Error {
    if err == nil {
        return nil
    }

    // Already a tygor.Error
    if e, ok := err.(*Error); ok {
        return e
    }

    // Context errors
    if errors.Is(err, context.DeadlineExceeded) {
        return &Error{Code: CodeDeadlineExceeded, Message: "request timeout"}
    }
    if errors.Is(err, context.Canceled) {
        return &Error{Code: CodeCancelled, Message: "request cancelled"}
    }

    // Validation errors (if go-playground/validator is available)
    if _, ok := err.(validator.ValidationErrors); ok {
        return &Error{Code: CodeInvalidArgument, Message: "validation failed"}
    }

    // Database errors
    if errors.Is(err, sql.ErrNoRows) {
        return &Error{Code: CodeNotFound, Message: "not found"}
    }

    // Default: internal error
    return &Error{Code: CodeInternal, Message: "internal server error"}
}
```

**Configuration:**
```go
reg := tygor.NewRegistry().
    WithErrorTransformer(func(err error) *Error {
        // Custom transformation
        if errors.Is(err, sql.ErrNoRows) {
            return tygor.NotFound("resource not found")
        }
        return tygor.DefaultErrorTransformer(err)
    })
```

---

## 7. Request Decoding

### 7.1 Query String Decoding (GET)

For GET requests, the library MUST decode query parameters using a decoder compatible with `gorilla/schema`.

**Important:** GET requests use `schema` struct tags, NOT `json` tags. The `json` tags are ignored by the query parameter decoder.

**Case-Insensitive Matching:**
Query parameter names are matched case-insensitively. Without a `schema` tag, the field name is used (e.g., field `Limit` matches query param `limit`, `Limit`, or `LIMIT`). For clarity, always use explicit `schema` tags.

**Array Handling:**
Arrays MUST be decoded from the "repeat" format: `?ids=1&ids=2&ids=3`

**Example:**
```go
type ListParams struct {
    Limit  int      `schema:"limit"`
    Offset int      `schema:"offset"`
    Tags   []string `schema:"tags"`
}

// GET /News/List?limit=10&offset=0&tags=go&tags=tech
// Decodes to: ListParams{Limit: 10, Offset: 0, Tags: []string{"go", "tech"}}
```

### 7.2 JSON Body Decoding (POST)

For POST requests, the library MUST decode JSON bodies using `encoding/json`.

**Pointer Field Handling:**
Pointer fields SHOULD support `omitempty` to distinguish between "not provided" and "explicitly null":

```go
type UpdateUserParams struct {
    Name  *string `json:"name,omitempty"`
    Email *string `json:"email,omitempty"`
}
```

---

## 8. Validation

### 8.1 Automatic Validation

If `github.com/go-playground/validator/v10` is available, the library SHOULD automatically validate request structs before calling the handler.

**Validation Tags:**
```go
type CreateUserParams struct {
    Email    string `json:"email" validate:"required,email"`
    Username string `json:"username" validate:"required,min=3,max=20"`
    Age      int    `json:"age" validate:"gte=0,lte=130"`
}
```

**Validation Errors:**
Validation failures MUST be transformed to `CodeInvalidArgument` errors with field-level details.

### 8.2 Disabling Validation

Validation can be disabled globally or per-handler:

```go
// Global
reg := tygor.NewRegistry().WithSkipValidation()

// Per-handler
h := tygor.NewHandler(fn).WithSkipValidation()
```

---

## 9. Internal Architecture

### 9.1 Sealed `RPCMethod` Interface

The `RPCMethod` interface MUST be sealed to prevent external implementations. This is achieved by having the `Metadata()` method return a type from an internal package.

**`internal/meta` Package:**
```go
package meta

type MethodMetadata struct {
    HTTPMethod   string
    CacheDuration time.Duration
    ReqType      reflect.Type
    ResType      reflect.Type
}
```

**Root Package:**
```go
package tygor

type RPCMethod interface {
    ServeHTTP(http.ResponseWriter, *http.Request)
    Metadata() *meta.MethodMetadata // Cannot be implemented outside this module
}
```

This ensures that only handlers created via `tygor.NewHandler` can be registered.

---

## 10. Interceptors

### 10.1 Interceptor Signature

The library MUST support interceptors (middleware) with the following signature:

```go
type UnaryInterceptor func(ctx context.Context, req any, info *RPCInfo, handler HandlerFunc) (res any, err error)

type HandlerFunc func(ctx context.Context, req any) (res any, err error)

type RPCInfo struct {
    Service string
    Method  string
}
```

### 10.2 Interceptor Scopes

Interceptors can be registered at three levels (executed in order):

1. **Registry-level (global):** Applied to all operations
2. **Service-level:** Applied to all operations in a service
3. **Handler-level:** Applied to a specific operation

**Example:**
```go
// Logging interceptor
func LoggingInterceptor(ctx context.Context, req any, info *RPCInfo, handler HandlerFunc) (any, error) {
    start := time.Now()
    res, err := handler(ctx, req)
    log.Printf("%s.%s took %v", info.Service, info.Method, time.Since(start))
    return res, err
}

reg := tygor.NewRegistry().WithInterceptor(LoggingInterceptor)
```

### 10.3 Execution Order

Interceptors MUST execute in the following order:
1. Registry interceptors (first registered → last registered)
2. Service interceptors (first registered → last registered)
3. Handler interceptors (first registered → last registered)
4. Actual handler
5. Interceptors return in reverse order (unwinding stack)

---

## 11. Context API

### 11.1 Request Access

The library MUST provide a function to access the underlying HTTP request:

```go
func RequestFromContext(ctx context.Context) *http.Request
```

**Example:**
```go
func MyHandler(ctx context.Context, req *MyRequest) (*MyResponse, error) {
    httpReq := tygor.RequestFromContext(ctx)
    userAgent := httpReq.Header.Get("User-Agent")
    // ...
}
```

### 11.2 Operation Info Access

The library MUST provide a function to access operation metadata:

```go
func MethodFromContext(ctx context.Context) (service, method string)
```

### 11.3 Response Header Manipulation

The library MUST provide a function to set response headers:

```go
func SetHeader(ctx context.Context, key, value string)
```

**Example:**
```go
func MyHandler(ctx context.Context, req *MyRequest) (*MyResponse, error) {
    tygor.SetHeader(ctx, "X-Custom-Header", "value")
    return &MyResponse{}, nil
}
```

---

## 12. Type Generation

### 12.1 Generator API

The library MUST provide a standalone `Generate` function in the `tygorgen` package:

```go
func Generate(reg *tygor.Registry, config *Config) error
```

This function:
- Takes the registry to extract registered routes
- Uses the provided configuration to control type generation
- Generates TypeScript types and manifest files

### 12.2 Generation Configuration

The `tygorgen.Config` struct controls TypeScript type generation:

```go
type Config struct {
    // REQUIRED: Output directory for generated files
    OutDir string

    // OPTIONAL: Custom Go-to-TypeScript type mappings
    TypeMappings map[string]string

    // OPTIONAL: Comment preservation ("default", "types", "none")
    PreserveComments string

    // OPTIONAL: Enum generation style ("union", "enum", "const")
    EnumStyle string

    // OPTIONAL: Optional field typing ("undefined", "null")
    OptionalType string

    // OPTIONAL: Custom frontmatter for generated files
    Frontmatter string
}
```

**Defaults:**
```go
{
    PreserveComments: "default",
    EnumStyle: "union",
    OptionalType: "undefined",
    TypeMappings: map[string]string{
        "time.Time":          "string",
        "pgtype.Timestamptz": "string | null",
        "pgtype.Text":        "string",
        "pgtype.Int4":        "number",
    },
}
```

### 12.3 Generated Files

The generator MUST produce:

1. **`types.ts`**: TypeScript interfaces for all request/response types
2. **`manifest.ts`**: Operation manifest (see TYPESCRIPT-CLIENT.md)

**Example:**
```go
err := tygorgen.Generate(reg, &tygorgen.Config{
    OutDir: "./client/src/rpc",
    TypeMappings: map[string]string{
        "uuid.UUID": "string",
    },
})
```

---

## 13. Logging

### 13.1 Logger Interface

The library SHOULD support a logger interface compatible with `log/slog`:

```go
type Logger interface {
    Info(msg string, args ...any)
    Error(msg string, args ...any)
    Debug(msg string, args ...any)
}
```

### 13.2 Logger Configuration

```go
reg := tygor.NewRegistry().WithLogger(slog.Default())
```

**Logged Events:**
- Request start/end
- Errors (with error details)
- Panics (recovered)

---

## 14. Complete Example

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/broady/tygor"
    "myapp/internal/db"
)

// Handler function
func ListNews(ctx context.Context, req *db.ListNewsParams) ([]*db.News, error) {
    queries := db.New(dbPool)
    return queries.ListNews(ctx, req)
}

func CreateNews(ctx context.Context, req *db.CreateNewsParams) (*db.News, error) {
    queries := db.New(dbPool)
    return queries.CreateNews(ctx, req)
}

func main() {
    // Create registry
    reg := tygor.NewRegistry().
        WithLogger(slog.Default()).
        WithErrorTransformer(customErrors)

    // Register services
    news := reg.Service("News")
    news.Register("List", tygor.NewHandler(ListNews).Method("GET").Cache(5*time.Minute))
    news.Register("Create", tygor.NewHandler(CreateNews))

    // Generate TypeScript types
    if err := tygorgen.Generate(reg, &tygorgen.Config{
        OutDir: "./client/src/rpc",
    }); err != nil {
        log.Fatal(err)
    }

    // Start server
    log.Fatal(http.ListenAndServe(":8080", reg))
}
```

---

## 15. Compliance Checklist

A conforming Go implementation MUST:

- ✅ Support `func(ctx context.Context, req Req) (Res, error)` handlers
- ✅ Provide `NewHandler` with fluent configuration API
- ✅ Provide `Registry` with `Service` namespacing
- ✅ Implement `http.Handler` interface
- ✅ Support GET (query string) and POST (JSON body) serialization
- ✅ Implement default error transformer with standard error codes
- ✅ Support custom error transformers
- ✅ Provide sealed `RPCMethod` interface
- ✅ Support interceptors at registry/service/handler levels
- ✅ Provide context API for request metadata
- ✅ Generate TypeScript types and manifest
