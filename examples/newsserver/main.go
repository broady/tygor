package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/broady/tygor"
	"github.com/broady/tygor/examples/newsserver/api"
)

// --- Handlers ---

func ListNews(ctx context.Context, req *api.ListNewsParams) ([]*api.News, error) {
	// Simulate DB
	body := "This is the body"
	now := time.Now()
	return []*api.News{
		{ID: 1, Title: "News 1", Body: &body, CreatedAt: &now},
		{ID: 2, Title: "News 2", CreatedAt: &now},
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
		CreatedAt: &now,
	}, nil
}

// --- Main ---

func main() {
	genFlag := flag.Bool("gen", false, "Generate TypeScript types and manifest")
	outDir := flag.String("out", "./client/src/rpc", "Output directory for generation")
	flag.Parse()

	// 1. Create Registry
	reg := tygor.NewRegistry().
		WithErrorTransformer(func(err error) *tygor.Error {
			// Example custom error mapping
			if err.Error() == "database connection failed" {
				return tygor.NewError(tygor.CodeUnavailable, "db down")
			}
			return nil
		}).
		WithInterceptor(func(ctx context.Context, req any, info *tygor.RPCInfo, handler tygor.HandlerFunc) (any, error) {
			start := time.Now()
			log.Printf("[START] %s.%s", info.Service, info.Method)
			res, err := handler(ctx, req)
			duration := time.Since(start)

			if err != nil {
				log.Printf("[ERROR] %s.%s (%v) - %v", info.Service, info.Method, duration, err)
			} else {
				log.Printf("[OK] %s.%s (%v)", info.Service, info.Method, duration)
			}
			return res, err
		})

	// 2. Register Services
	news := reg.Service("News")

	news.Register("List", tygor.NewHandler(ListNews).
		Method("GET").
		Cache(1*time.Minute))

	news.Register("Create", tygor.NewHandler(CreateNews).
		Method("POST").
		WithInterceptor(func(ctx context.Context, req any, info *tygor.RPCInfo, handler tygor.HandlerFunc) (any, error) {
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
		if err := reg.Generate(&tygor.GenConfig{
			OutDir: *outDir,
		}); err != nil {
			log.Fatalf("Generation failed: %v", err)
		}
		fmt.Println("Done.")
		return
	}

	// 4. Server Mode
	fmt.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", reg); err != nil {
		log.Fatal(err)
	}
}
