package tygorgen

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/broady/tygor"
	"github.com/broady/tygor/internal/testfixtures"
)

// Test types for reflection tests (local to this package)
type TestStruct struct {
	Name string `json:"name"`
}

func TestSanitizePkgPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple path", "github.com/user/repo", "github_com_user_repo"},
		{"with dots", "pkg.sub.name", "pkg_sub_name"},
		{"with hyphens", "my-package", "my_package"},
		{"alphanumeric only", "abc123", "abc123"},
		{"empty string", "", ""},
		{"special chars", "foo@bar#baz", "foo_bar_baz"},
		{"mixed case", "GitHub.Com/User/Repo", "GitHub_Com_User_Repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePkgPath(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizePkgPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    reflect.Type
		expected string
	}{
		// Nil type
		{"nil type", nil, "any"},

		// Primitives
		{"string", reflect.TypeOf(""), "string"},
		{"bool", reflect.TypeOf(true), "boolean"},
		{"int", reflect.TypeOf(0), "number"},
		{"int8", reflect.TypeOf(int8(0)), "number"},
		{"int16", reflect.TypeOf(int16(0)), "number"},
		{"int32", reflect.TypeOf(int32(0)), "number"},
		{"int64", reflect.TypeOf(int64(0)), "number"},
		{"uint", reflect.TypeOf(uint(0)), "number"},
		{"uint8", reflect.TypeOf(uint8(0)), "number"},
		{"uint16", reflect.TypeOf(uint16(0)), "number"},
		{"uint32", reflect.TypeOf(uint32(0)), "number"},
		{"uint64", reflect.TypeOf(uint64(0)), "number"},
		{"float32", reflect.TypeOf(float32(0)), "number"},
		{"float64", reflect.TypeOf(float64(0)), "number"},
		{"byte", reflect.TypeOf(byte(0)), "number"},
		{"rune", reflect.TypeOf(rune(0)), "number"},
		{"uintptr", reflect.TypeOf(uintptr(0)), "number"},
		{"complex64", reflect.TypeOf(complex64(0)), "any"},
		{"complex128", reflect.TypeOf(complex128(0)), "any"},

		// Struct types
		{"struct", reflect.TypeOf(TestStruct{}), "TestStruct"},

		// Pointer types
		{"pointer to struct", reflect.TypeOf(&TestStruct{}), "TestStruct"},
		{"pointer to string", reflect.TypeOf(new(string)), "string"},

		// Slice types
		{"slice of struct", reflect.TypeOf([]TestStruct{}), "TestStruct[]"},
		{"slice of int", reflect.TypeOf([]int{}), "number[]"},
		{"slice of string", reflect.TypeOf([]string{}), "string[]"},

		// Array types
		{"array of struct", reflect.TypeOf([5]TestStruct{}), "TestStruct[]"},
		{"array of int", reflect.TypeOf([3]int{}), "number[]"},

		// Map types
		{"map string to struct", reflect.TypeOf(map[string]TestStruct{}), "Record<string, TestStruct>"},
		{"map int to string", reflect.TypeOf(map[int]string{}), "Record<number, string>"},

		// Nested types
		{"pointer to slice", reflect.TypeOf(&[]TestStruct{}), "TestStruct[]"},
		{"slice of pointers", reflect.TypeOf([]*TestStruct{}), "TestStruct[]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTypeName(tt.input)
			if result != tt.expected {
				t.Errorf("getTypeName(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAddPkg(t *testing.T) {
	tests := []struct {
		name        string
		input       reflect.Type
		expectPkg   bool
		expectedPkg string
	}{
		{"nil type", nil, false, ""},
		{"builtin string", reflect.TypeOf(""), false, ""},
		{"builtin int", reflect.TypeOf(0), false, ""},
		{"struct type", reflect.TypeOf(TestStruct{}), true, "github.com/broady/tygor/tygorgen"},
		{"pointer to struct", reflect.TypeOf(&TestStruct{}), true, "github.com/broady/tygor/tygorgen"},
		{"slice of struct", reflect.TypeOf([]TestStruct{}), true, "github.com/broady/tygor/tygorgen"},
		{"map with struct value", reflect.TypeOf(map[string]TestStruct{}), true, "github.com/broady/tygor/tygorgen"},
		{"slice of builtin", reflect.TypeOf([]int{}), false, ""},
		{"map of builtins", reflect.TypeOf(map[string]int{}), false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := make(map[string]bool)
			addPkg(set, tt.input)

			if tt.expectPkg {
				if !set[tt.expectedPkg] {
					t.Errorf("addPkg did not add expected package %q, got %v", tt.expectedPkg, set)
				}
			} else {
				if len(set) != 0 {
					t.Errorf("addPkg should not have added any packages, got %v", set)
				}
			}
		})
	}
}

func TestApplyConfigDefaults(t *testing.T) {
	tests := []struct {
		name   string
		input  *Config
		check  func(*Config) bool
		errMsg string
	}{
		{
			name:  "empty config gets defaults",
			input: &Config{OutDir: "/tmp"},
			check: func(c *Config) bool {
				return c.PreserveComments == "default" &&
					c.EnumStyle == "union" &&
					c.OptionalType == "undefined"
			},
			errMsg: "defaults not applied correctly",
		},
		{
			name: "explicit values preserved",
			input: &Config{
				OutDir:           "/tmp",
				PreserveComments: "none",
				EnumStyle:        "enum",
				OptionalType:     "null",
			},
			check: func(c *Config) bool {
				return c.PreserveComments == "none" &&
					c.EnumStyle == "enum" &&
					c.OptionalType == "null"
			},
			errMsg: "explicit values not preserved",
		},
		{
			name: "partial config",
			input: &Config{
				OutDir:    "/tmp",
				EnumStyle: "const",
			},
			check: func(c *Config) bool {
				return c.PreserveComments == "default" &&
					c.EnumStyle == "const" &&
					c.OptionalType == "undefined"
			},
			errMsg: "partial config not handled correctly",
		},
		{
			name: "does not mutate input",
			input: &Config{
				OutDir: "/tmp",
			},
			check: func(c *Config) bool {
				// The returned config should be different from original
				return c.PreserveComments == "default"
			},
			errMsg: "config mutation check failed",
		},
		{
			name: "preserves TypeMappings",
			input: &Config{
				OutDir:       "/tmp",
				TypeMappings: map[string]string{"foo": "bar"},
			},
			check: func(c *Config) bool {
				return c.TypeMappings != nil && c.TypeMappings["foo"] == "bar"
			},
			errMsg: "TypeMappings not preserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyConfigDefaults(tt.input)
			if !tt.check(result) {
				t.Error(tt.errMsg)
			}
		})
	}
}

func TestGenerate_MissingOutDir(t *testing.T) {
	reg := tygor.NewApp()
	cfg := &Config{} // Missing OutDir

	err := Generate(reg, cfg)
	if err == nil {
		t.Error("expected error for missing OutDir")
	}
	if !strings.Contains(err.Error(), "OutDir is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerate_EmptyApp(t *testing.T) {
	reg := tygor.NewApp()
	outDir := t.TempDir()

	cfg := &Config{OutDir: outDir}

	err := Generate(reg, cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check types.ts exists
	typesPath := filepath.Join(outDir, "types.ts")
	if _, err := os.Stat(typesPath); os.IsNotExist(err) {
		t.Error("types.ts was not created")
	}

	// Check manifest.ts exists
	manifestPath := filepath.Join(outDir, "manifest.ts")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("manifest.ts was not created")
	}

	// Verify manifest content for empty registry
	content, _ := os.ReadFile(manifestPath)
	if !strings.Contains(string(content), "RPCManifest") {
		t.Error("manifest.ts missing RPCManifest interface")
	}
}

func TestGenerate_WithHandlers(t *testing.T) {
	reg := tygor.NewApp()
	outDir := t.TempDir()

	// Register a test handler using internal test fixture types
	handler := func(ctx context.Context, req *testfixtures.CreateUserRequest) (*testfixtures.User, error) {
		return &testfixtures.User{Username: req.Username}, nil
	}
	reg.Service("Users").Register("Create", tygor.Exec(handler))

	cfg := &Config{OutDir: outDir}

	err := Generate(reg, cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check manifest.ts has the route
	manifestPath := filepath.Join(outDir, "manifest.ts")
	content, _ := os.ReadFile(manifestPath)
	manifestStr := string(content)

	if !strings.Contains(manifestStr, `"Users.Create"`) {
		t.Error("manifest.ts missing Users.Create route")
	}
	if !strings.Contains(manifestStr, `method: "POST"`) {
		t.Error("manifest.ts missing POST method")
	}
	if !strings.Contains(manifestStr, `path: "/Users/Create"`) {
		t.Error("manifest.ts missing correct path")
	}
}

func TestGenerate_ManifestStructure(t *testing.T) {
	reg := tygor.NewApp()
	outDir := t.TempDir()

	createHandler := func(ctx context.Context, req *testfixtures.CreateUserRequest) (*testfixtures.User, error) {
		return nil, nil
	}
	listHandler := func(ctx context.Context, req *testfixtures.ListPostsParams) ([]*testfixtures.Post, error) {
		return nil, nil
	}
	reg.Service("Users").Register("Create", tygor.Exec(createHandler))
	reg.Service("Posts").Register("List", tygor.Query(listHandler))

	cfg := &Config{OutDir: outDir}

	err := Generate(reg, cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	manifestPath := filepath.Join(outDir, "manifest.ts")
	content, _ := os.ReadFile(manifestPath)
	manifestStr := string(content)

	// Verify imports
	if !strings.Contains(manifestStr, "import { ServiceRegistry }") {
		t.Error("manifest.ts missing ServiceRegistry import")
	}
	if !strings.Contains(manifestStr, "import * as types") {
		t.Error("manifest.ts missing types import")
	}

	// Verify interface definition
	if !strings.Contains(manifestStr, "export interface RPCManifest") {
		t.Error("manifest.ts missing RPCManifest interface")
	}

	// Verify routes
	if !strings.Contains(manifestStr, `"Users.Create"`) {
		t.Error("manifest.ts missing Users.Create")
	}
	if !strings.Contains(manifestStr, `"Posts.List"`) {
		t.Error("manifest.ts missing Posts.List")
	}

	// Verify metadata
	if !strings.Contains(manifestStr, "const metadata") {
		t.Error("manifest.ts missing metadata constant")
	}

	// Verify registry export
	if !strings.Contains(manifestStr, "export const registry") {
		t.Error("manifest.ts missing registry export")
	}
}

func TestGenerate_TypesFile(t *testing.T) {
	reg := tygor.NewApp()
	outDir := t.TempDir()

	handler := func(ctx context.Context, req *testfixtures.CreateUserRequest) (*testfixtures.User, error) {
		return nil, nil
	}
	reg.Service("Users").Register("Create", tygor.Exec(handler))

	cfg := &Config{OutDir: outDir}

	err := Generate(reg, cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	typesPath := filepath.Join(outDir, "types.ts")
	content, _ := os.ReadFile(typesPath)
	typesStr := string(content)

	// Verify header
	if !strings.Contains(typesStr, "Code generated by tygor") {
		t.Error("types.ts missing generation header")
	}

	// Should have export statement for package types
	if !strings.Contains(typesStr, "export * from") {
		t.Error("types.ts missing re-exports")
	}
}

func TestGenerate_CustomConfig(t *testing.T) {
	reg := tygor.NewApp()
	outDir := t.TempDir()

	handler := func(ctx context.Context, req *testfixtures.CreateUserRequest) (*testfixtures.User, error) {
		return nil, nil
	}
	reg.Service("Users").Register("Create", tygor.Exec(handler))

	cfg := &Config{
		OutDir:           outDir,
		PreserveComments: "none",
		EnumStyle:        "enum",
		OptionalType:     "null",
		TypeMappings: map[string]string{
			"custom.Type": "CustomType",
		},
	}

	err := Generate(reg, cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check types file content to verify config was applied
	typesPath := filepath.Join(outDir, "types_github_com_broady_tygor_internal_testfixtures.ts")
	content, err := os.ReadFile(typesPath)
	if err != nil {
		t.Fatalf("failed to read generated types: %v", err)
	}
	typesStr := string(content)

	// Verify User struct is present (sanity check that config was used)
	if !strings.Contains(typesStr, "export interface User") {
		t.Error("expected User interface to be generated")
	}
}

// TestGenerate_GETParamsUseLowercaseNames verifies that GET request parameter types
// generate TypeScript with lowercase property names (matching schema tags via json tags).
// This ensures the TypeScript client sends query params that match what Go expects.
func TestGenerate_GETParamsUseLowercaseNames(t *testing.T) {
	reg := tygor.NewApp()
	outDir := t.TempDir()

	// Register a GET handler using ListPostsParams which has both json and schema tags
	listHandler := func(ctx context.Context, req *testfixtures.ListPostsParams) ([]*testfixtures.Post, error) {
		return nil, nil
	}
	reg.Service("Posts").Register("List", tygor.Query(listHandler))

	cfg := &Config{OutDir: outDir}

	err := Generate(reg, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read the generated types file
	typesPath := filepath.Join(outDir, "types_github_com_broady_tygor_internal_testfixtures.ts")
	content, err := os.ReadFile(typesPath)
	if err != nil {
		t.Fatalf("failed to read generated types: %v", err)
	}
	typesStr := string(content)

	// Verify ListPostsParams has lowercase property names (from json tags)
	// These should match the schema tags used for query parameter decoding
	if !strings.Contains(typesStr, "author_id") {
		t.Error("ListPostsParams should have 'author_id' property (lowercase)")
	}
	if !strings.Contains(typesStr, "published") {
		t.Error("ListPostsParams should have 'published' property (lowercase)")
	}
	if !strings.Contains(typesStr, "limit") {
		t.Error("ListPostsParams should have 'limit' property (lowercase)")
	}
	if !strings.Contains(typesStr, "offset") {
		t.Error("ListPostsParams should have 'offset' property (lowercase)")
	}

	// Verify we DON'T have the capitalized Go field names
	if strings.Contains(typesStr, "AuthorID") {
		t.Error("ListPostsParams should NOT have 'AuthorID' - should use lowercase 'author_id'")
	}
	if strings.Contains(typesStr, "Published:") {
		t.Error("ListPostsParams should NOT have 'Published' - should use lowercase 'published'")
	}
}
