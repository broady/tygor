# Multi-Package Example

This example demonstrates how to handle same-named types from different packages using the `StripPackagePrefix` configuration option.

## The Problem

In Go, it's common to have types with the same name in different packages:

```go
// api/v1/types.go
type User struct {
    ID   int64
    Name string
}

// api/v2/types.go
type User struct {
    ID        int64
    Name      string
    Email     string
    CreatedAt string
}
```

Without disambiguation, both would become `User` in TypeScript, causing a collision.

## The Solution: StripPackagePrefix

The `StripPackagePrefix` configuration strips a common prefix from package paths and uses the remainder to qualify type names:

```go
tygorgen.Generate(app, &tygorgen.Config{
    OutDir: *out,
    // Types from the api package itself get no prefix (MigrationRequest).
    StripPackagePrefix: "github.com/broady/tygor/examples/multipackage/api",
})
```

This produces:

```typescript
// From api/v1/types.go
export interface v1_User {
  id: number;
  name: string;
}

// From api/v2/types.go
export interface v2_User {
  id: number;
  name: string;
  email: string;
  created_at: string;
}

// From api/types.go (no prefix - matches the StripPackagePrefix exactly)
export interface MigrationRequest {
  v1_user: v1_User;
  v2_user: v2_User;
}
```

## How It Works

1. **Package path matching**: For each type, the generator compares its package path against `StripPackagePrefix`
2. **Exact match**: If the package path equals the prefix, no qualifier is added (e.g., `api` package types)
3. **Subpackage match**: If the package path starts with the prefix, the remaining path becomes a qualifier:
   - `api/v1` - prefix `api` = `/v1` -> sanitized to `v1_`
   - `api/v2` - prefix `api` = `/v2` -> sanitized to `v2_`
4. **No match**: Types from packages that don't match the prefix use the full sanitized package path

## Running the Example

```bash
cd examples/multipackage

# Generate TypeScript types
make gen

# Run tests
make test
```

## Code Structure

```
examples/multipackage/
|-- api/
|   |-- types.go          # MigrationRequest/Response (references v1 and v2 types)
|   |-- v1/
|   |   |-- types.go      # v1.User, v1.GetUserRequest
|   |-- v2/
|       |-- types.go      # v2.User, v2.GetUserRequest
|-- main.go               # Server with StripPackagePrefix config
|-- client/
|   |-- index.ts          # TypeScript client example
|   |-- tests/            # Integration tests
|   |-- src/rpc/          # Generated TypeScript types
|-- README.md             # This file
```

## Generated Types

The generated `types.ts` demonstrates proper disambiguation:

```typescript
export interface v1_GetUserRequest {
  id: number;
}
export interface v2_GetUserRequest {
  id: number;
}
export interface MigrationRequest {
  v1_user: v1_User;
  v2_user: v2_User;
}
export interface MigrationResponse {
  success: boolean;
  v1_user: v1_User;
  v2_user: v2_User;
}
export interface v1_User {
  id: number;
  name: string;
}
export interface v2_User {
  id: number;
  name: string;
  email: string;
  created_at: string;
}
```

## TypeScript Client Usage

The client can use properly typed v1 and v2 types:

```typescript
import type { v1_User, v2_User, MigrationRequest } from "./src/rpc/types";

// Both types are distinct
const v1User: v1_User = { id: 1, name: "Old" };
const v2User: v2_User = { id: 1, name: "New", email: "new@example.com", created_at: "2024-01-01" };

// MigrationRequest properly references both
const req: MigrationRequest = {
  v1_user: v1User,
  v2_user: v2User,
};
```
