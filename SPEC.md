# Specification: tygorpc

## 1. Introduction

This document specifies the architecture and implementation requirements for **tygorpc** (`github.com/broady/tygorpc`), a code-first Remote Procedure Call system designed for Go backends and TypeScript frontends. The system MUST utilize Go structs as the single source of truth, leveraging Go Generics for type safety and `sqlc` for database interactions.

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in RFC 2119.

-----

## 2. Data Layer Specification (`sqlc`)

The system MUST use `sqlc` to generate type-safe Go structs from SQL queries.

### 2.1 Configuration Requirements

The `sqlc.yaml` configuration MUST adhere to the following constraints:

1.  **`query_parameter_limit: 0`**: This MUST be set to force `sqlc` to generate a dedicated parameters struct for *every* query.
2.  **`emit_result_struct_pointers: true`**: Nullable SQL rows MUST map to Go pointers.
3.  **`emit_params_struct_pointers: true`**: Input parameters MUST use pointers.
4.  **`emit_json_tags: true`**: Generated structs MUST include standard JSON tags.

### 2.2 Query Naming Convention

Query names defined in SQL comments (e.g., `-- name: GetUser :one`) MUST follow `PascalCase`.

-----

## 3. Application Layer Specification

### 3.1 Handler Signature

All RPC handlers MUST conform to the following generic signature:

```go
func(ctx context.Context, req Req) (Res, error)
```

### 3.2 Handler Construction

The system MUST provide a fluent builder API for defining handlers. The handler encapsulates the generic types, allowing them to be registered without type parameters on the registration method.

```go
// NewHandler creates a handler from a generic function.
// Defaults: Method="POST"
h := tygorpc.NewHandler(ListNews).
    Method("GET").
    Cache(5 * time.Minute)
```

-----

## 4. Transport Layer Specification (The Registry)

The Registry is responsible for mapping Operation IDs to generic HTTP handlers.

### 4.1 Registry & Services

The system MUST provide a `Registry` that manages services and routes.

```go
reg := tygorpc.NewRegistry()
news := reg.Service("News") // Returns a namespaced Service

// Register takes the operation name and the handler.
// The generic type parameters are encapsulated in the handler interface.
news.Register("List", h)
```

### 4.2 Configuration (Chaining)

The Registry MUST support configuration via chaining methods that return the registry (or a modified copy).

```go
reg := tygorpc.NewRegistry().
    WithErrorTransformer(customErrorHandler)
```

### 4.3 HTTP Mapping Rules

1.  **Path Construction**: The URL path MUST be constructed as `/{service_name}/{method_name}` (e.g., `/News/List`).
2.  **Method Constraints**:
      * **GET**: MUST be used for read-only operations. Request parameters MUST be encoded in the URL Query String.
      * **POST**: MUST be used for state-changing operations. Request parameters MUST be encoded in the JSON Body.
3.  **Query Serialization**:
      * **GET Requests**: MUST use a decoder compatible with `gorilla/schema`.
      * **Arrays**: MUST be serialized using the "Repeat" format (`ids=1&ids=2`).

### 4.4 Error Handling

The system MUST define a standard JSON error envelope and support custom transformations.

**Error Structure:**
```go
type Error struct {
    Code    ErrorCode      `json:"code"`
    Message string         `json:"message"`
    Details map[string]any `json:"details,omitempty"`
}
```

**Error Codes (`ErrorCode`)**:
- `invalid_argument` (400)
- `unauthenticated` (401)
- `permission_denied` (403)
- `not_found` (404)
- `unavailable` (503)
- `internal` (500)
- ... and others as needed.

**Error Transformer:**
The system MUST allow users to provide a `func(error) *Error` to map application errors (e.g., `sql.ErrNoRows`) to RPC status codes.

**Default Behavior:**
- `context.DeadlineExceeded` -> `unavailable` (503)
- `validator.ValidationErrors` -> `invalid_argument` (400)
- `nil` -> `nil`
- `*tygorpc.Error` -> Returned as-is
- Other -> `internal` (500) (with message masked in production, or passed through depending on config)

-----

## 5. Frontend Layer Specification

### 5.1 Type Generation

The system MUST provide a `Generate(reg *Registry, config GenConfig)` function that:
1.  **Types**: Uses `tygo` to generate TypeScript interfaces for all Request/Response structs found in the registry.
2.  **Manifest**: Generates a `manifest.ts` file mapping Operation IDs to metadata.

### 5.2 Manifest Format

The manifest MUST export a TypeScript interface mapping Operation IDs to their request/response types and runtime metadata.

```typescript
import * as types from './types';

export interface RPCManifest {
  "News.List": {
    req: types.ListNewsParams;
    res: types.News;
    method: "GET";
    path: "/News/List";
  };
}

export const RPCMetadata = {
  "News.List": { method: "GET", path: "/News/List" },
} as const;
```

### 5.3 Client Runtime

The TypeScript client MUST use an ES6 `Proxy` to dynamically resolve method calls based on `RPCMetadata`. It MUST NOT contain generated code for individual methods.

**Syntax**: `client.Service.Method(params)`

-----

## 6. Internal Architecture

### 6.1 Sealed Interfaces

The `RPCMethod` interface used in `Register` MUST be sealed to prevent external implementation. This is achieved by having the `Metadata()` method return a type defined in an internal package.

**`internal/meta` Package**:
```go
package meta
type MethodMetadata struct {
    Method string
    // ... reflection types ...
}
```

**Root Package**:
```go
type RPCMethod interface {
    ServeHTTP(...)
    Metadata() *meta.MethodMetadata // Cannot be implemented outside this module
}
```

-----

## 7. Interceptors & Metadata

### 7.1 Interceptors

The system MUST support a universal interceptor mechanism that wraps RPC execution.

**Signature:**
```go
// Interceptor is a generic hook.
// req/res are pointers to the structs.
type Interceptor func(ctx context.Context, req any, info *RPCInfo, handler HandlerFunc) (res any, err error)

type HandlerFunc func(ctx context.Context, req any) (res any, err error)

type RPCInfo struct {
    Service string
    Method  string
}
```

**Registration Scopes:**
1.  **Global**: `Registry.WithInterceptor(i)`
2.  **Service**: `Service.WithInterceptor(i)`
3.  **Handler**: `Handler.WithInterceptor(i)`

### 7.2 Context Metadata

The system MUST provide mechanisms to access and modify HTTP-layer metadata via `context.Context`.

**Request Metadata:**
-   `RequestFromContext(ctx) *http.Request`: Access to the underlying HTTP request (headers, remote addr, etc.).
-   `MethodFromContext(ctx) (service, method string)`: Access to operation info.

**Response Metadata:**
-   `SetHeader(ctx, key, val)`: Sets an HTTP response header.
