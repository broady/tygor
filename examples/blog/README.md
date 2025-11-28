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
├── client/
│   ├── index.ts          # TypeScript client example
│   └── src/rpc/          # Generated TypeScript types
└── README.md             # This file
```

## Key Patterns

### Authentication Interceptor

<!-- [snippet:auth-interceptor] -->
```go title="main.go"
func requireAuth(ctx tygor.Context, req any, handler tygor.HandlerFunc) (any, error) {
	// ...
}

```
<!-- [/snippet:auth-interceptor] -->

Applied at different levels (handler-level with `WithUnaryInterceptor` on specific endpoints, service-level on the entire `Comments` service):

<!-- [snippet:mixed-endpoints] -->
```go title="main.go"
// Post Service (mixed public/private endpoints)
postService := app.Service("Posts")

// Public endpoints
postService.Register("Get", tygor.Query(GetPost))
postService.Register("List", tygor.Query(ListPosts).CacheControl(tygor.CacheConfig{
	MaxAge: 30 * time.Second,
	Public: true,
}))

// Private endpoints (require authentication)
postService.Register("Create",
	tygor.Exec(CreatePost).WithUnaryInterceptor(requireAuth))
postService.Register("Update",
	tygor.Exec(UpdatePost).WithUnaryInterceptor(requireAuth))
postService.Register("Publish",
	tygor.Exec(PublishPost).WithUnaryInterceptor(requireAuth))

// Comment Service (requires authentication)
commentService := app.Service("Comments").WithUnaryInterceptor(requireAuth)
commentService.Register("Create", tygor.Exec(CreateComment))
commentService.Register("List", tygor.Query(ListComments))
```
<!-- [/snippet:mixed-endpoints] -->

### Authorization in Handlers

<!-- [snippet:authorization-check] -->
```go title="main.go"
func UpdatePost(ctx context.Context, req *api.UpdatePostRequest) (*api.Post, error) {
	// ...
}

```
<!-- [/snippet:authorization-check] -->

### Mixed Public/Private Endpoints

<!-- [snippet:mixed-endpoints] -->
```go title="main.go"
// Post Service (mixed public/private endpoints)
postService := app.Service("Posts")

// Public endpoints
postService.Register("Get", tygor.Query(GetPost))
postService.Register("List", tygor.Query(ListPosts).CacheControl(tygor.CacheConfig{
	MaxAge: 30 * time.Second,
	Public: true,
}))

// Private endpoints (require authentication)
postService.Register("Create",
	tygor.Exec(CreatePost).WithUnaryInterceptor(requireAuth))
postService.Register("Update",
	tygor.Exec(UpdatePost).WithUnaryInterceptor(requireAuth))
postService.Register("Publish",
	tygor.Exec(PublishPost).WithUnaryInterceptor(requireAuth))

// Comment Service (requires authentication)
commentService := app.Service("Comments").WithUnaryInterceptor(requireAuth)
commentService.Register("Create", tygor.Exec(CreateComment))
commentService.Register("List", tygor.Query(ListComments))
```
<!-- [/snippet:mixed-endpoints] -->

### Query Parameters with Validation

<!-- [snippet:api:query-params] -->
```go title="types.go"
// ListPostsParams are the query parameters for listing posts.
type ListPostsParams struct {
	AuthorID  *int64 `json:"author_id" schema:"author_id"`
	Published *bool  `json:"published" schema:"published"`
	Limit     int32  `json:"limit" schema:"limit"`
	Offset    int32  `json:"offset" schema:"offset"`
}

```
<!-- [/snippet:api:query-params] -->

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

## TypeScript Client

### Client Setup

<!-- [snippet:client:client-setup] -->
```typescript title="index.ts"
// Create a basic client (for public endpoints)
const client = createClient(registry, {
  baseUrl: "http://localhost:8080",
});
```
<!-- [/snippet:client:client-setup] -->

### Authenticated Client

<!-- [snippet:client:client-auth] -->
```typescript title="index.ts"
// Create an authenticated client
function createAuthClient(token: string) {
  return createClient(registry, {
    baseUrl: "http://localhost:8080",
    headers: () => ({
      Authorization: `Bearer ${token}`,
    }),
  });
}
```
<!-- [/snippet:client:client-auth] -->

### Login Flow

<!-- [snippet:client:client-login] -->
```typescript title="index.ts"
// Login to get a token
const loginResult = await client.Users.Login({
  email: "alice@example.com",
  password: "anything",
});
console.log("Logged in as:", loginResult.user?.username);

// Create authenticated client with the token
const authClient = createAuthClient(loginResult.token);
```
<!-- [/snippet:client:client-login] -->

### Making RPC Calls

<!-- [snippet:client:client-calls] -->
```typescript title="index.ts"
// Public endpoint: list published posts
const posts = await client.Posts.List({
  published: true,
  limit: 10,
  offset: 0,
});
console.log(`Found ${posts.length} published posts`);

// Authenticated endpoint: create a new post
const newPost = await authClient.Posts.Create({
  title: "My New Blog Post",
  content: "This is the content of my blog post.",
});
console.log("Created post:", newPost.id, newPost.title);

// Publish the post
const published = await authClient.Posts.Publish({
  post_id: newPost.id,
});
console.log("Published:", published.published);
```
<!-- [/snippet:client:client-calls] -->

### Error Handling

<!-- [snippet:client:client-errors] -->
```typescript title="index.ts"
// Error handling example
async function handleErrors() {
  try {
    await client.Posts.Get({ post_id: 99999 });
  } catch (e) {
    if (e instanceof ServerError) {
      // Structured error from the server
      console.error(`Error [${e.code}]: ${e.message}`);
      // e.details contains validation errors, etc.
    } else {
      throw e; // Re-throw unexpected errors
    }
  }
}
```
<!-- [/snippet:client:client-errors] -->
