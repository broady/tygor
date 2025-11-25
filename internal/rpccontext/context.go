// Package rpccontext provides shared context key definitions for tygor RPC handlers.
package rpccontext

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

// RPCInfo contains metadata about the current RPC call.
// The public type tygor.RPCInfo is an alias for this type.
type RPCInfo struct {
	Service string
	Method  string
}

// NewContext creates a context with RPC metadata.
// The info parameter should be a *tygor.RPCInfo.
func NewContext(ctx context.Context, w http.ResponseWriter, r *http.Request, info any) context.Context {
	ctx = context.WithValue(ctx, writerKey, w)
	ctx = context.WithValue(ctx, requestKey, r)
	ctx = context.WithValue(ctx, rpcInfoKey, info)
	return ctx
}

// RequestFromContext returns the HTTP request from the context.
func RequestFromContext(ctx context.Context) *http.Request {
	if r, ok := ctx.Value(requestKey).(*http.Request); ok {
		return r
	}
	return nil
}

// WriterFromContext returns the HTTP response writer from the context.
func WriterFromContext(ctx context.Context) http.ResponseWriter {
	if w, ok := ctx.Value(writerKey).(http.ResponseWriter); ok {
		return w
	}
	return nil
}

// InfoFromContext returns the RPC info from the context.
// The returned value should be cast to *tygor.RPCInfo.
func InfoFromContext(ctx context.Context) any {
	return ctx.Value(rpcInfoKey)
}
