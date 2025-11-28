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
	baseConfig := tygorgen.Config{
		SingleFile:       true,
		PreserveComments: "default",
	}

	configs := []struct {
		name   string
		outDir string
		modify func(*tygorgen.Config)
	}{
		{
			name:   "EnumStyle: union (default)",
			outDir: "./client/src/union",
			modify: func(c *tygorgen.Config) {
				c.EnumStyle = "union"
			},
		},
		{
			name:   "EnumStyle: enum (with member docs)",
			outDir: "./client/src/enum",
			modify: func(c *tygorgen.Config) {
				c.EnumStyle = "enum"
			},
		},
		{
			name:   "EnumStyle: const_enum (inlined)",
			outDir: "./client/src/const",
			modify: func(c *tygorgen.Config) {
				c.EnumStyle = "const_enum"
			},
		},
		{
			name:   "EnumStyle: object (runtime accessible)",
			outDir: "./client/src/object",
			modify: func(c *tygorgen.Config) {
				c.EnumStyle = "object"
			},
		},
		{
			name:   "OptionalType: default (§4.9 spec)",
			outDir: "./client/src/opt-default",
			modify: func(c *tygorgen.Config) {
				c.EnumStyle = "union"
				c.OptionalType = "default"
			},
		},
		{
			name:   "OptionalType: null (all | null)",
			outDir: "./client/src/opt-null",
			modify: func(c *tygorgen.Config) {
				c.EnumStyle = "union"
				c.OptionalType = "null"
			},
		},
		{
			name:   "OptionalType: undefined (all ?:)",
			outDir: "./client/src/opt-undef",
			modify: func(c *tygorgen.Config) {
				c.EnumStyle = "union"
				c.OptionalType = "undefined"
			},
		},
		{
			name:   "PreserveComments: none",
			outDir: "./client/src/no-comments",
			modify: func(c *tygorgen.Config) {
				c.EnumStyle = "union"
				c.PreserveComments = "none"
			},
		},
	}

	for _, cfg := range configs {
		config := baseConfig
		config.OutDir = cfg.outDir
		cfg.modify(&config)

		fmt.Printf("Generating: %s -> %s\n", cfg.name, cfg.outDir)
		if err := tygorgen.Generate(app, &config); err != nil {
			log.Fatalf("Failed to generate %s: %v", cfg.name, err)
		}
	}

	fmt.Println("\nDone! Compare the outputs in client/src/*/types.ts")
}
