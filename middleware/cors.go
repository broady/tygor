package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

// CORSConfig holds the configuration for CORS middleware.
type CORSConfig struct {
	// AllowOrigins is a list of origins a cross-domain request can be executed from.
	// If the list contains "*", all origins are allowed.
	// Default: ["*"]
	AllowOrigins []string

	// AllowMethods is a list of methods the client is allowed to use.
	// Default: ["GET", "POST", "OPTIONS"]
	AllowMethods []string

	// AllowHeaders is a list of headers the client is allowed to use.
	// Default: ["Content-Type", "Authorization"]
	AllowHeaders []string

	// ExposeHeaders indicates which headers are safe to expose.
	// Default: []
	ExposeHeaders []string

	// AllowCredentials indicates whether the request can include credentials.
	// Default: false
	AllowCredentials bool

	// MaxAge indicates how long (in seconds) the results of a preflight request can be cached.
	// Default: 0 (not set)
	MaxAge int
}

// CORSAllowAll is a permissive CORS configuration suitable for development.
// It allows all origins (*), standard methods (GET, POST, OPTIONS),
// and common headers (Content-Type, Authorization).
var CORSAllowAll *CORSConfig = nil

// CORS returns an HTTP middleware that handles CORS preflight requests and sets CORS headers.
// This is an HTTP middleware, not an RPC interceptor, so it wraps the entire http.Handler.
func CORS(cfg *CORSConfig) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = &CORSConfig{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "OPTIONS"},
			AllowHeaders: []string{"Content-Type", "Authorization"},
		}
	}

	allowedOrigins := cfg.AllowOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	allowedMethods := cfg.AllowMethods
	if len(allowedMethods) == 0 {
		allowedMethods = []string{"GET", "POST", "OPTIONS"}
	}

	allowedHeaders := cfg.AllowHeaders
	if len(allowedHeaders) == 0 {
		allowedHeaders = []string{"Content-Type", "Authorization"}
	}

	allowedMethodsStr := strings.Join(allowedMethods, ", ")
	allowedHeadersStr := strings.Join(allowedHeaders, ", ")
	exposedHeadersStr := strings.Join(cfg.ExposeHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			if contains(allowedOrigins, "*") {
				allowed = true
			} else if origin != "" {
				for _, o := range allowedOrigins {
					if o == origin {
						allowed = true
						break
					}
				}
			}

			if allowed {
				// CORS spec forbids using Access-Control-Allow-Origin: * with
				// Access-Control-Allow-Credentials: true. When credentials are enabled
				// and wildcard origins are configured, echo back the specific requesting
				// origin instead of "*". This effectively allows all origins while
				// remaining spec-compliant.
				if origin != "" && !contains(allowedOrigins, "*") {
					// Specific origins configured: echo back the matched origin
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else if origin != "" && cfg.AllowCredentials {
					// Wildcard with credentials: echo back the requesting origin
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					// Wildcard without credentials: use "*"
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}

				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Allow-Methods", allowedMethodsStr)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeadersStr)
				if exposedHeadersStr != "" {
					w.Header().Set("Access-Control-Expose-Headers", exposedHeadersStr)
				}
				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
