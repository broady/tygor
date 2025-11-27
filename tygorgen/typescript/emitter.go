package typescript

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/broady/tygor/tygorgen/ir"
)

// Emitter handles TypeScript code emission for IR type descriptors.
type Emitter struct {
	schema   *ir.Schema
	config   GeneratorConfig
	tsConfig TypeScriptConfig
	indent   string
}

// EmitType emits a top-level type declaration.
func (e *Emitter) EmitType(buf *bytes.Buffer, typ ir.TypeDescriptor) ([]ir.Warning, error) {
	// Emit documentation comments if enabled
	if e.config.EmitComments && !typ.Doc().IsZero() {
		e.emitJSDoc(buf, typ.Doc())
	}

	switch t := typ.(type) {
	case *ir.StructDescriptor:
		return e.emitStruct(buf, t)
	case *ir.AliasDescriptor:
		return e.emitAlias(buf, t)
	case *ir.EnumDescriptor:
		return e.emitEnum(buf, t)
	default:
		return nil, fmt.Errorf("unsupported top-level type kind: %s", typ.Kind())
	}
}

// emitStruct emits a struct as an interface or type.
func (e *Emitter) emitStruct(buf *bytes.Buffer, s *ir.StructDescriptor) ([]ir.Warning, error) {
	var warnings []ir.Warning

	// Apply name transforms
	typeName := applyNameTransforms(s.Name.Name, e.config)
	typeName = escapeReservedWord(typeName)

	// Emit export keyword
	if e.tsConfig.EmitExport {
		buf.WriteString("export ")
	}
	if e.tsConfig.EmitDeclare {
		buf.WriteString("declare ")
	}

	// Emit type parameters
	typeParams := ""
	if len(s.TypeParameters) > 0 {
		typeParams = e.emitTypeParameters(s.TypeParameters)
	}

	// Decide whether to use interface or type
	useInterface := e.tsConfig.UseInterface && len(s.Extends) == 0

	if useInterface {
		buf.WriteString("interface ")
		buf.WriteString(typeName)
		buf.WriteString(typeParams)
		buf.WriteString(" ")

		// Handle extends
		if len(s.Extends) > 0 {
			buf.WriteString("extends ")
			for i, ext := range s.Extends {
				if i > 0 {
					buf.WriteString(", ")
				}
				extName := applyNameTransforms(ext.Name, e.config)
				extName = escapeReservedWord(extName)
				buf.WriteString(extName)
			}
			buf.WriteString(" ")
		}

		buf.WriteString("{\n")
	} else {
		buf.WriteString("type ")
		buf.WriteString(typeName)
		buf.WriteString(typeParams)
		buf.WriteString(" = ")

		// Handle extends with intersection types
		if len(s.Extends) > 0 {
			for _, ext := range s.Extends {
				extName := applyNameTransforms(ext.Name, e.config)
				extName = escapeReservedWord(extName)
				buf.WriteString(extName)
				buf.WriteString(" & ")
			}
		}

		buf.WriteString("{\n")
	}

	// Emit fields
	for _, field := range s.Fields {
		if field.Skip {
			continue
		}

		// Field documentation
		if e.config.EmitComments && !field.Documentation.IsZero() {
			buf.WriteString(e.indent)
			buf.WriteString("  ")
			e.emitJSDoc(buf, field.Documentation)
		}

		buf.WriteString(e.indent)
		buf.WriteString("  ")

		// Field name (escape if reserved word or needs quoting)
		fieldName := e.getPropertyName(field)
		if needsQuoting(fieldName) {
			buf.WriteString(fmt.Sprintf("%q", fieldName))
		} else {
			fieldName = escapeReservedWord(fieldName)
			buf.WriteString(fieldName)
		}

		// Determine optional vs nullable based on §4.9
		optional, nullable, err := e.determineOptionalNullable(field)
		if err != nil {
			return nil, err
		}

		// Optional marker
		if optional {
			buf.WriteString("?")
		}

		buf.WriteString(": ")

		// Emit type
		typeExpr, err := e.EmitTypeExpr(field.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to emit field %s type: %w", field.Name, err)
		}
		buf.WriteString(typeExpr)

		// Nullable marker
		if nullable {
			buf.WriteString(" | null")
		}

		buf.WriteString(";\n")
	}

	buf.WriteString(e.indent)
	if useInterface {
		buf.WriteString("}")
	} else {
		buf.WriteString("};")
	}

	return warnings, nil
}

// emitAlias emits a type alias.
func (e *Emitter) emitAlias(buf *bytes.Buffer, a *ir.AliasDescriptor) ([]ir.Warning, error) {
	// Apply name transforms
	typeName := applyNameTransforms(a.Name.Name, e.config)
	typeName = escapeReservedWord(typeName)

	// Emit export keyword
	if e.tsConfig.EmitExport {
		buf.WriteString("export ")
	}
	if e.tsConfig.EmitDeclare {
		buf.WriteString("declare ")
	}

	buf.WriteString("type ")
	buf.WriteString(typeName)

	// Emit type parameters
	if len(a.TypeParameters) > 0 {
		buf.WriteString(e.emitTypeParameters(a.TypeParameters))
	}

	buf.WriteString(" = ")

	// Emit underlying type
	underlying, err := e.EmitTypeExpr(a.Underlying)
	if err != nil {
		return nil, fmt.Errorf("failed to emit alias underlying type: %w", err)
	}
	buf.WriteString(underlying)
	buf.WriteString(";")

	return nil, nil
}

// emitEnum emits an enum based on the configured style.
func (e *Emitter) emitEnum(buf *bytes.Buffer, enum *ir.EnumDescriptor) ([]ir.Warning, error) {
	// Apply name transforms
	typeName := applyNameTransforms(enum.Name.Name, e.config)
	typeName = escapeReservedWord(typeName)

	switch e.tsConfig.EnumStyle {
	case "union":
		return e.emitEnumAsUnion(buf, typeName, enum)
	case "enum":
		return e.emitEnumAsEnum(buf, typeName, enum, false)
	case "const_enum":
		return e.emitEnumAsEnum(buf, typeName, enum, true)
	case "object":
		return e.emitEnumAsObject(buf, typeName, enum)
	default:
		// Default to union
		return e.emitEnumAsUnion(buf, typeName, enum)
	}
}

// emitEnumAsUnion emits an enum as a union type.
func (e *Emitter) emitEnumAsUnion(buf *bytes.Buffer, typeName string, enum *ir.EnumDescriptor) ([]ir.Warning, error) {
	if e.tsConfig.EmitExport {
		buf.WriteString("export ")
	}
	if e.tsConfig.EmitDeclare {
		buf.WriteString("declare ")
	}

	buf.WriteString("type ")
	buf.WriteString(typeName)
	buf.WriteString(" = ")

	for i, member := range enum.Members {
		if i > 0 {
			buf.WriteString(" | ")
		}
		buf.WriteString(formatEnumValue(member.Value))
	}

	buf.WriteString(";")
	return nil, nil
}

// emitEnumAsEnum emits an enum as a TypeScript enum.
func (e *Emitter) emitEnumAsEnum(buf *bytes.Buffer, typeName string, enum *ir.EnumDescriptor, isConst bool) ([]ir.Warning, error) {
	if e.tsConfig.EmitExport {
		buf.WriteString("export ")
	}
	if e.tsConfig.EmitDeclare {
		buf.WriteString("declare ")
	}

	if isConst {
		buf.WriteString("const ")
	}

	buf.WriteString("enum ")
	buf.WriteString(typeName)
	buf.WriteString(" {\n")

	for i, member := range enum.Members {
		if i > 0 {
			buf.WriteString(",\n")
		}

		// Member documentation
		if e.config.EmitComments && !member.Documentation.IsZero() {
			buf.WriteString("  ")
			e.emitJSDoc(buf, member.Documentation)
			buf.WriteString("  ")
		} else {
			buf.WriteString("  ")
		}

		memberName := escapeReservedWord(member.Name)
		buf.WriteString(memberName)
		buf.WriteString(" = ")
		buf.WriteString(formatEnumValue(member.Value))
	}

	buf.WriteString(",\n")
	buf.WriteString("}")
	return nil, nil
}

// emitEnumAsObject emits an enum as a const object.
func (e *Emitter) emitEnumAsObject(buf *bytes.Buffer, typeName string, enum *ir.EnumDescriptor) ([]ir.Warning, error) {
	if e.tsConfig.EmitExport {
		buf.WriteString("export ")
	}
	if e.tsConfig.EmitDeclare {
		buf.WriteString("declare ")
	}

	buf.WriteString("const ")
	buf.WriteString(typeName)
	buf.WriteString(" = {\n")

	for i, member := range enum.Members {
		if i > 0 {
			buf.WriteString(",\n")
		}

		buf.WriteString("  ")
		memberName := escapeReservedWord(member.Name)
		buf.WriteString(memberName)
		buf.WriteString(": ")
		buf.WriteString(formatEnumValue(member.Value))
	}

	buf.WriteString(",\n")
	buf.WriteString("} as const;")
	return nil, nil
}

// EmitTypeExpr emits a type expression (non-top-level types).
func (e *Emitter) EmitTypeExpr(typ ir.TypeDescriptor) (string, error) {
	switch t := typ.(type) {
	case *ir.PrimitiveDescriptor:
		return e.emitPrimitive(t), nil
	case *ir.ArrayDescriptor:
		return e.emitArray(t)
	case *ir.MapDescriptor:
		return e.emitMap(t)
	case *ir.ReferenceDescriptor:
		return e.emitReference(t), nil
	case *ir.PtrDescriptor:
		return e.emitPtr(t)
	case *ir.UnionDescriptor:
		return e.emitUnion(t)
	case *ir.TypeParameterDescriptor:
		return e.emitTypeParameter(t), nil
	default:
		return "", fmt.Errorf("unsupported type expression kind: %s", typ.Kind())
	}
}

// emitPrimitive emits a primitive type.
func (e *Emitter) emitPrimitive(p *ir.PrimitiveDescriptor) string {
	switch p.PrimitiveKind {
	case ir.PrimitiveBool:
		return "boolean"
	case ir.PrimitiveInt, ir.PrimitiveUint, ir.PrimitiveFloat:
		return "number"
	case ir.PrimitiveString:
		return "string"
	case ir.PrimitiveBytes:
		return "string" // base64
	case ir.PrimitiveTime:
		return "string" // RFC 3339
	case ir.PrimitiveDuration:
		return "number" // nanoseconds
	case ir.PrimitiveAny:
		return e.tsConfig.UnknownType
	case ir.PrimitiveEmpty:
		return "Record<string, never>"
	default:
		return e.tsConfig.UnknownType
	}
}

// emitArray emits an array type.
func (e *Emitter) emitArray(a *ir.ArrayDescriptor) (string, error) {
	elemType, err := e.EmitTypeExpr(a.Element)
	if err != nil {
		return "", err
	}

	if a.Length > 0 {
		// Fixed-length array: emit as tuple if small, otherwise regular array
		if a.Length <= 10 {
			// Emit as tuple: [T, T, T]
			parts := make([]string, a.Length)
			for i := 0; i < a.Length; i++ {
				parts[i] = elemType
			}
			return "[" + strings.Join(parts, ", ") + "]", nil
		}
		// For large fixed arrays, just use T[]
	}

	// Slice or large array: T[] or readonly T[]
	if e.tsConfig.UseReadonlyArrays {
		return "readonly " + elemType + "[]", nil
	}
	return elemType + "[]", nil
}

// emitMap emits a map type.
func (e *Emitter) emitMap(m *ir.MapDescriptor) (string, error) {
	keyType, err := e.EmitTypeExpr(m.Key)
	if err != nil {
		return "", err
	}
	valueType, err := e.EmitTypeExpr(m.Value)
	if err != nil {
		return "", err
	}

	// All maps serialize to string keys in JSON
	// But preserve the key type for named string types
	if _, ok := m.Key.(*ir.ReferenceDescriptor); ok {
		// Named type - preserve it
		return fmt.Sprintf("Record<%s, %s>", keyType, valueType), nil
	}
	// Primitive key - always use string
	return fmt.Sprintf("Record<string, %s>", valueType), nil
}

// emitReference emits a reference to a named type.
func (e *Emitter) emitReference(r *ir.ReferenceDescriptor) string {
	typeName := applyNameTransforms(r.Target.Name, e.config)
	return escapeReservedWord(typeName)
}

// emitPtr emits a pointer type.
// Note: PtrDescriptor nullability is context-dependent (§4.9).
// This method only handles nested pointers; field-level nullability
// is handled by determineOptionalNullable.
func (e *Emitter) emitPtr(p *ir.PtrDescriptor) (string, error) {
	elemType, err := e.EmitTypeExpr(p.Element)
	if err != nil {
		return "", err
	}
	// In nested contexts, pointer always adds null
	return elemType, nil
}

// emitUnion emits a union type.
func (e *Emitter) emitUnion(u *ir.UnionDescriptor) (string, error) {
	var parts []string
	for _, t := range u.Types {
		part, err := e.EmitTypeExpr(t)
		if err != nil {
			return "", err
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " | "), nil
}

// emitTypeParameter emits a type parameter reference.
func (e *Emitter) emitTypeParameter(tp *ir.TypeParameterDescriptor) string {
	return tp.ParamName
}

// emitTypeParameters emits type parameter declarations.
func (e *Emitter) emitTypeParameters(params []ir.TypeParameterDescriptor) string {
	if len(params) == 0 {
		return ""
	}

	var parts []string
	for _, param := range params {
		part := param.ParamName
		if param.Constraint != nil {
			constraintType, err := e.EmitTypeExpr(param.Constraint)
			if err == nil {
				part += " extends " + constraintType
			}
		}
		parts = append(parts, part)
	}

	return "<" + strings.Join(parts, ", ") + ">"
}

// determineOptionalNullable determines if a field should be optional and/or nullable.
// Implements the decision tree from §4.9.
func (e *Emitter) determineOptionalNullable(field ir.FieldDescriptor) (optional, nullable bool, err error) {
	// Decision tree from §4.9:
	// 1. If Optional is true → field is optional (field?: T)
	// 2. Else if field type is pointer (*T) → field can be null (field: T | null)
	// 3. Else if field type is slice or map → field can be null (field: T | null)
	// 4. Else → field is required (field: T)

	if field.Optional {
		optional = true
		// Check for the rare case of pointer to collection
		if ptr, ok := field.Type.(*ir.PtrDescriptor); ok {
			if _, isArray := ptr.Element.(*ir.ArrayDescriptor); isArray {
				nullable = true
			} else if _, isMap := ptr.Element.(*ir.MapDescriptor); isMap {
				nullable = true
			}
		}
		return
	}

	// Check if pointer
	if _, ok := field.Type.(*ir.PtrDescriptor); ok {
		nullable = true
		return
	}

	// Check if slice or map
	if _, ok := field.Type.(*ir.ArrayDescriptor); ok {
		nullable = true
		return
	}
	if _, ok := field.Type.(*ir.MapDescriptor); ok {
		nullable = true
		return
	}

	// Required field
	return false, false, nil
}

// getPropertyName returns the property name for a field.
func (e *Emitter) getPropertyName(field ir.FieldDescriptor) string {
	switch e.config.PropertyNameSource {
	case "field":
		return applyCaseTransform(field.Name, e.config.FieldCase)
	default: // "tag:json" or empty (default to json)
		name := field.JSONName
		if name == "" {
			name = field.Name
		}
		return applyCaseTransform(name, e.config.FieldCase)
	}
}

// emitJSDoc emits JSDoc-style documentation comments.
func (e *Emitter) emitJSDoc(buf *bytes.Buffer, doc ir.Documentation) {
	if doc.IsZero() {
		return
	}

	lines := strings.Split(doc.Body, "\n")
	if len(lines) == 1 && doc.Deprecated == nil {
		// Single line
		buf.WriteString("/** ")
		buf.WriteString(strings.TrimSpace(lines[0]))
		buf.WriteString(" */\n")
		return
	}

	// Multi-line
	buf.WriteString("/**\n")
	for _, line := range lines {
		buf.WriteString(" * ")
		buf.WriteString(strings.TrimSpace(line))
		buf.WriteString("\n")
	}

	if doc.Deprecated != nil {
		buf.WriteString(" * @deprecated")
		if *doc.Deprecated != "" {
			buf.WriteString(" ")
			buf.WriteString(*doc.Deprecated)
		}
		buf.WriteString("\n")
	}

	buf.WriteString(" */\n")
}

// formatEnumValue formats an enum member value for output.
func formatEnumValue(value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
