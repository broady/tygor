package tygor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/broady/tygor/internal/meta"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
)

var (
	validate      = validator.New()
	schemaDecoder = schema.NewDecoder()
)

func init() {
	schemaDecoder.IgnoreUnknownKeys(true)
}

// HandlerConfig contains configuration passed from Registry to handlers.
type HandlerConfig struct {
	ErrorTransformer   ErrorTransformer
	MaskInternalErrors bool
	Interceptors       []Interceptor
}

// RPCMethod is the interface for registered handlers.
// It is exported so users can pass it to Register, but sealed so they cannot implement it.
type RPCMethod interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, config HandlerConfig)
	Metadata() *meta.MethodMetadata
}

// Handler implements RPCMethod for a specific Request/Response pair.
type Handler[Req any, Res any] struct {
	fn           func(context.Context, Req) (Res, error)
	method       string
	cacheTTL     time.Duration
	interceptors []Interceptor
}

// NewHandler creates a new handler from a generic function.
// Default HTTP Method is "POST".
func NewHandler[Req any, Res any](fn func(context.Context, Req) (Res, error)) *Handler[Req, Res] {
	return &Handler[Req, Res]{
		fn:     fn,
		method: "POST",
	}
}

// Method sets the HTTP method (e.g., "GET", "POST").
func (h *Handler[Req, Res]) Method(m string) *Handler[Req, Res] {
	h.method = m
	return h
}

// Cache sets the cache TTL for the handler.
func (h *Handler[Req, Res]) Cache(d time.Duration) *Handler[Req, Res] {
	h.cacheTTL = d
	return h
}

// WithInterceptor adds an interceptor to this handler.
func (h *Handler[Req, Res]) WithInterceptor(i Interceptor) *Handler[Req, Res] {
	h.interceptors = append(h.interceptors, i)
	return h
}

// Metadata returns the runtime metadata for the handler.
func (h *Handler[Req, Res]) Metadata() *meta.MethodMetadata {
	var req Req
	var res Res
	return &meta.MethodMetadata{
		Method:   h.method,
		Request:  reflect.TypeOf(req),
		Response: reflect.TypeOf(res),
		CacheTTL: h.cacheTTL,
	}
}

// ServeHTTP implements the generic glue code.
func (h *Handler[Req, Res]) ServeHTTP(w http.ResponseWriter, r *http.Request, config HandlerConfig) {
	// 1. Prepare Context & Info
	ctx := r.Context()
	// MethodFromContext is already set by Registry.
	service, method, _ := MethodFromContext(ctx)
	info := &RPCInfo{
		Service: service,
		Method:  method,
	}

	// 2. Combine Interceptors
	// Config contains: Global + Service interceptors
	// We append Handler-level interceptors
	allInterceptors := make([]Interceptor, 0, len(config.Interceptors)+len(h.interceptors))
	allInterceptors = append(allInterceptors, config.Interceptors...)
	allInterceptors = append(allInterceptors, h.interceptors...)

	chain := chainInterceptors(allInterceptors)

	// 3. Decode Request
	// We must decode first because the interceptor expects a typed struct pointer.
	var req Req

	// Decoding logic
	decodeErr := func() error {
		if h.method == "GET" {
			// schemaDecoder.Decode requires a pointer to a struct.
			// &req is a *Req.
			// If Req is already a pointer type (e.g. *ListNewsParams), then req is *ListNewsParams.
			// &req is **ListNewsParams.
			// schemaDecoder wants interface{} that is a pointer to a struct.
			// If Req is a struct (ListNewsParams), &req is *ListNewsParams. Correct.
			// If Req is a pointer (*ListNewsParams), &req is **ListNewsParams. Incorrect?

			// We need to detect if Req is a pointer type.
			// But we can't do that easily at runtime with generics without reflection on the value?
			// Actually, we can check h.Metadata().Request.Kind().

			// But schema/decoder wants the struct pointer.
			// If Req is *T, we need to allocate T and pass &T to Decode.
			// Then set req = &T.

			// Wait, in standard usage: func(ctx, req *ListNewsParams). Req is *ListNewsParams.
			// So req is *ListNewsParams (which is nil initially).
			// We need to initialize it?
			// var req Req -> nil.

			// We need to instantiate the underlying struct.
			reqType := reflect.TypeOf(req)
			if reqType.Kind() == reflect.Ptr {
				// Instantiate the element
				val := reflect.New(reqType.Elem())
				// val is *Elem.
				// Decode into it
				if err := schemaDecoder.Decode(val.Interface(), r.URL.Query()); err != nil {
					return Errorf(CodeInvalidArgument, "failed to decode query: %v", err)
				}
				// req = val.Interface().(Req) ??
				req = val.Interface().(Req)
			} else {
				// Req is a struct. &req is *Req.
				if err := schemaDecoder.Decode(&req, r.URL.Query()); err != nil {
					return Errorf(CodeInvalidArgument, "failed to decode query: %v", err)
				}
			}
		} else {
			if r.Body != nil {
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					return Errorf(CodeInvalidArgument, "failed to decode body: %v", err)
				}
			}
		}

		if err := validate.Struct(req); err != nil {
			return err
		}
		return nil
	}()

	if decodeErr != nil {
		handleError(w, decodeErr, config)
		return
	}

	// 4. Execute Chain
	// The chain eventually calls the user function.

	finalHandler := func(ctx context.Context, reqAny any) (any, error) {
		// Type assertion should be safe here because we only pass 'req' (type Req) into the chain.
		reqTyped, ok := reqAny.(Req)
		if !ok {
			return nil, Errorf(CodeInternal, "interceptor modified request type incorrectly")
		}
		return h.fn(ctx, reqTyped)
	}

	var res any
	var err error

	if chain != nil {
		res, err = chain(ctx, req, info, finalHandler)
	} else {
		res, err = finalHandler(ctx, req)
	}

	if err != nil {
		handleError(w, err, config)
		return
	}

	// 5. Write Response
	w.Header().Set("Content-Type", "application/json")
	if h.cacheTTL > 0 {
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(h.cacheTTL.Seconds())))
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		// Response may be partially written, nothing we can do. Log for debugging.
		fmt.Fprintf(os.Stderr, "FATAL: failed to encode response: %v\n", err)
	}
}

func handleError(w http.ResponseWriter, err error, config HandlerConfig) {
	var rpcErr *Error
	if config.ErrorTransformer != nil {
		rpcErr = config.ErrorTransformer(err)
	}
	if rpcErr == nil {
		rpcErr = DefaultErrorTransformer(err)
	}
	if config.MaskInternalErrors && rpcErr.Code == CodeInternal {
		rpcErr.Message = "internal server error"
	}
	writeError(w, rpcErr)
}
