package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/broady/tygor/cmd/tygor/internal/check"
	"github.com/broady/tygor/cmd/tygor/internal/dev"
	"github.com/broady/tygor/cmd/tygor/internal/gen"
)

type CLI struct {
	Version VersionCmd `cmd:"" help:"Print version information."`
	Gen     gen.Cmd    `cmd:"" help:"Generate TypeScript types and manifest."`
	Check   check.Cmd  `cmd:"" help:"Validate exports and types without generating files."`
	Dev     dev.Cmd    `cmd:"" help:"Start standalone devtools server."`
}

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Println(Version())
	return nil
}

func main() {
	cli := &CLI{}
	ctx := kong.Parse(cli,
		kong.Name("tygor"),
		kong.Description("Tygor CLI for code generation and development tools."),
		kong.UsageOnError(),
	)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
