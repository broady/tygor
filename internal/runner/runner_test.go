package runner

import (
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/broady/tygor/internal/directive"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name     string
		opts     Options
		contains []string // strings that must appear in output
		excludes []string // strings that must not appear in output
	}{
		{
			name: "app export simple",
			opts: Options{
				Export: directive.Export{
					Directive: directive.Directive{
						FuncName: "setupApp",
						Pos:      token.Position{},
					},
					Type: directive.ExportTypeApp,
				},
				OutDir: "./src/rpc",
			},
			contains: []string{
				"//go:build tygor_gen_runner",
				"tygorgen.FromApp(setupApp())",
				`g.ToDir("./src/rpc")`,
			},
			excludes: []string{
				"WithFlavor",
				"WithDiscovery",
			},
		},
		{
			name: "app export with flavor",
			opts: Options{
				Export: directive.Export{
					Directive: directive.Directive{
						FuncName: "setupApp",
					},
					Type: directive.ExportTypeApp,
				},
				OutDir: "./src/rpc",
				Flavor: "zod",
			},
			contains: []string{
				"tygorgen.FromApp(setupApp())",
				`WithFlavor(tygorgen.Flavor("zod"))`,
			},
		},
		{
			name: "app export with discovery",
			opts: Options{
				Export: directive.Export{
					Directive: directive.Directive{
						FuncName: "setupApp",
					},
					Type: directive.ExportTypeApp,
				},
				OutDir:    "./src/rpc",
				Discovery: true,
			},
			contains: []string{
				"tygorgen.FromApp(setupApp())",
				"WithDiscovery()",
			},
		},
		{
			name: "app export with config",
			opts: Options{
				Export: directive.Export{
					Directive: directive.Directive{
						FuncName: "setupApp",
					},
					Type: directive.ExportTypeApp,
				},
				Config: &directive.Config{
					Directive: directive.Directive{
						FuncName: "configure",
					},
				},
				OutDir: "./src/rpc",
			},
			contains: []string{
				"tygorgen.FromApp(setupApp())",
				"g = configure(g)",
			},
		},
		{
			name: "app export with config but no-config flag",
			opts: Options{
				Export: directive.Export{
					Directive: directive.Directive{
						FuncName: "setupApp",
					},
					Type: directive.ExportTypeApp,
				},
				Config: &directive.Config{
					Directive: directive.Directive{
						FuncName: "configure",
					},
				},
				OutDir:   "./src/rpc",
				NoConfig: true,
			},
			contains: []string{
				"tygorgen.FromApp(setupApp())",
			},
			excludes: []string{
				"configure(g)",
			},
		},
		{
			name: "app export with all flags",
			opts: Options{
				Export: directive.Export{
					Directive: directive.Directive{
						FuncName: "setupApp",
					},
					Type: directive.ExportTypeApp,
				},
				Config: &directive.Config{
					Directive: directive.Directive{
						FuncName: "configure",
					},
				},
				OutDir:    "./src/rpc",
				Flavor:    "zod",
				Discovery: true,
			},
			contains: []string{
				"tygorgen.FromApp(setupApp())",
				`WithFlavor(tygorgen.Flavor("zod"))`,
				"WithDiscovery()",
				"g = configure(g)",
			},
		},
		{
			name: "generator export",
			opts: Options{
				Export: directive.Export{
					Directive: directive.Directive{
						FuncName: "gen",
					},
					Type: directive.ExportTypeGenerator,
				},
				OutDir: "./src/rpc",
			},
			contains: []string{
				"//go:build tygor_gen_runner",
				"g := gen()",
				`g.ToDir("./src/rpc")`,
			},
			excludes: []string{
				"tygorgen.FromApp",
				"WithFlavor",
				"WithDiscovery",
			},
		},
		{
			name: "generator export ignores flags",
			opts: Options{
				Export: directive.Export{
					Directive: directive.Directive{
						FuncName: "gen",
					},
					Type: directive.ExportTypeGenerator,
				},
				OutDir:    "./src/rpc",
				Flavor:    "zod", // should be ignored
				Discovery: true,  // should be ignored
			},
			contains: []string{
				"g := gen()",
			},
			excludes: []string{
				"WithFlavor",
				"WithDiscovery",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := Generate(tt.opts)
			if err != nil {
				t.Fatalf("Generate() error: %v", err)
			}

			code := string(output)

			for _, want := range tt.contains {
				if !strings.Contains(code, want) {
					t.Errorf("output missing %q\n\nGot:\n%s", want, code)
				}
			}

			for _, unwant := range tt.excludes {
				if strings.Contains(code, unwant) {
					t.Errorf("output should not contain %q\n\nGot:\n%s", unwant, code)
				}
			}
		})
	}
}

func TestExec(t *testing.T) {
	// Disable go.work so temp directories work as standalone modules
	t.Setenv("GOWORK", "off")

	// Create a temp directory with a simple tygor app
	dir := t.TempDir()

	// Write go.mod
	tygorRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	goMod := `module testapp

go 1.21

require github.com/broady/tygor v0.7.4

replace github.com/broady/tygor => ` + tygorRoot + `
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a simple app
	mainGo := `package main

import "github.com/broady/tygor"

//tygor:export
func setupApp() *tygor.App {
	app := tygor.NewApp()
	svc := app.Service("Test")
	svc.Register("Hello", tygor.Query(func(_ context.Context, req struct{}) (string, error) {
		return "hello", nil
	}))
	return app
}

func main() {
	setupApp().Handler()
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Use a non-main package that can be imported
	// The export function goes in an api package
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	apiGo := `package api

import (
	"context"
	"github.com/broady/tygor"
)

type HelloRequest struct {
	Name string
}

type HelloResponse struct {
	Message string
}

//tygor:export
func SetupApp() *tygor.App {
	app := tygor.NewApp()
	svc := app.Service("Test")
	svc.Register("Hello", tygor.Query(func(_ context.Context, req HelloRequest) (HelloResponse, error) {
		return HelloResponse{Message: "hello " + req.Name}, nil
	}))
	return app
}
`
	if err := os.WriteFile(filepath.Join(apiDir, "api.go"), []byte(apiGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Main package imports api
	mainGo = `package main

import "testapp/api"

func main() {
	api.SetupApp().Handler()
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, out)
	}

	// Create output directory
	outDir := filepath.Join(dir, "rpc")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Run Exec
	output, err := Exec(ExecOptions{
		Options: Options{
			Export: directive.Export{
				Directive: directive.Directive{
					FuncName: "SetupApp",
				},
				Type: directive.ExportTypeApp,
			},
			OutDir: outDir,
		},
		ModuleDir:  dir,
		ModulePath: "testapp",
		PkgPath:    "testapp/api",
	})
	if err != nil {
		t.Fatalf("Exec() error: %v\nOutput: %s", err, output)
	}

	// Verify output was generated
	typesPath := filepath.Join(outDir, "types.ts")
	if _, err := os.Stat(typesPath); os.IsNotExist(err) {
		t.Errorf("types.ts was not generated")
	}

	manifestPath := filepath.Join(outDir, "manifest.ts")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Errorf("manifest.ts was not generated")
	}
}
