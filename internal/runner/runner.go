// Package runner generates temporary Go files for tygor code generation.
//
// The generated runner file has a build tag so it doesn't interfere with
// normal builds. It calls the user's export function and writes output
// to the specified directory.
package runner

import (
	"bytes"
	"fmt"
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

// Filename is the name of the generated runner file.
const Filename = "_tygor_gen_runner.go"
