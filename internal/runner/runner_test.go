package runner

import (
	"go/token"
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
