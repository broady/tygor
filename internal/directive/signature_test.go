package directive

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseWithTypes(t *testing.T) {
	// Disable go.work so temp directories work as standalone modules
	t.Setenv("GOWORK", "off")

	tests := []struct {
		name        string
		files       map[string]string
		wantExports []struct {
			funcName   string
			exportType ExportType
		}
		wantConfig string // expected config func name, empty if none
		wantErr    string // expected error substring, empty if none
	}{
		{
			name: "export returning *tygor.App",
			files: map[string]string{
				"main.go": `package main

import "github.com/broady/tygor"

//tygor:export
func setupApp() *tygor.App {
	return tygor.NewApp()
}
`,
			},
			wantExports: []struct {
				funcName   string
				exportType ExportType
			}{
				{funcName: "setupApp", exportType: ExportTypeApp},
			},
		},
		{
			name: "export returning *tygorgen.Generator",
			files: map[string]string{
				"main.go": `package main

import (
	"github.com/broady/tygor"
	"github.com/broady/tygor/tygorgen"
)

func setupApp() *tygor.App {
	return tygor.NewApp()
}

//tygor:export
func gen() *tygorgen.Generator {
	return tygorgen.FromApp(setupApp())
}
`,
			},
			wantExports: []struct {
				funcName   string
				exportType ExportType
			}{
				{funcName: "gen", exportType: ExportTypeGenerator},
			},
		},
		{
			name: "export with config",
			files: map[string]string{
				"main.go": `package main

import (
	"github.com/broady/tygor"
	"github.com/broady/tygor/tygorgen"
)

//tygor:export
func setupApp() *tygor.App {
	return tygor.NewApp()
}

//tygor:config
func configure(g *tygorgen.Generator) *tygorgen.Generator {
	return g
}
`,
			},
			wantExports: []struct {
				funcName   string
				exportType ExportType
			}{
				{funcName: "setupApp", exportType: ExportTypeApp},
			},
			wantConfig: "configure",
		},
		{
			name: "export with wrong return type",
			files: map[string]string{
				"main.go": `package main

//tygor:export
func wrong() string {
	return "oops"
}
`,
			},
			wantErr: "invalid return type",
		},
		{
			name: "export with parameters",
			files: map[string]string{
				"main.go": `package main

import "github.com/broady/tygor"

//tygor:export
func setupApp(name string) *tygor.App {
	return tygor.NewApp()
}
`,
			},
			wantErr: "must have no parameters",
		},
		{
			name: "export on method",
			files: map[string]string{
				"main.go": `package main

import "github.com/broady/tygor"

type Builder struct{}

//tygor:export
func (b *Builder) Build() *tygor.App {
	return tygor.NewApp()
}
`,
			},
			wantErr: "must be on a package-level function",
		},
		{
			name: "config with wrong parameter type",
			files: map[string]string{
				"main.go": `package main

//tygor:config
func configure(s string) string {
	return s
}
`,
			},
			wantErr: "parameter must be *tygorgen.Generator",
		},
		{
			name: "config with wrong return type",
			files: map[string]string{
				"main.go": `package main

import "github.com/broady/tygor/tygorgen"

//tygor:config
func configure(g *tygorgen.Generator) string {
	return "oops"
}
`,
			},
			wantErr: "must return *tygorgen.Generator",
		},
		{
			name: "multiple exports with different types",
			files: map[string]string{
				"main.go": `package main

import (
	"github.com/broady/tygor"
	"github.com/broady/tygor/tygorgen"
)

//tygor:export app
func setupApp() *tygor.App {
	return tygor.NewApp()
}

//tygor:export gen
func makeGen() *tygorgen.Generator {
	return tygorgen.FromApp(setupApp())
}
`,
			},
			wantExports: []struct {
				funcName   string
				exportType ExportType
			}{
				{funcName: "setupApp", exportType: ExportTypeApp},
				{funcName: "makeGen", exportType: ExportTypeGenerator},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			// Write go.mod with tygor dependency
			goMod := `module test

go 1.21

require github.com/broady/tygor v0.7.4

replace github.com/broady/tygor => ` + mustAbs(t, "../..") + `
`
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
				t.Fatal(err)
			}

			// Write test files first, then run go mod tidy
			for name, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Run go mod tidy to resolve dependencies
			if err := runGoModTidy(t, dir); err != nil {
				t.Fatalf("go mod tidy failed: %v", err)
			}

			result, err := ParseWithTypesDir(".", dir)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Exports) != len(tt.wantExports) {
				t.Fatalf("got %d exports, want %d", len(result.Exports), len(tt.wantExports))
			}

			// Build a map for easier comparison
			gotExports := make(map[string]ExportType)
			for _, e := range result.Exports {
				gotExports[e.FuncName] = e.Type
			}

			for _, want := range tt.wantExports {
				gotType, ok := gotExports[want.funcName]
				if !ok {
					t.Errorf("missing export for func %s", want.funcName)
					continue
				}
				if gotType != want.exportType {
					t.Errorf("export %s: got type %v, want %v", want.funcName, gotType, want.exportType)
				}
			}

			if tt.wantConfig != "" {
				if result.Config == nil {
					t.Fatal("expected config directive, got nil")
				}
				if result.Config.FuncName != tt.wantConfig {
					t.Errorf("config func: got %q, want %q", result.Config.FuncName, tt.wantConfig)
				}
			} else if result.Config != nil {
				t.Errorf("unexpected config directive: %s", result.Config.FuncName)
			}
		})
	}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func runGoModTidy(t *testing.T, dir string) error {
	t.Helper()
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", string(output), err)
	}
	return nil
}
