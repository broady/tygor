# Reflection Provider Example

This example demonstrates tygor's **reflection provider**, which uses runtime reflection to extract type information. The reflection provider is particularly useful for handling **generic type instantiation** and types from external packages where source code analysis isn't available.

## When to Use the Reflection Provider

Use the **reflection provider** when:
- You need to instantiate generic types (e.g., `PagedResponse[User]`, `Result[Post]`)
- Working with types from external packages where source isn't available
- Source analysis would be too complex (deeply nested generics, constraints)

Use the **source provider** (default) when:
- You have access to source code
- You want better comment preservation
- Types are in your own codebase
- You need advanced features like deprecated warnings

## Overview

This example shows a simple API with:
- **Generic response wrappers**: `PagedResponse[T]` and `Result[T]`
- **Multiple instantiations**: `PagedResponse[User]`, `PagedResponse[Post]`, `Result[User]`, `Result[Post]`
- **Type-safe client**: Fully typed TypeScript client with autocomplete

## Key Concepts

### Generic Types

<!-- [snippet:generic-types] -->
```go title="types.go"
// PagedResponse wraps paginated data with metadata.
// This generic type demonstrates reflection provider's ability to handle
// instantiated generics (e.g., PagedResponse[User], PagedResponse[Post]).
type PagedResponse[T any] struct {
	// Data contains the page of results
	Data []T `json:"data"`
	// Total is the total number of items across all pages
	Total int `json:"total"`
	// Page is the current page number (1-indexed)
	Page int `json:"page"`
	// PageSize is the number of items per page
	PageSize int `json:"page_size"`
	// HasMore indicates if there are more pages available
	HasMore bool `json:"has_more"`
}

// Result wraps operation results with success/error status.
// This pattern is common when working with types from external packages
// where source analysis isn't available.
type Result[T any] struct {
	// Success indicates if the operation succeeded
	Success bool `json:"success"`
	// Data contains the result if successful
	Data *T `json:"data,omitempty"`
	// Error contains the error message if failed
	Error *string `json:"error,omitempty"`
}

```
<!-- [/snippet:generic-types] -->

### Concrete Types

<!-- [snippet:concrete-types] -->
```go title="types.go"
// User represents a user in the system.
type User struct {
	ID       int32  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// Post represents a blog post.
type Post struct {
	ID       int32  `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	AuthorID int32  `json:"author_id"`
}

```
<!-- [/snippet:concrete-types] -->

### Request Types

<!-- [snippet:request-types] -->
```go title="types.go"
// ListUsersParams contains pagination parameters for listing users.
type ListUsersParams struct {
	Page     int    `json:"page" schema:"page"`
	PageSize int    `json:"page_size" schema:"page_size"`
	Role     string `json:"role,omitempty" schema:"role"`
}

// GetUserParams contains parameters for fetching a single user.
type GetUserParams struct {
	ID int32 `json:"id" schema:"id"`
}

// CreatePostParams contains parameters for creating a post.
type CreatePostParams struct {
	Title    string `json:"title" validate:"required,min=3"`
	Content  string `json:"content" validate:"required,min=10"`
	AuthorID int32  `json:"author_id" validate:"required"`
}

```
<!-- [/snippet:request-types] -->

## Handlers

<!-- [snippet:handlers] -->
```go title="main.go"
func ListUsers(ctx context.Context, req *api.ListUsersParams) (*api.PagedResponse[api.User], error) {
	// ...
}

func GetUser(ctx context.Context, req *api.GetUserParams) (*api.Result[api.User], error) {
	// ...
}

func CreatePost(ctx context.Context, req *api.CreatePostParams) (*api.Result[api.Post], error) {
	// ...
}

```
<!-- [/snippet:handlers] -->

## Reflection-Based Generation

The key difference from the default source provider is how types are extracted:

<!-- [snippet:reflection-generation] -->
```go title="main.go"
// The reflection provider extracts types from registered handlers
// and automatically handles generic type instantiation.
_, err := tygorgen.FromApp(app).
	Provider("reflection").
	PreserveComments("default").
	EnumStyle("union").
	OptionalType("undefined").
	StripPackagePrefix("github.com/broady/tygor/examples/reflection/").
	ToDir(*outDir)
if err != nil {
	log.Fatalf("Generation failed: %v", err)
}

```
<!-- [/snippet:reflection-generation] -->

**Key points:**
- You must explicitly provide **instantiated types** (e.g., `reflect.TypeOf(api.PagedResponse[api.User]{})`)
- Each instantiation becomes a separate TypeScript type
- Generic type parameters are resolved at runtime

## TypeScript Client

### Setup

<!-- [snippet:client-setup] -->
```typescript title="index.ts"
const client = createClient(registry, {
  baseUrl: 'http://localhost:8080',
});

```
<!-- [/snippet:client-setup] -->

### Usage

<!-- [snippet:client-calls] -->
```typescript title="index.ts"
async function demonstrateGenerics() {
  // PagedResponse[User] - Generic type instantiated for users
  const usersPage = await client.Users.List({
    page: 1,
    page_size: 10,
    role: 'admin'
  });

  console.log(`Users page ${usersPage.page}/${Math.ceil(usersPage.total / usersPage.page_size)}`);
  if (usersPage.data) {
    console.log(`Found ${usersPage.data.length} of ${usersPage.total} total`);
    usersPage.data.forEach(user => {
      console.log(`- ${user.username} (${user.email}) - ${user.role}`);
    });
  }

  // Result[User] - Generic type for success/error results
  const userResult = await client.Users.Get({ id: 1 });
  if (userResult.success && userResult.data) {
    console.log('User found:', userResult.data.username);
  } else if (userResult.error) {
    console.log('Error:', userResult.error);
  }

  // Result[Post] - Same generic type with different type parameter
  const postResult = await client.Posts.Create({
    title: 'My First Post',
    content: 'This demonstrates generic type instantiation',
    author_id: 1
  });

  if (postResult.success && postResult.data) {
    console.log('Post created:', postResult.data.title);
  }
}

```
<!-- [/snippet:client-calls] -->

## Running the Example

```bash
# Generate TypeScript types
make gen

# Run integration tests
make test

# Start the server
make run

# Run the TypeScript demo
cd client && bun run index.ts
```

## Generated TypeScript

The reflection provider generates properly typed instantiations:

```typescript
// PagedResponse is instantiated for each type parameter
export interface PagedResponseUser {
  data: User[];
  total: number;
  page: number;
  page_size: number;
  has_more: boolean;
}

export interface PagedResponsePost {
  data: Post[];
  total: number;
  page: number;
  page_size: number;
  has_more: boolean;
}

// Result types are similarly instantiated
export interface ResultUser {
  success: boolean;
  data?: User;
  error?: string;
}

export interface ResultPost {
  success: boolean;
  data?: Post;
  error?: string;
}
```

## Comparison: Reflection vs Source Provider

| Feature | Reflection Provider | Source Provider (Default) |
|---------|-------------------|--------------------------|
| Generic instantiation | ✅ Explicit instantiation required | ✅ Automatic from usage |
| External packages | ✅ Works with any type | ❌ Needs source code |
| Comment preservation | ⚠️ Limited | ✅ Full support |
| Type constraints | ⚠️ Runtime only | ✅ Compile-time aware |
| Setup complexity | More explicit | Automatic |
| Use case | Generic wrappers, external types | Standard API types |

## Best Practices

1. **Explicitly list all instantiations**: The reflection provider needs `reflect.TypeOf()` for each generic instantiation
2. **Use for wrapper types**: Perfect for `Result[T]`, `PagedResponse[T]`, `Envelope[T]` patterns
3. **Combine with source provider**: Use reflection for generics, source for regular types (advanced)
4. **Keep it simple**: If you don't need generics, use the default source provider

## Related Examples

- [newsserver](../newsserver) - Basic CRUD with source provider (default)
- [protobuf](../protobuf) - Using protobuf-generated types
- [blog](../blog) - Multi-service with authentication
