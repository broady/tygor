// Package provider implements input providers that extract type information
// from Go code and convert it to the intermediate representation.
package provider

import (
	"context"
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"strings"

	"github.com/broady/tygor/tygorgen/ir"
	"golang.org/x/tools/go/packages"
)

// SourceProvider extracts types by analyzing Go source code.
type SourceProvider struct{}

// SourceInputOptions configures source-based type extraction.
type SourceInputOptions struct {
	// Packages are the Go package paths to analyze.
	Packages []string

	// RootTypes are the type names to extract (e.g., "User", "CreateRequest").
	// If empty, all exported types in the packages are extracted.
	RootTypes []string
}

// BuildSchema analyzes source code and returns a Schema.
// The provider recursively extracts all types reachable from RootTypes.
func (p *SourceProvider) BuildSchema(ctx context.Context, opts SourceInputOptions) (*ir.Schema, error) {
	if len(opts.Packages) == 0 {
		return nil, fmt.Errorf("no packages specified")
	}

	// Load packages using go/packages
	cfg := &packages.Config{
		Context: ctx,
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
	}

	pkgs, err := packages.Load(cfg, opts.Packages...)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	// Check for errors in loaded packages
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return nil, fmt.Errorf("package %s has errors: %v", pkg.PkgPath, pkg.Errors)
		}
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found")
	}

	// Create a builder to extract types
	builder := &schemaBuilder{
		pkgs:           pkgs,
		schema:         &ir.Schema{},
		seen:           make(map[types.Type]bool),
		namedTypes:     make(map[string]ir.TypeDescriptor),
		enumCandidates: make(map[*types.Named][]enumConstant),
	}

	// Populate package info from first INPUT package (not first loaded package)
	// packages.Load returns packages in dependency order, not input order
	mainPkg := pkgs[0]
	for _, pkg := range pkgs {
		if pkg.PkgPath == opts.Packages[0] {
			mainPkg = pkg
			break
		}
	}
	builder.schema.Package = ir.PackageInfo{
		Path: mainPkg.PkgPath,
		Name: mainPkg.Name,
		Dir:  mainPkg.PkgPath,
	}

	// Find and process root types
	if len(opts.RootTypes) > 0 {
		// Extract specific root types
		for _, rootName := range opts.RootTypes {
			if err := builder.extractRootType(rootName); err != nil {
				return nil, fmt.Errorf("failed to extract root type %s: %w", rootName, err)
			}
		}
	} else {
		// Extract all exported types
		if err := builder.extractAllExportedTypes(); err != nil {
			return nil, fmt.Errorf("failed to extract exported types: %w", err)
		}
	}

	return builder.schema, nil
}

// schemaBuilder accumulates types and manages the extraction process.
type schemaBuilder struct {
	pkgs           []*packages.Package
	schema         *ir.Schema
	seen           map[types.Type]bool
	namedTypes     map[string]ir.TypeDescriptor // key: pkgPath.Name
	enumCandidates map[*types.Named][]enumConstant
}

// enumConstant represents a const declaration that might be an enum member.
type enumConstant struct {
	name  string
	value constant.Value
}

// extractRootType finds and extracts a named type by name.
func (b *schemaBuilder) extractRootType(name string) error {
	for _, pkg := range b.pkgs {
		obj := pkg.Types.Scope().Lookup(name)
		if obj == nil {
			continue
		}

		typeName, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}

		if err := b.extractNamedType(typeName); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("type %s not found in any package", name)
}

// extractAllExportedTypes extracts all exported types from all packages.
func (b *schemaBuilder) extractAllExportedTypes() error {
	for _, pkg := range b.pkgs {
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if !obj.Exported() {
				continue
			}

			typeName, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}

			if err := b.extractNamedType(typeName); err != nil {
				return err
			}
		}
	}
	return nil
}

// extractNamedType extracts a named type and recursively processes dependencies.
func (b *schemaBuilder) extractNamedType(tn *types.TypeName) error {
	named, ok := tn.Type().(*types.Named)
	if !ok {
		return nil
	}

	// Check if already processed
	key := b.typeKey(named)
	if _, exists := b.namedTypes[key]; exists {
		return nil
	}

	// First, scan for enum constants before processing the type
	b.scanEnumConstants(tn)

	// Extract documentation and source location
	doc := b.extractDocumentation(tn)
	src := b.extractSource(tn)

	// Check if this is an enum type
	if consts, isEnum := b.enumCandidates[named]; isEnum && len(consts) > 0 {
		enumDesc := b.buildEnumDescriptor(tn.Name(), named.Obj().Pkg().Path(), consts, doc, src)
		b.namedTypes[key] = enumDesc
		b.schema.AddType(enumDesc)
		return nil
	}

	// Check for custom marshalers on the named type itself
	if b.hasCustomMarshaler(named) {
		b.schema.AddWarning(ir.Warning{
			Code:     "CUSTOM_MARSHALER",
			Message:  fmt.Sprintf("type %s implements custom marshaler, mapped to 'any'", tn.Name()),
			TypeName: tn.Name(),
		})
		// Create an alias to PrimitiveAny
		aliasDesc := &ir.AliasDescriptor{
			Name:          ir.GoIdentifier{Name: tn.Name(), Package: named.Obj().Pkg().Path()},
			Underlying:    ir.Any(),
			Documentation: doc,
			Source:        src,
		}
		b.namedTypes[key] = aliasDesc
		b.schema.AddType(aliasDesc)
		return nil
	}

	// Check the underlying type
	switch underlyingType := named.Underlying().(type) {
	case *types.Struct:
		structDesc, err := b.buildStructDescriptor(named, tn.Name(), doc, src)
		if err != nil {
			return err
		}
		b.namedTypes[key] = structDesc
		b.schema.AddType(structDesc)

	case *types.Interface:
		// Interfaces are emitted as PrimitiveAny with a warning
		b.schema.AddWarning(ir.Warning{
			Code:     "INTERFACE_TYPE",
			Message:  fmt.Sprintf("interface type %s mapped to 'any'", tn.Name()),
			TypeName: tn.Name(),
		})
		// Create an alias to PrimitiveAny
		aliasDesc := &ir.AliasDescriptor{
			Name:          ir.GoIdentifier{Name: tn.Name(), Package: named.Obj().Pkg().Path()},
			Underlying:    ir.Any(),
			Documentation: doc,
			Source:        src,
		}
		b.namedTypes[key] = aliasDesc
		b.schema.AddType(aliasDesc)

	default:
		// Regular type alias
		underlying, err := b.convertType(underlyingType)
		if err != nil {
			return err
		}
		aliasDesc := &ir.AliasDescriptor{
			Name:          ir.GoIdentifier{Name: tn.Name(), Package: named.Obj().Pkg().Path()},
			Underlying:    underlying,
			Documentation: doc,
			Source:        src,
		}
		b.namedTypes[key] = aliasDesc
		b.schema.AddType(aliasDesc)
	}

	return nil
}

// typeKey generates a unique key for a named type.
func (b *schemaBuilder) typeKey(named *types.Named) string {
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return named.String()
	}
	return obj.Pkg().Path() + "." + obj.Name()
}

// convertType converts a Go type to an IR TypeDescriptor.
func (b *schemaBuilder) convertType(t types.Type) (ir.TypeDescriptor, error) {
	// Handle special cases first
	if desc := b.handleSpecialType(t); desc != nil {
		return desc, nil
	}

	switch typ := t.(type) {
	case *types.Basic:
		return b.convertBasicType(typ), nil

	case *types.Named:
		// Reference to a named type
		obj := typ.Obj()
		pkgPath := ""
		if obj.Pkg() != nil {
			pkgPath = obj.Pkg().Path()
		}
		return ir.Ref(obj.Name(), pkgPath), nil

	case *types.Pointer:
		elem, err := b.convertType(typ.Elem())
		if err != nil {
			return nil, err
		}
		return ir.Ptr(elem), nil

	case *types.Slice:
		elem, err := b.convertType(typ.Elem())
		if err != nil {
			return nil, err
		}
		return ir.Slice(elem), nil

	case *types.Array:
		elem, err := b.convertType(typ.Elem())
		if err != nil {
			return nil, err
		}
		return ir.Array(elem, int(typ.Len())), nil

	case *types.Map:
		key, err := b.convertType(typ.Key())
		if err != nil {
			return nil, err
		}
		value, err := b.convertType(typ.Elem())
		if err != nil {
			return nil, err
		}
		// Validate map key type
		if !b.isValidMapKey(typ.Key()) {
			return nil, fmt.Errorf("unsupported map key type: %s", typ.Key())
		}
		return ir.Map(key, value), nil

	case *types.Interface:
		// Empty interface or any
		if typ.Empty() {
			return ir.Any(), nil
		}
		// Non-empty interface
		b.schema.AddWarning(ir.Warning{
			Code:    "INTERFACE_TYPE",
			Message: fmt.Sprintf("interface type %s mapped to 'any'", typ.String()),
		})
		return ir.Any(), nil

	case *types.Struct:
		// Anonymous struct - need to generate a synthetic name
		return nil, fmt.Errorf("anonymous structs should be handled by caller")

	case *types.TypeParam:
		// Generic type parameter
		return ir.TypeParam(typ.Obj().Name(), nil), nil

	case *types.Alias:
		// Type alias - follow to the actual type
		return b.convertType(typ.Rhs())

	case *types.Chan, *types.Signature:
		return nil, fmt.Errorf("unsupported type: %s", t.String())

	default:
		return nil, fmt.Errorf("unknown type: %T", t)
	}
}

// handleSpecialType handles special types like time.Time, []byte, etc.
func (b *schemaBuilder) handleSpecialType(t types.Type) ir.TypeDescriptor {
	switch typ := t.(type) {
	case *types.Slice:
		// Check for []byte or []uint8
		if basic, ok := typ.Elem().(*types.Basic); ok {
			if basic.Kind() == types.Byte || basic.Kind() == types.Uint8 {
				return ir.Bytes()
			}
		}

	case *types.Named:
		obj := typ.Obj()
		if obj == nil || obj.Pkg() == nil {
			return nil
		}

		pkgPath := obj.Pkg().Path()
		name := obj.Name()

		// time.Time
		if pkgPath == "time" && name == "Time" {
			return ir.Time()
		}

		// time.Duration
		if pkgPath == "time" && name == "Duration" {
			return ir.Duration()
		}

		// Check for custom marshalers
		if b.hasCustomMarshaler(typ) {
			b.schema.AddWarning(ir.Warning{
				Code:     "CUSTOM_MARSHALER",
				Message:  fmt.Sprintf("type %s implements custom marshaler, mapped to 'any'", name),
				TypeName: name,
			})
			return ir.Any()
		}
	}

	return nil
}

// hasCustomMarshaler checks if a type implements json.Marshaler or encoding.TextMarshaler.
func (b *schemaBuilder) hasCustomMarshaler(named *types.Named) bool {
	// Look for MarshalJSON() ([]byte, error) method
	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)
		if method.Name() == "MarshalJSON" {
			sig := method.Type().(*types.Signature)
			if sig.Params().Len() == 0 && sig.Results().Len() == 2 {
				// Check return types: []byte, error
				return true
			}
		}
		if method.Name() == "MarshalText" {
			sig := method.Type().(*types.Signature)
			if sig.Params().Len() == 0 && sig.Results().Len() == 2 {
				return true
			}
		}
	}
	return false
}

// isValidMapKey checks if a type is a valid JSON map key.
func (b *schemaBuilder) isValidMapKey(t types.Type) bool {
	switch typ := t.(type) {
	case *types.Basic:
		kind := typ.Kind()
		// String and integer types are valid
		return kind == types.String ||
			kind >= types.Int && kind <= types.Uint64

	case *types.Named:
		// Check underlying type and TextMarshaler
		if b.hasTextMarshaler(typ) {
			return true
		}
		return b.isValidMapKey(typ.Underlying())

	default:
		return false
	}
}

// hasTextMarshaler checks if a type implements encoding.TextMarshaler.
func (b *schemaBuilder) hasTextMarshaler(named *types.Named) bool {
	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)
		if method.Name() == "MarshalText" {
			sig := method.Type().(*types.Signature)
			if sig.Params().Len() == 0 && sig.Results().Len() == 2 {
				return true
			}
		}
	}
	return false
}

// convertBasicType converts a Go basic type to an IR primitive.
func (b *schemaBuilder) convertBasicType(basic *types.Basic) ir.TypeDescriptor {
	switch basic.Kind() {
	case types.Bool:
		return ir.Bool()
	case types.String:
		return ir.String()
	case types.Int:
		return ir.Int(0)
	case types.Int8:
		return ir.Int(8)
	case types.Int16:
		return ir.Int(16)
	case types.Int32:
		return ir.Int(32)
	case types.Int64:
		return ir.Int(64)
	case types.Uint, types.Uintptr:
		return ir.Uint(0)
	case types.Uint8: // types.Byte is an alias for Uint8
		return ir.Uint(8)
	case types.Uint16:
		return ir.Uint(16)
	case types.Uint32:
		return ir.Uint(32)
	case types.Uint64:
		return ir.Uint(64)
	case types.Float32:
		return ir.Float(32)
	case types.Float64:
		return ir.Float(64)
	case types.UntypedNil:
		return ir.Any()
	default:
		// Unsupported basic types
		return ir.Any()
	}
}

// extractDocumentation extracts documentation from an object.
func (b *schemaBuilder) extractDocumentation(obj types.Object) ir.Documentation {
	// Find the declaration in the AST
	for _, pkg := range b.pkgs {
		if pkg.Types != obj.Pkg() {
			continue
		}

		// Find the object's position
		pos := obj.Pos()
		for _, file := range pkg.Syntax {
			if file.Pos() > pos || file.End() < pos {
				continue
			}

			// Search for the declaration
			var docGroup *ast.CommentGroup
			ast.Inspect(file, func(n ast.Node) bool {
				switch decl := n.(type) {
				case *ast.GenDecl:
					for _, spec := range decl.Specs {
						if ts, ok := spec.(*ast.TypeSpec); ok {
							if ts.Name.Pos() == pos {
								docGroup = decl.Doc
								if docGroup == nil {
									docGroup = ts.Doc
								}
								return false
							}
						}
					}
				}
				return true
			})

			if docGroup != nil {
				return b.parseDocumentation(docGroup)
			}
		}
	}

	return ir.Documentation{}
}

// parseDocumentation parses a comment group into Documentation.
func (b *schemaBuilder) parseDocumentation(cg *ast.CommentGroup) ir.Documentation {
	if cg == nil {
		return ir.Documentation{}
	}

	text := cg.Text()
	lines := strings.Split(strings.TrimSpace(text), "\n")

	var summary string
	var deprecated *string

	// Check for deprecated marker
	for i, line := range lines {
		if strings.HasPrefix(line, "Deprecated:") {
			msg := strings.TrimSpace(strings.TrimPrefix(line, "Deprecated:"))
			deprecated = &msg
			// Remove this line from the body
			lines = append(lines[:i], lines[i+1:]...)
			break
		}
	}

	// First non-empty line is the summary
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			summary = trimmed
			break
		}
	}

	body := strings.Join(lines, "\n")

	return ir.Documentation{
		Summary:    summary,
		Body:       body,
		Deprecated: deprecated,
	}
}

// extractSource extracts source location information.
func (b *schemaBuilder) extractSource(obj types.Object) ir.Source {
	pos := obj.Pos()
	if !pos.IsValid() {
		return ir.Source{}
	}

	for _, pkg := range b.pkgs {
		if pkg.Fset != nil {
			position := pkg.Fset.Position(pos)
			return ir.Source{
				File:   position.Filename,
				Line:   position.Line,
				Column: position.Column,
			}
		}
	}

	return ir.Source{}
}

// scanEnumConstants scans for const declarations that might be enum members.
func (b *schemaBuilder) scanEnumConstants(tn *types.TypeName) {
	named, ok := tn.Type().(*types.Named)
	if !ok {
		return
	}

	// Only consider defined types with primitive underlying types
	underlying := named.Underlying()
	if _, ok := underlying.(*types.Basic); !ok {
		return
	}

	pkg := tn.Pkg()
	if pkg == nil {
		return
	}

	// Scan all const declarations in the package
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		cnst, ok := obj.(*types.Const)
		if !ok {
			continue
		}

		// Check if const has the same type as our named type
		if types.Identical(cnst.Type(), named) {
			b.enumCandidates[named] = append(b.enumCandidates[named], enumConstant{
				name:  cnst.Name(),
				value: cnst.Val(),
			})
		}
	}
}

// buildEnumDescriptor creates an EnumDescriptor from constants.
func (b *schemaBuilder) buildEnumDescriptor(name, pkgPath string, consts []enumConstant, doc ir.Documentation, src ir.Source) *ir.EnumDescriptor {
	members := make([]ir.EnumMember, len(consts))
	for i, c := range consts {
		value := b.constantValue(c.value)
		members[i] = ir.EnumMember{
			Name:          c.name,
			Value:         value,
			Documentation: ir.Documentation{}, // TODO: extract const documentation
		}
	}

	return &ir.EnumDescriptor{
		Name:          ir.GoIdentifier{Name: name, Package: pkgPath},
		Members:       members,
		Documentation: doc,
		Source:        src,
	}
}

// constantValue converts a constant.Value to string, int64, or float64.
func (b *schemaBuilder) constantValue(v constant.Value) any {
	switch v.Kind() {
	case constant.String:
		return constant.StringVal(v)
	case constant.Int:
		i64, _ := constant.Int64Val(v)
		return i64
	case constant.Float:
		f64, _ := constant.Float64Val(v)
		return f64
	case constant.Bool:
		return constant.BoolVal(v)
	default:
		return v.String()
	}
}

// buildStructDescriptor creates a StructDescriptor from a named struct type.
func (b *schemaBuilder) buildStructDescriptor(named *types.Named, name string, doc ir.Documentation, src ir.Source) (*ir.StructDescriptor, error) {
	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("not a struct type")
	}

	pkgPath := ""
	if named.Obj() != nil && named.Obj().Pkg() != nil {
		pkgPath = named.Obj().Pkg().Path()
	}

	// Extract type parameters for generic types
	var typeParams []ir.TypeParameterDescriptor
	if tparams := named.TypeParams(); tparams != nil && tparams.Len() > 0 {
		for i := 0; i < tparams.Len(); i++ {
			tp := tparams.At(i)
			// tp.Constraint() returns the constraint interface
			// For [T any], this returns a non-nil interface but it's the universal constraint
			// We need to check if it's effectively unconstrained
			constraintType := tp.Constraint()
			var constraint ir.TypeDescriptor
			if constraintType != nil {
				constraint = b.convertTypeParamConstraint(constraintType)
			}
			typeParams = append(typeParams, ir.TypeParameterDescriptor{
				ParamName:  tp.Obj().Name(),
				Constraint: constraint,
			})
		}
	}

	descriptor := &ir.StructDescriptor{
		Name:           ir.GoIdentifier{Name: name, Package: pkgPath},
		TypeParameters: typeParams,
		Fields:         []ir.FieldDescriptor{},
		Extends:        []ir.GoIdentifier{},
		Documentation:  doc,
		Source:         src,
	}

	// Process struct fields
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		// Skip unexported fields
		if !field.Exported() {
			continue
		}

		// Parse struct tags
		jsonTag, jsonOpts := b.parseJSONTag(tag)

		// Skip fields with json:"-"
		if jsonTag == "-" {
			continue
		}

		// Handle embedded fields
		if field.Embedded() {
			if jsonTag == "" {
				// No JSON tag - this is inheritance (Extends)
				fieldType := field.Type()
				// Dereference pointer
				if ptr, ok := fieldType.(*types.Pointer); ok {
					fieldType = ptr.Elem()
				}
				if named, ok := fieldType.(*types.Named); ok {
					obj := named.Obj()
					descriptor.Extends = append(descriptor.Extends, ir.GoIdentifier{
						Name:    obj.Name(),
						Package: obj.Pkg().Path(),
					})
				}
				continue
			}
			// Has JSON tag - treat as regular field
		}

		// Convert field type
		fieldTypeDesc, err := b.convertType(field.Type())
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %w", field.Name(), err)
		}

		// Determine JSON name
		jsonName := jsonTag
		if jsonName == "" {
			jsonName = field.Name()
		}

		// Check for omitempty/omitzero
		optional := false
		for _, opt := range jsonOpts {
			if opt == "omitempty" || opt == "omitzero" {
				optional = true
				break
			}
		}

		// Check for string encoding
		stringEncoded := false
		for _, opt := range jsonOpts {
			if opt == "string" {
				stringEncoded = true
				break
			}
		}

		// Extract validate tag
		validateTag := b.extractTag(tag, "validate")

		fieldDesc := ir.FieldDescriptor{
			Name:          field.Name(),
			Type:          fieldTypeDesc,
			JSONName:      jsonName,
			Optional:      optional,
			StringEncoded: stringEncoded,
			Skip:          false,
			ValidateTag:   validateTag,
			RawTags:       b.parseAllTags(tag),
			Documentation: ir.Documentation{}, // TODO: extract field documentation
		}

		descriptor.Fields = append(descriptor.Fields, fieldDesc)
	}

	return descriptor, nil
}

// parseJSONTag parses the json struct tag.
func (b *schemaBuilder) parseJSONTag(tag string) (name string, opts []string) {
	jsonTag := b.extractTag(tag, "json")
	if jsonTag == "" {
		return "", nil
	}

	parts := strings.Split(jsonTag, ",")
	name = parts[0]
	if len(parts) > 1 {
		opts = parts[1:]
	}
	return
}

// extractTag extracts a specific tag value from a struct tag.
func (b *schemaBuilder) extractTag(tag, key string) string {
	// Parse struct tag
	st := parseStructTag(tag)
	return st[key]
}

// parseAllTags parses all struct tags into a map.
func (b *schemaBuilder) parseAllTags(tag string) map[string]string {
	return parseStructTag(tag)
}

// convertTypeParamConstraint converts a type parameter constraint to IR.
func (b *schemaBuilder) convertTypeParamConstraint(constraint types.Type) ir.TypeDescriptor {
	if constraint == nil {
		return nil
	}

	// Check the string representation first for special cases
	constraintStr := constraint.String()

	// "any" and "comparable" are special built-in constraints
	// Per spec §3.4, these are NOT preserved in the IR
	if constraintStr == "any" || constraintStr == "comparable" {
		return nil
	}

	// Handle type aliases (e.g., "any" which is an alias to interface{})
	if alias, ok := constraint.(*types.Alias); ok {
		// Follow the alias to its actual type
		underlying := alias.Rhs()
		underlyingStr := underlying.String()
		if underlyingStr == "interface{}" || underlyingStr == "any" {
			return nil
		}
		// Recursively check the underlying type
		return b.convertTypeParamConstraint(underlying)
	}

	// Check for interface constraints
	iface, ok := constraint.(*types.Interface)
	if !ok {
		// Non-interface constraint - try to convert
		desc, err := b.convertType(constraint)
		if err == nil {
			return desc
		}
		return nil
	}

	// Handle interface{} / any (empty interface)
	if iface.Empty() {
		return nil
	}

	// Check if this is the "comparable" interface
	ifaceStr := iface.String()
	if ifaceStr == "comparable" || ifaceStr == "interface{comparable}" {
		return nil
	}

	// For other interface constraints with type sets (unions), we should extract them
	// For now, return nil since union constraint handling is complex
	// TODO: properly extract union constraints from type sets using iface.EmbeddedType()
	return nil
}

// parseStructTag parses a struct tag string into a map.
func parseStructTag(tag string) map[string]string {
	result := make(map[string]string)
	for tag != "" {
		// Skip leading space
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Find key
		i = 0
		for i < len(tag) && tag[i] != ':' && tag[i] != ' ' {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' {
			break
		}
		key := tag[:i]
		tag = tag[i+1:]

		// Find value (quoted string)
		if tag[0] != '"' {
			break
		}
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		value := tag[1:i]
		tag = tag[i+1:]

		result[key] = value
	}
	return result
}
