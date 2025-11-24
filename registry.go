package tygor

import (
	"net/http"
	"strings"
	"sync"

	"github.com/broady/tygor/internal/meta"
)

type Registry struct {
	mu               sync.RWMutex
	routes           map[string]RPCMethod
	errorTransformer ErrorTransformer
	interceptors     []Interceptor
}

func NewRegistry() *Registry {
	return &Registry{
		routes: make(map[string]RPCMethod),
	}
}

// WithErrorTransformer adds a custom error transformer.
// It returns the registry for chaining.
func (r *Registry) WithErrorTransformer(fn ErrorTransformer) *Registry {
	r.errorTransformer = fn
	return r
}

// WithInterceptor adds a global interceptor.
func (r *Registry) WithInterceptor(i Interceptor) *Registry {
	r.interceptors = append(r.interceptors, i)
	return r
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
			writeError(w, NewError(CodeInternal, "internal server error (panic)"))
		}
	}()

	// Inject Writer/Request into Context for SetHeader helper
	ctx := newContext(req.Context(), w, req, nil) // rpcInfo is not known yet
	req = req.WithContext(ctx)

	// Basic CORS support for development (optional, but good to have)
	if req.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/")
	// Path format: Service/Method or Service.Method?
	// Spec says: /{service_name}/{method_name}

	// Normalize path
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeError(w, NewError(CodeNotFound, "route not found"))
		return
	}

	// We store keys as "Service.Method" internally to match the Manifest format
	// But the URL is /Service/Method.
	key := parts[0] + "." + parts[1]

	r.mu.RLock()
	handler, ok := r.routes[key]
	r.mu.RUnlock()

	if !ok {
		writeError(w, NewError(CodeNotFound, "route not found"))
		return
	}

	// Check HTTP Method
	meta := handler.Metadata()
	if req.Method != meta.Method {
		writeError(w, Errorf(CodeInvalidArgument, "method %s not allowed, expected %s", req.Method, meta.Method))
		return
	}

	// Update context with RPC Info
	info := &RPCInfo{Service: parts[0], Method: parts[1]}
	ctx = newContext(ctx, w, req, info)
	req = req.WithContext(ctx)

	// Execute handler with Global Interceptors
	handler.ServeHTTP(w, req, r.errorTransformer, r.interceptors)
}

type Service struct {
	registry     *Registry
	name         string
	interceptors []Interceptor
}

// WithInterceptor adds an interceptor to this service.
func (s *Service) WithInterceptor(i Interceptor) *Service {
	s.interceptors = append(s.interceptors, i)
	return s
}

// Register registers a handler for the given operation name.
func (s *Service) Register(name string, handler RPCMethod) {
	key := s.name + "." + name
	s.registry.mu.Lock()
	defer s.registry.mu.Unlock()

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
	interceptors []Interceptor
}

func (h *serviceWrappedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, errTx ErrorTransformer, prefixInterceptors []Interceptor) {
	// Combine: Global (prefix) + Service (h.interceptors)
	combined := make([]Interceptor, 0, len(prefixInterceptors)+len(h.interceptors))
	combined = append(combined, prefixInterceptors...)
	combined = append(combined, h.interceptors...)

	h.inner.ServeHTTP(w, r, errTx, combined)
}

func (h *serviceWrappedHandler) Metadata() *meta.MethodMetadata {
	return h.inner.Metadata()
}
