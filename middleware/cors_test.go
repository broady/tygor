package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultCORSConfig(t *testing.T) {
	cfg := DefaultCORSConfig()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "*" {
		t.Errorf("expected AllowedOrigins to be [*], got %v", cfg.AllowedOrigins)
	}

	expectedMethods := []string{"GET", "POST", "OPTIONS"}
	if len(cfg.AllowedMethods) != len(expectedMethods) {
		t.Errorf("expected %d methods, got %d", len(expectedMethods), len(cfg.AllowedMethods))
	}

	expectedHeaders := []string{"Content-Type", "Authorization"}
	if len(cfg.AllowedHeaders) != len(expectedHeaders) {
		t.Errorf("expected %d headers, got %d", len(expectedHeaders), len(cfg.AllowedHeaders))
	}
}

func TestCORS_WithDefaultConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	corsHandler := CORS(DefaultCORSConfig())(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_NilConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Should use default config
	corsHandler := CORS(nil)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected default Access-Control-Allow-Origin *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_Preflight(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for preflight request")
	})

	corsHandler := CORS(DefaultCORSConfig())(handler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods header to be set")
	}

	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("expected Access-Control-Allow-Headers header to be set")
	}
}

func TestCORS_SpecificOrigin(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &CORSConfig{
		AllowedOrigins: []string{"http://example.com", "http://test.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	corsHandler := CORS(cfg)(handler)

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
	}{
		{"allowed origin 1", "http://example.com", "http://example.com"},
		{"allowed origin 2", "http://test.com", "http://test.com"},
		{"disallowed origin", "http://evil.com", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			corsHandler.ServeHTTP(w, req)

			gotOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if gotOrigin != tt.expectedOrigin {
				t.Errorf("expected origin %s, got %s", tt.expectedOrigin, gotOrigin)
			}
		})
	}
}

func TestCORS_NoOrigin(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &CORSConfig{
		AllowedOrigins: []string{"http://example.com"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	corsHandler := CORS(cfg)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	// No Origin header
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no Access-Control-Allow-Origin header, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_Credentials(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}

	corsHandler := CORS(cfg)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials true, got %s", w.Header().Get("Access-Control-Allow-Credentials"))
	}
}

func TestCORS_MaxAge(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         3600,
	}

	corsHandler := CORS(cfg)(handler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Max-Age") != "3600" {
		t.Errorf("expected Access-Control-Max-Age 3600, got %s", w.Header().Get("Access-Control-Max-Age"))
	}
}

func TestCORS_ExposedHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
		ExposedHeaders: []string{"X-Custom-Header", "X-Another-Header"},
	}

	corsHandler := CORS(cfg)(handler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	exposedHeaders := w.Header().Get("Access-Control-Expose-Headers")
	if exposedHeaders != "X-Custom-Header, X-Another-Header" {
		t.Errorf("expected exposed headers, got %s", exposedHeaders)
	}
}

func TestCORS_EmptyAllowedOrigins(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &CORSConfig{
		AllowedOrigins: []string{}, // Empty, should default to *
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	corsHandler := CORS(cfg)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected default origin *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_EmptyAllowedMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{}, // Empty, should use default
		AllowedHeaders: []string{"Content-Type"},
	}

	corsHandler := CORS(cfg)(handler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	allowedMethods := w.Header().Get("Access-Control-Allow-Methods")
	if allowedMethods == "" {
		t.Error("expected default allowed methods to be set")
	}
}

func TestCORS_EmptyAllowedHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{}, // Empty, should use default
	}

	corsHandler := CORS(cfg)(handler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	allowedHeaders := w.Header().Get("Access-Control-Allow-Headers")
	if allowedHeaders == "" {
		t.Error("expected default allowed headers to be set")
	}
}

func TestCORS_NonPreflightRequest(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := CORS(DefaultCORSConfig())(handler)

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("expected handler to be called for non-preflight request")
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS headers to be set even for non-preflight requests")
	}
}

func TestCORS_AllowedOriginNotWildcard(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &CORSConfig{
		AllowedOrigins:   []string{"http://example.com"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}

	corsHandler := CORS(cfg)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	// When not using wildcard, should return the specific origin
	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("expected origin http://example.com, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_WildcardWithCredentials(t *testing.T) {
	// This test verifies that when AllowedOrigins is ["*"] AND AllowCredentials is true,
	// the middleware echoes back the specific requesting origin instead of "*".
	// This is required by the CORS spec, which forbids using Access-Control-Allow-Origin: *
	// with Access-Control-Allow-Credentials: true.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := &CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}

	corsHandler := CORS(cfg)(handler)

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
	}{
		{
			name:           "with origin header",
			origin:         "http://example.com",
			expectedOrigin: "http://example.com",
		},
		{
			name:           "different origin",
			origin:         "https://another-domain.com",
			expectedOrigin: "https://another-domain.com",
		},
		{
			name:           "no origin header",
			origin:         "",
			expectedOrigin: "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()

			corsHandler.ServeHTTP(w, req)

			gotOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if gotOrigin != tt.expectedOrigin {
				t.Errorf("expected origin %s, got %s", tt.expectedOrigin, gotOrigin)
			}

			// When an origin is present, credentials header should always be set
			if tt.origin != "" {
				gotCredentials := w.Header().Get("Access-Control-Allow-Credentials")
				if gotCredentials != "true" {
					t.Errorf("expected credentials 'true', got %s", gotCredentials)
				}
			}
		})
	}

	// Also test preflight requests to ensure the behavior is consistent
	t.Run("preflight with origin", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://preflight-test.com")
		w := httptest.NewRecorder()

		corsHandler.ServeHTTP(w, req)

		gotOrigin := w.Header().Get("Access-Control-Allow-Origin")
		if gotOrigin != "http://preflight-test.com" {
			t.Errorf("expected origin http://preflight-test.com, got %s", gotOrigin)
		}

		gotCredentials := w.Header().Get("Access-Control-Allow-Credentials")
		if gotCredentials != "true" {
			t.Errorf("expected credentials 'true', got %s", gotCredentials)
		}
	})
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"exact match", []string{"test"}, "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
