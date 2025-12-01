package gen

import (
	"bytes"
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
	Check     bool   `help:"Check if generated files are up-to-date (exit 1 if stale)." short:"c"`
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

	// In check mode, generate to temp dir and compare
	genDir := outDir
	if c.Check {
		tmpDir, err := os.MkdirTemp("", "tygor-gen-check-*")
		if err != nil {
			return fmt.Errorf("create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)
		genDir = tmpDir
	}

	// Build runner options
	opts := runner.Options{
		Export:     *export,
		OutDir:     genDir,
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

	// In check mode, compare generated files with existing
	if c.Check {
		return c.compareFiles(genDir, outDir)
	}

	return nil
}

// compareFiles compares generated files in genDir with existing files in outDir.
// Returns an error if any files differ, are missing, or are stale.
func (c *Cmd) compareFiles(genDir, outDir string) error {
	var stale []string

	// Check all generated files exist and match in outDir
	entries, err := os.ReadDir(genDir)
	if err != nil {
		return fmt.Errorf("read generated dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		genPath := filepath.Join(genDir, name)
		outPath := filepath.Join(outDir, name)

		genContent, err := os.ReadFile(genPath)
		if err != nil {
			return fmt.Errorf("read generated file %s: %w", name, err)
		}

		outContent, err := os.ReadFile(outPath)
		if err != nil {
			if os.IsNotExist(err) {
				stale = append(stale, name+" (missing)")
				continue
			}
			return fmt.Errorf("read existing file %s: %w", name, err)
		}

		if !bytes.Equal(genContent, outContent) {
			stale = append(stale, name)
		}
	}

	if len(stale) > 0 {
		fmt.Fprintf(os.Stderr, "Generated files are out of date in %s:\n", outDir)
		for _, f := range stale {
			fmt.Fprintf(os.Stderr, "  %s\n", f)
		}
		fmt.Fprintf(os.Stderr, "\nRun 'tygor gen %s' to update.\n", outDir)
		return fmt.Errorf("generated files are stale")
	}

	fmt.Printf("Generated files in %s are up to date.\n", outDir)
	return nil
}
