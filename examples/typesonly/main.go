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
	"flag"
	"log"
	"os"

	"github.com/broady/tygor/examples/typesonly/api"
	"github.com/broady/tygor/tygorgen"
)

// [snippet:main]

func main() {
	gen := flag.Bool("gen", false, "generate TypeScript types")
	out := flag.String("out", "./client/src/types", "output directory")
	flag.Parse()

	if *gen {
		generate(*out)
		return
	}

	log.Println("Run with -gen to generate TypeScript types")
}

func generate(dir string) {
	// Pass root types - referenced types are followed automatically.
	_, err := tygorgen.FromTypes(
		api.User{},
		api.Team{},
		api.PaginatedUsers{},
		api.AuditEvent{},
		api.Pagination{},
	).ToDir(dir)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Generated types to", dir)
	os.Exit(0)
}

// [/snippet:main]
