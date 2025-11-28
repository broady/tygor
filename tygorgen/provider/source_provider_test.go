package provider

import (
	"context"
	"testing"

	"github.com/broady/tygor/tygorgen/ir"
)

func TestSourceProvider_BasicTypes(t *testing.T) {
	provider := &SourceProvider{}
	schema, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages:  []string{"github.com/broady/tygor/tygorgen/provider/testdata"},
		RootTypes: []string{"User"},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	if schema == nil {
		t.Fatal("schema is nil")
	}

	// Find User type
	userType := findType(schema, "User")
	if userType == nil {
		t.Fatal("User type not found")
	}

	userStruct, ok := userType.(*ir.StructDescriptor)
	if !ok {
		t.Fatalf("User is not a StructDescriptor, got %T", userType)
	}

	// Check documentation
	if userStruct.Documentation.Summary == "" {
		t.Error("User should have documentation summary")
	}

	// Check fields
	expectedFields := map[string]struct {
		jsonName      string
		optional      bool
		primitiveKind ir.PrimitiveKind
	}{
		"ID":        {"id", false, ir.PrimitiveString},
		"Name":      {"name", false, ir.PrimitiveString},
		"Email":     {"email", true, ir.PrimitiveString},
		"CreatedAt": {"created_at", false, ir.PrimitiveTime},
	}

	for fieldName, expected := range expectedFields {
		field := findFieldByName(userStruct.Fields, fieldName)
		if field == nil {
			t.Errorf("Field %s not found", fieldName)
			continue
		}

		if field.JSONName != expected.jsonName {
			t.Errorf("Field %s: expected JSON name %q, got %q", fieldName, expected.jsonName, field.JSONName)
		}

		if field.Optional != expected.optional {
			t.Errorf("Field %s: expected Optional=%v, got %v", fieldName, expected.optional, field.Optional)
		}
	}

	// Check Age field (pointer type)
	ageField := findFieldByName(userStruct.Fields, "Age")
	if ageField == nil {
		t.Fatal("Age field not found")
	}
	if ageField.Type.Kind() != ir.KindPtr {
		t.Errorf("Age field should be a pointer type, got %v", ageField.Type.Kind())
	}
}

func TestSourceProvider_EnumTypes(t *testing.T) {
	provider := &SourceProvider{}
	schema, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages:  []string{"github.com/broady/tygor/tygorgen/provider/testdata"},
		RootTypes: []string{"Status", "Priority"},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	// Check Status enum (string-based)
	statusType := findType(schema, "Status")
	if statusType == nil {
		t.Fatal("Status type not found")
	}

	statusEnum, ok := statusType.(*ir.EnumDescriptor)
	if !ok {
		t.Fatalf("Status is not an EnumDescriptor, got %T", statusType)
	}

	if len(statusEnum.Members) != 3 {
		t.Errorf("Status should have 3 members, got %d", len(statusEnum.Members))
	}

	// Check member values
	expectedValues := map[string]string{
		"StatusActive":   "active",
		"StatusInactive": "inactive",
		"StatusPending":  "pending",
	}

	for _, member := range statusEnum.Members {
		expectedValue, exists := expectedValues[member.Name]
		if !exists {
			t.Errorf("Unexpected enum member: %s", member.Name)
			continue
		}

		if strVal, ok := member.Value.(string); !ok || strVal != expectedValue {
			t.Errorf("Member %s: expected value %q, got %v", member.Name, expectedValue, member.Value)
		}
	}

	// Check Priority enum (int-based)
	priorityType := findType(schema, "Priority")
	if priorityType == nil {
		t.Fatal("Priority type not found")
	}

	priorityEnum, ok := priorityType.(*ir.EnumDescriptor)
	if !ok {
		t.Fatalf("Priority is not an EnumDescriptor, got %T", priorityType)
	}

	if len(priorityEnum.Members) != 3 {
		t.Errorf("Priority should have 3 members, got %d", len(priorityEnum.Members))
	}
}

func TestSourceProvider_SliceAndArrayTypes(t *testing.T) {
	provider := &SourceProvider{}
	schema, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages:  []string{"github.com/broady/tygor/tygorgen/provider/testdata"},
		RootTypes: []string{"SliceAndArrayTypes"},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	sliceType := findType(schema, "SliceAndArrayTypes")
	if sliceType == nil {
		t.Fatal("SliceAndArrayTypes not found")
	}

	sliceStruct, ok := sliceType.(*ir.StructDescriptor)
	if !ok {
		t.Fatalf("SliceAndArrayTypes is not a StructDescriptor, got %T", sliceType)
	}

	// Check Slice field
	sliceField := findFieldByName(sliceStruct.Fields, "Slice")
	if sliceField == nil {
		t.Fatal("Slice field not found")
	}
	if sliceField.Type.Kind() != ir.KindArray {
		t.Errorf("Slice field should be KindArray, got %v", sliceField.Type.Kind())
	}
	arrayDesc := sliceField.Type.(*ir.ArrayDescriptor)
	if arrayDesc.Length != 0 {
		t.Errorf("Slice should have Length=0, got %d", arrayDesc.Length)
	}

	// Check Array field (fixed-length)
	arrayField := findFieldByName(sliceStruct.Fields, "Array")
	if arrayField == nil {
		t.Fatal("Array field not found")
	}
	if arrayField.Type.Kind() != ir.KindArray {
		t.Errorf("Array field should be KindArray, got %v", arrayField.Type.Kind())
	}
	fixedArrayDesc := arrayField.Type.(*ir.ArrayDescriptor)
	if fixedArrayDesc.Length != 3 {
		t.Errorf("Array should have Length=3, got %d", fixedArrayDesc.Length)
	}

	// Check ByteSlice field (should be PrimitiveBytes)
	byteSliceField := findFieldByName(sliceStruct.Fields, "ByteSlice")
	if byteSliceField == nil {
		t.Fatal("ByteSlice field not found")
	}
	if byteSliceField.Type.Kind() != ir.KindPrimitive {
		t.Errorf("ByteSlice should be KindPrimitive, got %v", byteSliceField.Type.Kind())
	}
	primDesc := byteSliceField.Type.(*ir.PrimitiveDescriptor)
	if primDesc.PrimitiveKind != ir.PrimitiveBytes {
		t.Errorf("ByteSlice should be PrimitiveBytes, got %v", primDesc.PrimitiveKind)
	}

	// Check ByteArray field (should be array of bytes, NOT PrimitiveBytes)
	byteArrayField := findFieldByName(sliceStruct.Fields, "ByteArray")
	if byteArrayField == nil {
		t.Fatal("ByteArray field not found")
	}
	if byteArrayField.Type.Kind() != ir.KindArray {
		t.Errorf("ByteArray should be KindArray, got %v", byteArrayField.Type.Kind())
	}
}

func TestSourceProvider_MapTypes(t *testing.T) {
	provider := &SourceProvider{}
	schema, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages:  []string{"github.com/broady/tygor/tygorgen/provider/testdata"},
		RootTypes: []string{"MapTypes"},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	mapType := findType(schema, "MapTypes")
	if mapType == nil {
		t.Fatal("MapTypes not found")
	}

	mapStruct, ok := mapType.(*ir.StructDescriptor)
	if !ok {
		t.Fatalf("MapTypes is not a StructDescriptor, got %T", mapType)
	}

	// Check StringMap field
	stringMapField := findFieldByName(mapStruct.Fields, "StringMap")
	if stringMapField == nil {
		t.Fatal("StringMap field not found")
	}
	if stringMapField.Type.Kind() != ir.KindMap {
		t.Errorf("StringMap should be KindMap, got %v", stringMapField.Type.Kind())
	}

	// Check IntMap field (int keys should be valid)
	intMapField := findFieldByName(mapStruct.Fields, "IntMap")
	if intMapField == nil {
		t.Fatal("IntMap field not found")
	}
	if intMapField.Type.Kind() != ir.KindMap {
		t.Errorf("IntMap should be KindMap, got %v", intMapField.Type.Kind())
	}
}

func TestSourceProvider_PointerTypes(t *testing.T) {
	provider := &SourceProvider{}
	schema, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages:  []string{"github.com/broady/tygor/tygorgen/provider/testdata"},
		RootTypes: []string{"PointerTypes"},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	ptrType := findType(schema, "PointerTypes")
	if ptrType == nil {
		t.Fatal("PointerTypes not found")
	}

	ptrStruct, ok := ptrType.(*ir.StructDescriptor)
	if !ok {
		t.Fatalf("PointerTypes is not a StructDescriptor, got %T", ptrType)
	}

	// Check RequiredPtr (pointer without omitempty)
	requiredPtrField := findFieldByName(ptrStruct.Fields, "RequiredPtr")
	if requiredPtrField == nil {
		t.Fatal("RequiredPtr field not found")
	}
	if requiredPtrField.Type.Kind() != ir.KindPtr {
		t.Errorf("RequiredPtr should be KindPtr, got %v", requiredPtrField.Type.Kind())
	}
	if requiredPtrField.Optional {
		t.Error("RequiredPtr should not be optional")
	}

	// Check OptionalPtr (pointer with omitempty)
	optionalPtrField := findFieldByName(ptrStruct.Fields, "OptionalPtr")
	if optionalPtrField == nil {
		t.Fatal("OptionalPtr field not found")
	}
	if !optionalPtrField.Optional {
		t.Error("OptionalPtr should be optional")
	}

	// Check DoublePtr (pointer to pointer)
	doublePtrField := findFieldByName(ptrStruct.Fields, "DoublePtr")
	if doublePtrField == nil {
		t.Fatal("DoublePtr field not found")
	}
	if doublePtrField.Type.Kind() != ir.KindPtr {
		t.Errorf("DoublePtr should be KindPtr, got %v", doublePtrField.Type.Kind())
	}
	// The element should also be a pointer
	ptrDesc := doublePtrField.Type.(*ir.PtrDescriptor)
	if ptrDesc.Element.Kind() != ir.KindPtr {
		t.Errorf("DoublePtr element should be KindPtr, got %v", ptrDesc.Element.Kind())
	}
}

func TestSourceProvider_EmbeddedTypes(t *testing.T) {
	provider := &SourceProvider{}
	schema, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages:  []string{"github.com/broady/tygor/tygorgen/provider/testdata"},
		RootTypes: []string{"EmbeddedTypes", "NamedEmbedding"},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	// Check EmbeddedTypes (inheritance)
	embeddedType := findType(schema, "EmbeddedTypes")
	if embeddedType == nil {
		t.Fatal("EmbeddedTypes not found")
	}

	embeddedStruct, ok := embeddedType.(*ir.StructDescriptor)
	if !ok {
		t.Fatalf("EmbeddedTypes is not a StructDescriptor, got %T", embeddedType)
	}

	// Should have BaseType in Extends
	if len(embeddedStruct.Extends) != 1 {
		t.Errorf("EmbeddedTypes should extend 1 type, got %d", len(embeddedStruct.Extends))
	} else {
		if embeddedStruct.Extends[0].Name != "BaseType" {
			t.Errorf("EmbeddedTypes should extend BaseType, got %s", embeddedStruct.Extends[0].Name)
		}
	}

	// Should have OwnField
	ownField := findFieldByName(embeddedStruct.Fields, "OwnField")
	if ownField == nil {
		t.Error("OwnField not found in EmbeddedTypes")
	}

	// Check NamedEmbedding (nested, not inheritance)
	namedType := findType(schema, "NamedEmbedding")
	if namedType == nil {
		t.Fatal("NamedEmbedding not found")
	}

	namedStruct, ok := namedType.(*ir.StructDescriptor)
	if !ok {
		t.Fatalf("NamedEmbedding is not a StructDescriptor, got %T", namedType)
	}

	// Should NOT have anything in Extends
	if len(namedStruct.Extends) != 0 {
		t.Errorf("NamedEmbedding should not extend any types, got %d", len(namedStruct.Extends))
	}

	// Should have Base as a regular field
	baseField := findFieldByName(namedStruct.Fields, "Base")
	if baseField == nil {
		t.Error("Base field not found in NamedEmbedding")
	} else {
		if baseField.JSONName != "base" {
			t.Errorf("Base field should have JSON name 'base', got %q", baseField.JSONName)
		}
	}
}

func TestSourceProvider_TaggedFields(t *testing.T) {
	provider := &SourceProvider{}
	schema, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages:  []string{"github.com/broady/tygor/tygorgen/provider/testdata"},
		RootTypes: []string{"TaggedFields"},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	taggedType := findType(schema, "TaggedFields")
	if taggedType == nil {
		t.Fatal("TaggedFields not found")
	}

	taggedStruct, ok := taggedType.(*ir.StructDescriptor)
	if !ok {
		t.Fatalf("TaggedFields is not a StructDescriptor, got %T", taggedType)
	}

	// Skipped field should not appear
	skippedField := findFieldByName(taggedStruct.Fields, "Skipped")
	if skippedField != nil {
		t.Error("Skipped field should not be in Fields")
	}

	// StringEncoded field
	stringEncodedField := findFieldByName(taggedStruct.Fields, "StringEncoded")
	if stringEncodedField == nil {
		t.Fatal("StringEncoded field not found")
	}
	if !stringEncodedField.StringEncoded {
		t.Error("StringEncoded field should have StringEncoded=true")
	}

	// Validated field
	validatedField := findFieldByName(taggedStruct.Fields, "Validated")
	if validatedField == nil {
		t.Fatal("Validated field not found")
	}
	if validatedField.ValidateTag != "required,email" {
		t.Errorf("Validated field should have ValidateTag='required,email', got %q", validatedField.ValidateTag)
	}

	// OmitZero field
	omitZeroField := findFieldByName(taggedStruct.Fields, "OmitZero")
	if omitZeroField == nil {
		t.Fatal("OmitZero field not found")
	}
	if !omitZeroField.Optional {
		t.Error("OmitZero field should be Optional")
	}

	// CustomTags field
	customTagsField := findFieldByName(taggedStruct.Fields, "CustomTags")
	if customTagsField == nil {
		t.Fatal("CustomTags field not found")
	}
	if customTagsField.RawTags["db"] != "custom_db" {
		t.Errorf("CustomTags field should have db tag 'custom_db', got %q", customTagsField.RawTags["db"])
	}
	if customTagsField.RawTags["xml"] != "CustomXML" {
		t.Errorf("CustomTags field should have xml tag 'CustomXML', got %q", customTagsField.RawTags["xml"])
	}
}

func TestSourceProvider_DeprecatedType(t *testing.T) {
	provider := &SourceProvider{}
	schema, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages:  []string{"github.com/broady/tygor/tygorgen/provider/testdata"},
		RootTypes: []string{"OldStruct"},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	oldType := findType(schema, "OldStruct")
	if oldType == nil {
		t.Fatal("OldStruct not found")
	}

	oldStruct, ok := oldType.(*ir.StructDescriptor)
	if !ok {
		t.Fatalf("OldStruct is not a StructDescriptor, got %T", oldType)
	}

	if oldStruct.Documentation.Deprecated == nil {
		t.Error("OldStruct should be marked as deprecated")
	} else {
		if *oldStruct.Documentation.Deprecated == "" {
			t.Error("Deprecated message should not be empty")
		}
	}
}

func TestSourceProvider_AllExportedTypes(t *testing.T) {
	provider := &SourceProvider{}
	schema, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages: []string{"github.com/broady/tygor/tygorgen/provider/testdata"},
		// RootTypes is empty - should extract all exported types
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	// Should have many types
	if len(schema.Types) < 10 {
		t.Errorf("Expected at least 10 types, got %d", len(schema.Types))
	}

	// Check that various types exist
	expectedTypes := []string{"User", "Status", "Priority", "SimpleStruct", "MapTypes"}
	for _, typeName := range expectedTypes {
		if findType(schema, typeName) == nil {
			t.Errorf("Expected type %s not found", typeName)
		}
	}
}

func TestSourceProvider_NoPackages(t *testing.T) {
	provider := &SourceProvider{}
	_, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages: []string{},
	})

	if err == nil {
		t.Error("Expected error when no packages specified")
	}
}

func TestSourceProvider_NonexistentPackage(t *testing.T) {
	provider := &SourceProvider{}
	_, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages: []string{"github.com/nonexistent/package"},
	})

	if err == nil {
		t.Error("Expected error for nonexistent package")
	}
}

func TestSourceProvider_InterfaceTypes(t *testing.T) {
	provider := &SourceProvider{}
	schema, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages:  []string{"github.com/broady/tygor/tygorgen/provider/testdata"},
		RootTypes: []string{"InterfaceField"},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	interfaceType := findType(schema, "InterfaceField")
	if interfaceType == nil {
		t.Fatal("InterfaceField not found")
	}

	interfaceStruct, ok := interfaceType.(*ir.StructDescriptor)
	if !ok {
		t.Fatalf("InterfaceField is not a StructDescriptor, got %T", interfaceType)
	}

	// Any field should be PrimitiveAny
	anyField := findFieldByName(interfaceStruct.Fields, "Any")
	if anyField == nil {
		t.Fatal("Any field not found")
	}
	if anyField.Type.Kind() != ir.KindPrimitive {
		t.Errorf("Any field should be KindPrimitive, got %v", anyField.Type.Kind())
	}
	primDesc := anyField.Type.(*ir.PrimitiveDescriptor)
	if primDesc.PrimitiveKind != ir.PrimitiveAny {
		t.Errorf("Any field should be PrimitiveAny, got %v", primDesc.PrimitiveKind)
	}
}

// Helper functions

func findType(schema *ir.Schema, name string) ir.TypeDescriptor {
	for _, t := range schema.Types {
		if t.TypeName().Name == name {
			return t
		}
	}
	return nil
}

func findFieldByName(fields []ir.FieldDescriptor, name string) *ir.FieldDescriptor {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

func TestSourceProvider_FieldDocumentation(t *testing.T) {
	provider := &SourceProvider{}
	schema, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages:  []string{"github.com/broady/tygor/tygorgen/provider/testdata"},
		RootTypes: []string{"User"},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	userType := findType(schema, "User")
	if userType == nil {
		t.Fatal("User type not found")
	}

	structDesc, ok := userType.(*ir.StructDescriptor)
	if !ok {
		t.Fatal("User should be a struct")
	}

	// Check field documentation
	tests := []struct {
		fieldName   string
		wantSummary string
	}{
		{"ID", "ID is the unique identifier"},
		{"Name", "Name is the user's display name"},
		{"Email", "Email is optional"},
		{"Age", "Age may be nil"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			field := findFieldByName(structDesc.Fields, tt.fieldName)
			if field == nil {
				t.Fatalf("Field %s not found", tt.fieldName)
			}
			if field.Documentation.Summary != tt.wantSummary {
				t.Errorf("Field %s doc summary = %q, want %q", tt.fieldName, field.Documentation.Summary, tt.wantSummary)
			}
		})
	}
}

func TestSourceProvider_EnumMemberDocumentation(t *testing.T) {
	provider := &SourceProvider{}
	schema, err := provider.BuildSchema(context.Background(), SourceInputOptions{
		Packages:  []string{"github.com/broady/tygor/tygorgen/provider/testdata"},
		RootTypes: []string{"Status"},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	statusType := findType(schema, "Status")
	if statusType == nil {
		t.Fatal("Status type not found")
	}

	enumDesc, ok := statusType.(*ir.EnumDescriptor)
	if !ok {
		t.Fatal("Status should be an enum")
	}

	// Check enum member documentation
	tests := []struct {
		memberName  string
		wantSummary string
	}{
		{"StatusActive", "StatusActive means the user is active"},
		{"StatusInactive", "StatusInactive means the user is inactive"},
		{"StatusPending", "StatusPending means awaiting approval"},
	}

	for _, tt := range tests {
		t.Run(tt.memberName, func(t *testing.T) {
			var found *ir.EnumMember
			for i := range enumDesc.Members {
				if enumDesc.Members[i].Name == tt.memberName {
					found = &enumDesc.Members[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("Enum member %s not found", tt.memberName)
			}
			if found.Documentation.Summary != tt.wantSummary {
				t.Errorf("Enum member %s doc summary = %q, want %q", tt.memberName, found.Documentation.Summary, tt.wantSummary)
			}
		})
	}
}
