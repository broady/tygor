package tygor

import (
	"github.com/broady/tygor/generator"
)

// GenConfig holds the configuration for code generation.
type GenConfig struct {
	// OutDir is the directory where generated files will be written.
	// e.g. "./client/src/rpc"
	OutDir string

	// TypeMappings allows overriding type mappings for tygo.
	// e.g. map[string]string{"time.Time": "Date", "CustomType": "string"}
	TypeMappings map[string]string

	// PreserveComments controls whether Go doc comments are preserved in TypeScript output.
	// Supported values: "default" (preserve package and type comments), "types" (only type comments), "none".
	// Default: "default"
	PreserveComments string

	// EnumStyle controls how Go const groups are generated in TypeScript.
	// Supported values: "union" (type unions), "enum" (TS enums), "const" (individual consts).
	// Default: "union"
	EnumStyle string

	// OptionalType controls how optional fields (Go pointers) are typed in TypeScript.
	// Supported values: "undefined" (T | undefined), "null" (T | null).
	// Default: "undefined"
	OptionalType string

	// Frontmatter is content added to the top of each generated TypeScript file.
	// Useful for custom type definitions or imports.
	// e.g. "export type DateTime = string & { __brand: 'DateTime' };"
	Frontmatter string
}

// Generate generates the TypeScript types and manifest for the registered services.
func (r *Registry) Generate(cfg *GenConfig) error {
	// Extract routes metadata
	r.mu.RLock()
	routes := make(map[string]generator.RouteMetadata)
	for k, v := range r.routes {
		meta := v.Metadata()
		routes[k] = generator.RouteMetadata{
			Request:  meta.Request,
			Response: meta.Response,
			Method:   meta.Method,
		}
	}
	r.mu.RUnlock()

	// Convert GenConfig to generator.Config
	genCfg := &generator.Config{
		OutDir:           cfg.OutDir,
		TypeMappings:     cfg.TypeMappings,
		PreserveComments: cfg.PreserveComments,
		EnumStyle:        cfg.EnumStyle,
		OptionalType:     cfg.OptionalType,
		Frontmatter:      cfg.Frontmatter,
	}

	return generator.Generate(routes, genCfg)
}
