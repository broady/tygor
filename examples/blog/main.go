package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/broady/tygor"
	"github.com/broady/tygor/examples/blog/api"
	"github.com/broady/tygor/middleware"
)

// In-memory database (for demo purposes)
var (
	dbMu          sync.RWMutex
	users         = make(map[int64]*api.User)
	posts         = make(map[int64]*api.Post)
	comments      = make(map[int64]*api.Comment)
	tokens        = make(map[string]int64) // token -> userID
	nextUserID    int64
	nextPostID    int64
	nextCommentID int64
)

func init() {
	// Seed with demo data
	now := time.Now()

	// Create a demo user
	users[1] = &api.User{
		ID:        1,
		Username:  "alice",
		Email:     "alice@example.com",
		CreatedAt: now,
	}
	nextUserID = 2

	// Create demo token for alice
	tokens["demo-token-alice"] = 1

	// Create demo posts
	posts[1] = &api.Post{
		ID:        1,
		AuthorID:  1,
		Title:     "Welcome to the Blog",
		Content:   "This is the first post on our new blog platform!",
		Published: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	posts[2] = &api.Post{
		ID:        2,
		AuthorID:  1,
		Title:     "Draft Post",
		Content:   "This is a draft post that hasn't been published yet.",
		Published: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	nextPostID = 3

	// Create demo comment
	comments[1] = &api.Comment{
		ID:        1,
		PostID:    1,
		AuthorID:  1,
		Content:   "First comment!",
		CreatedAt: now,
	}
	nextCommentID = 2
}

// --- Authentication Helpers ---

type contextKey string

const userIDKey contextKey = "user_id"

func getUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}

func requireAuth(ctx context.Context, req any, info *tygor.RPCInfo, handler tygor.HandlerFunc) (any, error) {
	// Extract token from request headers
	httpReq := tygor.RequestFromContext(ctx)
	if httpReq == nil {
		return nil, tygor.NewError(tygor.CodeUnauthenticated, "no request in context")
	}

	authHeader := httpReq.Header.Get("Authorization")
	if authHeader == "" {
		return nil, tygor.NewError(tygor.CodeUnauthenticated, "missing authorization header")
	}

	// Extract bearer token
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, tygor.NewError(tygor.CodeUnauthenticated, "invalid authorization header format")
	}

	token := parts[1]

	// Look up user by token
	dbMu.RLock()
	userID, ok := tokens[token]
	dbMu.RUnlock()

	if !ok {
		return nil, tygor.NewError(tygor.CodeUnauthenticated, "invalid token")
	}

	// Add user ID to context
	ctx = context.WithValue(ctx, userIDKey, userID)

	return handler(ctx, req)
}

// --- User Service Handlers ---

func CreateUser(ctx context.Context, req *api.CreateUserRequest) (*api.User, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	// Check if email already exists
	for _, u := range users {
		if u.Email == req.Email {
			return nil, tygor.NewError(tygor.CodeInvalidArgument, "email already registered")
		}
		if u.Username == req.Username {
			return nil, tygor.NewError(tygor.CodeInvalidArgument, "username already taken")
		}
	}

	user := &api.User{
		ID:        nextUserID,
		Username:  req.Username,
		Email:     req.Email,
		CreatedAt: time.Now(),
	}
	users[nextUserID] = user
	nextUserID++

	return user, nil
}

func Login(ctx context.Context, req *api.LoginRequest) (*api.LoginResponse, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()

	// Find user by email
	var user *api.User
	for _, u := range users {
		if u.Email == req.Email {
			user = u
			break
		}
	}

	if user == nil {
		return nil, tygor.NewError(tygor.CodeUnauthenticated, "invalid credentials")
	}

	// In production, verify password hash here
	// For demo, just generate a token
	tokenBytes := make([]byte, 16)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	dbMu.RUnlock()
	dbMu.Lock()
	tokens[token] = user.ID
	dbMu.Unlock()
	dbMu.RLock()

	return &api.LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

// --- Post Service Handlers ---

func CreatePost(ctx context.Context, req *api.CreatePostRequest) (*api.Post, error) {
	userID, ok := getUserID(ctx)
	if !ok {
		return nil, tygor.NewError(tygor.CodeInternal, "user ID not in context")
	}

	dbMu.Lock()
	defer dbMu.Unlock()

	now := time.Now()
	post := &api.Post{
		ID:        nextPostID,
		AuthorID:  userID,
		Title:     req.Title,
		Content:   req.Content,
		Published: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	posts[nextPostID] = post
	nextPostID++

	return post, nil
}

func GetPost(ctx context.Context, req *api.GetPostParams) (*api.Post, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()

	post, ok := posts[req.PostID]
	if !ok {
		return nil, tygor.NewError(tygor.CodeNotFound, "post not found")
	}

	// Check if post is published or user is the author
	if !post.Published {
		if userID, ok := getUserID(ctx); !ok || userID != post.AuthorID {
			return nil, tygor.NewError(tygor.CodePermissionDenied, "cannot view unpublished post")
		}
	}

	return post, nil
}

func ListPosts(ctx context.Context, req *api.ListPostsParams) ([]*api.Post, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Filter posts
	var result []*api.Post
	for _, post := range posts {
		// Filter by author if specified
		if req.AuthorID != nil && post.AuthorID != *req.AuthorID {
			continue
		}

		// Filter by published status if specified
		if req.Published != nil && post.Published != *req.Published {
			continue
		}

		// Skip unpublished posts unless user is the author
		if !post.Published {
			if userID, ok := getUserID(ctx); !ok || userID != post.AuthorID {
				continue
			}
		}

		result = append(result, post)
	}

	// Apply offset and limit
	start := int(req.Offset)
	if start >= len(result) {
		return []*api.Post{}, nil
	}

	end := start + int(req.Limit)
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

func UpdatePost(ctx context.Context, req *api.UpdatePostRequest) (*api.Post, error) {
	userID, ok := getUserID(ctx)
	if !ok {
		return nil, tygor.NewError(tygor.CodeInternal, "user ID not in context")
	}

	dbMu.Lock()
	defer dbMu.Unlock()

	post, ok := posts[req.PostID]
	if !ok {
		return nil, tygor.NewError(tygor.CodeNotFound, "post not found")
	}

	// Check authorization
	if post.AuthorID != userID {
		return nil, tygor.NewError(tygor.CodePermissionDenied, "not authorized to edit this post")
	}

	// Update fields
	if req.Title != nil {
		post.Title = *req.Title
	}
	if req.Content != nil {
		post.Content = *req.Content
	}
	post.UpdatedAt = time.Now()

	return post, nil
}

func PublishPost(ctx context.Context, req *api.PublishPostRequest) (*api.Post, error) {
	userID, ok := getUserID(ctx)
	if !ok {
		return nil, tygor.NewError(tygor.CodeInternal, "user ID not in context")
	}

	dbMu.Lock()
	defer dbMu.Unlock()

	post, ok := posts[req.PostID]
	if !ok {
		return nil, tygor.NewError(tygor.CodeNotFound, "post not found")
	}

	// Check authorization
	if post.AuthorID != userID {
		return nil, tygor.NewError(tygor.CodePermissionDenied, "not authorized to publish this post")
	}

	post.Published = true
	post.UpdatedAt = time.Now()

	return post, nil
}

// --- Comment Service Handlers ---

func CreateComment(ctx context.Context, req *api.CreateCommentRequest) (*api.Comment, error) {
	userID, ok := getUserID(ctx)
	if !ok {
		return nil, tygor.NewError(tygor.CodeInternal, "user ID not in context")
	}

	dbMu.Lock()
	defer dbMu.Unlock()

	// Verify post exists and is published
	post, ok := posts[req.PostID]
	if !ok {
		return nil, tygor.NewError(tygor.CodeNotFound, "post not found")
	}
	if !post.Published {
		return nil, tygor.NewError(tygor.CodePermissionDenied, "cannot comment on unpublished post")
	}

	comment := &api.Comment{
		ID:        nextCommentID,
		PostID:    req.PostID,
		AuthorID:  userID,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}
	comments[nextCommentID] = comment
	nextCommentID++

	return comment, nil
}

func ListComments(ctx context.Context, req *api.ListCommentsParams) ([]*api.Comment, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()

	// Verify post exists and is published
	post, ok := posts[req.PostID]
	if !ok {
		return nil, tygor.NewError(tygor.CodeNotFound, "post not found")
	}

	// Check if user can view this post
	if !post.Published {
		if userID, ok := getUserID(ctx); !ok || userID != post.AuthorID {
			return nil, tygor.NewError(tygor.CodePermissionDenied, "cannot view comments on unpublished post")
		}
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 50
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Filter comments
	var result []*api.Comment
	for _, comment := range comments {
		if comment.PostID == req.PostID {
			result = append(result, comment)
		}
	}

	// Apply offset and limit
	start := int(req.Offset)
	if start >= len(result) {
		return []*api.Comment{}, nil
	}

	end := start + int(req.Limit)
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], nil
}

// --- Main ---

func main() {
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	// Configure structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create registry with global middleware and interceptors
	reg := tygor.NewRegistry().
		WithUnaryInterceptor(middleware.LoggingInterceptor(logger)).
		WithMiddleware(middleware.CORS(middleware.DefaultCORSConfig()))

	// User Service (public endpoints)
	userService := reg.Service("Users")
	userService.Register("Create", tygor.Unary(CreateUser))
	userService.Register("Login", tygor.Unary(Login))

	// Post Service (mixed public/private endpoints)
	postService := reg.Service("Posts")

	// Public endpoints
	postService.Register("Get", tygor.UnaryGet(GetPost))
	postService.Register("List", tygor.UnaryGet(ListPosts).Cache(30*time.Second))

	// Private endpoints (require authentication)
	postService.Register("Create",
		tygor.Unary(CreatePost).WithUnaryInterceptor(requireAuth))
	postService.Register("Update",
		tygor.Unary(UpdatePost).WithUnaryInterceptor(requireAuth))
	postService.Register("Publish",
		tygor.Unary(PublishPost).WithUnaryInterceptor(requireAuth))

	// Comment Service (requires authentication)
	commentService := reg.Service("Comments").WithUnaryInterceptor(requireAuth)
	commentService.Register("Create", tygor.Unary(CreateComment))
	commentService.Register("List", tygor.UnaryGet(ListComments))

	// Start server
	addr := ":" + *port
	fmt.Printf("Blog server listening on %s\n", addr)
	fmt.Println("\nExample requests:")
	fmt.Println("  # Create user:")
	fmt.Printf("  curl -X POST http://localhost:%s/Users/Create -H 'Content-Type: application/json' -d '{\"username\":\"bob\",\"email\":\"bob@example.com\",\"password\":\"password123\"}'\n", *port)
	fmt.Println("\n  # Login:")
	fmt.Printf("  curl -X POST http://localhost:%s/Users/Login -H 'Content-Type: application/json' -d '{\"email\":\"alice@example.com\",\"password\":\"anything\"}'\n", *port)
	fmt.Println("\n  # List posts:")
	fmt.Printf("  curl http://localhost:%s/Posts/List?limit=10\n", *port)
	fmt.Println("\n  # Create post (requires auth):")
	fmt.Printf("  curl -X POST http://localhost:%s/Posts/Create -H 'Authorization: Bearer demo-token-alice' -H 'Content-Type: application/json' -d '{\"title\":\"My New Post\",\"content\":\"This is the content of my post.\"}'\n", *port)
	fmt.Println()

	if err := http.ListenAndServe(addr, reg.Handler()); err != nil {
		log.Fatal(err)
	}
}
