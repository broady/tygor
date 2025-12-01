// Package runner executes tygor code generation by building and running
// a modified version of the user's package.
//
// It uses Go's -overlay flag to replace the user's main() with a runner
// that calls the export function and generates output.
package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/broady/tygor/internal/discover"
)

// Options configures the runner.
type Options struct {
	// Export is the function to call.
	Export discover.Export

	// OutDir is the output directory for generated files.
	OutDir string

	// Flavor is the optional flavor flag (e.g., "zod").
	// Only used when Export.Type is ExportTypeApp.
	Flavor string

	// Discovery enables discovery.json generation.
	// Only used when Export.Type is ExportTypeApp.
	Discovery bool

	// ConfigFunc is the optional config function name.
	// Only used when Export.Type is ExportTypeApp.
	ConfigFunc string

	// NoConfig disables the config function even if one exists.
	NoConfig bool

	// PkgDir is the directory containing the package.
	PkgDir string
}

// Exec builds and runs the generator.
//
// It creates an overlay that:
// 1. Replaces files containing func main() with versions that have main() removed
// 2. Adds a runner file with our own main()
//
// The overlay approach lets us work with package main and unexported functions.
func Exec(opts Options) (output []byte, err error) {
	// Create temp directory for overlay files
	tmpDir, err := os.MkdirTemp("", "tygor-gen-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Find and process files with main()
	overlay := make(map[string]string)

	files, err := filepath.Glob(filepath.Join(opts.PkgDir, "*.go"))
	if err != nil {
		return nil, fmt.Errorf("glob: %w", err)
	}

	for _, file := range files {
		// Skip test files
		if filepath.Base(file) == "_test.go" || len(file) > 8 && file[len(file)-8:] == "_test.go" {
			continue
		}

		hasMain, modified, err := removeMain(file)
		if err != nil {
			return nil, fmt.Errorf("process %s: %w", file, err)
		}

		if hasMain {
			// Write modified file to temp dir
			tmpFile := filepath.Join(tmpDir, filepath.Base(file))
			if err := os.WriteFile(tmpFile, modified, 0644); err != nil {
				return nil, fmt.Errorf("write modified %s: %w", file, err)
			}
			overlay[file] = tmpFile
		}
	}

	// Generate runner
	runnerSrc, err := generateRunner(opts)
	if err != nil {
		return nil, fmt.Errorf("generate runner: %w", err)
	}

	runnerFile := filepath.Join(tmpDir, "tygor_runner_main_.go")
	if err := os.WriteFile(runnerFile, runnerSrc, 0644); err != nil {
		return nil, fmt.Errorf("write runner: %w", err)
	}

	// Add runner to overlay (maps to a "new" file in the package)
	overlay[filepath.Join(opts.PkgDir, "tygor_runner_main_.go")] = runnerFile

	// Write overlay JSON
	overlayData := struct {
		Replace map[string]string `json:"Replace"`
	}{Replace: overlay}

	overlayJSON, err := json.Marshal(overlayData)
	if err != nil {
		return nil, fmt.Errorf("marshal overlay: %w", err)
	}

	overlayFile := filepath.Join(tmpDir, "overlay.json")
	if err := os.WriteFile(overlayFile, overlayJSON, 0644); err != nil {
		return nil, fmt.Errorf("write overlay: %w", err)
	}

	// Build with overlay
	// Use -mod=mod to allow updating go.mod/go.sum if needed
	binaryPath := filepath.Join(tmpDir, "runner")
	buildCmd := exec.Command("go", "build", "-mod=mod", "-overlay", overlayFile, "-o", binaryPath, ".")
	buildCmd.Dir = opts.PkgDir
	buildCmd.Env = append(os.Environ(), "GOWORK=off")
	if buildOut, err := buildCmd.CombinedOutput(); err != nil {
		return buildOut, fmt.Errorf("build: %w\n%s", err, buildOut)
	}

	// Run the binary
	runCmd := exec.Command(binaryPath)
	runCmd.Dir = opts.PkgDir
	output, err = runCmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("run: %w\n%s", err, output)
	}

	return output, nil
}

// removeMain parses a Go file and returns a version with func main() removed.
// Returns (hasMain, modifiedSource, error).
func removeMain(filename string) (bool, []byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return false, nil, err
	}

	// Find and remove main function
	hasMain := false
	var newDecls []ast.Decl
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "main" && fn.Recv == nil {
			hasMain = true
			continue // skip main()
		}
		newDecls = append(newDecls, decl)
	}

	if !hasMain {
		return false, nil, nil
	}

	f.Decls = newDecls

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return false, nil, err
	}

	return true, buf.Bytes(), nil
}

// generateRunner creates the runner main() source.
func generateRunner(opts Options) ([]byte, error) {
	var tmplStr string
	switch opts.Export.Type {
	case discover.ExportTypeApp:
		tmplStr = appRunnerTemplate
	case discover.ExportTypeGenerator:
		tmplStr = generatorRunnerTemplate
	default:
		return nil, fmt.Errorf("unknown export type: %v", opts.Export.Type)
	}

	tmpl, err := template.New("runner").Parse(tmplStr)
	if err != nil {
		return nil, err
	}

	configFunc := ""
	if opts.ConfigFunc != "" && !opts.NoConfig {
		configFunc = opts.ConfigFunc
	}

	data := struct {
		ExportFunc string
		OutDir     string
		Flavor     string
		Discovery  bool
		ConfigFunc string
	}{
		ExportFunc: opts.Export.Name,
		OutDir:     opts.OutDir,
		Flavor:     opts.Flavor,
		Discovery:  opts.Discovery,
		ConfigFunc: configFunc,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

const appRunnerTemplate = `package main

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
{{if .ConfigFunc}}
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

const generatorRunnerTemplate = `package main

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
