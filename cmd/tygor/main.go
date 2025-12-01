package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/broady/tygor/internal/discover"
	"github.com/broady/tygor/internal/runner"
)

type CLI struct {
	Version VersionCmd `cmd:"" help:"Print version information."`
	Gen     GenCmd     `cmd:"" help:"Generate TypeScript types and manifest."`
	Check   CheckCmd   `cmd:"" help:"Validate exports and types without generating files."`
	Dev     DevCmd     `cmd:"" help:"Start standalone devtools server."`
}

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Println(Version())
	return nil
}

type GenCmd struct {
	Out       string `arg:"" help:"Output directory for generated files."`
	Export    string `help:"Export function name (required if multiple exports exist)." short:"e"`
	Flavor    string `help:"Output flavor (e.g., zod)." short:"f"`
	Discovery bool   `help:"Generate discovery.json." short:"d"`
	NoConfig  bool   `help:"Ignore config function."`
	Package   string `help:"Package to scan (default: current directory)." short:"p" default:"."`
}

func (c *GenCmd) Run() error {
	// Discover exports in the package
	result, err := discover.Find(c.Package)
	if err != nil {
		return fmt.Errorf("discover: %w", err)
	}

	// Select the export to use
	export, err := discover.SelectExport(result.Exports, c.Export)
	if err != nil {
		return err
	}

	// Validate flags for *tygorgen.Generator exports
	if export.Type == discover.ExportTypeGenerator {
		if c.Flavor != "" {
			return fmt.Errorf("--flavor not supported with *tygorgen.Generator export\n\nYour export returns *tygorgen.Generator - configuration is in code.\nAdd flavors in your generator function:\n\n    return tygorgen.FromApp(setupApp()).\n        WithFlavor(tygorgen.FlavorZod)")
		}
		if c.Discovery {
			return fmt.Errorf("--discovery not supported with *tygorgen.Generator export\n\nYour export returns *tygorgen.Generator - configuration is in code.\nEnable discovery in your generator function:\n\n    return tygorgen.FromApp(setupApp()).\n        WithDiscovery()")
		}
	}

	// Resolve output directory to absolute path
	outDir, err := filepath.Abs(c.Out)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	// Build runner options
	opts := runner.Options{
		Export:    *export,
		OutDir:    outDir,
		Flavor:    c.Flavor,
		Discovery: c.Discovery,
		NoConfig:  c.NoConfig,
		PkgDir:    result.Dir,
	}

	// Add config function if present and applicable
	if result.ConfigFunc != nil && export.Type == discover.ExportTypeApp {
		opts.ConfigFunc = result.ConfigFunc.Name
	}

	// Run the generator
	output, err := runner.Exec(opts)
	if err != nil {
		if len(output) > 0 {
			fmt.Fprint(os.Stderr, string(output))
		}
		return err
	}

	// Print any output (warnings, etc.)
	if len(output) > 0 {
		fmt.Print(string(output))
	}

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
