// Typesonly demonstrates standalone TypeScript type generation without a tygor app.
//
// This is useful when you want to share Go types with TypeScript clients but don't
// need the full RPC infrastructure. Common use cases:
//   - Shared type libraries
//   - Event schemas for messaging
//   - Config file schemas
//   - API types for external codegen
package main

import (
	"fmt"

	"github.com/broady/tygor/examples/typesonly/api"
	"github.com/broady/tygor/tygorgen"
)

// [snippet:main]

// TygorConfig configures the TypeScript generator for types-only generation.
// This export is used by `tygor gen` for type generation.
func TygorConfig() *tygorgen.Generator {
	// Pass root types - referenced types are followed automatically.
	return tygorgen.FromTypes(
		api.User{},
		api.Page[api.User]{}, // generic instantiation
	)
}

func main() {
	fmt.Println("This is a types-only example. To generate TypeScript types, run:")
	fmt.Println("  tygor gen ./client/src/types")
}

// [/snippet:main]
