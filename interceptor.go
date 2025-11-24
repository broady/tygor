package tygorpc

import "context"

// RPCInfo provides metadata about the current operation.
type RPCInfo struct {
	Service string
	Method  string
}

// HandlerFunc represents the next handler in the chain.
type HandlerFunc func(ctx context.Context, req any) (res any, err error)

// Interceptor is a generic hook that wraps the RPC handler execution.
// req/res are pointers to the structs.
type Interceptor func(ctx context.Context, req any, info *RPCInfo, handler HandlerFunc) (res any, err error)

// chainInterceptors combines multiple interceptors into a single one.
// The first interceptor in the slice is the outer-most one (runs first).
func chainInterceptors(interceptors []Interceptor) Interceptor {
	if len(interceptors) == 0 {
		return nil
	}
	if len(interceptors) == 1 {
		return interceptors[0]
	}
	return func(ctx context.Context, req any, info *RPCInfo, handler HandlerFunc) (any, error) {
		// Chain: i[0] -> i[1] -> ... -> handler
		// We recursively build the chain
		var chain HandlerFunc = handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			current := interceptors[i]
			next := chain
			chain = func(ctx context.Context, req any) (any, error) {
				return current(ctx, req, info, next)
			}
		}
		return chain(ctx, req)
	}
}
