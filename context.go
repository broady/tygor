package tygor

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/broady/tygor/internal/tgrcontext"
)

// Context provides type-safe access to request metadata and HTTP primitives.
// It embeds context.Context, so it can be used anywhere a context.Context is expected.
//
// Interceptors receive *Context directly for convenient access to request metadata.
// Handlers receive context.Context but can use FromContext to get the *Context if needed.
type Context struct {
	context.Context
	service string
	name    string
	request *http.Request
	writer  http.ResponseWriter

	// Internal fields for handler execution (not exposed via public methods)
	errorTransformer   ErrorTransformer
	maskInternalErrors bool
	interceptors       []UnaryInterceptor
	logger             *slog.Logger
	maxRequestBodySize uint64
}

// Service returns the name of the service being called.
func (c *Context) Service() string { return c.service }

// EndpointID returns the full identifier for the endpoint being called (e.g., "Users.Create").
func (c *Context) EndpointID() string { return c.service + "." + c.name }

// HTTPRequest returns the underlying HTTP request.
func (c *Context) HTTPRequest() *http.Request { return c.request }

// HTTPWriter returns the underlying HTTP response writer.
// Use with caution in handlers - prefer returning errors to writing directly.
// This is useful for setting response headers.
func (c *Context) HTTPWriter() http.ResponseWriter { return c.writer }

// FromContext extracts the *Context from a context.Context.
// Returns the Context and true if found, or nil and false if not in a tygor handler context.
//
// This is useful in handlers that receive context.Context but need access to request metadata:
//
//	func (s *MyService) GetThing(ctx context.Context, req *GetThingRequest) (*GetThingResponse, error) {
//	    tc, ok := tygor.FromContext(ctx)
//	    if ok {
//	        log.Printf("handling %s", tc.EndpointID())
//	    }
//	    // ...
//	}
func FromContext(ctx context.Context) (*Context, bool) {
	v := ctx.Value(tgrcontext.ContextKey)
	if v == nil {
		return nil, false
	}

	// Try our type first (production path)
	if tc, ok := v.(*Context); ok {
		return tc, true
	}

	// Try the internal type (test utilities path)
	if rc, ok := v.(*tgrcontext.Context); ok {
		return &Context{
			Context: rc.Context,
			service: rc.Service,
			name:    rc.Name,
			request: rc.Request,
			writer:  rc.Writer,
		}, true
	}

	return nil, false
}

// NewContext creates a Context for testing interceptors and handlers.
// In production code, the framework creates contexts automatically.
func NewContext(parent context.Context, service, name string) *Context {
	return newContext(parent, nil, nil, service, name)
}

// newContext creates a new Context with all fields.
func newContext(parent context.Context, w http.ResponseWriter, r *http.Request, service, name string) *Context {
	ctx := &Context{
		service: service,
		name:    name,
		request: r,
		writer:  w,
	}
	// Store self using the shared key so FromContext works
	ctx.Context = context.WithValue(parent, tgrcontext.ContextKey, ctx)
	return ctx
}
