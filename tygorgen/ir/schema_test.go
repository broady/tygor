package ir

import "testing"

func TestSchema_AddType(t *testing.T) {
	s := &Schema{}

	s.AddType(&StructDescriptor{
		Name: GoIdentifier{Name: "User", Package: "api"},
	})
	s.AddType(&AliasDescriptor{
		Name:       GoIdentifier{Name: "UserID", Package: "api"},
		Underlying: String(),
	})

	if len(s.Types) != 2 {
		t.Errorf("Schema.Types length = %d, want 2", len(s.Types))
	}
}

func TestSchema_AddService(t *testing.T) {
	s := &Schema{}

	s.AddService(ServiceDescriptor{
		Name: "Users",
		Endpoints: []EndpointDescriptor{
			{Name: "Create", FullName: "Users.Create"},
		},
	})
	s.AddService(ServiceDescriptor{
		Name: "Posts",
	})

	if len(s.Services) != 2 {
		t.Errorf("Schema.Services length = %d, want 2", len(s.Services))
	}
}

func TestSchema_AddWarning(t *testing.T) {
	s := &Schema{}

	s.AddWarning(Warning{Code: "W001", Message: "warning 1"})
	s.AddWarning(Warning{Code: "W002", Message: "warning 2"})

	if len(s.Warnings) != 2 {
		t.Errorf("Schema.Warnings length = %d, want 2", len(s.Warnings))
	}
}

func TestSchema_FindType(t *testing.T) {
	s := &Schema{}
	userID := GoIdentifier{Name: "User", Package: "api"}
	postID := GoIdentifier{Name: "Post", Package: "api"}

	s.AddType(&StructDescriptor{Name: userID})
	s.AddType(&StructDescriptor{Name: postID})

	// Find existing type
	found := s.FindType(userID)
	if found == nil {
		t.Fatal("FindType should find User")
	}
	if found.TypeName() != userID {
		t.Errorf("FindType returned wrong type: %v", found.TypeName())
	}

	// Find non-existing type
	notFound := s.FindType(GoIdentifier{Name: "NotExist", Package: "api"})
	if notFound != nil {
		t.Error("FindType should return nil for non-existing type")
	}
}

func TestSchema_FindService(t *testing.T) {
	s := &Schema{}
	s.AddService(ServiceDescriptor{Name: "Users"})
	s.AddService(ServiceDescriptor{Name: "Posts"})

	// Find existing service
	found := s.FindService("Users")
	if found == nil {
		t.Fatal("FindService should find Users")
	}
	if found.Name != "Users" {
		t.Errorf("FindService returned wrong service: %s", found.Name)
	}

	// Find non-existing service
	notFound := s.FindService("NotExist")
	if notFound != nil {
		t.Error("FindService should return nil for non-existing service")
	}
}

func TestSchema_Package(t *testing.T) {
	s := &Schema{
		Package: PackageInfo{
			Path: "github.com/example/api",
			Name: "api",
			Dir:  "/home/user/go/src/github.com/example/api",
		},
	}

	if s.Package.Path != "github.com/example/api" {
		t.Errorf("Schema.Package.Path = %q", s.Package.Path)
	}
	if s.Package.Name != "api" {
		t.Errorf("Schema.Package.Name = %q", s.Package.Name)
	}
}

func TestSchema_Full(t *testing.T) {
	s := &Schema{
		Package: PackageInfo{
			Path: "github.com/example/api",
			Name: "api",
		},
	}

	// Add types
	s.AddType(&StructDescriptor{
		Name: GoIdentifier{Name: "User", Package: "api"},
		Fields: []FieldDescriptor{
			{Name: "ID", JSONName: "id", Type: Int(64)},
			{Name: "Name", JSONName: "name", Type: String()},
		},
	})
	s.AddType(&AliasDescriptor{
		Name:       GoIdentifier{Name: "UserID", Package: "api"},
		Underlying: Int(64),
	})
	s.AddType(&EnumDescriptor{
		Name: GoIdentifier{Name: "Status", Package: "api"},
		Members: []EnumMember{
			{Name: "StatusActive", Value: "active"},
			{Name: "StatusInactive", Value: "inactive"},
		},
	})

	// Add services
	s.AddService(ServiceDescriptor{
		Name: "Users",
		Endpoints: []EndpointDescriptor{
			{
				Name:       "Get",
				FullName:   "Users.Get",
				HTTPMethod: "GET",
				Path:       "/Users/Get",
				Request:    Ref("GetUserRequest", "api"),
				Response:   Ref("User", "api"),
			},
		},
	})

	// Verify counts
	if len(s.Types) != 3 {
		t.Errorf("Schema.Types length = %d, want 3", len(s.Types))
	}
	if len(s.Services) != 1 {
		t.Errorf("Schema.Services length = %d, want 1", len(s.Services))
	}

	// Verify type kinds
	kinds := make(map[DescriptorKind]int)
	for _, typ := range s.Types {
		kinds[typ.Kind()]++
	}
	if kinds[KindStruct] != 1 {
		t.Errorf("expected 1 struct, got %d", kinds[KindStruct])
	}
	if kinds[KindAlias] != 1 {
		t.Errorf("expected 1 alias, got %d", kinds[KindAlias])
	}
	if kinds[KindEnum] != 1 {
		t.Errorf("expected 1 enum, got %d", kinds[KindEnum])
	}
}
