package tygorgen

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/broady/tygor"
	"github.com/broady/tygor/internal"
	"github.com/broady/tygor/tygorgen/ir"
	"github.com/broady/tygor/tygorgen/provider"
	"github.com/broady/tygor/tygorgen/sink"
	"github.com/broady/tygor/tygorgen/typescript"
)

// Config holds the configuration for code generation.
type Config struct {
	// OutDir is the directory where generated files will be written.
	// e.g. "./client/src/rpc"
	OutDir string

	// Provider selects the type extraction strategy.
	// "source" (default) - uses go/packages for full type info including enums and comments
	// "reflection" - uses runtime reflection (faster, but no enum values or comments)
	Provider string

	// Packages are the Go package paths to analyze when using source provider.
	// Required when Provider is "source".
	// e.g. []string{"github.com/myorg/myapp/api"}
	Packages []string

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

	// StripPackagePrefix removes this prefix from package paths when qualifying type names.
	// Use this when you have same-named types in different packages (e.g., v1.User and v2.User).
	// Example: "github.com/myorg/myrepo/" makes "github.com/myorg/myrepo/api/v1.User" → "v1_User"
	// Without this, types from different packages with the same name will collide.
	StripPackagePrefix string

	// SingleFile emits all types in a single types.ts file.
	// Default (false) generates one file per Go package with a barrel types.ts that re-exports all.
	SingleFile bool
}

// Generate generates the TypeScript types and manifest for the registered services.
func Generate(app *tygor.App, cfg *Config) error {
	routes := app.Routes()
	if cfg.OutDir == "" {
		return fmt.Errorf("OutDir is required")
	}

	// Apply defaults
	cfg = applyConfigDefaults(cfg)

	ctx := context.Background()

	// 1. Build schema using configured provider
	var schema *ir.Schema
	var err error

	switch cfg.Provider {
	case "source":
		schema, err = buildSchemaFromSource(ctx, routes, cfg.Packages)
	case "reflection":
		schema, err = buildSchemaFromReflection(ctx, routes)
	default:
		return fmt.Errorf("unknown provider: %q (expected \"source\" or \"reflection\")", cfg.Provider)
	}

	if err != nil {
		return fmt.Errorf("failed to build schema: %w", err)
	}

	// 2. Build service descriptors from routes
	services := buildServiceDescriptors(routes)
	schema.Services = services

	// 3. Configure TypeScript generator
	tsConfig := typescript.GeneratorConfig{
		TypePrefix:         "",
		TypeSuffix:         "",
		FieldCase:          "preserve",
		TypeCase:           "preserve",
		StripPackagePrefix: cfg.StripPackagePrefix,
		SingleFile:         cfg.SingleFile,
		IndentStyle:        "space",
		IndentSize:         2,
		LineEnding:         "lf",
		TrailingNewline:    true,
		EmitComments:       cfg.PreserveComments != "none",
		Frontmatter:        cfg.Frontmatter,
		TypeMappings:       cfg.TypeMappings,
		Custom: map[string]any{
			"EmitExport":        true,
			"EmitDeclare":       false,
			"UseInterface":      true,
			"UseReadonlyArrays": false,
			"EnumStyle":         cfg.EnumStyle,
			"OptionalType":      cfg.OptionalType,
			"UnknownType":       "unknown",
		},
	}

	// 4. Create filesystem sink
	filesystemSink := sink.NewFilesystemSink(cfg.OutDir)

	// 5. Generate TypeScript
	gen := &typescript.TypeScriptGenerator{}
	result, err := gen.Generate(ctx, schema, typescript.GenerateOptions{
		Sink:   filesystemSink,
		Config: tsConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to generate TypeScript: %w", err)
	}

	// Report warnings if any
	if len(result.Warnings) > 0 {
		for _, w := range result.Warnings {
			fmt.Fprintf(os.Stderr, "Warning: %s: %s\n", w.Code, w.Message)
		}
	}

	return nil
}

// buildServiceDescriptors converts route metadata to IR service descriptors.
func buildServiceDescriptors(routes internal.RouteMap) []ir.ServiceDescriptor {
	// Group routes by service
	serviceMap := make(map[string]*ir.ServiceDescriptor)

	// Sort route keys for deterministic output
	keys := make([]string, 0, len(routes))
	for k := range routes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		route := routes[key]

		// Parse service and method name from key (e.g., "Users.Create")
		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 {
			continue // Skip malformed keys
		}
		serviceName, methodName := parts[0], parts[1]

		// Get or create service
		service, exists := serviceMap[serviceName]
		if !exists {
			service = &ir.ServiceDescriptor{
				Name:      serviceName,
				Endpoints: []ir.EndpointDescriptor{},
			}
			serviceMap[serviceName] = service
		}

		// Build endpoint descriptor
		endpoint := ir.EndpointDescriptor{
			Name:       methodName,
			FullName:   key,
			HTTPMethod: route.HTTPMethod,
			Path:       "/" + strings.Replace(key, ".", "/", 1),
		}

		// Convert request type to descriptor
		if route.Request != nil {
			endpoint.Request = reflectTypeToIRRef(route.Request)
		}

		// Convert response type to descriptor
		if route.Response != nil {
			endpoint.Response = reflectTypeToIRRef(route.Response)
		} else {
			// No response type means void/empty
			endpoint.Response = ir.Ptr(ir.Empty())
		}

		service.Endpoints = append(service.Endpoints, endpoint)
	}

	// Convert map to sorted slice
	serviceNames := make([]string, 0, len(serviceMap))
	for name := range serviceMap {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	services := make([]ir.ServiceDescriptor, 0, len(serviceMap))
	for _, name := range serviceNames {
		services = append(services, *serviceMap[name])
	}

	return services
}

// reflectTypeToIRRef converts a reflect.Type to an IR TypeDescriptor reference.
// This handles the basic mapping from Go types to IR type expressions.
func reflectTypeToIRRef(t reflect.Type) ir.TypeDescriptor {
	if t == nil {
		return ir.Any()
	}

	// Unwrap pointers
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Handle slices/arrays
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		elem := reflectTypeToIRRef(t.Elem())
		if t.Kind() == reflect.Slice {
			return ir.Slice(elem)
		}
		return ir.Array(elem, t.Len())
	}

	// Handle maps
	if t.Kind() == reflect.Map {
		key := reflectTypeToIRRef(t.Key())
		value := reflectTypeToIRRef(t.Elem())
		return ir.Map(key, value)
	}

	// For named types (structs, aliases), create a reference
	if t.Name() != "" {
		// Sanitize generic type names to match what the reflection provider generates
		name := sanitizeTypeName(t.Name())
		return ir.Ref(name, t.PkgPath())
	}

	// Fallback for primitives and unnamed types
	return ir.Any()
}

// sanitizeTypeName applies the synthetic naming algorithm for generic instantiations.
// This must match the logic in provider/reflection.go generateSyntheticName.
func sanitizeTypeName(name string) string {
	result := strings.ReplaceAll(name, ".", "_")
	result = strings.ReplaceAll(result, "/", "_")
	result = strings.ReplaceAll(result, "[", "_")
	result = strings.ReplaceAll(result, "]", "")
	result = strings.ReplaceAll(result, ",", "_")
	result = strings.ReplaceAll(result, " ", "")
	result = strings.ReplaceAll(result, "*", "Ptr")
	return result
}

// applyConfigDefaults applies default values to Config.
func applyConfigDefaults(cfg *Config) *Config {
	// Make a copy to avoid mutating the input
	result := *cfg

	if result.Provider == "" {
		result.Provider = "source"
	}
	if result.PreserveComments == "" {
		result.PreserveComments = "default"
	}
	if result.EnumStyle == "" {
		result.EnumStyle = "union"
	}
	if result.OptionalType == "" {
		result.OptionalType = "undefined"
	}

	return &result
}

// buildSchemaFromSource uses the source provider to extract types.
func buildSchemaFromSource(ctx context.Context, routes internal.RouteMap, packages []string) (*ir.Schema, error) {
	if len(packages) == 0 {
		return nil, fmt.Errorf("packages is required when using source provider")
	}

	// Collect root type names from routes
	rootTypes := collectRootTypeNames(routes)

	p := &provider.SourceProvider{}
	opts := provider.SourceInputOptions{
		Packages:  packages,
		RootTypes: rootTypes,
	}
	return p.BuildSchema(ctx, opts)
}

// buildSchemaFromReflection uses the reflection provider to extract types.
func buildSchemaFromReflection(ctx context.Context, routes internal.RouteMap) (*ir.Schema, error) {
	// Collect reflect.Types from routes
	rootTypes := make([]reflect.Type, 0, len(routes)*2)
	for _, route := range routes {
		if route.Request != nil {
			rootTypes = append(rootTypes, route.Request)
		}
		if route.Response != nil {
			rootTypes = append(rootTypes, route.Response)
		}
	}

	if len(rootTypes) == 0 {
		// Empty schema for apps with no handlers
		return &ir.Schema{
			Types:    []ir.TypeDescriptor{},
			Services: []ir.ServiceDescriptor{},
		}, nil
	}

	p := &provider.ReflectionProvider{}
	opts := provider.ReflectionInputOptions{
		RootTypes: rootTypes,
	}
	return p.BuildSchema(ctx, opts)
}

// collectRootTypeNames extracts type names from routes for source provider.
func collectRootTypeNames(routes internal.RouteMap) []string {
	seen := make(map[string]bool)
	var names []string

	for _, route := range routes {
		if route.Request != nil {
			name := route.Request.Name()
			if name != "" && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
		if route.Response != nil {
			name := route.Response.Name()
			if name != "" && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}

	sort.Strings(names)
	return names
}
