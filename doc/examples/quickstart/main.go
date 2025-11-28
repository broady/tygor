// Package quickstart provides simple example code for documentation.
package quickstart

import (
	"context"
	"log"
	"net/http"

	"github.com/broady/tygor"
	"github.com/broady/tygor/tygorgen"
)

// [snippet:handlers collapse]
func ListNews(ctx context.Context, req *ListNewsParams) ([]*News, error) {
	// Your implementation
	return nil, nil
}

func CreateNews(ctx context.Context, req *CreateNewsParams) (*News, error) {
	// Your implementation
	return nil, nil
}

// [/snippet:handlers]

func exampleRegistration() {
	// [snippet:registration]
	app := tygor.NewApp()

	news := app.Service("News")
	news.Register("List", tygor.Query(ListNews))
	news.Register("Create", tygor.Exec(CreateNews))

	http.ListenAndServe(":8080", app.Handler())
	// [/snippet:registration]
}

func exampleGeneration() {
	app := tygor.NewApp()
	// [snippet:generation]
	if _, err := tygorgen.FromApp(app).ToDir("./client/src/rpc"); err != nil {
		log.Fatal(err)
	}
	// [/snippet:generation]
}

// Keep imports used.
var (
	_ = context.Background
	_ = exampleRegistration
	_ = exampleGeneration
)
