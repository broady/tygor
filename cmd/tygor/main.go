package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Version VersionCmd `cmd:"" help:"Print version information."`
	Gen     GenCmd     `cmd:"" help:"Generate TypeScript types and discovery.json."`
	Check   CheckCmd   `cmd:"" help:"Validate exports and types without generating files."`
	Dev     DevCmd     `cmd:"" help:"Start standalone devtools server."`
}

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Println(Version())
	return nil
}

type GenCmd struct {
	Out      string `arg:"" help:"Output directory for generated files."`
	Flavor   string `help:"Output flavor (e.g., zod)." short:"f"`
	Watch    bool   `help:"Watch for changes and regenerate." short:"w"`
	NoConfig bool   `help:"Ignore //tygor:config function."`
}

func (c *GenCmd) Run() error {
	fmt.Fprintf(os.Stderr, "tygor gen: not yet implemented\n")
	return nil
}

type CheckCmd struct{}

func (c *CheckCmd) Run() error {
	fmt.Fprintf(os.Stderr, "tygor check: not yet implemented\n")
	return nil
}

type DevCmd struct {
	RPCDir string `help:"Directory containing generated RPC files." default:"./src/rpc" name:"rpc-dir"`
	Port   int    `help:"Port to listen on." default:"9000" short:"p"`
	Watch  bool   `help:"Watch for file changes." short:"w"`
}

func (c *DevCmd) Run() error {
	fmt.Fprintf(os.Stderr, "tygor dev: not yet implemented\n")
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
