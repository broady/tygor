package tygor

import (
	"context"
	"net/http"

	"github.com/broady/tygor/internal/rpccontext"
)

// RequestFromContext returns the HTTP request from the context.
func RequestFromContext(ctx context.Context) *http.Request {
	return rpccontext.RequestFromContext(ctx)
}

// SetHeader sets an HTTP response header.
// It requires that the handler was called via the Registry.
func SetHeader(ctx context.Context, key, value string) {
	if w := rpccontext.WriterFromContext(ctx); w != nil {
		w.Header().Set(key, value)
	}
}

// MethodFromContext returns the service and method name of the current RPC.
func MethodFromContext(ctx context.Context) (service, method string, ok bool) {
	if info, ok := rpccontext.InfoFromContext(ctx).(*rpccontext.RPCInfo); ok {
		return info.Service, info.Method, true
	}
	return "", "", false
}

func newContext(ctx context.Context, w http.ResponseWriter, r *http.Request, info *RPCInfo) context.Context {
	return rpccontext.NewContext(ctx, w, r, info)
}
