package ir

// Schema represents a complete set of types and services to generate.
type Schema struct {
	// Package is the source Go package information.
	Package PackageInfo

	// Types contains top-level named type descriptors to generate.
	// Only Struct, Alias, and Enum descriptors appear here.
	// Expression types (Primitive, Array, Map, etc.) appear nested
	// within these named types' fields and type expressions.
	//
	// Ordering: Providers emit types in topological order (dependencies before
	// dependents) as a convenience. However, generators MUST NOT rely on this
	// ordering for correctness—they MUST handle types in any order, including
	// circular references. See §7.1 for declaration order requirements.
	Types []TypeDescriptor

	// Services contains service descriptors with their endpoints.
	// This field is OPTIONAL - schemas containing only types (no services)
	// are valid. Generators that only emit type definitions MAY ignore this.
	// When present, endpoint Request/Response fields reference types in Types.
	Services []ServiceDescriptor

	// Warnings contains non-fatal issues encountered during schema building.
	Warnings []Warning
}

// AddType adds a named type descriptor to the schema.
func (s *Schema) AddType(t TypeDescriptor) {
	s.Types = append(s.Types, t)
}

// AddService adds a service descriptor to the schema.
func (s *Schema) AddService(svc ServiceDescriptor) {
	s.Services = append(s.Services, svc)
}

// AddWarning adds a warning to the schema.
func (s *Schema) AddWarning(w Warning) {
	s.Warnings = append(s.Warnings, w)
}

// FindType looks up a type by name. Returns nil if not found.
func (s *Schema) FindType(name GoIdentifier) TypeDescriptor {
	for _, t := range s.Types {
		if t.TypeName() == name {
			return t
		}
	}
	return nil
}

// FindService looks up a service by name. Returns nil if not found.
func (s *Schema) FindService(name string) *ServiceDescriptor {
	for i := range s.Services {
		if s.Services[i].Name == name {
			return &s.Services[i]
		}
	}
	return nil
}
