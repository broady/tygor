# News Server Example

A simple CRUD API demonstrating tygor's basic features.

## Features Demonstrated

- **Type-safe handlers**: Go generics for compile-time safety
- **Mixed HTTP methods**: GET for queries, POST for mutations
- **Query parameter decoding**: Automatic parsing from URL query strings
- **JSON request/response**: Seamless serialization
- **TypeScript client**: Auto-generated types and client code
- **Branded types**: Custom DateTime type with helper functions
- **Type-safe enums**: NewsStatus enum with constants

## Running the Example

```bash
make run    # Start server on :8080
make gen    # Generate TypeScript types
make test   # Build and test
```

Or manually:

```bash
go run main.go                    # Start server
go run main.go -gen               # Generate types
curl http://localhost:8080/News/List?limit=10
```

## Code Overview

### Type Definitions

<!-- [snippet:enum-type] -->
```go title="types.go"
// NewsStatus represents the publication status of a news article.
type NewsStatus string

const (
	// NewsStatusDraft indicates the article is not yet published.
	NewsStatusDraft NewsStatus = "draft"
	// NewsStatusPublished indicates the article is publicly visible.
	NewsStatusPublished NewsStatus = "published"
	// NewsStatusArchived indicates the article has been archived.
	NewsStatusArchived NewsStatus = "archived"
)

```
<!-- [/snippet:enum-type] -->

<!-- [snippet:list-params] -->
```go title="types.go"
// ListNewsParams contains pagination parameters for listing news articles.
type ListNewsParams struct {
	Limit  *int32 `json:"limit" schema:"limit"`
	Offset *int32 `json:"offset" schema:"offset"`
}

```
<!-- [/snippet:list-params] -->

<!-- [snippet:create-params] -->
```go title="types.go"
// CreateNewsParams contains the parameters for creating a new news article.
type CreateNewsParams struct {
	Title string  `json:"title" validate:"required,min=3"`
	Body  *string `json:"body,omitempty"`
}

```
<!-- [/snippet:create-params] -->

### Handlers

<!-- [snippet:handlers] -->
```go title="main.go"
func ListNews(ctx context.Context, req *api.ListNewsParams) ([]*api.News, error) {
	// ...
}

```
<!-- [/snippet:handlers] -->

### App Setup

<!-- [snippet:error-transformer] -->
```go title="main.go"
app := tygor.NewApp().
	WithErrorTransformer(func(err error) *tygor.Error {
		if err.Error() == "database connection failed" {
			return tygor.NewError(tygor.CodeUnavailable, "service unavailable")
		}
		return nil
	})

```
<!-- [/snippet:error-transformer] -->

<!-- [snippet:global-interceptor] -->
```go title="main.go"
app = app.WithUnaryInterceptor(middleware.LoggingInterceptor(logger))

```
<!-- [/snippet:global-interceptor] -->

<!-- [snippet:middleware] -->
```go title="main.go"
app = app.WithMiddleware(middleware.CORS(middleware.CORSAllowAll))

```
<!-- [/snippet:middleware] -->

### Service Registration

<!-- [snippet:cache-control] -->
```go title="main.go"
news.Register("List", tygor.Query(ListNews).
	CacheControl(tygor.CacheConfig{
		MaxAge: 1 * time.Minute,
		Public: true,
	}))

```
<!-- [/snippet:cache-control] -->

<!-- [snippet:handler-interceptor] -->
```go title="main.go"
news.Register("Create", tygor.Exec(CreateNews).
	WithUnaryInterceptor(func(ctx tygor.Context, req any, handler tygor.HandlerFunc) (any, error) {
		ctx.HTTPWriter().Header().Set("X-Created-By", "tygor")
		return handler(ctx, req)
	}))

```
<!-- [/snippet:handler-interceptor] -->

### TypeScript Client

<!-- [snippet:client-setup] -->
```typescript title="index.ts"
const client = createClient(registry, {
  baseUrl: 'http://localhost:8080',
  headers: () => ({
    'Authorization': 'Bearer my-token'
  })
});
```
<!-- [/snippet:client-setup] -->

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/News/List` | List news items with pagination |
| POST | `/News/Create` | Create a new news item |

## File Structure

```
newsserver/
├── main.go           # Server, handlers, registration
├── api/types.go      # Request/response types
├── client/
│   ├── index.ts      # TypeScript client example
│   └── src/rpc/      # Generated types
└── README.md
```
