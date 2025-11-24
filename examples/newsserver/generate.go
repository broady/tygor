//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/broady/tygor"
	"github.com/broady/tygor/examples/newsserver/api"
)

// Dummy handlers for type registration only
func ListNews(ctx context.Context, req *api.ListNewsParams) ([]*api.News, error) {
	return nil, nil
}

func CreateNews(ctx context.Context, req *api.CreateNewsParams) (*api.News, error) {
	return nil, nil
}

func main() {
	// Create registry and register all services (same structure as main.go)
	reg := tygor.NewRegistry()

	news := reg.Service("News")
	news.Register("List", tygor.NewHandler(ListNews).Method("GET"))
	news.Register("Create", tygor.NewHandler(CreateNews).Method("POST"))

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
		// Define custom branded types with helper functions
		Frontmatter: `// Branded types for enhanced type safety
export type DateTime = string & { readonly __brand: 'DateTime' };

// DateTime helper functions
export const DateTime = {
  // Create DateTime from string (assumes valid ISO 8601)
  from: (s: string): DateTime => s as DateTime,

  // Get current timestamp as DateTime
  now: (): DateTime => new Date().toISOString() as DateTime,

  // Parse DateTime to Date object
  toDate: (dt: DateTime): Date => new Date(dt),

  // Format DateTime for display
  format: (dt: DateTime, locale = 'en-US'): string => {
    return new Date(dt).toLocaleString(locale);
  },
};
`,
		TypeMappings: map[string]string{
			"time.Time": "DateTime",
		},
	}); err != nil {
		log.Fatalf("Generation failed: %v", err)
	}

	fmt.Println("Done.")
}
