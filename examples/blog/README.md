# Blog Example

This example demonstrates a more complex tygor RPC application with multiple services, authentication, authorization, and different HTTP methods.

## Features Demonstrated

- **Multiple Services**: Users, Posts, and Comments services
- **Authentication**: Token-based authentication using interceptors
- **Authorization**: Role-based access control (post ownership)
- **Mixed HTTP Methods**: GET for queries, POST for mutations
- **Validation**: Request validation using struct tags
- **Caching**: Response caching for list endpoints
- **Error Handling**: Proper error codes and messages
- **Middleware**: CORS and structured logging
- **Service-level Interceptors**: Authentication applied to entire service
- **Handler-level Interceptors**: Fine-grained auth on specific endpoints

## Running the Server

```bash
cd examples/blog
go run main.go
```

Server will start on `http://localhost:8080`

## API Endpoints

### User Service (Public)

**Create User**
```bash
curl -X POST http://localhost:8080/Users/Create \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","email":"alice@example.com","password":"password123"}'
```

**Login**
```bash
curl -X POST http://localhost:8080/Users/Login \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"anything"}'
```

Response includes a token:
```json
{
  "token": "abc123...",
  "user": {"id": 1, "username": "alice", ...}
}
```

### Post Service (Mixed)

**List Posts (Public, Cached)**
```bash
# All published posts
curl http://localhost:8080/Posts/List?limit=10

# Posts by specific author
curl http://localhost:8080/Posts/List?author_id=1&limit=10

# Only published or drafts
curl http://localhost:8080/Posts/List?published=true
```

**Get Post (Public)**
```bash
curl http://localhost:8080/Posts/Get?post_id=1
```

**Create Post (Requires Auth)**
```bash
curl -X POST http://localhost:8080/Posts/Create \
  -H 'Authorization: Bearer demo-token-alice' \
  -H 'Content-Type: application/json' \
  -d '{"title":"My First Post","content":"This is my first blog post!"}'
```

**Update Post (Requires Auth + Ownership)**
```bash
curl -X POST http://localhost:8080/Posts/Update \
  -H 'Authorization: Bearer demo-token-alice' \
  -H 'Content-Type: application/json' \
  -d '{"post_id":1,"title":"Updated Title"}'
```

**Publish Post (Requires Auth + Ownership)**
```bash
curl -X POST http://localhost:8080/Posts/Publish \
  -H 'Authorization: Bearer demo-token-alice' \
  -H 'Content-Type: application/json' \
  -d '{"post_id":1}'
```

### Comment Service (Requires Auth)

**Create Comment**
```bash
curl -X POST http://localhost:8080/Comments/Create \
  -H 'Authorization: Bearer demo-token-alice' \
  -H 'Content-Type: application/json' \
  -d '{"post_id":1,"content":"Great post!"}'
```

**List Comments**
```bash
curl http://localhost:8080/Comments/List?post_id=1 \
  -H 'Authorization: Bearer demo-token-alice'
```

## Code Structure

```
examples/blog/
├── api/
│   └── types.go          # Request/Response types
├── main.go               # Application setup and handlers
└── README.md             # This file
```

## Key Patterns

### Authentication Interceptor

```go
func requireAuth(ctx *tygor.Context, req any, handler tygor.HandlerFunc) (any, error) {
    // Extract and validate token from Authorization header via ctx.HTTPRequest()
    // Add user ID to context with context.WithValue
    // Call next handler
}
```

Applied at different levels:

```go
// Handler-level (specific endpoints)
postService.Register("Create",
    tygor.Exec(CreatePost).WithUnaryInterceptor(requireAuth))

// Service-level (all endpoints)
commentService := app.Service("Comments").WithUnaryInterceptor(requireAuth)
```

### Authorization in Handlers

```go
func UpdatePost(ctx context.Context, req *api.UpdatePostRequest) (*api.Post, error) {
    userID, _ := getUserID(ctx)

    post := findPost(req.PostID)
    if post.AuthorID != userID {
        return nil, tygor.NewError(tygor.CodePermissionDenied,
            "not authorized to edit this post")
    }
    // ... update post
}
```

### Mixed Public/Private Endpoints

```go
postService := app.Service("Posts")

// Public endpoints
postService.Register("Get", tygor.Query(GetPost))
postService.Register("List", tygor.Query(ListPosts))

// Private endpoints (require auth)
postService.Register("Create",
    tygor.Exec(CreatePost).WithUnaryInterceptor(requireAuth))
```

### Query Parameters with Validation

```go
type ListPostsParams struct {
    AuthorID  *int64 `schema:"author_id"`
    Published *bool  `schema:"published"`
    Limit     int32  `schema:"limit"`
    Offset    int32  `schema:"offset"`
}

func ListPosts(ctx context.Context, req *ListPostsParams) ([]*Post, error) {
    // Parameters automatically decoded and validated
}
```

## Demo Data

The server starts with demo data:
- User: `alice@example.com` (ID: 1)
- Token: `demo-token-alice`
- Post 1: "Welcome to the Blog" (published)
- Post 2: "Draft Post" (unpublished, only visible to alice)
- Comment 1: "First comment!" on Post 1

## Error Handling

The example demonstrates proper error codes:

- `invalid_argument`: Validation failures, duplicate emails
- `unauthenticated`: Missing/invalid tokens
- `permission_denied`: Unauthorized actions (editing others' posts)
- `not_found`: Missing resources
- `internal`: Server errors

Example error response:
```json
{
  "code": "permission_denied",
  "message": "not authorized to edit this post"
}
```
