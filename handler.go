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
	MaxBodySize        int64
}

// RPCMethod is the interface for registered handlers.
// It is exported so users can pass it to Register, but sealed so they cannot implement it.
type RPCMethod interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, config HandlerConfig)
	Metadata() *meta.MethodMetadata
}

// Handler implements RPCMethod for POST requests (state-changing operations).
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
type Handler[Req any, Res any] struct {
	fn                func(context.Context, Req) (Res, error)
	method            string
	interceptors      []UnaryInterceptor
	skipValidation    bool
	strictQueryParams bool
	maxBodySize       *int64 // nil means use registry default
}

// Unary creates a new POST handler from a generic function for unary (non-streaming) RPCs.
//
// The handler function signature is func(context.Context, Req) (Res, error).
// Requests are decoded from JSON body.
//
// For GET requests (cacheable reads), use UnaryGet instead.
func Unary[Req any, Res any](fn func(context.Context, Req) (Res, error)) *Handler[Req, Res] {
	return &Handler[Req, Res]{
		fn:     fn,
		method: "POST",
	}
}

// GetHandler implements RPCMethod for GET requests (cacheable read operations).
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
//	UnaryGet(ListPosts).Cache(5 * time.Minute)
//
//	func GetPost(ctx context.Context, req *GetPostParams) (*Post, error) { ... }
//	UnaryGet(GetPost).CacheControl(tygor.CacheConfig{
//	    MaxAge: 5 * time.Minute,
//	    StaleWhileRevalidate: 1 * time.Minute,
//	    Public: true,
//	})
type GetHandler[Req any, Res any] struct {
	Handler[Req, Res]
	cacheConfig *CacheConfig
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
// Use Cache() or CacheControl() to configure HTTP caching behavior.
func UnaryGet[Req any, Res any](fn func(context.Context, Req) (Res, error)) *GetHandler[Req, Res] {
	return &GetHandler[Req, Res]{
		Handler: Handler[Req, Res]{
			fn:     fn,
			method: "GET",
		},
	}
}

// Cache sets a simple max-age cache TTL for the handler.
// This is a convenience method equivalent to CacheControl(CacheConfig{MaxAge: d}).
//
// Example:
//
//	UnaryGet(ListPosts).Cache(5 * time.Minute).WithUnaryInterceptor(...)
//	// Sets: Cache-Control: private, max-age=300
func (h *GetHandler[Req, Res]) Cache(d time.Duration) *GetHandler[Req, Res] {
	h.cacheConfig = &CacheConfig{MaxAge: d}
	return h
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
func (h *GetHandler[Req, Res]) CacheControl(cfg CacheConfig) *GetHandler[Req, Res] {
	h.cacheConfig = &cfg
	return h
}

// WithUnaryInterceptor adds an interceptor to this handler.
// Handler interceptors execute after global and service interceptors.
// See Registry.WithUnaryInterceptor for the complete execution order.
func (h *Handler[Req, Res]) WithUnaryInterceptor(i UnaryInterceptor) *Handler[Req, Res] {
	h.interceptors = append(h.interceptors, i)
	return h
}

// WithSkipValidation disables validation for this handler.
// By default, all handlers validate requests using the validator package.
// Use this when you need to handle validation manually or when the request
// type has no validation tags.
func (h *Handler[Req, Res]) WithSkipValidation() *Handler[Req, Res] {
	h.skipValidation = true
	return h
}

// WithStrictQueryParams enables strict query parameter validation for GET requests.
// By default, unknown query parameters are ignored (lenient mode).
// When enabled, requests with unknown query parameters will return an error.
// This helps catch typos and enforces exact parameter expectations.
// Only affects GET requests; has no effect on POST/PUT/PATCH requests.
func (h *Handler[Req, Res]) WithStrictQueryParams() *Handler[Req, Res] {
	h.strictQueryParams = true
	return h
}

// WithMaxBodySize sets the maximum request body size for this handler.
// This overrides the registry-level default.
// A value of 0 means no limit. Negative values are invalid and will be ignored.
func (h *Handler[Req, Res]) WithMaxBodySize(size int64) *Handler[Req, Res] {
	if size >= 0 {
		h.maxBodySize = &size
	}
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
		CacheTTL: 0,
	}
}

// Metadata returns the runtime metadata for the GET handler.
func (h *GetHandler[Req, Res]) Metadata() *meta.MethodMetadata {
	meta := h.Handler.Metadata()
	if h.cacheConfig != nil {
		meta.CacheTTL = h.cacheConfig.MaxAge
	}
	return meta
}

// getCacheControlHeader returns the Cache-Control header value.
// Handler returns empty (no caching for POST requests).
func (h *Handler[Req, Res]) getCacheControlHeader() string {
	return ""
}

// getCacheControlHeader builds the Cache-Control header value from the cache config.
// Returns empty string if no cache config is set.
func (h *GetHandler[Req, Res]) getCacheControlHeader() string {
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
func (h *GetHandler[Req, Res]) ServeHTTP(w http.ResponseWriter, r *http.Request, config HandlerConfig) {
	h.serveHTTPWithCache(w, r, config, h.getCacheControlHeader())
}

// ServeHTTP implements the RPC handler for POST requests.
func (h *Handler[Req, Res]) ServeHTTP(w http.ResponseWriter, r *http.Request, config HandlerConfig) {
	h.serveHTTPWithCache(w, r, config, "")
}

// serveHTTPWithCache implements the generic glue code for both Handler and GetHandler.
func (h *Handler[Req, Res]) serveHTTPWithCache(w http.ResponseWriter, r *http.Request, config HandlerConfig, cacheControl string) {
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
	// We must decode first because the interceptor expects a typed struct pointer.
	var req Req

	// Decoding logic
	decodeErr := func() error {
		if h.method == "GET" {
			// Select decoder based on strictness setting
			decoder := schemaDecoder
			if h.strictQueryParams {
				decoder = strictSchemaDecoder
			}

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
			if reqType.Kind() == reflect.Pointer {
				// Instantiate the element
				val := reflect.New(reqType.Elem())
				// val is *Elem.
				// Decode into it
				if err := decoder.Decode(val.Interface(), r.URL.Query()); err != nil {
					return Errorf(CodeInvalidArgument, "failed to decode query: %v", err)
				}
				// req = val.Interface().(Req) ??
				req = val.Interface().(Req)
			} else {
				// Req is a struct. &req is *Req.
				if err := decoder.Decode(&req, r.URL.Query()); err != nil {
					return Errorf(CodeInvalidArgument, "failed to decode query: %v", err)
				}
			}
		} else {
			if r.Body != nil {
				// Determine effective body size limit
				effectiveLimit := config.MaxBodySize
				if h.maxBodySize != nil {
					effectiveLimit = *h.maxBodySize
				}

				// Apply body size limit if > 0
				// 0 means unlimited for backwards compatibility
				if effectiveLimit > 0 {
					r.Body = http.MaxBytesReader(w, r.Body, effectiveLimit)
				}

				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					return Errorf(CodeInvalidArgument, "failed to decode body: %v", err)
				}
			}
		}

		if !h.skipValidation {
			if err := validate.Struct(req); err != nil {
				return err
			}
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
	if cacheControl != "" {
		w.Header().Set("Cache-Control", cacheControl)
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
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
