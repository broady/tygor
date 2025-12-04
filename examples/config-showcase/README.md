# Config Showcase

Demonstrates different TypeScript generation configurations side-by-side.

This example generates the same Go types with different config options to show how each setting affects the output. Unlike other examples, this one runs custom generation logic to produce multiple output variants.

## Generated Outputs

| Directory | Configuration | Description |
|-----------|--------------|-------------|
| `client/src/union/` | `EnumStyle("union")` | String union types (default, cleanest for most uses) |
| `client/src/enum/` | `EnumStyle("enum")` | TypeScript enums with member docs |
| `client/src/const/` | `EnumStyle("const_enum")` | Inlined at compile time |
| `client/src/object/` | `EnumStyle("object")` | Const objects, runtime accessible |
| `client/src/opt-default/` | `OptionalType("default")` | omitempty uses `?:`, pointers use `\| null` |
| `client/src/opt-null/` | `OptionalType("null")` | All optional fields use `\| null` |
| `client/src/opt-undef/` | `OptionalType("undefined")` | All optional fields use `?:` |
| `client/src/no-comments/` | `PreserveComments("none")` | No JSDoc comments |

## Quick Start

```bash
go run . gen    # Generate all TypeScript variants
```

Then compare the outputs in `client/src/*/types.ts`.

## Go Types

The example uses these types to demonstrate different generation options:

```go
type Status string

const (
    StatusPending    Status = "pending"
    StatusInProgress Status = "in_progress"
    StatusCompleted  Status = "completed"
    StatusCancelled  Status = "cancelled"
)

type Task struct {
    ID          string     `json:"id"`
    Title       string     `json:"title"`
    Description *string    `json:"description,omitempty"`
    Status      Status     `json:"status"`
    Priority    Priority   `json:"priority"`
    Assignee    *string    `json:"assignee,omitempty"`
    DueDate     *time.Time `json:"due_date,omitempty"`
    Tags        []string   `json:"tags"`
    CreatedAt   time.Time  `json:"created_at"`
}
```

## Output Comparison

### EnumStyle: union (default)
```typescript
export type Status = "pending" | "in_progress" | "completed" | "cancelled";
```

### EnumStyle: enum
```typescript
export enum Status {
  /** StatusPending indicates the task is waiting to be started. */
  Pending = "pending",
  /** StatusInProgress indicates the task is currently being worked on. */
  InProgress = "in_progress",
  // ...
}
```

### EnumStyle: const_enum
```typescript
export const enum Status {
  Pending = "pending",
  InProgress = "in_progress",
  // ...
}
```

### EnumStyle: object
```typescript
export const Status = {
  Pending: "pending",
  InProgress: "in_progress",
  // ...
} as const;
export type Status = (typeof Status)[keyof typeof Status];
```

### OptionalType: default
```typescript
// omitempty fields get ?:, bare pointers get | null
description?: string;      // *string with omitempty → optional
due_date?: string;         // *time.Time with omitempty → optional
assignee?: string;         // *string with omitempty → optional
```

### OptionalType: null
```typescript
// All optional fields use | null
description: string | null;
due_date: string | null;
```

### OptionalType: undefined
```typescript
// All optional fields use ?:
description?: string;
due_date?: string;
```

## File Structure

```
config-showcase/
├── main.go           # Custom generation logic
├── api/types.go      # Go types with comments
└── client/src/
    ├── union/        # EnumStyle: union
    ├── enum/         # EnumStyle: enum
    ├── const/        # EnumStyle: const_enum
    ├── object/       # EnumStyle: object
    ├── opt-default/  # OptionalType: default
    ├── opt-null/     # OptionalType: null
    ├── opt-undef/    # OptionalType: undefined
    └── no-comments/  # PreserveComments: none
```
