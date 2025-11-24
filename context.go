package tygor

import (
	"context"
	"net/http"
)

type contextKey struct {
	name string
}

var (
	requestKey = &contextKey{"request"}
	writerKey  = &contextKey{"writer"}
	rpcInfoKey = &contextKey{"rpc_info"}
)

// RequestFromContext returns the HTTP request from the context.
func RequestFromContext(ctx context.Context) *http.Request {
	if r, ok := ctx.Value(requestKey).(*http.Request); ok {
		return r
	}
	return nil
}

// SetHeader sets an HTTP response header.
// It requires that the handler was called via the Registry.
func SetHeader(ctx context.Context, key, value string) {
	if w, ok := ctx.Value(writerKey).(http.ResponseWriter); ok {
		w.Header().Set(key, value)
	}
}

// MethodFromContext returns the service and method name of the current RPC.
func MethodFromContext(ctx context.Context) (service, method string, ok bool) {
	if info, ok := ctx.Value(rpcInfoKey).(*RPCInfo); ok {
		return info.Service, info.Method, true
	}
	return "", "", false
}

func newContext(ctx context.Context, w http.ResponseWriter, r *http.Request, info *RPCInfo) context.Context {
	ctx = context.WithValue(ctx, writerKey, w)
	ctx = context.WithValue(ctx, requestKey, r)
	ctx = context.WithValue(ctx, rpcInfoKey, info)
	return ctx
}

// NewTestContext creates a context with RPC metadata for testing.
// This is useful when testing handlers directly without going through the Registry.
//
// Example:
//
//	req := httptest.NewRequest("POST", "/test", body)
//	w := httptest.NewRecorder()
//	info := &RPCInfo{Service: "MyService", Method: "MyMethod"}
//	ctx := NewTestContext(req.Context(), w, req, info)
//	req = req.WithContext(ctx)
func NewTestContext(ctx context.Context, w http.ResponseWriter, r *http.Request, info *RPCInfo) context.Context {
	return newContext(ctx, w, r, info)
}

// TestContextSetup returns a context setup function for use with testutil.NewRequest().
// This provides a convenient way to set up tygor RPC context when testing from external packages.
//
// Example usage:
//
//	req, w := testutil.NewRequest(tygor.TestContextSetup()).
//	    POST("/test").
//	    WithJSON(&MyRequest{...}).
//	    Build()
//
//	handler.ServeHTTP(w, req, tygor.HandlerConfig{})
func TestContextSetup() func(ctx context.Context, w http.ResponseWriter, r *http.Request, service, method string) context.Context {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, service, method string) context.Context {
		info := &RPCInfo{
			Service: service,
			Method:  method,
		}
		return NewTestContext(ctx, w, r, info)
	}
}
