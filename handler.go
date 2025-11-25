package tygor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	"github.com/broady/tygor/internal/meta"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
)

var (
	validate            = validator.New()
	schemaDecoder       = schema.NewDecoder() // lenient: ignores unknown keys
	strictSchemaDecoder = schema.NewDecoder() // strict: errors on unknown keys
)

func init() {
	schemaDecoder.IgnoreUnknownKeys(true)
	strictSchemaDecoder.IgnoreUnknownKeys(false)
}

// HandlerConfig contains configuration passed from Registry to handlers.
type HandlerConfig struct {
	ErrorTransformer   ErrorTransformer
	MaskInternalErrors bool
	Interceptors       []UnaryInterceptor
	Logger             *slog.Logger
	MaxRequestBodySize uint64
}

// RPCMethod is the interface for registered handlers.
// It is exported so users can pass it to Register, but sealed so they cannot implement it.
type RPCMethod interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, config HandlerConfig)
	Metadata() *meta.MethodMetadata
}

// UnaryHandler contains common configuration for all unary handlers.
//
// This is a base type. See UnaryPostHandler and UnaryGetHandler for specific implementations.
type UnaryHandler[Req any, Res any] struct {
	fn             func(context.Context, Req) (Res, error)
	httpMethod     string
	interceptors   []UnaryInterceptor
	skipValidation bool
}

// UnaryPostHandler implements RPCMethod for POST requests (state-changing operations).
//
// It embeds UnaryHandler, inheriting methods like WithUnaryInterceptor and WithSkipValidation.
//
// Request Type Guidelines:
//   - Use struct or pointer types
//   - Request is decoded from JSON body
//
// Example:
//
//	func CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) { ... }
//	Unary(CreateUser)
//
//	func UpdatePost(ctx context.Context, req *UpdatePostRequest) (*Post, error) { ... }
//	Unary(UpdatePost).WithUnaryInterceptor(requireAuth)
type UnaryPostHandler[Req any, Res any] struct {
	UnaryHandler[Req, Res]
	maxRequestBodySize *uint64 // nil means use registry default
}

// Unary creates a new POST handler from a generic function for unary (non-streaming) RPCs.
//
// The handler function signature is func(context.Context, Req) (Res, error).
// Requests are decoded from JSON body.
//
// For GET requests (cacheable reads), use UnaryGet instead.
//
// The returned UnaryPostHandler supports:
//   - WithUnaryInterceptor (from UnaryHandler)
//   - WithSkipValidation (from UnaryHandler)
//   - WithMaxRequestBodySize (specific to POST)
func Unary[Req any, Res any](fn func(context.Context, Req) (Res, error)) *UnaryPostHandler[Req, Res] {
	return &UnaryPostHandler[Req, Res]{
		UnaryHandler: UnaryHandler[Req, Res]{
			fn:         fn,
			httpMethod: "POST",
		},
	}
}

// UnaryGetHandler implements RPCMethod for GET requests (cacheable read operations).
//
// It embeds UnaryHandler, inheriting methods like WithUnaryInterceptor and WithSkipValidation.
//
// Request Type Guidelines:
//   - Use struct types for simple cases, pointer types when you need optional fields
//   - Request parameters are decoded from URL query string
//
// Struct vs Pointer Types:
//   - Struct types (e.g., ListParams): Query parameters are decoded directly into the struct
//   - Pointer types (e.g., *ListParams): A new instance is created and query parameters are decoded into it
//
// Example:
//
//	func ListPosts(ctx context.Context, req ListPostsParams) ([]*Post, error) { ... }
//	UnaryGet(ListPosts).CacheControl(tygor.CacheConfig{
//	    MaxAge: 5 * time.Minute,
//	    Public: true,
//	})
type UnaryGetHandler[Req any, Res any] struct {
	UnaryHandler[Req, Res]
	cacheConfig       *CacheConfig
	strictQueryParams bool
}

// CacheConfig defines HTTP cache directives for GET requests.
// See RFC 9111 (HTTP Caching) for detailed semantics.
//
// Common patterns:
//   - Simple caching: CacheConfig{MaxAge: 5*time.Minute}
//   - Public CDN caching: CacheConfig{MaxAge: 5*time.Minute, Public: true}
//   - Stale-while-revalidate: CacheConfig{MaxAge: 1*time.Minute, StaleWhileRevalidate: 5*time.Minute}
//   - Immutable assets: CacheConfig{MaxAge: 365*24*time.Hour, Immutable: true}
type CacheConfig struct {
	// MaxAge specifies the maximum time a resource is considered fresh (RFC 9111 Section 5.2.2.1).
	// After this time, caches must revalidate before serving the cached response.
	MaxAge time.Duration

	// SMaxAge is like MaxAge but only applies to shared caches like CDNs (RFC 9111 Section 5.2.2.10).
	// Overrides MaxAge for shared caches. Private caches ignore this directive.
	SMaxAge time.Duration

	// StaleWhileRevalidate allows serving stale content while revalidating in the background (RFC 5861).
	// Example: MaxAge=60s, StaleWhileRevalidate=300s means serve from cache for 60s,
	// then serve stale content for up to 300s more while fetching fresh data in background.
	StaleWhileRevalidate time.Duration

	// StaleIfError allows serving stale content if the origin server is unavailable (RFC 5861).
	// Example: StaleIfError=86400 allows serving day-old stale content if origin returns 5xx errors.
	StaleIfError time.Duration

	// Public indicates the response may be cached by any cache, including CDNs (RFC 9111 Section 5.2.2.9).
	// Default is false (private), meaning only the user's browser cache may store it.
	// Set to true for responses that are safe to cache publicly.
	Public bool

	// MustRevalidate requires caches to revalidate stale responses with the origin before serving (RFC 9111 Section 5.2.2.2).
	// Prevents serving stale content. Useful when stale data could cause problems.
	MustRevalidate bool

	// Immutable indicates the response will never change during its freshness lifetime (RFC 8246).
	// Browsers won't send conditional requests for immutable resources within MaxAge period.
	// Useful for content-addressed assets like "bundle.abc123.js".
	Immutable bool
}

// UnaryGet creates a new GET handler from a generic function for cacheable read operations.
//
// The handler function signature is func(context.Context, Req) (Res, error).
// Requests are decoded from URL query parameters.
//
// Use CacheControl() to configure HTTP caching behavior.
//
// The returned UnaryGetHandler supports:
//   - WithUnaryInterceptor (from UnaryHandler)
//   - WithSkipValidation (from UnaryHandler)
//   - CacheControl (specific to GET)
//   - WithStrictQueryParams (specific to GET)
func UnaryGet[Req any, Res any](fn func(context.Context, Req) (Res, error)) *UnaryGetHandler[Req, Res] {
	return &UnaryGetHandler[Req, Res]{
		UnaryHandler: UnaryHandler[Req, Res]{
			fn:         fn,
			httpMethod: "GET",
		},
	}
}

// CacheControl sets detailed HTTP cache directives for the handler.
// See CacheConfig documentation and RFC 9111 for directive semantics.
//
// Example:
//
//	UnaryGet(ListPosts).CacheControl(tygor.CacheConfig{
//	    MaxAge:               5 * time.Minute,
//	    StaleWhileRevalidate: 1 * time.Minute,
//	    Public:               true,
//	}).WithUnaryInterceptor(...)
//	// Sets: Cache-Control: public, max-age=300, stale-while-revalidate=60
func (h *UnaryGetHandler[Req, Res]) CacheControl(cfg CacheConfig) *UnaryGetHandler[Req, Res] {
	h.cacheConfig = &cfg
	return h
}

// WithUnaryInterceptor adds an interceptor to this handler.
// Handler interceptors execute after global and service interceptors.
// See Registry.WithUnaryInterceptor for the complete execution order.
func (h *UnaryPostHandler[Req, Res]) WithUnaryInterceptor(i UnaryInterceptor) *UnaryPostHandler[Req, Res] {
	h.interceptors = append(h.interceptors, i)
	return h
}

// WithUnaryInterceptor adds an interceptor to this handler.
// Handler interceptors execute after global and service interceptors.
// See Registry.WithUnaryInterceptor for the complete execution order.
func (h *UnaryGetHandler[Req, Res]) WithUnaryInterceptor(i UnaryInterceptor) *UnaryGetHandler[Req, Res] {
	h.interceptors = append(h.interceptors, i)
	return h
}

// WithSkipValidation disables validation for this handler.
// By default, all handlers validate requests using the validator package.
// Use this when you need to handle validation manually or when the request
// type has no validation tags.
func (h *UnaryPostHandler[Req, Res]) WithSkipValidation() *UnaryPostHandler[Req, Res] {
	h.skipValidation = true
	return h
}

// WithSkipValidation disables validation for this handler.
// By default, all handlers validate requests using the validator package.
// Use this when you need to handle validation manually or when the request
// type has no validation tags.
func (h *UnaryGetHandler[Req, Res]) WithSkipValidation() *UnaryGetHandler[Req, Res] {
	h.skipValidation = true
	return h
}

// WithStrictQueryParams enables strict query parameter validation for GET requests.
// By default, unknown query parameters are ignored (lenient mode).
// When enabled, requests with unknown query parameters will return an error.
// This helps catch typos and enforces exact parameter expectations.
func (h *UnaryGetHandler[Req, Res]) WithStrictQueryParams() *UnaryGetHandler[Req, Res] {
	h.strictQueryParams = true
	return h
}

// WithMaxRequestBodySize sets the maximum request body size for this handler.
// This overrides the registry-level default. A value of 0 means no limit.
func (h *UnaryPostHandler[Req, Res]) WithMaxRequestBodySize(size uint64) *UnaryPostHandler[Req, Res] {
	h.maxRequestBodySize = &size
	return h
}

// Metadata returns the runtime metadata for the handler.
func (h *UnaryHandler[Req, Res]) Metadata() *meta.MethodMetadata {
	var req Req
	var res Res
	return &meta.MethodMetadata{
		HTTPMethod: h.httpMethod,
		Request:    reflect.TypeOf(req),
		Response:   reflect.TypeOf(res),
		CacheTTL:   0,
	}
}

// Metadata returns the runtime metadata for the GET handler.
func (h *UnaryGetHandler[Req, Res]) Metadata() *meta.MethodMetadata {
	meta := h.UnaryHandler.Metadata()
	if h.cacheConfig != nil {
		meta.CacheTTL = h.cacheConfig.MaxAge
	}
	return meta
}

// Metadata returns the runtime metadata for the POST handler.
func (h *UnaryPostHandler[Req, Res]) Metadata() *meta.MethodMetadata {
	return h.UnaryHandler.Metadata()
}

// getCacheControlHeader returns the Cache-Control header value.
// Handler returns empty (no caching for POST requests).
func (h *UnaryHandler[Req, Res]) getCacheControlHeader() string {
	return ""
}

// getCacheControlHeader builds the Cache-Control header value from the cache config.
// Returns empty string if no cache config is set.
func (h *UnaryGetHandler[Req, Res]) getCacheControlHeader() string {
	if h.cacheConfig == nil {
		return ""
	}

	cfg := h.cacheConfig
	var parts []string

	// Visibility directive
	if cfg.Public {
		parts = append(parts, "public")
	} else {
		parts = append(parts, "private")
	}

	// max-age (required if any caching is configured)
	if cfg.MaxAge > 0 {
		parts = append(parts, fmt.Sprintf("max-age=%d", int(cfg.MaxAge.Seconds())))
	}

	// s-maxage (shared cache specific)
	if cfg.SMaxAge > 0 {
		parts = append(parts, fmt.Sprintf("s-maxage=%d", int(cfg.SMaxAge.Seconds())))
	}

	// stale-while-revalidate (RFC 5861)
	if cfg.StaleWhileRevalidate > 0 {
		parts = append(parts, fmt.Sprintf("stale-while-revalidate=%d", int(cfg.StaleWhileRevalidate.Seconds())))
	}

	// stale-if-error (RFC 5861)
	if cfg.StaleIfError > 0 {
		parts = append(parts, fmt.Sprintf("stale-if-error=%d", int(cfg.StaleIfError.Seconds())))
	}

	// must-revalidate
	if cfg.MustRevalidate {
		parts = append(parts, "must-revalidate")
	}

	// immutable (RFC 8246)
	if cfg.Immutable {
		parts = append(parts, "immutable")
	}

	if len(parts) == 0 {
		return ""
	}

	// Join with ", "
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	return result
}

// ServeHTTP implements the RPC handler for GET requests with caching support.
func (h *UnaryGetHandler[Req, Res]) ServeHTTP(w http.ResponseWriter, r *http.Request, config HandlerConfig) {
	decoder := func() (Req, error) {
		var req Req
		// Select decoder based on strictness setting
		decoder := schemaDecoder
		if h.strictQueryParams {
			decoder = strictSchemaDecoder
		}

		reqType := reflect.TypeOf(req)
		if reqType.Kind() == reflect.Pointer {
			// Instantiate the element
			val := reflect.New(reqType.Elem())
			// val is *Elem.
			// Decode into it
			if err := decoder.Decode(val.Interface(), r.URL.Query()); err != nil {
				return req, Errorf(CodeInvalidArgument, "failed to decode query: %v", err)
			}
			req = val.Interface().(Req)
		} else {
			// Req is a struct. &req is *Req.
			if err := decoder.Decode(&req, r.URL.Query()); err != nil {
				return req, Errorf(CodeInvalidArgument, "failed to decode query: %v", err)
			}
		}
		return req, nil
	}
	h.serve(w, r, config, h.getCacheControlHeader(), decoder)
}

// ServeHTTP implements the RPC handler for POST requests.
func (h *UnaryPostHandler[Req, Res]) ServeHTTP(w http.ResponseWriter, r *http.Request, config HandlerConfig) {
	decoder := func() (Req, error) {
		var req Req
		if r.Body != nil {
			// Determine effective body size limit
			effectiveLimit := config.MaxRequestBodySize
			if h.maxRequestBodySize != nil {
				effectiveLimit = *h.maxRequestBodySize
			}

			// Apply body size limit if > 0
			// 0 means unlimited for backwards compatibility
			if effectiveLimit > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, int64(effectiveLimit))
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				return req, Errorf(CodeInvalidArgument, "failed to decode body: %v", err)
			}
		}
		return req, nil
	}
	h.serve(w, r, config, "", decoder)
}

// serve implements the generic glue code for both UnaryPostHandler and UnaryGetHandler.
func (h *UnaryHandler[Req, Res]) serve(w http.ResponseWriter, r *http.Request, config HandlerConfig, cacheControl string, decodeFunc func() (Req, error)) {
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
	allInterceptors := make([]UnaryInterceptor, 0, len(config.Interceptors)+len(h.interceptors))
	allInterceptors = append(allInterceptors, config.Interceptors...)
	allInterceptors = append(allInterceptors, h.interceptors...)

	chain := chainInterceptors(allInterceptors)

	// 3. Decode Request
	req, decodeErr := func() (Req, error) {
		req, err := decodeFunc()
		if err != nil {
			return req, err
		}

		if !h.skipValidation {
			if err := validate.Struct(req); err != nil {
				return req, err
			}
		}
		return req, nil
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
	if cacheControl != "" {
		w.Header().Set("Cache-Control", cacheControl)
	}

	if err := encodeResponse(w, res); err != nil {
		// Response may be partially written, nothing we can do. Log for debugging.
		logger := config.Logger
		if logger == nil {
			logger = slog.Default()
		}
		logger.Error("failed to encode response",
			slog.String("service", service),
			slog.String("method", method),
			slog.Any("error", err))
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
	writeError(w, rpcErr, config.Logger)
}
