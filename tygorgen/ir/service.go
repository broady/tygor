package ir

// ServiceDescriptor represents a group of related endpoints.
type ServiceDescriptor struct {
	// Name is the service identifier (e.g., "Users", "Posts").
	Name string

	// Endpoints contains all endpoints in this service.
	Endpoints []EndpointDescriptor

	// Documentation for this service.
	Documentation Documentation
}

// EndpointDescriptor represents a single API endpoint.
type EndpointDescriptor struct {
	// Name is the endpoint identifier within the service (e.g., "Create", "List").
	Name string

	// FullName is the qualified name: "ServiceName.EndpointName" (e.g., "Users.Create").
	FullName string

	// HTTPMethod is the HTTP verb: "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD".
	HTTPMethod string

	// Path is the URL path: "/{ServiceName}/{EndpointName}".
	// Example: "/Users/Create", "/News/List"
	Path string

	// Request describes the request payload type.
	// Typically a ReferenceDescriptor pointing to a type in Schema.Types.
	// For GET endpoints, fields become query parameters.
	// For POST/PUT/PATCH endpoints, this is the JSON request body.
	// May be nil for endpoints with no request parameters.
	Request TypeDescriptor

	// Response describes the response payload type.
	// May be a ReferenceDescriptor, ArrayDescriptor, MapDescriptor, etc.
	Response TypeDescriptor

	// Documentation for this endpoint.
	Documentation Documentation
}
