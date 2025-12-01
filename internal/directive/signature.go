package directive

import (
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ExportType represents the return type of an export function.
type ExportType int

const (
	ExportTypeUnknown   ExportType = iota
	ExportTypeApp                  // *tygor.App
	ExportTypeGenerator            // *tygorgen.Generator
)

func (t ExportType) String() string {
	switch t {
	case ExportTypeApp:
		return "*tygor.App"
	case ExportTypeGenerator:
		return "*tygorgen.Generator"
	default:
		return "unknown"
	}
}

// Export represents a validated export directive with type information.
type Export struct {
	Directive
	Type ExportType // The return type of the export function
}

// Config represents a validated config directive.
type Config struct {
	Directive
}

// TypedResult contains directives with validated type information.
type TypedResult struct {
	Exports     []Export
	Config      *Config
	PackagePath string
	Dir         string
}

// ParseWithTypes scans a Go package for tygor directives and validates their signatures.
//
// Export functions must have signature:
//   - func() *tygor.App
//   - func() *tygorgen.Generator
//
// Config functions must have signature:
//   - func(*tygorgen.Generator) *tygorgen.Generator
func ParseWithTypes(pattern string) (*TypedResult, error) {
	return ParseWithTypesDir(pattern, "")
}

// ParseWithTypesDir is like ParseWithTypes but allows specifying a working directory.
func ParseWithTypesDir(pattern, dir string) (*TypedResult, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo,
		Dir: dir,
	}

	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, fmt.Errorf("load package: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found matching %q", pattern)
	}

	if len(pkgs) > 1 {
		return nil, fmt.Errorf("multiple packages found matching %q; specify a single package", pattern)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package errors: %v", pkg.Errors[0])
	}

	result := &TypedResult{
		PackagePath: pkg.PkgPath,
	}

	if len(pkg.GoFiles) > 0 {
		result.Dir = filepath.Dir(pkg.GoFiles[0])
	}

	// First pass: find directive comments and match to functions
	directives, err := findDirectives(pkg)
	if err != nil {
		return nil, err
	}

	// Second pass: validate signatures using type info
	var configDirective *Directive
	for _, d := range directives {
		switch d.Kind {
		case KindExport:
			export, err := validateExport(pkg, d)
			if err != nil {
				return nil, err
			}
			result.Exports = append(result.Exports, *export)

		case KindConfig:
			if configDirective != nil {
				return nil, fmt.Errorf("multiple //tygor:config directives found:\n  %s\n  %s",
					configDirective.Pos, d.Pos)
			}
			configDirective = &d
			config, err := validateConfig(pkg, d)
			if err != nil {
				return nil, err
			}
			result.Config = config
		}
	}

	return result, nil
}

// findDirectives extracts directives from AST and matches them to functions.
// It also validates that directives are not placed on methods.
func findDirectives(pkg *packages.Package) ([]Directive, error) {
	var directives []Directive

	for _, f := range pkg.Syntax {
		// Build a map of comment end positions to directives
		type pending struct {
			kind Kind
			name string
			pos  string
		}
		commentToDirective := make(map[int]pending) // line number -> pending

		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if !strings.HasPrefix(c.Text, "//tygor:") {
					continue
				}

				text := strings.TrimPrefix(c.Text, "//tygor:")
				parts := strings.Fields(text)
				if len(parts) == 0 {
					continue
				}

				pos := pkg.Fset.Position(c.Pos())
				switch parts[0] {
				case "export":
					name := ""
					if len(parts) > 1 {
						name = parts[1]
					}
					// Store by the line after the comment group
					endLine := pkg.Fset.Position(cg.End()).Line
					commentToDirective[endLine] = pending{
						kind: KindExport,
						name: name,
						pos:  pos.String(),
					}
				case "config":
					endLine := pkg.Fset.Position(cg.End()).Line
					commentToDirective[endLine] = pending{
						kind: KindConfig,
						pos:  pos.String(),
					}
				default:
					return nil, fmt.Errorf("%s: unknown directive //tygor:%s", pos, parts[0])
				}
			}
		}

		// Match directives to function declarations
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			// Check if there's a directive in the doc comment
			if fn.Doc != nil {
				endLine := pkg.Fset.Position(fn.Doc.End()).Line
				if p, ok := commentToDirective[endLine]; ok {
					// Methods are not allowed
					if fn.Recv != nil {
						return nil, fmt.Errorf("%s: //tygor:%s must be on a package-level function, not a method",
							p.pos, p.kind)
					}

					pos := pkg.Fset.Position(fn.Pos())
					directives = append(directives, Directive{
						Kind:     p.kind,
						Name:     p.name,
						FuncName: fn.Name.Name,
						Pos:      pos,
					})
					delete(commentToDirective, endLine)
				}
			}
		}

		// Check for unmatched directives
		for _, p := range commentToDirective {
			return nil, fmt.Errorf("%s: //tygor:%s directive must be followed by a function declaration", p.pos, p.kind)
		}
	}

	return directives, nil
}

// validateExport checks that an export function has a valid signature.
func validateExport(pkg *packages.Package, d Directive) (*Export, error) {
	obj := pkg.Types.Scope().Lookup(d.FuncName)
	if obj == nil {
		return nil, fmt.Errorf("%s: function %s not found in package scope", d.Pos, d.FuncName)
	}

	fn, ok := obj.(*types.Func)
	if !ok {
		return nil, fmt.Errorf("%s: %s is not a function", d.Pos, d.FuncName)
	}

	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("%s: %s has invalid type", d.Pos, d.FuncName)
	}

	// Note: receiver check is done in findDirectives (AST level)

	// Check: no parameters
	if sig.Params().Len() != 0 {
		return nil, fmt.Errorf("%s: export function %s must have no parameters\n  got: func(%s)",
			d.Pos, d.FuncName, formatParams(sig.Params()))
	}

	// Check: exactly one return value
	if sig.Results().Len() != 1 {
		return nil, fmt.Errorf("%s: export function %s must return exactly one value\n  got: %d return values",
			d.Pos, d.FuncName, sig.Results().Len())
	}

	// Check: return type is *tygor.App or *tygorgen.Generator
	ret := sig.Results().At(0).Type()
	exportType, err := classifyExportType(ret)
	if err != nil {
		return nil, fmt.Errorf("%s: export function %s has invalid return type\n  got: %s\n  expected: *tygor.App or *tygorgen.Generator",
			d.Pos, d.FuncName, ret)
	}

	return &Export{
		Directive: d,
		Type:      exportType,
	}, nil
}

// validateConfig checks that a config function has a valid signature.
func validateConfig(pkg *packages.Package, d Directive) (*Config, error) {
	obj := pkg.Types.Scope().Lookup(d.FuncName)
	if obj == nil {
		return nil, fmt.Errorf("%s: function %s not found in package scope", d.Pos, d.FuncName)
	}

	fn, ok := obj.(*types.Func)
	if !ok {
		return nil, fmt.Errorf("%s: %s is not a function", d.Pos, d.FuncName)
	}

	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("%s: %s has invalid type", d.Pos, d.FuncName)
	}

	// Note: receiver check is done in findDirectives (AST level)

	// Check: exactly one parameter of type *tygorgen.Generator
	if sig.Params().Len() != 1 {
		return nil, fmt.Errorf("%s: config function %s must have exactly one parameter\n  expected: func(*tygorgen.Generator) *tygorgen.Generator\n  got: func(%s)",
			d.Pos, d.FuncName, formatParams(sig.Params()))
	}

	param := sig.Params().At(0).Type()
	if !isGeneratorPtr(param) {
		return nil, fmt.Errorf("%s: config function %s parameter must be *tygorgen.Generator\n  got: %s",
			d.Pos, d.FuncName, param)
	}

	// Check: exactly one return value of type *tygorgen.Generator
	if sig.Results().Len() != 1 {
		return nil, fmt.Errorf("%s: config function %s must return exactly one value\n  expected: *tygorgen.Generator\n  got: %d return values",
			d.Pos, d.FuncName, sig.Results().Len())
	}

	ret := sig.Results().At(0).Type()
	if !isGeneratorPtr(ret) {
		return nil, fmt.Errorf("%s: config function %s must return *tygorgen.Generator\n  got: %s",
			d.Pos, d.FuncName, ret)
	}

	return &Config{Directive: d}, nil
}

// classifyExportType determines if a type is *tygor.App or *tygorgen.Generator.
func classifyExportType(t types.Type) (ExportType, error) {
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return ExportTypeUnknown, fmt.Errorf("not a pointer type")
	}

	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return ExportTypeUnknown, fmt.Errorf("not a named type")
	}

	pkg := named.Obj().Pkg()
	if pkg == nil {
		return ExportTypeUnknown, fmt.Errorf("no package")
	}

	typeName := named.Obj().Name()
	pkgPath := pkg.Path()

	switch {
	case pkgPath == "github.com/broady/tygor" && typeName == "App":
		return ExportTypeApp, nil
	case pkgPath == "github.com/broady/tygor/tygorgen" && typeName == "Generator":
		return ExportTypeGenerator, nil
	default:
		return ExportTypeUnknown, fmt.Errorf("unknown type %s.%s", pkgPath, typeName)
	}
}

// isGeneratorPtr checks if a type is *tygorgen.Generator.
func isGeneratorPtr(t types.Type) bool {
	et, err := classifyExportType(t)
	return err == nil && et == ExportTypeGenerator
}

// formatParams formats a types.Tuple as a parameter list string.
func formatParams(params *types.Tuple) string {
	if params.Len() == 0 {
		return ""
	}
	var parts []string
	for i := 0; i < params.Len(); i++ {
		parts = append(parts, params.At(i).Type().String())
	}
	return strings.Join(parts, ", ")
}
