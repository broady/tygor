// Example demonstrating tygor with protobuf-generated Go types as request/response.
//
// This shows that tygor works seamlessly with protobuf messages - you can use
// proto-generated structs directly in your handlers.
//
// To regenerate the proto types:
//
//	protoc --go_out=. --go_opt=paths=source_relative api/messages.proto
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/broady/tygor"
	"github.com/broady/tygor/examples/protobuf/api"
	"github.com/broady/tygor/tygorgen"
)

var idCounter atomic.Int64

// --- Handlers using protobuf types ---

// [snippet:proto-handler collapse]
func CreateItem(ctx context.Context, req *api.CreateItemRequest) (*api.Item, error) {
	if req.Name == "" {
		return nil, tygor.NewError(tygor.CodeInvalidArgument, "name is required")
	}

	id := idCounter.Add(1)
	return &api.Item{
		Id:          id,
		Name:        req.Name,
		Description: req.Description,
		PriceCents:  req.PriceCents,
		Tags:        req.Tags,
	}, nil
}

// [/snippet:proto-handler]

func ListItems(ctx context.Context, req *api.ListItemsRequest) (*api.ListItemsResponse, error) {
	// Simulate some items
	items := []*api.Item{
		{Id: 1, Name: "Widget", Description: "A useful widget", PriceCents: 999, Tags: []string{"gadget", "tool"}},
		{Id: 2, Name: "Gadget", Description: "A fancy gadget", PriceCents: 1999, Tags: []string{"gadget", "fancy"}},
		{Id: 3, Name: "Gizmo", Description: "A mysterious gizmo", PriceCents: 2999, Tags: []string{"mystery"}},
	}

	// Apply pagination
	offset := int(req.Offset)
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}

	total := len(items)
	if offset >= total {
		return &api.ListItemsResponse{Items: nil, Total: int32(total)}, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return &api.ListItemsResponse{
		Items: items[offset:end],
		Total: int32(total),
	}, nil
}

func GetItem(ctx context.Context, req *api.GetItemRequest) (*api.Item, error) {
	// Simulate lookup
	if req.Id <= 0 || req.Id > 3 {
		return nil, tygor.NewError(tygor.CodeNotFound, "item not found")
	}

	items := map[int64]*api.Item{
		1: {Id: 1, Name: "Widget", Description: "A useful widget", PriceCents: 999, Tags: []string{"gadget", "tool"}},
		2: {Id: 2, Name: "Gadget", Description: "A fancy gadget", PriceCents: 1999, Tags: []string{"gadget", "fancy"}},
		3: {Id: 3, Name: "Gizmo", Description: "A mysterious gizmo", PriceCents: 2999, Tags: []string{"mystery"}},
	}

	return items[req.Id], nil
}

// --- Main ---

// SetupApp configures the tygor application.
// This export is used by `tygor gen` for type generation.
func SetupApp() *tygor.App {
	app := tygor.NewApp()

	// Register handlers
	items := app.Service("Items")
	items.Register("Create", tygor.Exec(CreateItem))
	items.Register("List", tygor.Query(ListItems))
	items.Register("Get", tygor.Query(GetItem))

	return app
}

// TygorConfig configures the TypeScript generator.
func TygorConfig(g *tygorgen.Generator) *tygorgen.Generator {
	return g.
		EnumStyle("union").
		OptionalType("undefined").
		WithDiscovery().
		WithFlavor(tygorgen.FlavorZod)
}

func main() {
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	if p := os.Getenv("PORT"); p != "" {
		*port = p
	}

	app := SetupApp()

	addr := ":" + *port
	fmt.Printf("Server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, app.Handler()); err != nil {
		log.Fatal(err)
	}
}
