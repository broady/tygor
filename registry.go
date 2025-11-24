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

type Registry struct {
	mu                 sync.RWMutex
	routes             map[string]RPCMethod
	errorTransformer   ErrorTransformer
	maskInternalErrors bool
	interceptors       []UnaryInterceptor
	middlewares        []func(http.Handler) http.Handler
	logger             *slog.Logger
	maxRequestBodySize uint64
}

func NewRegistry() *Registry {
	return &Registry{
		routes:             make(map[string]RPCMethod),
		maxRequestBodySize: 1 << 20, // 1MB default
	}
}

// WithErrorTransformer adds a custom error transformer.
// It returns the registry for chaining.
func (r *Registry) WithErrorTransformer(fn ErrorTransformer) *Registry {
	r.errorTransformer = fn
	return r
}

// WithMaskInternalErrors enables masking of internal error messages.
// This is useful in production to avoid leaking sensitive information.
// The original error is still available to interceptors and logging.
func (r *Registry) WithMaskInternalErrors() *Registry {
	r.maskInternalErrors = true
	return r
}

// WithUnaryInterceptor adds a global interceptor.
// Global interceptors are executed before service-level and handler-level interceptors.
//
// Interceptor execution order:
//  1. Global interceptors (added via Registry.WithUnaryInterceptor)
//  2. Service interceptors (added via Service.WithUnaryInterceptor)
//  3. Handler interceptors (added via Handler.WithUnaryInterceptor)
//  4. Handler function
//
// Within each level, interceptors execute in the order they were added.
func (r *Registry) WithUnaryInterceptor(i UnaryInterceptor) *Registry {
	r.interceptors = append(r.interceptors, i)
	return r
}

// WithMiddleware adds an HTTP middleware to wrap the registry.
// Middleware is applied in the order added (first added is outermost).
// Use Handler() to get the wrapped handler.
func (r *Registry) WithMiddleware(mw func(http.Handler) http.Handler) *Registry {
	r.middlewares = append(r.middlewares, mw)
	return r
}

// WithLogger sets a custom logger for the registry.
// If not set, slog.Default() will be used.
func (r *Registry) WithLogger(logger *slog.Logger) *Registry {
	r.logger = logger
	return r
}

// WithMaxRequestBodySize sets the default maximum request body size for all handlers.
// Individual handlers can override this with Handler.WithMaxRequestBodySize.
// A value of 0 means no limit. Default is 1MB (1 << 20).
func (r *Registry) WithMaxRequestBodySize(size uint64) *Registry {
	r.maxRequestBodySize = size
	return r
}

// Handler returns the registry wrapped with all configured middleware.
// The middleware is applied in the order it was added via WithMiddleware.
func (r *Registry) Handler() http.Handler {
	var h http.Handler = http.HandlerFunc(r.ServeHTTP)
	// Apply middleware in reverse order so first added is outermost
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}
	return h
}

// Service returns a Service namespace.
func (r *Registry) Service(name string) *Service {
	return &Service{
		registry: r,
		name:     name,
	}
}

// ServeHTTP implements http.Handler.
func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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

	// Inject Writer/Request into Context for SetHeader helper
	ctx := newContext(req.Context(), w, req, nil) // rpcInfo is not known yet
	req = req.WithContext(ctx)

	path := strings.TrimPrefix(req.URL.Path, "/")
	// Path format: Service/Method or Service.Method?
	// Spec says: /{service_name}/{method_name}

	// Normalize path
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeError(w, NewError(CodeNotFound, "route not found"), r.logger)
		return
	}

	// We store keys as "Service.Method" internally to match the Manifest format
	// But the URL is /Service/Method.
	key := parts[0] + "." + parts[1]

	r.mu.RLock()
	handler, ok := r.routes[key]
	r.mu.RUnlock()

	if !ok {
		writeError(w, NewError(CodeNotFound, "route not found"), r.logger)
		return
	}

	// Check HTTP Method
	meta := handler.Metadata()
	if req.Method != meta.Method {
		writeError(w, Errorf(CodeMethodNotAllowed, "method %s not allowed, expected %s", req.Method, meta.Method), r.logger)
		return
	}

	// Update context with RPC Info
	info := &RPCInfo{Service: parts[0], Method: parts[1]}
	ctx = newContext(ctx, w, req, info)
	req = req.WithContext(ctx)

	// Build config with registry-level settings
	config := HandlerConfig{
		ErrorTransformer:   r.errorTransformer,
		MaskInternalErrors: r.maskInternalErrors,
		Interceptors:       r.interceptors,
		Logger:             r.logger,
		MaxRequestBodySize: r.maxRequestBodySize,
	}

	// Execute handler
	handler.ServeHTTP(w, req, config)
}

type Service struct {
	registry     *Registry
	name         string
	interceptors []UnaryInterceptor
}

// WithUnaryInterceptor adds an interceptor to this service.
// Service interceptors execute after global interceptors but before handler interceptors.
// See Registry.WithUnaryInterceptor for the complete execution order.
func (s *Service) WithUnaryInterceptor(i UnaryInterceptor) *Service {
	s.interceptors = append(s.interceptors, i)
	return s
}

// Register registers a handler for the given operation name.
// If a handler is already registered for this service and method, it will be replaced
// and a warning will be logged.
func (s *Service) Register(name string, handler RPCMethod) {
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

	// We need to wrap the handler or somehow attach the service interceptors?
	// The handler.ServeHTTP takes a list of prefix interceptors.
	// But `Register` is called ONCE. `ServeHTTP` is called MANY times.
	// The `prefixInterceptors` passed to `handler.ServeHTTP` in `Registry.ServeHTTP`
	// are the Global ones.
	// We need to include the Service ones too.
	// But Registry doesn't store Service objects, it stores routes map[string]RPCMethod.
	// We lose the Service object after registration.
	// So we must Wrap the RPCMethod to include the Service interceptors.

	wrappedHandler := &serviceWrappedHandler{
		inner:        handler,
		interceptors: s.interceptors,
	}

	s.registry.routes[key] = wrappedHandler
}

type serviceWrappedHandler struct {
	inner        RPCMethod
	interceptors []UnaryInterceptor
}

func (h *serviceWrappedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, config HandlerConfig) {
	// Combine: Global (config.Interceptors) + Service (h.interceptors)
	combined := make([]UnaryInterceptor, 0, len(config.Interceptors)+len(h.interceptors))
	combined = append(combined, config.Interceptors...)
	combined = append(combined, h.interceptors...)

	// Build new config with combined interceptors
	serviceConfig := HandlerConfig{
		ErrorTransformer:   config.ErrorTransformer,
		MaskInternalErrors: config.MaskInternalErrors,
		Interceptors:       combined,
		Logger:             config.Logger,
	}

	h.inner.ServeHTTP(w, r, serviceConfig)
}

func (h *serviceWrappedHandler) Metadata() *meta.MethodMetadata {
	return h.inner.Metadata()
}
