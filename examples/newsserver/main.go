package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/broady/tygor"
	"github.com/broady/tygor/examples/newsserver/api"
	"github.com/broady/tygor/middleware"
	"github.com/broady/tygor/tygorgen"
)

// --- Handlers ---

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

// --- Main ---

func main() {
	genFlag := flag.Bool("gen", false, "Generate TypeScript types and manifest")
	outDir := flag.String("out", "./client/src/rpc", "Output directory for generation")
	flag.Parse()

	// Configure structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// 1. Create Registry
	reg := tygor.NewRegistry().
		WithErrorTransformer(func(err error) *tygor.Error {
			// Example custom error mapping
			if err.Error() == "database connection failed" {
				return tygor.NewError(tygor.CodeUnavailable, "db down")
			}
			return nil
		}).
		WithUnaryInterceptor(middleware.LoggingInterceptor(logger)).
		WithMiddleware(middleware.CORS(middleware.DefaultCORSConfig()))

	// 2. Register Services
	news := reg.Service("News")

	news.Register("List", tygor.UnaryGet(ListNews).
		CacheControl(tygor.CacheConfig{
			MaxAge: 1 * time.Minute,
			Public: true,
		}))

	news.Register("Create", tygor.Unary(CreateNews).
		WithUnaryInterceptor(func(ctx context.Context, req any, info *tygor.RPCInfo, handler tygor.HandlerFunc) (any, error) {
			// Example: Set a custom header
			tygor.SetHeader(ctx, "X-Created-By", "Tygorpc")
			return handler(ctx, req)
		}))

	// 3. Generation Mode
	if *genFlag {
		fmt.Printf("Generating types to %s...\n", *outDir)
		if err := os.MkdirAll(*outDir, 0755); err != nil {
			log.Fatal(err)
		}
		if err := tygorgen.Generate(reg, &tygorgen.Config{
			OutDir:           *outDir,
			PreserveComments: "default",
			EnumStyle:        "union",
			OptionalType:     "undefined",
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
		return
	}

	// 4. Start Server
	fmt.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", reg.Handler()); err != nil {
		log.Fatal(err)
	}
}
