package directive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	// Disable go.work so temp directories work as standalone modules
	t.Setenv("GOWORK", "off")
	tests := []struct {
		name        string
		files       map[string]string
		wantExports []struct {
			name     string
			funcName string
		}
		wantConfig string // expected config func name, empty if none
		wantErr    string // expected error substring, empty if none
	}{
		{
			name: "single unnamed export",
			files: map[string]string{
				"main.go": `package main

//tygor:export
func setupApp() *App {
	return nil
}
`,
			},
			wantExports: []struct {
				name     string
				funcName string
			}{
				{name: "", funcName: "setupApp"},
			},
		},
		{
			name: "named export",
			files: map[string]string{
				"main.go": `package main

//tygor:export public
func setupPublic() *App {
	return nil
}
`,
			},
			wantExports: []struct {
				name     string
				funcName string
			}{
				{name: "public", funcName: "setupPublic"},
			},
		},
		{
			name: "multiple exports",
			files: map[string]string{
				"main.go": `package main

//tygor:export public
func setupPublic() *App {
	return nil
}

//tygor:export admin
func setupAdmin() *App {
	return nil
}
`,
			},
			wantExports: []struct {
				name     string
				funcName string
			}{
				{name: "public", funcName: "setupPublic"},
				{name: "admin", funcName: "setupAdmin"},
			},
		},
		{
			name: "export with config",
			files: map[string]string{
				"main.go": `package main

//tygor:export
func setupApp() *App {
	return nil
}

//tygor:config
func configure(g *Generator) *Generator {
	return g
}
`,
			},
			wantExports: []struct {
				name     string
				funcName string
			}{
				{name: "", funcName: "setupApp"},
			},
			wantConfig: "configure",
		},
		{
			name: "multiple configs error",
			files: map[string]string{
				"main.go": `package main

//tygor:config
func config1(g *Generator) *Generator {
	return g
}

//tygor:config
func config2(g *Generator) *Generator {
	return g
}
`,
			},
			wantErr: "multiple //tygor:config directives",
		},
		{
			name: "directive not on function",
			files: map[string]string{
				"main.go": `package main

//tygor:export
var x = 1
`,
			},
			wantErr: "must be followed by a function declaration",
		},
		{
			name: "unknown directive",
			files: map[string]string{
				"main.go": `package main

//tygor:unknown
func foo() {}
`,
			},
			wantErr: "unknown directive //tygor:unknown",
		},
		{
			name: "exports across files",
			files: map[string]string{
				"public.go": `package main

//tygor:export public
func setupPublic() *App {
	return nil
}
`,
				"admin.go": `package main

//tygor:export admin
func setupAdmin() *App {
	return nil
}
`,
			},
			wantExports: []struct {
				name     string
				funcName string
			}{
				{name: "admin", funcName: "setupAdmin"},
				{name: "public", funcName: "setupPublic"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			// Write go.mod
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
				t.Fatal(err)
			}

			// Write test files
			for name, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}

			result, err := ParseDir(".", dir)

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

			// Build a map for easier comparison (order may vary across files)
			gotExports := make(map[string]string)
			for _, e := range result.Exports {
				gotExports[e.FuncName] = e.Name
			}

			for _, want := range tt.wantExports {
				gotName, ok := gotExports[want.funcName]
				if !ok {
					t.Errorf("missing export for func %s", want.funcName)
					continue
				}
				if gotName != want.name {
					t.Errorf("export %s: got name %q, want %q", want.funcName, gotName, want.name)
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
