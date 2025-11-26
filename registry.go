package tygor

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/broady/tygor/internal/meta"
)

// App is the central router for RPC handlers.
// It manages route registration, middleware, interceptors, and error handling.
// Use Handler() to get an http.Handler for use with http.ListenAndServe.
type App struct {
	mu                 sync.RWMutex
	routes             map[string]RPCMethod
	errorTransformer   ErrorTransformer
	maskInternalErrors bool
	interceptors       []UnaryInterceptor
	middlewares        []func(http.Handler) http.Handler
	logger             *slog.Logger
	maxRequestBodySize uint64
}

func NewApp() *App {
	return &App{
		routes:             make(map[string]RPCMethod),
		maxRequestBodySize: 1 << 20, // 1MB default
	}
}

// WithErrorTransformer adds a custom error transformer.
// It returns the app for chaining.
func (r *App) WithErrorTransformer(fn ErrorTransformer) *App {
	r.errorTransformer = fn
	return r
}

// WithMaskInternalErrors enables masking of internal error messages.
// This is useful in production to avoid leaking sensitive information.
// The original error is still available to interceptors and logging.
func (r *App) WithMaskInternalErrors() *App {
	r.maskInternalErrors = true
	return r
}

// WithUnaryInterceptor adds a global interceptor.
// Global interceptors are executed before service-level and handler-level interceptors.
//
// Interceptor execution order:
//  1. Global interceptors (added via App.WithUnaryInterceptor)
//  2. Service interceptors (added via Service.WithUnaryInterceptor)
//  3. Handler interceptors (added via Handler.WithUnaryInterceptor)
//  4. Handler function
//
// Within each level, interceptors execute in the order they were added.
func (r *App) WithUnaryInterceptor(i UnaryInterceptor) *App {
	r.interceptors = append(r.interceptors, i)
	return r
}

// WithMiddleware adds an HTTP middleware to wrap the app.
// Middleware is applied in the order added (first added is outermost).
func (r *App) WithMiddleware(mw func(http.Handler) http.Handler) *App {
	r.middlewares = append(r.middlewares, mw)
	return r
}

// WithLogger sets a custom logger for the app.
// If not set, slog.Default() will be used.
func (r *App) WithLogger(logger *slog.Logger) *App {
	r.logger = logger
	return r
}

// WithMaxRequestBodySize sets the default maximum request body size for all handlers.
// Individual handlers can override this with Handler.WithMaxRequestBodySize.
// A value of 0 means no limit. Default is 1MB (1 << 20).
func (r *App) WithMaxRequestBodySize(size uint64) *App {
	r.maxRequestBodySize = size
	return r
}

// Handler returns an http.Handler for use with http.ListenAndServe or other
// HTTP servers. The returned handler includes all configured middleware.
//
// Example:
//
//	app := tygor.NewApp().WithMiddleware(cors)
//	http.ListenAndServe(":8080", app.Handler())
func (r *App) Handler() http.Handler {
	var h http.Handler = http.HandlerFunc(r.serveHTTP)
	// Apply middleware in reverse order so first added is outermost
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}
	return h
}

// Service returns a Service namespace.
func (r *App) Service(name string) *Service {
	return &Service{
		registry: r,
		name:     name,
	}
}

// serveHTTP handles incoming RPC requests (internal, called via Handler()).
func (r *App) serveHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			stack := debug.Stack()
			logger := r.logger
			if logger == nil {
				logger = slog.Default()
			}
			logger.Error("PANIC recovered",
				slog.Any("panic", rec),
				slog.String("stack", string(stack)))
			writeError(w, NewError(CodeInternal, fmt.Sprintf("internal server error (panic): %v", rec)), r.logger)
		}
	}()

	path := strings.TrimPrefix(req.URL.Path, "/")
	// Path format: /{service_name}/{method_name}

	// Normalize path
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeError(w, NewError(CodeNotFound, "route not found"), r.logger)
		return
	}

	service, method := parts[0], parts[1]

	// We store keys as "Service.Method" internally to match the Manifest format
	// But the URL is /Service/Method.
	key := service + "." + method

	r.mu.RLock()
	handler, ok := r.routes[key]
	r.mu.RUnlock()

	if !ok {
		writeError(w, NewError(CodeNotFound, "route not found"), r.logger)
		return
	}

	// Type assert to internal handler interface
	h, ok := handler.(rpcHandler)
	if !ok {
		writeError(w, NewError(CodeInternal, "invalid handler type"), r.logger)
		return
	}

	// Check HTTP Method
	meta := h.metadata()
	if req.Method != meta.HTTPMethod {
		writeError(w, Errorf(CodeMethodNotAllowed, "method %s not allowed, expected %s", req.Method, meta.HTTPMethod), r.logger)
		return
	}

	// Create tygor Context with RPC metadata and config
	ctx := newContext(req.Context(), w, req, service, method)
	ctx.errorTransformer = r.errorTransformer
	ctx.maskInternalErrors = r.maskInternalErrors
	ctx.interceptors = r.interceptors
	ctx.logger = r.logger
	ctx.maxRequestBodySize = r.maxRequestBodySize

	// Execute handler
	h.serveHTTP(ctx)
}

type Service struct {
	registry     *App
	name         string
	interceptors []UnaryInterceptor
}

// WithUnaryInterceptor adds an interceptor to this service.
// Service interceptors execute after global interceptors but before handler interceptors.
// See App.WithUnaryInterceptor for the complete execution order.
func (s *Service) WithUnaryInterceptor(i UnaryInterceptor) *Service {
	s.interceptors = append(s.interceptors, i)
	return s
}

// Register registers a handler for the given operation name.
// If a handler is already registered for this service and method, it will be replaced
// and a warning will be logged.
func (s *Service) Register(name string, handler RPCMethod) {
	// Type assert to internal handler interface
	h, ok := handler.(rpcHandler)
	if !ok {
		panic("tygor: handler must be created with Unary() or UnaryGet()")
	}

	key := s.name + "." + name
	s.registry.mu.Lock()
	defer s.registry.mu.Unlock()

	// Check for duplicate registration
	if _, exists := s.registry.routes[key]; exists {
		logger := s.registry.logger
		if logger == nil {
			logger = slog.Default()
		}
		logger.Warn("duplicate route registration",
			slog.String("service", s.name),
			slog.String("method", name),
			slog.String("route", key))
	}

	// Wrap the handler to include service interceptors
	wrappedHandler := &serviceWrappedHandler{
		inner:        h,
		interceptors: s.interceptors,
	}

	s.registry.routes[key] = wrappedHandler
}

type serviceWrappedHandler struct {
	inner        rpcHandler
	interceptors []UnaryInterceptor
}

func (h *serviceWrappedHandler) IsRPCMethod() {}

func (h *serviceWrappedHandler) serveHTTP(ctx *Context) {
	// Combine: Global (ctx.interceptors) + Service (h.interceptors)
	combined := make([]UnaryInterceptor, 0, len(ctx.interceptors)+len(h.interceptors))
	combined = append(combined, ctx.interceptors...)
	combined = append(combined, h.interceptors...)

	// Update context with combined interceptors
	ctx.interceptors = combined

	h.inner.serveHTTP(ctx)
}

func (h *serviceWrappedHandler) metadata() *meta.MethodMetadata {
	return h.inner.metadata()
}
