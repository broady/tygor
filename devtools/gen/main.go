//go:build ignore

package main

import (
	"log"
	"os"

	"github.com/broady/tygor"
	"github.com/broady/tygor/devtools"
	"github.com/broady/tygor/tygorgen"
)

func main() {
	outDir := "../client"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	app := tygor.NewApp()
	devtools.New(app, 0).Register()

	_, err := tygorgen.FromApp(app).
		EnumStyle("union").
		OptionalType("undefined").
		ToDir(outDir)
	if err != nil {
		log.Fatalf("Generation failed: %v", err)
	}
}
