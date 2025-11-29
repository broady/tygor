# tygorgen

Go to TypeScript code generator. Generates TypeScript types and Zod schemas from Go structs.

## Features

- **Zod schema generation** from Go `validate` tags
- **Nullable vs optional** correctly distinguished (`*T` vs `omitempty`)
- **Enum support** from Go const groups with iota
- **Doc comments** preserved in TypeScript output
- **Custom type mappings** for `time.Time`, `uuid.UUID`, etc.

## Usage

### With tygor app

```go
import "github.com/broady/tygor/tygorgen"

tygorgen.FromApp(app).
    WithFlavor(tygorgen.FlavorZod).
    ToDir("./client/src/rpc")
```

### Standalone type generation

Generate TypeScript from Go types without the full RPC framework:

```go
tygorgen.FromTypes(
    User{},
    CreateUserRequest{},
    ListUsersResponse{},
).ToDir("./client/src/types")
```

### With Zod schemas

```go
tygorgen.FromTypes(User{}).
    WithFlavor(tygorgen.FlavorZod).
    ToDir("./client/src/types")
```

## Example

**Go input:**
```go
type User struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name" validate:"required,min=2"`
    Email     string    `json:"email" validate:"required,email"`
    Avatar    *string   `json:"avatar"`           // nullable
    CreatedAt time.Time `json:"created_at"`
}
```

**TypeScript output:**
```typescript
export interface User {
  id: number;
  name: string;
  email: string;
  avatar: string | null;
  created_at: string;
}
```

**Zod output:**
```typescript
export const UserSchema = z.object({
  id: z.number().int(),
  name: z.string().min(1).min(2),
  email: z.string().min(1).email(),
  avatar: z.nullable(z.string()),
  created_at: z.string().datetime(),
});
```

## Configuration

```go
tygorgen.FromApp(app).
    WithFlavor(tygorgen.FlavorZod).      // Generate Zod schemas
    WithFlavor(tygorgen.FlavorZodMini).  // Or use zod/mini for smaller bundles
    SingleFile().                         // All types in one file
    EnumStyle("enum").                    // "union" | "enum" | "const"
    OptionalType("null").                 // "undefined" | "null"
    TypeMapping("time.Time", "Date").     // Custom type mappings
    PreserveComments("types").            // "default" | "types" | "none"
    ToDir("./client/src/rpc")
```

## Providers

Two extraction strategies are available:

| Provider | Speed | Enums | Comments | Use case |
|----------|-------|-------|----------|----------|
| `source` (default) | Slower | Yes | Yes | Production builds |
| `reflection` | Fast | No | No | Development iteration |

```go
tygorgen.FromTypes(User{}).
    Provider("reflection").  // Fast mode
    ToDir("./client/src/types")
```

## See also

Other Go to TypeScript generators:

- [tygo](https://github.com/gzuidhof/tygo) - CLI with YAML config, enums, comments
- [guts](https://github.com/coder/guts) - Library using TypeScript compiler API
- [typescriptify-golang-structs](https://github.com/tkrajina/typescriptify-golang-structs) - Library with enum support
- [go2ts](https://github.com/StirlingMarketingGroup/go2ts) - Online converter tool
