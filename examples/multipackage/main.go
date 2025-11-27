// Example: multipackage demonstrates handling same-named types from different packages.
//
// This example shows how to use StripPackagePrefix to disambiguate types:
//   - v1.User becomes v1_User in TypeScript
//   - v2.User becomes v2_User in TypeScript
//
// Run: go run . -gen -out ./client/src/rpc
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/broady/tygor"
	"github.com/broady/tygor/examples/multipackage/api"
	v1 "github.com/broady/tygor/examples/multipackage/api/v1"
	v2 "github.com/broady/tygor/examples/multipackage/api/v2"
	"github.com/broady/tygor/tygorgen"
)

func main() {
	gen := flag.Bool("gen", false, "Generate TypeScript types")
	out := flag.String("out", "./client/src/rpc", "Output directory for generated files")
	flag.Parse()

	app := tygor.NewApp()

	// Register v1 endpoints
	app.Service("V1Users").Register("Get", tygor.Query(func(ctx context.Context, req *v1.GetUserRequest) (*v1.User, error) {
		return &v1.User{ID: req.ID, Name: "V1 User"}, nil
	}))

	// Register v2 endpoints
	app.Service("V2Users").Register("Get", tygor.Query(func(ctx context.Context, req *v2.GetUserRequest) (*v2.User, error) {
		return &v2.User{ID: req.ID, Name: "V2 User", Email: "v2@example.com", CreatedAt: "2024-01-01T00:00:00Z"}, nil
	}))

	// Register migration endpoint that uses both
	app.Service("Migration").Register("Migrate", tygor.Exec(func(ctx context.Context, req *api.MigrationRequest) (*api.MigrationResponse, error) {
		return &api.MigrationResponse{
			Success: true,
			V1User:  req.V1User,
			V2User:  req.V2User,
		}, nil
	}))

	if *gen {
		fmt.Println("Generating types to", *out)
		if err := tygorgen.Generate(app, &tygorgen.Config{
			OutDir: *out,
			// StripPackagePrefix disambiguates same-named types from different packages.
			// Without this, both v1.User and v2.User would become "User" (collision!).
			// With this, they become "v1_User" and "v2_User".
			// Types from the api package itself get no prefix (MigrationRequest).
			StripPackagePrefix: "github.com/broady/tygor/examples/multipackage/api",
		}); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Done.")
		os.Exit(0)
	}

	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", app.Handler()))
}
