// Package directive parses tygor directives from Go source files.
//
// Directives are line comments in the form:
//
//	//tygor:export [name]
//	//tygor:config
//
// The export directive marks a function that returns *tygor.App or *tygorgen.Generator.
// The optional name allows selecting between multiple exports.
//
// The config directive marks a function that modifies the generator configuration.
// Only one config directive is allowed per package.
package directive

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Directive represents a parsed tygor directive.
type Directive struct {
	Kind     Kind           // export or config
	Name     string         // optional name for exports (empty if unnamed)
	FuncName string         // name of the function
	Pos      token.Position // source location
}

// Kind represents the type of directive.
type Kind string

const (
	KindExport Kind = "export"
	KindConfig Kind = "config"
)

// Result contains all directives found in a package.
type Result struct {
	// Exports contains all //tygor:export directives found.
	Exports []Directive

	// Config contains the //tygor:config directive, if any.
	// At most one config directive is allowed.
	Config *Directive

	// PackagePath is the import path of the parsed package.
	PackagePath string

	// Dir is the directory containing the package.
	Dir string
}

// Parse scans a Go package for tygor directives.
//
// The pattern follows go command semantics:
//   - "." for current directory
//   - "./..." for current directory and subdirectories (not supported yet)
//   - Import path like "github.com/foo/bar"
//   - Absolute or relative directory path
//
// Returns an error if:
//   - The package cannot be loaded
//   - Multiple //tygor:config directives are found
//   - A directive is not immediately followed by a function declaration
func Parse(pattern string) (*Result, error) {
	return ParseDir(pattern, "")
}

// ParseDir is like Parse but allows specifying a working directory.
// If dir is empty, the current directory is used.
func ParseDir(pattern, dir string) (*Result, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax,
		Dir:  dir,
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

	result := &Result{
		PackagePath: pkg.PkgPath,
	}

	if len(pkg.GoFiles) > 0 {
		result.Dir = filepath.Dir(pkg.GoFiles[0])
	}

	fset := token.NewFileSet()
	for _, filename := range pkg.GoFiles {
		f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filename, err)
		}

		directives, err := parseFile(fset, f)
		if err != nil {
			return nil, err
		}

		for _, d := range directives {
			switch d.Kind {
			case KindExport:
				result.Exports = append(result.Exports, d)
			case KindConfig:
				if result.Config != nil {
					return nil, fmt.Errorf("multiple //tygor:config directives found:\n  %s\n  %s",
						result.Config.Pos, d.Pos)
				}
				result.Config = &d
			}
		}
	}

	return result, nil
}

// parseFile extracts directives from a single file.
func parseFile(fset *token.FileSet, f *ast.File) ([]Directive, error) {
	var directives []Directive

	// Build a map of comment end positions to directives
	// so we can match them to the following function declarations.
	type pending struct {
		kind Kind
		name string
		pos  token.Position
	}
	commentToDirective := make(map[token.Pos]pending)

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

			pos := fset.Position(c.Pos())
			switch parts[0] {
			case "export":
				name := ""
				if len(parts) > 1 {
					name = parts[1]
				}
				commentToDirective[cg.End()] = pending{
					kind: KindExport,
					name: name,
					pos:  pos,
				}
			case "config":
				commentToDirective[cg.End()] = pending{
					kind: KindConfig,
					pos:  pos,
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

		// Check if there's a directive comment group ending just before this function
		if fn.Doc != nil {
			if p, ok := commentToDirective[fn.Doc.End()]; ok {
				directives = append(directives, Directive{
					Kind:     p.kind,
					Name:     p.name,
					FuncName: fn.Name.Name,
					Pos:      p.pos,
				})
				delete(commentToDirective, fn.Doc.End())
			}
		}
	}

	// Check for unmatched directives
	for _, p := range commentToDirective {
		return nil, fmt.Errorf("%s: //tygor:%s directive must be followed by a function declaration", p.pos, p.kind)
	}

	return directives, nil
}
