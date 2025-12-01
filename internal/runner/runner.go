// Package runner generates and executes temporary Go files for tygor code generation.
//
// The generated runner file has a build tag so it doesn't interfere with
// normal builds. It calls the user's export function and writes output
// to the specified directory.
package runner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/broady/tygor/internal/directive"
)

// Options configures the generated runner.
type Options struct {
	// Export is the export directive to use.
	Export directive.Export

	// Config is the optional config directive.
	// Only used when Export.Type is ExportTypeApp.
	Config *directive.Config

	// OutDir is the output directory (passed as command line arg).
	// This is embedded in the generated code.
	OutDir string

	// Flavor is the optional flavor flag (e.g., "zod").
	// Only used when Export.Type is ExportTypeApp.
	Flavor string

	// Discovery enables discovery.json generation.
	// Only used when Export.Type is ExportTypeApp.
	Discovery bool

	// NoConfig skips calling the config function even if present.
	NoConfig bool
}

// Generate creates the runner source code.
func Generate(opts Options) ([]byte, error) {
	var tmpl *template.Template
	var err error

	switch opts.Export.Type {
	case directive.ExportTypeApp:
		tmpl, err = template.New("runner").Parse(appRunnerTemplate)
	case directive.ExportTypeGenerator:
		tmpl, err = template.New("runner").Parse(generatorRunnerTemplate)
	default:
		return nil, fmt.Errorf("unknown export type: %v", opts.Export.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	data := templateData{
		ExportFunc: opts.Export.FuncName,
		OutDir:     opts.OutDir,
		Flavor:     opts.Flavor,
		Discovery:  opts.Discovery,
		HasConfig:  opts.Config != nil && !opts.NoConfig,
		ConfigFunc: "",
	}
	if opts.Config != nil {
		data.ConfigFunc = opts.Config.FuncName
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	return buf.Bytes(), nil
}

type templateData struct {
	ExportFunc string
	OutDir     string
	Flavor     string
	Discovery  bool
	HasConfig  bool
	ConfigFunc string
}

// appRunnerTemplate is used when the export returns *tygor.App.
// It creates a generator from the app, applies flags, optionally calls config.
const appRunnerTemplate = `//go:build tygor_gen_runner

package main

import (
	"fmt"
	"os"

	"github.com/broady/tygor/tygorgen"
)

func main() {
	g := tygorgen.FromApp({{.ExportFunc}}())
{{if .Flavor}}
	g = g.WithFlavor(tygorgen.Flavor("{{.Flavor}}"))
{{end}}
{{if .Discovery}}
	g = g.WithDiscovery()
{{end}}
{{if .HasConfig}}
	g = {{.ConfigFunc}}(g)
{{end}}
	result, err := g.ToDir("{{.OutDir}}")
	if err != nil {
		fmt.Fprintf(os.Stderr, "tygor gen: %v\n", err)
		os.Exit(1)
	}
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s: %s\n", w.Code, w.Message)
	}
}
`

// generatorRunnerTemplate is used when the export returns *tygorgen.Generator.
// The user has full control over the generator, so no flags are applied.
const generatorRunnerTemplate = `//go:build tygor_gen_runner

package main

import (
	"fmt"
	"os"
)

func main() {
	g := {{.ExportFunc}}()
	result, err := g.ToDir("{{.OutDir}}")
	if err != nil {
		fmt.Fprintf(os.Stderr, "tygor gen: %v\n", err)
		os.Exit(1)
	}
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s: %s\n", w.Code, w.Message)
	}
}
`

// ExecOptions configures the runner execution.
type ExecOptions struct {
	Options

	// ModuleDir is the directory containing the go.mod file.
	ModuleDir string

	// ModulePath is the module path from go.mod.
	ModulePath string

	// PkgPath is the full import path of the package containing the export.
	PkgPath string
}

// Exec generates a runner in a temporary directory, executes it, and cleans up.
//
// The runner is created as a separate package that imports the user's package.
// This avoids conflicts with the user's main function.
//
// Returns the combined stdout/stderr output and any error.
func Exec(opts ExecOptions) (output []byte, err error) {
	// Create a temporary directory for the runner
	tmpDir, err := os.MkdirTemp("", "tygor-gen-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate the runner source (with import)
	src, err := GenerateWithImport(opts.Options, opts.PkgPath)
	if err != nil {
		return nil, fmt.Errorf("generate runner: %w", err)
	}

	// Write go.mod that requires the user's module
	goMod := fmt.Sprintf(`module tygor_gen_runner

go 1.21

require %s v0.0.0

replace %s => %s
`, opts.ModulePath, opts.ModulePath, opts.ModuleDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		return nil, fmt.Errorf("write go.mod: %w", err)
	}

	// Write the runner file
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), src, 0644); err != nil {
		return nil, fmt.Errorf("write runner: %w", err)
	}

	// Run go mod tidy to resolve dependencies
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	tidyCmd.Env = append(os.Environ(), "GOWORK=off")
	if tidyOut, err := tidyCmd.CombinedOutput(); err != nil {
		return tidyOut, fmt.Errorf("go mod tidy: %w\n%s", err, tidyOut)
	}

	// Run the generated code
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("run generator: %w\n%s", err, output)
	}

	return output, nil
}

// GenerateWithImport creates runner source that imports the user's package.
func GenerateWithImport(opts Options, pkgPath string) ([]byte, error) {
	var tmpl *template.Template
	var err error

	switch opts.Export.Type {
	case directive.ExportTypeApp:
		tmpl, err = template.New("runner").Parse(appRunnerWithImportTemplate)
	case directive.ExportTypeGenerator:
		tmpl, err = template.New("runner").Parse(generatorRunnerWithImportTemplate)
	default:
		return nil, fmt.Errorf("unknown export type: %v", opts.Export.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	data := templateDataWithImport{
		PkgPath:    pkgPath,
		ExportFunc: opts.Export.FuncName,
		OutDir:     opts.OutDir,
		Flavor:     opts.Flavor,
		Discovery:  opts.Discovery,
		HasConfig:  opts.Config != nil && !opts.NoConfig,
		ConfigFunc: "",
	}
	if opts.Config != nil {
		data.ConfigFunc = opts.Config.FuncName
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	return buf.Bytes(), nil
}

type templateDataWithImport struct {
	PkgPath    string
	ExportFunc string
	OutDir     string
	Flavor     string
	Discovery  bool
	HasConfig  bool
	ConfigFunc string
}

const appRunnerWithImportTemplate = `package main

import (
	"fmt"
	"os"

	"github.com/broady/tygor/tygorgen"
	pkg "{{.PkgPath}}"
)

func main() {
	g := tygorgen.FromApp(pkg.{{.ExportFunc}}())
{{if .Flavor}}
	g = g.WithFlavor(tygorgen.Flavor("{{.Flavor}}"))
{{end}}
{{if .Discovery}}
	g = g.WithDiscovery()
{{end}}
{{if .HasConfig}}
	g = pkg.{{.ConfigFunc}}(g)
{{end}}
	result, err := g.ToDir("{{.OutDir}}")
	if err != nil {
		fmt.Fprintf(os.Stderr, "tygor gen: %v\n", err)
		os.Exit(1)
	}
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s: %s\n", w.Code, w.Message)
	}
}
`

const generatorRunnerWithImportTemplate = `package main

import (
	"fmt"
	"os"

	pkg "{{.PkgPath}}"
)

func main() {
	g := pkg.{{.ExportFunc}}()
	result, err := g.ToDir("{{.OutDir}}")
	if err != nil {
		fmt.Fprintf(os.Stderr, "tygor gen: %v\n", err)
		os.Exit(1)
	}
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s: %s\n", w.Code, w.Message)
	}
}
`
