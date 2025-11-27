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

<!-- [snippet:request-types] -->
```go title="types.go"
// ListNewsParams contains pagination parameters for listing news articles.
type ListNewsParams struct {
	// Limit is the maximum number of articles to return.
	Limit *int32 `json:"limit" schema:"limit"`
	// Offset is the number of articles to skip.
	Offset *int32 `json:"offset" schema:"offset"`
}

// CreateNewsParams contains the parameters for creating a new news article.
type CreateNewsParams struct {
	// Title is the article headline (required, 3-100 characters).
	Title string `json:"title" validate:"required,min=3"`
	// Body is the optional article content.
	Body *string `json:"body,omitempty"`
}

```
<!-- [/snippet:request-types] -->

### Handlers

<!-- [snippet:handlers] -->
```go title="main.go"
func ListNews(ctx context.Context, req *api.ListNewsParams) ([]*api.News, error) {
	// Simulate DB
	body := "This is the body"
	now := time.Now()
	return []*api.News{
		{ID: 1, Title: "News 1", Body: &body, Status: api.NewsStatusPublished, CreatedAt: &now},
		{ID: 2, Title: "News 2", Status: api.NewsStatusDraft, CreatedAt: &now},
	}, nil
}

func CreateNews(ctx context.Context, req *api.CreateNewsParams) (*api.News, error) {
	if req.Title == "error" {
		return nil, tygor.NewError(tygor.CodeInvalidArgument, "simulated error")
	}
	now := time.Now()
	return &api.News{
		ID:        123,
		Title:     req.Title,
		Body:      req.Body,
		Status:    api.NewsStatusDraft, // New articles start as drafts
		CreatedAt: &now,
	}, nil
}

```
<!-- [/snippet:handlers] -->

### App Setup

<!-- [snippet:app-setup] -->
```go title="main.go"
// 1. Create App
app := tygor.NewApp().
	WithErrorTransformer(func(err error) *tygor.Error {
		// Example custom error mapping
		if err.Error() == "database connection failed" {
			return tygor.NewError(tygor.CodeUnavailable, "db down")
		}
		return nil
	}).
	WithUnaryInterceptor(middleware.LoggingInterceptor(logger)).
	WithMiddleware(middleware.CORS(middleware.DefaultCORSConfig()))
```
<!-- [/snippet:app-setup] -->

### Service Registration

<!-- [snippet:service-registration] -->
```go title="main.go"
// 2. Register Services
news := app.Service("News")

news.Register("List", tygor.Query(ListNews).
	CacheControl(tygor.CacheConfig{
		MaxAge: 1 * time.Minute,
		Public: true,
	}))

news.Register("Create", tygor.Exec(CreateNews).
	WithUnaryInterceptor(func(ctx tygor.Context, req any, handler tygor.HandlerFunc) (any, error) {
		// Example: Set a custom header
		ctx.HTTPWriter().Header().Set("X-Created-By", "Tygorpc")
		return handler(ctx, req)
	}))
```
<!-- [/snippet:service-registration] -->

### TypeScript Client

<!-- [snippet:client-setup] -->
```typescript title="index.ts"
// 1. Create the strictly typed client
const client = createClient(
  registry,
  {
    baseUrl: 'http://localhost:8080',
    headers: () => ({
      'Authorization': 'Bearer my-token'
    })
  }
);
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
