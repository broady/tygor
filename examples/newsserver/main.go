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

// [snippet:handlers collapse]
func ListNews(ctx context.Context, req *api.ListNewsParams) ([]*api.News, error) {
	// Simulate DB
	body := "This is the body"
	now := time.Now()
	return []*api.News{
		{ID: 1, Title: "News 1", Body: &body, Status: api.NewsStatusPublished, CreatedAt: &now},
		{ID: 2, Title: "News 2", Status: api.NewsStatusDraft, CreatedAt: &now},
	}, nil
}

// [/snippet:handlers]

// [snippet:error-handling collapse]

func CreateNews(ctx context.Context, req *api.CreateNewsParams) (*api.News, error) {
	if req.Title == "error" {
		return nil, tygor.NewError(tygor.CodeInvalidArgument, "simulated error")
	}
	now := time.Now()
	return &api.News{
		ID:        123,
		Title:     req.Title,
		Body:      req.Body,
		Status:    api.NewsStatusDraft,
		CreatedAt: &now,
	}, nil
}

// [/snippet:error-handling]

// --- Main ---

func main() {
	port := flag.String("port", "8080", "Server port")
	genFlag := flag.Bool("gen", false, "Generate TypeScript types and manifest")
	outDir := flag.String("out", "./client/src/rpc", "Output directory for generation")
	flag.Parse()

	// Configure structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// [snippet:error-transformer]

	app := tygor.NewApp().
		WithErrorTransformer(func(err error) *tygor.Error {
			if err.Error() == "database connection failed" {
				return tygor.NewError(tygor.CodeUnavailable, "service unavailable")
			}
			return nil
		})

	// [/snippet:error-transformer]

	// [snippet:global-interceptor]

	app = app.WithUnaryInterceptor(middleware.LoggingInterceptor(logger))

	// [/snippet:global-interceptor]

	// [snippet:middleware]

	app = app.WithMiddleware(middleware.CORS(middleware.CORSAllowAll))

	// [/snippet:middleware]

	news := app.Service("News")

	// [snippet:cache-control]

	news.Register("List", tygor.Query(ListNews).
		CacheControl(tygor.CacheConfig{
			MaxAge: 1 * time.Minute,
			Public: true,
		}))

	// [/snippet:cache-control]

	// [snippet:handler-interceptor]

	news.Register("Create", tygor.Exec(CreateNews).
		WithUnaryInterceptor(func(ctx tygor.Context, req any, handler tygor.HandlerFunc) (any, error) {
			ctx.HTTPWriter().Header().Set("X-Created-By", "tygor")
			return handler(ctx, req)
		}))

	// [/snippet:handler-interceptor]

	// 3. Generation Mode
	if *genFlag {
		fmt.Printf("Generating types to %s...\n", *outDir)
		if err := os.MkdirAll(*outDir, 0755); err != nil {
			log.Fatal(err)
		}
		_, err := tygorgen.FromApp(app).
			PreserveComments("default").
			EnumStyle("union").
			OptionalType("undefined").
			Frontmatter(`// Branded types for enhanced type safety
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
`).
			TypeMapping("time.Time", "DateTime").
			ToDir(*outDir)
		if err != nil {
			log.Fatalf("Generation failed: %v", err)
		}
		fmt.Println("Done.")
		return
	}

	// 4. Start Server
	addr := ":" + *port
	fmt.Printf("Server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, app.Handler()); err != nil {
		log.Fatal(err)
	}
}
