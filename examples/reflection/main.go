package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/broady/tygor"
	"github.com/broady/tygor/examples/reflection/api"
	"github.com/broady/tygor/tygorgen"
)

// --- Handlers ---

// [snippet:handlers]

func ListUsers(ctx context.Context, req *api.ListUsersParams) (*api.PagedResponse[api.User], error) {
	// Simulate database query with pagination
	users := []api.User{
		{ID: 1, Username: "alice", Email: "alice@example.com", Role: "admin"},
		{ID: 2, Username: "bob", Email: "bob@example.com", Role: "user"},
		{ID: 3, Username: "charlie", Email: "charlie@example.com", Role: "user"},
	}

	// Apply role filter if specified
	filtered := users
	if req.Role != "" {
		filtered = []api.User{}
		for _, u := range users {
			if u.Role == req.Role {
				filtered = append(filtered, u)
			}
		}
	}

	// Calculate pagination
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 10
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	if start > len(filtered) {
		start = len(filtered)
	}

	return &api.PagedResponse[api.User]{
		Data:     filtered[start:end],
		Total:    len(filtered),
		Page:     page,
		PageSize: pageSize,
		HasMore:  end < len(filtered),
	}, nil
}

func GetUser(ctx context.Context, req *api.GetUserParams) (*api.Result[api.User], error) {
	// Simulate database lookup
	if req.ID == 1 {
		user := api.User{
			ID:       1,
			Username: "alice",
			Email:    "alice@example.com",
			Role:     "admin",
		}
		return &api.Result[api.User]{
			Success: true,
			Data:    &user,
		}, nil
	}

	errMsg := "user not found"
	return &api.Result[api.User]{
		Success: false,
		Error:   &errMsg,
	}, nil
}

func CreatePost(ctx context.Context, req *api.CreatePostParams) (*api.Result[api.Post], error) {
	// Simulate post creation
	post := api.Post{
		ID:       123,
		Title:    req.Title,
		Content:  req.Content,
		AuthorID: req.AuthorID,
	}

	return &api.Result[api.Post]{
		Success: true,
		Data:    &post,
	}, nil
}

// [/snippet:handlers]

// --- Main ---

func main() {
	port := flag.String("port", "8080", "Server port")
	genFlag := flag.Bool("gen", false, "Generate TypeScript types using reflection provider")
	outDir := flag.String("out", "./client/src/rpc", "Output directory for generation")
	flag.Parse()

	app := tygor.NewApp()

	users := app.Service("Users")
	users.Register("List", tygor.Query(ListUsers))
	users.Register("Get", tygor.Query(GetUser))

	posts := app.Service("Posts")
	posts.Register("Create", tygor.Exec(CreatePost))

	// Generation Mode
	if *genFlag {
		fmt.Printf("Generating types to %s...\n", *outDir)
		if err := os.MkdirAll(*outDir, 0755); err != nil {
			log.Fatal(err)
		}

		// [snippet:reflection-generation]

		// tygor.Generate uses the reflection provider internally.
		// The reflection provider extracts types from the registered handlers
		// and automatically handles generic type instantiation.
		if err := tygorgen.Generate(app, &tygorgen.Config{
			OutDir:              *outDir,
			PreserveComments:    "default",
			EnumStyle:           "union",
			OptionalType:        "undefined",
			StripPackagePrefix:  "github.com/broady/tygor/examples/reflection/",
		}); err != nil {
			log.Fatalf("Generation failed: %v", err)
		}

		// [/snippet:reflection-generation]

		fmt.Println("Done.")
		return
	}

	// Start Server
	addr := ":" + *port
	fmt.Printf("Server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, app.Handler()); err != nil {
		log.Fatal(err)
	}
}
