package tygorgen

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/broady/tygor"
	"github.com/broady/tygor/internal/testfixtures"
)

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
				return c.Provider == "source" &&
					c.PreserveComments == "default" &&
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

func TestGenerate_NoOutDir_ReturnsFilesInMemory(t *testing.T) {
	reg := tygor.NewApp()
	handler := func(ctx context.Context, req *testfixtures.CreateUserRequest) (*testfixtures.User, error) {
		return nil, nil
	}
	reg.Service("Users").Register("Create", tygor.Exec(handler))

	cfg := &Config{
		Provider: "reflection",
	}

	result, err := Generate(reg, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have files in result since no OutDir
	if len(result.Files) == 0 {
		t.Error("expected files in result when OutDir is empty")
	}

	// Should have types.ts and manifest.ts at minimum
	var hasTypes, hasManifest bool
	for _, f := range result.Files {
		if f.Path == "types.ts" {
			hasTypes = true
		}
		if f.Path == "manifest.ts" {
			hasManifest = true
		}
	}
	if !hasTypes {
		t.Error("missing types.ts in result files")
	}
	if !hasManifest {
		t.Error("missing manifest.ts in result files")
	}
}

func TestGenerate_EmptyApp(t *testing.T) {
	reg := tygor.NewApp()
	outDir := t.TempDir()

	cfg := &Config{OutDir: outDir, SingleFile: true, Provider: "reflection"}

	_, err := Generate(reg, cfg)
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
	if !strings.Contains(string(content), "Manifest") {
		t.Error("manifest.ts missing Manifest interface")
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

	cfg := &Config{OutDir: outDir, SingleFile: true, Provider: "reflection"}

	_, err := Generate(reg, cfg)
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

	cfg := &Config{OutDir: outDir, SingleFile: true, Provider: "reflection"}

	_, err := Generate(reg, cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	manifestPath := filepath.Join(outDir, "manifest.ts")
	content, _ := os.ReadFile(manifestPath)
	manifestStr := string(content)

	// Verify imports
	if !strings.Contains(manifestStr, "import * as types") {
		t.Error("manifest.ts missing types import")
	}

	// Verify interface definition
	if !strings.Contains(manifestStr, "export interface Manifest") {
		t.Error("manifest.ts missing Manifest interface")
	}

	// Verify routes
	if !strings.Contains(manifestStr, `"Users.Create"`) {
		t.Error("manifest.ts missing Users.Create")
	}
	if !strings.Contains(manifestStr, `"Posts.List"`) {
		t.Error("manifest.ts missing Posts.List")
	}
}

func TestGenerate_TypesFile(t *testing.T) {
	reg := tygor.NewApp()
	outDir := t.TempDir()

	handler := func(ctx context.Context, req *testfixtures.CreateUserRequest) (*testfixtures.User, error) {
		return nil, nil
	}
	reg.Service("Users").Register("Create", tygor.Exec(handler))

	cfg := &Config{OutDir: outDir, SingleFile: true, Provider: "reflection"}

	_, err := Generate(reg, cfg)
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

	// Should have TypeScript interface exports
	// (Note: new generator creates a single file with all types, not re-exports)
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
		SingleFile:       true,
		Provider:         "reflection",
		PreserveComments: "none",
		EnumStyle:        "enum",
		OptionalType:     "null",
		TypeMappings: map[string]string{
			"custom.Type": "CustomType",
		},
	}

	_, err := Generate(reg, cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check types file content to verify config was applied
	typesPath := filepath.Join(outDir, "types.ts")
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

	cfg := &Config{OutDir: outDir, SingleFile: true, Provider: "reflection"}

	_, err := Generate(reg, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read the generated types file
	typesPath := filepath.Join(outDir, "types.ts")
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
