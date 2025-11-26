package tygor

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/broady/tygor/internal/rpccontext"
)

// Context provides type-safe access to RPC metadata and HTTP primitives.
// It embeds context.Context, so it can be used anywhere a context.Context is expected.
//
// Interceptors receive *Context directly for convenient access to RPC metadata.
// Handlers receive context.Context but can use FromContext to get the *Context if needed.
type Context struct {
	context.Context
	service string
	method  string
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

// Method returns the name of the RPC method being called.
func (c *Context) Method() string { return c.method }

// HTTPRequest returns the underlying HTTP request.
func (c *Context) HTTPRequest() *http.Request { return c.request }

// HTTPWriter returns the underlying HTTP response writer.
// Use with caution in handlers - prefer returning errors to writing directly.
// This is useful for setting response headers.
func (c *Context) HTTPWriter() http.ResponseWriter { return c.writer }

// FromContext extracts the *Context from a context.Context.
// Returns the Context and true if found, or nil and false if not in a tygor handler context.
//
// This is useful in handlers that receive context.Context but need access to RPC metadata:
//
//	func (s *MyService) GetThing(ctx context.Context, req *GetThingRequest) (*GetThingResponse, error) {
//	    tc, ok := tygor.FromContext(ctx)
//	    if ok {
//	        log.Printf("handling %s.%s", tc.Service(), tc.Method())
//	    }
//	    // ...
//	}
func FromContext(ctx context.Context) (*Context, bool) {
	v := ctx.Value(rpccontext.ContextKey)
	if v == nil {
		return nil, false
	}

	// Try our type first (production path)
	if tc, ok := v.(*Context); ok {
		return tc, true
	}

	// Try the internal type (test utilities path)
	if rc, ok := v.(*rpccontext.Context); ok {
		return &Context{
			Context: rc.Context,
			service: rc.Service,
			method:  rc.Method,
			request: rc.Request,
			writer:  rc.Writer,
		}, true
	}

	return nil, false
}

// NewContext creates a Context for testing interceptors and handlers.
// In production code, the framework creates contexts automatically.
func NewContext(parent context.Context, service, method string) *Context {
	return newContext(parent, nil, nil, service, method)
}

// newContext creates a new Context with all fields.
func newContext(parent context.Context, w http.ResponseWriter, r *http.Request, service, method string) *Context {
	ctx := &Context{
		service: service,
		method:  method,
		request: r,
		writer:  w,
	}
	// Store self using the shared key so FromContext works
	ctx.Context = context.WithValue(parent, rpccontext.ContextKey, ctx)
	return ctx
}
