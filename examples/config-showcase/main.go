// Example: config-showcase demonstrates different TypeScript generation configurations.
//
// This example generates the same Go types with different config options to show
// how each setting affects the output. Run with -gen to generate all variants.
//
// Generated outputs:
//   - client/src/union/       - EnumStyle: "union" (default, cleanest for most uses)
//   - client/src/enum/        - EnumStyle: "enum" (TypeScript enums with member docs)
//   - client/src/const/       - EnumStyle: "const_enum" (inlined at compile time)
//   - client/src/object/      - EnumStyle: "object" (const objects, runtime accessible)
//   - client/src/opt-default/ - OptionalType: "default" (§4.9: omitempty→?:, pointers→|null)
//   - client/src/opt-null/    - OptionalType: "null" (all optional fields use | null)
//   - client/src/opt-undef/   - OptionalType: "undefined" (all optional fields use ?:)
//   - client/src/no-comments/ - PreserveComments: "none" (no JSDoc comments)
//
// Run: go run . -gen
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/broady/tygor"
	"github.com/broady/tygor/examples/config-showcase/api"
	"github.com/broady/tygor/tygorgen"
)

func main() {
	gen := flag.Bool("gen", false, "Generate TypeScript types with different configs")
	flag.Parse()

	app := tygor.NewApp()

	// Register handlers
	tasks := app.Service("Tasks")
	tasks.Register("Create", tygor.Exec(func(ctx context.Context, req *api.CreateTaskRequest) (*api.Task, error) {
		return &api.Task{ID: "1", Title: req.Title, Status: api.StatusPending, Priority: req.Priority}, nil
	}))
	tasks.Register("List", tygor.Query(func(ctx context.Context, req *api.ListTasksParams) ([]*api.Task, error) {
		return nil, nil
	}))

	if *gen {
		generateAll(app)
		return
	}

	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", app.Handler()))
}

func generateAll(app *tygor.App) {
	// base returns a generator with common settings
	base := func() *tygorgen.Generator {
		return tygorgen.FromApp(app).
			SingleFile().
			PreserveComments("default")
	}

	configs := []struct {
		name string
		gen  func() (*tygorgen.GenerateResult, error)
	}{
		{
			name: "EnumStyle: union (default) -> ./client/src/union",
			gen:  func() (*tygorgen.GenerateResult, error) { return base().EnumStyle("union").ToDir("./client/src/union") },
		},
		{
			name: "EnumStyle: enum (with member docs) -> ./client/src/enum",
			gen:  func() (*tygorgen.GenerateResult, error) { return base().EnumStyle("enum").ToDir("./client/src/enum") },
		},
		{
			name: "EnumStyle: const_enum (inlined) -> ./client/src/const",
			gen: func() (*tygorgen.GenerateResult, error) {
				return base().EnumStyle("const_enum").ToDir("./client/src/const")
			},
		},
		{
			name: "EnumStyle: object (runtime accessible) -> ./client/src/object",
			gen: func() (*tygorgen.GenerateResult, error) {
				return base().EnumStyle("object").ToDir("./client/src/object")
			},
		},
		{
			name: "OptionalType: default (§4.9 spec) -> ./client/src/opt-default",
			gen: func() (*tygorgen.GenerateResult, error) {
				return base().EnumStyle("union").OptionalType("default").ToDir("./client/src/opt-default")
			},
		},
		{
			name: "OptionalType: null (all | null) -> ./client/src/opt-null",
			gen: func() (*tygorgen.GenerateResult, error) {
				return base().EnumStyle("union").OptionalType("null").ToDir("./client/src/opt-null")
			},
		},
		{
			name: "OptionalType: undefined (all ?:) -> ./client/src/opt-undef",
			gen: func() (*tygorgen.GenerateResult, error) {
				return base().EnumStyle("union").OptionalType("undefined").ToDir("./client/src/opt-undef")
			},
		},
		{
			name: "PreserveComments: none -> ./client/src/no-comments",
			gen: func() (*tygorgen.GenerateResult, error) {
				return base().EnumStyle("union").PreserveComments("none").ToDir("./client/src/no-comments")
			},
		},
	}

	for _, cfg := range configs {
		fmt.Printf("Generating: %s\n", cfg.name)
		if _, err := cfg.gen(); err != nil {
			log.Fatalf("Failed to generate %s: %v", cfg.name, err)
		}
	}

	fmt.Println("\nDone! Compare the outputs in client/src/*/types.ts")
}
