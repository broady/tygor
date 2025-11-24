# Specification: tygorpc

## 1. Introduction

This document specifies the architecture and implementation requirements for **tygorpc**, a code-first Remote Procedure Call system designed for Go backends and TypeScript frontends. The system MUST utilize Go structs as the single source of truth, leveraging Go Generics for type safety and `sqlc` for database interactions. The frontend MUST utilize a minimal, proxy-based runtime that derives all API contracts strictly from generated TypeScript definitions.

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in RFC 2119.

-----

## 2. Data Layer Specification (`sqlc`)

The system MUST use `sqlc` to generate type-safe Go structs from SQL queries.

### 2.1 Configuration Requirements

The `sqlc.yaml` configuration MUST adhere to the following constraints to ensure compatibility with the RPC transport layer and JSON serialization standards:

1.  **`query_parameter_limit: 0`**: This MUST be set to force `sqlc` to generate a dedicated parameters struct for *every* query, even those with single arguments. This ensures a uniform `func(ctx, ArgStruct) (ResStruct, error)` signature for generic wrapping.
2.  **`emit_result_struct_pointers: true`**: Nullable SQL rows MUST map to Go pointers, which serialize to `null` in JSON (as opposed to `sql.NullString` structs).
3.  **`emit_params_struct_pointers: true`**: Input parameters MUST use pointers for nullable columns to distinguish between `undefined` (missing) and `null` (explicit null) in JSON.
4.  **`emit_json_tags: true`**: Generated structs MUST include standard JSON tags.

### 2.2 Query Naming Convention

Query names defined in SQL comments (e.g., `-- name: GetUser :one`) MUST follow `PascalCase`. These names SHALL serve as the "Method" names in the RPC registry.

-----

## 3. Application Layer Specification

### 3.1 Request/Response Models

1.  **Read Operations**: Handlers SHOULD accept `sqlc` generated parameter structs directly if no additional validation is required.
2.  **Write Operations**: Handlers SHOULD accept a defined "Request Struct". If validation is required, this struct MAY embed the `sqlc` parameter struct.
      * **Validation**: Struct fields MAY use `validate:"..."` tags compatible with `github.com/go-playground/validator`.
      * **No Logic Tags**: Struct tags MUST NOT be used to define routing logic (e.g., no `api:"GET /path"`).

### 3.2 Handler Signature

All RPC handlers MUST conform to the following generic signature:

```go
type HandlerFunc func(ctx context.Context, req Req) (Res, error)
```

-----

## 4. Transport Layer Specification (The Registry)

The Registry is responsible for mapping Operation IDs to generic HTTP handlers.

### 4.1 Registration

The system MUST provide a programmatic registration API.

```go
// Service represents a namespace (e.g., "News")
type Service struct {... }

// Register adds a method to the service.
// Method: "GET" | "POST"
// Name: The operation name (e.g., "List")
// fn: The generic handler function
func Register(s *Service, method string, name string, fn HandlerFunc)
```

### 4.2 HTTP Mapping Rules

1.  **Path Construction**: The URL path MUST be constructed as `/{service_name}/{method_name}`.
2.  **Method Constraints**:
      * **GET**: MUST be used for read-only operations. Request parameters MUST be encoded in the URL Query String.
      * **POST**: MUST be used for state-changing operations. Request parameters MUST be encoded in the JSON Body.

### 4.3 Serialization Strategy

1.  **GET Requests (Query String)**:
      * The system MUST use a decoder compatible with `gorilla/schema`.
      * **Arrays**: MUST be serialized using the "Repeat" format (e.g., `ids=1&ids=2`). The decoder MUST NOT require brackets (..).
      * **Client Requirement**: The TypeScript client MUST serialise arrays in this format.
2.  **POST Requests (JSON)**:
      * The system MUST use the standard `encoding/json` decoder.
      * **Content-Type**: The server MUST enforce `Content-Type: application/json`.

### 4.4 Caching

1.  **Declarative**: The Registration API MUST accept an optional `CacheOptions` struct (e.g., `TTL time.Duration`).
2.  **Programmatic**: The Context passed to the handler MUST provide a mechanism to set HTTP headers dynamically.
      * The handler MUST NOT access `http.ResponseWriter` directly.
      * A middleware MUST extract these context values and apply `Cache-Control` headers after handler execution.

### 4.5 Error Handling

The system MUST define a standard JSON error envelope.

```json
{
  "code": "string",   // Machine-readable error code (e.g., "validation_failed")
  "message": "string", // Human-readable message
  "details": {... }   // Optional structured data (e.g., validation field errors)
}
```

  * **Validation Errors**: MUST return HTTP 400.
  * **Internal Errors**: MUST return HTTP 500 and mask internal details in production.
  * **Custom Errors**: The system MUST map Go error types to HTTP status codes via a central error mapper.

-----

## 5. Frontend Layer Specification

### 5.1 Type Generation (`tygo` Library)

The system MUST use `tygo` library programmatically to generate TypeScript interfaces from Go structs via reflection.

  * **Dependency**: `github.com/gzuidhof/tygo/tygo` MUST be imported as a Go dependency
  * **Programmatic Usage**: The generator tool MUST use `tygo.New(config).Generate()` rather than CLI execution
  * **Input**: The `db` package (sqlc output) and any custom API struct packages
  * **Output**: A single `types.ts` file containing all data shapes

### 5.2 Manifest Generation

The service binary MUST provide a generation capability that:

1. **Uses `tygo` library** to generate `types.ts` from Go packages
2. **Inspects Runtime Registry** via reflection to extract operation metadata
3. **Combines** the generated types with operation metadata to create `manifest.ts`

**Implementation Approach:**
- The generation logic MUST be part of the service binary itself (not a separate tool)
- The service accesses its own registry at runtime via reflection
- The trigger mechanism is implementation-defined (flags, subcommands, build tags, etc.)
- **RECOMMENDED**: Use `go generate` with a directive like `//go:generate go run . --gen-manifest`

The generation process MUST:
- Iterate over registered routes using reflection
- Extract `Req` and `Res` type information from handler signatures
- Generate `manifest.ts` with references to `tygo`-generated types

**Manifest Format (`manifest.ts`):**
This file MUST export a TypeScript interface mapping Operation IDs to their request/response types.

```typescript
import * as types from './types';

export interface RPCManifest {
  "News.List": {
    req: types.ListNewsParams;
    res: types.News;
    method: "GET";
    path: "/news/list";
  };
  "News.Create": {
    req: types.CreateNewsParams;
    res: types.News;
    method: "POST";
    path: "/news/create";
  };
}

export const RPCMetadata = {
  "News.List": { method: "GET", path: "/news/list" },
  "News.Create": { method: "POST", path: "/news/create" },
} as const;
```

### 5.2.1 Alternative: Annotation-Based Generation

The approach described in Section 5.2 (runtime reflection) couples generation to the service binary. An alternative approach using **source code annotations** would enable standalone generator tools, but is explicitly **deferred** for the initial implementation.

**Annotation-Based Approach (Future Consideration):**
- Handlers would be marked with annotations (e.g., struct tags or code comments)
- A standalone generator tool would parse Go source files to discover annotations
- The tool would generate both:
  - Go code for runtime registration (e.g., `init()` functions calling `Register()`)
  - TypeScript types and manifest from the same source definitions

**Tradeoffs:**
- **Benefit**: True standalone generation without running the service binary
- **Benefit**: Potential for better IDE integration and tooling
- **Cost**: Requires annotation syntax design and parsing infrastructure
- **Cost**: Introduces code generation for Go (not just TypeScript)
- **Cost**: Annotations duplicate information already present in runtime registration

This specification defers annotation-based approaches to maintain focus on the core runtime registration model. The annotation system represents a separate design space that can be explored once the runtime approach is validated in production use.

### 5.3 Client Runtime (The Proxy)

The TypeScript client MUST NOT contain generated code for individual methods. It MUST use an ES6 `Proxy` to dynamically resolve method calls based on the `RPCMetadata` and `RPCManifest`.

**Requirements:**

1.  **Syntax**: `client.Service.Method(params)`
2.  **Type Safety**: Usage MUST be strictly typed against `RPCManifest`.
3.  **Tree Shaking**: The client runtime size MUST remain constant regardless of the number of API methods.

### 5.4 Type Mapping Configuration

The `tygo` configuration MUST include mappings for common database types:

```yaml
type_mappings:
  pgtype.Timestamptz: "string | null"
  pgtype.Text: "string"
  pgtype.Int4: "number"
  time.Time: "string"
```

This ensures proper TypeScript representation of nullable database columns.

-----

## 6. Observability & Middleware

### 6.1 Metrics (Prometheus)

The generic handler MUST include instrumentation middleware.

  * **Metric**: `rpc_request_duration_seconds` (Histogram).
  * **Labels**:
      * `service`: The service name (e.g., "News").
      * `method`: The operation name (e.g., "List"). *Note: Do NOT use raw URL paths to avoid high cardinality.*
      * `status`: HTTP status code.

### 6.2 Authentication / Context

  * Middleware MUST validate headers (e.g., `Authorization`).
  * Valid credentials MUST be injected into the `context.Context`.
  * The system MUST provide a generic helper function `GetActor(ctx) (T, bool)` to retrieve typed auth data in handlers.

-----

## 7. Implementation Roadmap

### Phase 1: Core Plumbing (Go)

1.  Initialize generic `Registry` and `Service` structs.
2.  Implement `LiftHandler` function:
      * Decodes HTTP request to `Req`.
      * Runs `go-validator`.
      * Calls user function.
      * Encodes `Res` to HTTP response.
      * Handles errors.

### Phase 2: Generation Implementation

1.  Implement generation capability within the service binary:
       * Add generation logic to the service (triggered via flag/subcommand/build tag)
       * Import `tygo` library for type generation
       * Access the service's own `Registry` at runtime
       * Use `tygo.New(config).Generate()` to create `types.ts`
       * Iterate over registered routes using reflection
       * Extract `Req` and `Res` type information from handler signatures
       * Generate `manifest.ts` with type references
       * Optionally: Add `//go:generate` directive for idiomatic workflow

### Phase 3: Client Library (TypeScript)

1.  Implement the generic `createClient<Manifest>(metadata)` function using `Proxy`.
2.  Implement the query string serializer (Array format: Repeat).

### Phase 4: Service Integration

1.  Define `sqlc` queries.
2.  Generate Go code.
3.  Register handlers.
4.  Trigger generation (e.g., `go generate`, or run service binary with generation flag).

-----

## 8. Versioning Strategy

  * API Versioning MUST be handled at the Service level (e.g., `NewsV1`, `NewsV2`).
  * Each version MUST be a distinct Go package or namespace in the Registry.
  * The `manifest.ts` SHALL contain keys like `"NewsV1.List"` and `"NewsV2.List"`.
  * Old handlers/queries MAY remain indefinitely until deprecated.
