//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/broady/tygor"
	"github.com/broady/tygor/examples/blog/api"
)

// Dummy handlers for type registration only
func CreateUser(ctx context.Context, req *api.CreateUserRequest) (*api.User, error) {
	return nil, nil
}

func Login(ctx context.Context, req *api.LoginRequest) (*api.LoginResponse, error) {
	return nil, nil
}

func CreatePost(ctx context.Context, req *api.CreatePostRequest) (*api.Post, error) {
	return nil, nil
}

func GetPost(ctx context.Context, req *api.GetPostParams) (*api.Post, error) {
	return nil, nil
}

func ListPosts(ctx context.Context, req *api.ListPostsParams) ([]*api.Post, error) {
	return nil, nil
}

func UpdatePost(ctx context.Context, req *api.UpdatePostRequest) (*api.Post, error) {
	return nil, nil
}

func PublishPost(ctx context.Context, req *api.PublishPostRequest) (*api.Post, error) {
	return nil, nil
}

func CreateComment(ctx context.Context, req *api.CreateCommentRequest) (*api.Comment, error) {
	return nil, nil
}

func ListComments(ctx context.Context, req *api.ListCommentsParams) ([]*api.Comment, error) {
	return nil, nil
}

func main() {
	// Create registry and register all services (same structure as main.go)
	reg := tygor.NewRegistry()

	// User Service
	userService := reg.Service("Users")
	userService.Register("Create", tygor.NewHandler(CreateUser))
	userService.Register("Login", tygor.NewHandler(Login))

	// Post Service
	postService := reg.Service("Posts")
	postService.Register("Get", tygor.NewHandler(GetPost).Method("GET"))
	postService.Register("List", tygor.NewHandler(ListPosts).Method("GET"))
	postService.Register("Create", tygor.NewHandler(CreatePost))
	postService.Register("Update", tygor.NewHandler(UpdatePost))
	postService.Register("Publish", tygor.NewHandler(PublishPost))

	// Comment Service
	commentService := reg.Service("Comments")
	commentService.Register("Create", tygor.NewHandler(CreateComment))
	commentService.Register("List", tygor.NewHandler(ListComments).Method("GET"))

	// Generate TypeScript types
	outDir := "./client/src/rpc"
	fmt.Printf("Generating types to %s...\n", outDir)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatal(err)
	}

	if err := reg.Generate(&tygor.GenConfig{
		OutDir:           outDir,
		PreserveComments: "default",
		EnumStyle:        "union",
		OptionalType:     "undefined",
	}); err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	fmt.Println("Done.")
}
