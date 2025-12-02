package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/broady/tygor"
	"github.com/broady/tygor/tygorgen"

	"github.com/broady/tygor/examples/starter-solid/api"
)

// Atom holding message state - subscribers get current value and updates
var messageAtom = tygor.NewAtom(&api.MessageState{
	Message:  "hello",
	SetCount: 0,
})

// SetupApp configures the tygor application.
// This export is used by `tygor gen` for type generation.
func SetupApp() *tygor.App {
	app := tygor.NewApp()

	svc := app.Service("Message")
	svc.Register("State", messageAtom.Handler())
	svc.Register("Set", tygor.Exec(SetMessage))

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

func SetMessage(ctx context.Context, req *api.SetMessageParams) (*api.MessageState, error) {
	var newState *api.MessageState
	messageAtom.Update(func(state *api.MessageState) *api.MessageState {
		newState = &api.MessageState{
			Message:  req.Message,
			SetCount: state.SetCount + 1,
		}
		return newState
	})
	return newState, nil
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
