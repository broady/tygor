package provider

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/broady/tygor/tygorgen/ir"
)

// Test types for comprehensive coverage

type SimpleStruct struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email,omitempty"`
}

type AllPrimitives struct {
	Bool    bool    `json:"bool"`
	Int     int     `json:"int"`
	Int8    int8    `json:"int8"`
	Int16   int16   `json:"int16"`
	Int32   int32   `json:"int32"`
	Int64   int64   `json:"int64"`
	Uint    uint    `json:"uint"`
	Uint8   uint8   `json:"uint8"`
	Uint16  uint16  `json:"uint16"`
	Uint32  uint32  `json:"uint32"`
	Uint64  uint64  `json:"uint64"`
	Uintptr uintptr `json:"uintptr"`
	Float32 float32 `json:"float32"`
	Float64 float64 `json:"float64"`
	String  string  `json:"string"`
	// unexported string  // Should be skipped (commented out to pass linter)
}

type PointerFields struct {
	PtrString *string `json:"ptr_string"`
	PtrInt    *int    `json:"ptr_int,omitempty"`
}

type SliceAndArray struct {
	Slice     []string `json:"slice"`
	SliceOmit []int    `json:"slice_omit,omitempty"`
	Array     [3]int   `json:"array"`
	ByteSlice []byte   `json:"bytes"`
	ByteArray [16]byte `json:"byte_array"`
}

type MapTypes struct {
	StringMap map[string]int    `json:"string_map"`
	IntMap    map[int]string    `json:"int_map"`
	MapOmit   map[string]string `json:"map_omit,omitempty"`
}

type SpecialTypes struct {
	Time         time.Time       `json:"time"`
	Duration     time.Duration   `json:"duration"`
	JSONNumber   json.Number     `json:"json_number"`
	RawMessage   json.RawMessage `json:"raw_message"`
	AnyInterface interface{}     `json:"any"`
	EmptyStruct  struct{}        `json:"empty"`
}

type EmbeddedNoTag struct {
	EmbeddedField string `json:"embedded_field"`
}

type EmbeddedWithTag struct {
	TaggedField string `json:"tagged_field"`
}

type EmbeddingStruct struct {
	EmbeddedNoTag                 // Flattened (Extends)
	Nested        EmbeddedWithTag `json:"nested"` // Nested (Fields)
	OwnField      string          `json:"own_field"`
}

type RecursiveStruct struct {
	Name  string           `json:"name"`
	Child *RecursiveStruct `json:"child,omitempty"`
}

type MutuallyRecursive1 struct {
	Name  string              `json:"name"`
	Other *MutuallyRecursive2 `json:"other,omitempty"`
}

type MutuallyRecursive2 struct {
	Value string              `json:"value"`
	Back  *MutuallyRecursive1 `json:"back,omitempty"`
}

type WithAnonymousStruct struct {
	Inner struct {
		X int    `json:"x"`
		Y string `json:"y"`
	} `json:"inner"`
	Name string `json:"name"`
}

type NestedAnonymous struct {
	Level1 struct {
		Level2 struct {
			Value string `json:"value"`
		} `json:"level2"`
	} `json:"level1"`
}

type StringEncoded struct {
	NumberAsString int  `json:"number,string"`
	BoolAsString   bool `json:"bool,string"`
}

type ValidateTags struct {
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"gte=0,lte=120"`
	Password string `json:"password" validate:"required,min=8"`
}

type CustomTags struct {
	Field1 string `json:"field1" db:"field_1" xml:"Field1"`
	Field2 int    `json:"field2" schema:"field2" validate:"required"`
}

type SkipFields struct {
	Included string `json:"included"`
	Skipped  string `json:"-"`
	AlsoSkip string `json:"-,"`
}

type OmitZero struct {
	Field1 string `json:"field1,omitzero"`
	Field2 int    `json:"field2,omitempty"`
}

type GenericResponse[T any] struct {
	Data  T      `json:"data"`
	Error string `json:"error,omitempty"`
}

type ConcreteGeneric struct {
	Response GenericResponse[string] `json:"response"`
}

type StringAlias string

type IntAlias int

type AliasStruct struct {
	StringAlias StringAlias `json:"string_alias"`
	IntAlias    IntAlias    `json:"int_alias"`
}

// Test functions

func TestReflectionProvider_SimpleStruct(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(SimpleStruct{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	if len(schema.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(schema.Types))
	}

	structDesc, ok := schema.Types[0].(*ir.StructDescriptor)
	if !ok {
		t.Fatalf("expected StructDescriptor, got %T", schema.Types[0])
	}

	if structDesc.Name.Name != "SimpleStruct" {
		t.Errorf("expected name 'SimpleStruct', got %q", structDesc.Name.Name)
	}

	if len(structDesc.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(structDesc.Fields))
	}

	// Check fields
	nameField := structDesc.Fields[0]
	if nameField.Name != "Name" || nameField.JSONName != "name" {
		t.Errorf("unexpected Name field: %+v", nameField)
	}

	emailField := structDesc.Fields[2]
	if !emailField.Optional {
		t.Error("Email field should be optional")
	}
}

func TestReflectionProvider_AllPrimitives(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(AllPrimitives{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	structDesc := schema.Types[0].(*ir.StructDescriptor)

	// Should have 15 fields (all exported primitives)
	if len(structDesc.Fields) != 15 {
		t.Errorf("expected 15 fields, got %d", len(structDesc.Fields))
	}

	// Verify some specific types
	boolField := findField(structDesc.Fields, "Bool")
	if boolField == nil {
		t.Fatal("Bool field not found")
	}
	if boolField.Type.Kind() != ir.KindPrimitive {
		t.Errorf("Bool field should be primitive")
	}
	primDesc := boolField.Type.(*ir.PrimitiveDescriptor)
	if primDesc.PrimitiveKind != ir.PrimitiveBool {
		t.Errorf("Bool field should be PrimitiveBool")
	}

	int32Field := findField(structDesc.Fields, "Int32")
	if int32Field == nil {
		t.Fatal("Int32 field not found")
	}
	primDesc = int32Field.Type.(*ir.PrimitiveDescriptor)
	if primDesc.PrimitiveKind != ir.PrimitiveInt || primDesc.BitSize != 32 {
		t.Errorf("Int32 field should be PrimitiveInt with BitSize 32, got kind=%v bitsize=%d", primDesc.PrimitiveKind, primDesc.BitSize)
	}
}

func TestReflectionProvider_PointerFields(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(PointerFields{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	structDesc := schema.Types[0].(*ir.StructDescriptor)

	ptrStringField := findField(structDesc.Fields, "PtrString")
	if ptrStringField == nil {
		t.Fatal("PtrString field not found")
	}

	if ptrStringField.Type.Kind() != ir.KindPtr {
		t.Errorf("PtrString should be KindPtr, got %v", ptrStringField.Type.Kind())
	}

	ptrIntField := findField(structDesc.Fields, "PtrInt")
	if !ptrIntField.Optional {
		t.Error("PtrInt should be optional")
	}
}

func TestReflectionProvider_SliceAndArray(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(SliceAndArray{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	structDesc := schema.Types[0].(*ir.StructDescriptor)

	sliceField := findField(structDesc.Fields, "Slice")
	if sliceField.Type.Kind() != ir.KindArray {
		t.Errorf("Slice should be KindArray, got %v", sliceField.Type.Kind())
	}
	arrayDesc := sliceField.Type.(*ir.ArrayDescriptor)
	if arrayDesc.Length != 0 {
		t.Errorf("Slice should have Length 0, got %d", arrayDesc.Length)
	}

	arrayField := findField(structDesc.Fields, "Array")
	arrayDesc = arrayField.Type.(*ir.ArrayDescriptor)
	if arrayDesc.Length != 3 {
		t.Errorf("Array should have Length 3, got %d", arrayDesc.Length)
	}

	byteSliceField := findField(structDesc.Fields, "ByteSlice")
	if byteSliceField.Type.Kind() != ir.KindPrimitive {
		t.Errorf("ByteSlice should be KindPrimitive, got %v", byteSliceField.Type.Kind())
	}
	primDesc := byteSliceField.Type.(*ir.PrimitiveDescriptor)
	if primDesc.PrimitiveKind != ir.PrimitiveBytes {
		t.Errorf("ByteSlice should be PrimitiveBytes")
	}

	// ByteArray should be [16]byte - an array of bytes, NOT PrimitiveBytes
	byteArrayField := findField(structDesc.Fields, "ByteArray")
	if byteArrayField.Type.Kind() != ir.KindArray {
		t.Errorf("ByteArray should be KindArray, got %v", byteArrayField.Type.Kind())
	}
}

func TestReflectionProvider_MapTypes(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(MapTypes{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	structDesc := schema.Types[0].(*ir.StructDescriptor)

	stringMapField := findField(structDesc.Fields, "StringMap")
	if stringMapField.Type.Kind() != ir.KindMap {
		t.Errorf("StringMap should be KindMap, got %v", stringMapField.Type.Kind())
	}

	mapOmitField := findField(structDesc.Fields, "MapOmit")
	if !mapOmitField.Optional {
		t.Error("MapOmit should be optional")
	}
}

func TestReflectionProvider_SpecialTypes(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(SpecialTypes{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	structDesc := schema.Types[0].(*ir.StructDescriptor)

	timeField := findField(structDesc.Fields, "Time")
	primDesc := timeField.Type.(*ir.PrimitiveDescriptor)
	if primDesc.PrimitiveKind != ir.PrimitiveTime {
		t.Errorf("Time should be PrimitiveTime")
	}

	durationField := findField(structDesc.Fields, "Duration")
	primDesc = durationField.Type.(*ir.PrimitiveDescriptor)
	if primDesc.PrimitiveKind != ir.PrimitiveDuration {
		t.Errorf("Duration should be PrimitiveDuration")
	}

	jsonNumberField := findField(structDesc.Fields, "JSONNumber")
	primDesc = jsonNumberField.Type.(*ir.PrimitiveDescriptor)
	if primDesc.PrimitiveKind != ir.PrimitiveString {
		t.Errorf("JSONNumber should be PrimitiveString")
	}

	rawMessageField := findField(structDesc.Fields, "RawMessage")
	primDesc = rawMessageField.Type.(*ir.PrimitiveDescriptor)
	if primDesc.PrimitiveKind != ir.PrimitiveAny {
		t.Errorf("RawMessage should be PrimitiveAny")
	}

	anyField := findField(structDesc.Fields, "AnyInterface")
	primDesc = anyField.Type.(*ir.PrimitiveDescriptor)
	if primDesc.PrimitiveKind != ir.PrimitiveAny {
		t.Errorf("AnyInterface should be PrimitiveAny")
	}

	emptyField := findField(structDesc.Fields, "EmptyStruct")
	primDesc = emptyField.Type.(*ir.PrimitiveDescriptor)
	if primDesc.PrimitiveKind != ir.PrimitiveEmpty {
		t.Errorf("EmptyStruct should be PrimitiveEmpty")
	}
}

func TestReflectionProvider_Embedding(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(EmbeddingStruct{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	// Should have 3 types: EmbeddingStruct, EmbeddedNoTag, EmbeddedWithTag
	if len(schema.Types) != 3 {
		t.Fatalf("expected 3 types, got %d", len(schema.Types))
	}

	var embeddingStruct *ir.StructDescriptor
	for _, typ := range schema.Types {
		if typ.TypeName().Name == "EmbeddingStruct" {
			embeddingStruct = typ.(*ir.StructDescriptor)
			break
		}
	}

	if embeddingStruct == nil {
		t.Fatal("EmbeddingStruct not found")
	}

	// Check Extends (should have EmbeddedNoTag)
	if len(embeddingStruct.Extends) != 1 {
		t.Errorf("expected 1 extended type, got %d", len(embeddingStruct.Extends))
	} else {
		if embeddingStruct.Extends[0].Name != "EmbeddedNoTag" {
			t.Errorf("expected EmbeddedNoTag in Extends, got %s", embeddingStruct.Extends[0].Name)
		}
	}

	// Check Fields (should have Nested and OwnField)
	if len(embeddingStruct.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(embeddingStruct.Fields))
	}

	nestedField := findField(embeddingStruct.Fields, "Nested")
	if nestedField == nil {
		t.Fatal("Nested field not found")
	}
	if nestedField.JSONName != "nested" {
		t.Errorf("expected JSONName 'nested', got %q", nestedField.JSONName)
	}
}

func TestReflectionProvider_Recursive(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(RecursiveStruct{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	// Should handle recursion gracefully with cycle detection
	if len(schema.Types) != 1 {
		t.Errorf("expected 1 type, got %d", len(schema.Types))
	}

	// Should have a warning about cycle detection
	if len(schema.Warnings) == 0 {
		t.Error("expected cycle detection warning")
	}
}

func TestReflectionProvider_MutualRecursion(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{
			reflect.TypeOf(MutuallyRecursive1{}),
			reflect.TypeOf(MutuallyRecursive2{}),
		},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	// Should handle mutual recursion
	if len(schema.Types) != 2 {
		t.Errorf("expected 2 types, got %d", len(schema.Types))
	}
}

func TestReflectionProvider_AnonymousStruct(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(WithAnonymousStruct{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	// Should have 2 types: WithAnonymousStruct and synthetic name for Inner
	if len(schema.Types) != 2 {
		t.Errorf("expected 2 types, got %d", len(schema.Types))
	}

	var mainStruct *ir.StructDescriptor
	var anonStruct *ir.StructDescriptor

	for _, typ := range schema.Types {
		structDesc := typ.(*ir.StructDescriptor)
		if structDesc.Name.Name == "WithAnonymousStruct" {
			mainStruct = structDesc
		} else {
			anonStruct = structDesc
		}
	}

	if mainStruct == nil {
		t.Fatal("WithAnonymousStruct not found")
	}
	if anonStruct == nil {
		t.Fatal("Anonymous struct not found")
	}

	// Check that Inner field references the synthetic type
	innerField := findField(mainStruct.Fields, "Inner")
	if innerField == nil {
		t.Fatal("Inner field not found")
	}

	if innerField.Type.Kind() != ir.KindReference {
		t.Errorf("Inner field should be KindReference, got %v", innerField.Type.Kind())
	}
}

func TestReflectionProvider_StringEncoded(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(StringEncoded{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	structDesc := schema.Types[0].(*ir.StructDescriptor)

	numberField := findField(structDesc.Fields, "NumberAsString")
	if !numberField.StringEncoded {
		t.Error("NumberAsString should have StringEncoded=true")
	}

	boolField := findField(structDesc.Fields, "BoolAsString")
	if !boolField.StringEncoded {
		t.Error("BoolAsString should have StringEncoded=true")
	}
}

func TestReflectionProvider_ValidateTags(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(ValidateTags{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	structDesc := schema.Types[0].(*ir.StructDescriptor)

	emailField := findField(structDesc.Fields, "Email")
	if emailField.ValidateTag != "required,email" {
		t.Errorf("expected ValidateTag 'required,email', got %q", emailField.ValidateTag)
	}

	ageField := findField(structDesc.Fields, "Age")
	if ageField.ValidateTag != "gte=0,lte=120" {
		t.Errorf("expected ValidateTag 'gte=0,lte=120', got %q", ageField.ValidateTag)
	}
}

func TestReflectionProvider_CustomTags(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(CustomTags{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	structDesc := schema.Types[0].(*ir.StructDescriptor)

	field1 := findField(structDesc.Fields, "Field1")
	if field1.RawTags["db"] != "field_1" {
		t.Errorf("expected db tag 'field_1', got %q", field1.RawTags["db"])
	}
	if field1.RawTags["xml"] != "Field1" {
		t.Errorf("expected xml tag 'Field1', got %q", field1.RawTags["xml"])
	}

	field2 := findField(structDesc.Fields, "Field2")
	if field2.RawTags["schema"] != "field2" {
		t.Errorf("expected schema tag 'field2', got %q", field2.RawTags["schema"])
	}
}

func TestReflectionProvider_SkipFields(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(SkipFields{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	structDesc := schema.Types[0].(*ir.StructDescriptor)

	// Should have Included and AlsoSkip (which becomes "-")
	// Skipped should be skipped
	if len(structDesc.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(structDesc.Fields))
	}

	includedField := findField(structDesc.Fields, "Included")
	if includedField == nil {
		t.Error("Included field not found")
	}

	// AlsoSkip should become a field named "-"
	alsoSkipField := findField(structDesc.Fields, "AlsoSkip")
	if alsoSkipField == nil {
		t.Error("AlsoSkip field not found")
	} else if alsoSkipField.JSONName != "-" {
		t.Errorf("expected JSONName '-', got %q", alsoSkipField.JSONName)
	}

	// Skipped should not exist
	skippedField := findField(structDesc.Fields, "Skipped")
	if skippedField != nil {
		t.Error("Skipped field should not be present")
	}
}

func TestReflectionProvider_OmitZero(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(OmitZero{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	structDesc := schema.Types[0].(*ir.StructDescriptor)

	field1 := findField(structDesc.Fields, "Field1")
	if !field1.Optional {
		t.Error("Field1 with omitzero should be optional")
	}

	field2 := findField(structDesc.Fields, "Field2")
	if !field2.Optional {
		t.Error("Field2 with omitempty should be optional")
	}
}

func TestReflectionProvider_GenericInstantiation(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(ConcreteGeneric{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	// Should have ConcreteGeneric and the instantiated GenericResponse[string]
	if len(schema.Types) < 2 {
		t.Errorf("expected at least 2 types, got %d", len(schema.Types))
	}

	// Check that generic name is sanitized
	var genericFound bool
	for _, typ := range schema.Types {
		name := typ.TypeName().Name
		if strings.Contains(name, "GenericResponse") {
			genericFound = true
			// Name should not contain brackets
			if strings.Contains(name, "[") || strings.Contains(name, "]") {
				t.Errorf("generic name should not contain brackets: %q", name)
			}
		}
	}

	if !genericFound {
		t.Error("GenericResponse type not found")
	}
}

func TestReflectionProvider_Aliases(t *testing.T) {
	provider := &ReflectionProvider{}
	schema, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(AliasStruct{})},
	})

	if err != nil {
		t.Fatalf("BuildSchema failed: %v", err)
	}

	// Should have AliasStruct, StringAlias, IntAlias
	if len(schema.Types) != 3 {
		t.Errorf("expected 3 types, got %d", len(schema.Types))
	}

	// Find the alias types
	var stringAlias, intAlias *ir.AliasDescriptor
	for _, typ := range schema.Types {
		if alias, ok := typ.(*ir.AliasDescriptor); ok {
			if alias.Name.Name == "StringAlias" {
				stringAlias = alias
			} else if alias.Name.Name == "IntAlias" {
				intAlias = alias
			}
		}
	}

	if stringAlias == nil {
		t.Error("StringAlias not found")
	} else {
		if stringAlias.Underlying.Kind() != ir.KindPrimitive {
			t.Errorf("StringAlias should have primitive underlying type")
		}
	}

	if intAlias == nil {
		t.Error("IntAlias not found")
	}
}

func TestReflectionProvider_UnsupportedTypes(t *testing.T) {
	tests := []struct {
		name    string
		typ     reflect.Type
		wantErr string
	}{
		{
			name:    "chan type",
			typ:     reflect.TypeOf(make(chan int)),
			wantErr: "unsupported type: chan",
		},
		{
			name:    "complex64",
			typ:     reflect.TypeOf(complex64(0)),
			wantErr: "unsupported type: complex64",
		},
		{
			name:    "func type",
			typ:     reflect.TypeOf(func() {}),
			wantErr: "unsupported type: func",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a struct with the unsupported field
			structType := reflect.StructOf([]reflect.StructField{
				{
					Name: "Field",
					Type: tt.typ,
					Tag:  `json:"field"`,
				},
			})

			provider := &ReflectionProvider{}
			_, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
				RootTypes: []reflect.Type{structType},
			})

			if err == nil {
				t.Error("expected error, got nil")
			} else if !strings.Contains(err.Error(), "unsupported type") {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestReflectionProvider_UnsupportedMapKeys(t *testing.T) {
	tests := []struct {
		name    string
		keyType reflect.Type
		wantErr string
	}{
		{
			name:    "bool key",
			keyType: reflect.TypeOf(true),
			wantErr: "unsupported map key type: bool",
		},
		{
			name:    "float32 key",
			keyType: reflect.TypeOf(float32(0)),
			wantErr: "unsupported map key type",
		},
		{
			name:    "float64 key",
			keyType: reflect.TypeOf(float64(0)),
			wantErr: "unsupported map key type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapType := reflect.MapOf(tt.keyType, reflect.TypeOf(""))
			structType := reflect.StructOf([]reflect.StructField{
				{
					Name: "Field",
					Type: mapType,
					Tag:  `json:"field"`,
				},
			})

			provider := &ReflectionProvider{}
			_, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
				RootTypes: []reflect.Type{structType},
			})

			if err == nil {
				t.Error("expected error, got nil")
			} else if !strings.Contains(err.Error(), "unsupported map key type") {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestReflectionProvider_NoRootTypes(t *testing.T) {
	provider := &ReflectionProvider{}
	_, err := provider.BuildSchema(context.Background(), ReflectionInputOptions{
		RootTypes: nil,
	})

	if err == nil {
		t.Error("expected error for no root types")
	}
}

func TestReflectionProvider_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	provider := &ReflectionProvider{}
	_, err := provider.BuildSchema(ctx, ReflectionInputOptions{
		RootTypes: []reflect.Type{reflect.TypeOf(SimpleStruct{})},
	})

	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestParseJSONTag(t *testing.T) {
	tests := []struct {
		tag           string
		fieldName     string
		wantJSON      string
		wantOptional  bool
		wantSkip      bool
		wantStringEnc bool
	}{
		{"", "Field", "Field", false, false, false},
		{"name", "Field", "name", false, false, false},
		{"name,omitempty", "Field", "name", true, false, false},
		{"name,omitzero", "Field", "name", true, false, false},
		{"name,string", "Field", "name", false, false, true},
		{"name,omitempty,string", "Field", "name", true, false, true},
		{"-", "Field", "", false, true, false},    // Skip field
		{"-,", "Field", "-", false, false, false}, // Field named "-"
		{",omitempty", "Field", "Field", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			jsonName, optional, skip, stringEnc := parseJSONTag(tt.tag, tt.fieldName)

			if jsonName != tt.wantJSON {
				t.Errorf("jsonName: got %q, want %q", jsonName, tt.wantJSON)
			}
			if optional != tt.wantOptional {
				t.Errorf("optional: got %v, want %v", optional, tt.wantOptional)
			}
			if skip != tt.wantSkip {
				t.Errorf("skip: got %v, want %v", skip, tt.wantSkip)
			}
			if stringEnc != tt.wantStringEnc {
				t.Errorf("stringEncoded: got %v, want %v", stringEnc, tt.wantStringEnc)
			}
		})
	}
}

func TestGenerateSyntheticName(t *testing.T) {
	b := &reflectionSchemaBuilder{}

	tests := []struct {
		input string
		want  string
	}{
		{"Response[User]", "Response_User"},
		{"Map[string, int]", "Map_string_int"},
		{"Response[pkg.User]", "Response_pkg_User"},
		{"Nested[Outer[Inner]]", "Nested_Outer_Inner"},
		{"Pair[*Foo, Bar]", "Pair_PtrFoo_Bar"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := b.generateSyntheticName(tt.input)
			if got != tt.want {
				t.Errorf("generateSyntheticName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Helper function to find a field by name
func findField(fields []ir.FieldDescriptor, name string) *ir.FieldDescriptor {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}
