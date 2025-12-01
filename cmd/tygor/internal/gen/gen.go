package gen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/broady/tygor/internal/discover"
	"github.com/broady/tygor/internal/runner"
)

type Cmd struct {
	Out       string `arg:"" help:"Output directory for generated files."`
	Export    string `help:"Export function name (required if multiple exports exist)." short:"e"`
	Flavor    string `help:"Output flavor (e.g., zod)." short:"f"`
	Discovery bool   `help:"Generate discovery.json." short:"d"`
	NoConfig  bool   `help:"Ignore config function."`
	Package   string `help:"Package to scan (default: current directory)." short:"p" default:"."`
}

func (c *Cmd) Run() error {
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
		Export:     *export,
		OutDir:     outDir,
		Flavor:     c.Flavor,
		Discovery:  c.Discovery,
		NoConfig:   c.NoConfig,
		PkgDir:     result.Dir,
		PkgPath:    result.PackagePath,
		ModulePath: result.ModulePath,
		ModuleDir:  result.ModuleDir,
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
