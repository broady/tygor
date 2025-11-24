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
