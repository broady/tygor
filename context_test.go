package tygor

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestRequestFromContext(t *testing.T) {
	t.Run("with request in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
		ctx := newContext(context.Background(), w, req, info)

		result := RequestFromContext(ctx)
		if result != req {
			t.Error("expected request to be returned from context")
		}
	})

	t.Run("without request in context", func(t *testing.T) {
		ctx := context.Background()
		result := RequestFromContext(ctx)
		if result != nil {
			t.Error("expected nil when request not in context")
		}
	})
}

func TestSetHeader(t *testing.T) {
	t.Run("with writer in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
		ctx := newContext(context.Background(), w, req, info)

		SetHeader(ctx, "X-Custom-Header", "custom-value")

		if w.Header().Get("X-Custom-Header") != "custom-value" {
			t.Errorf("expected header to be set, got %s", w.Header().Get("X-Custom-Header"))
		}
	})

	t.Run("without writer in context", func(t *testing.T) {
		ctx := context.Background()
		// Should not panic
		SetHeader(ctx, "X-Custom-Header", "custom-value")
	})
}

func TestMethodFromContext(t *testing.T) {
	t.Run("with RPC info in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
		ctx := newContext(context.Background(), w, req, info)

		service, method, ok := MethodFromContext(ctx)
		if !ok {
			t.Error("expected ok to be true")
		}
		if service != "TestService" {
			t.Errorf("expected service 'TestService', got %s", service)
		}
		if method != "TestMethod" {
			t.Errorf("expected method 'TestMethod', got %s", method)
		}
	})

	t.Run("without RPC info in context", func(t *testing.T) {
		ctx := context.Background()
		service, method, ok := MethodFromContext(ctx)
		if ok {
			t.Error("expected ok to be false")
		}
		if service != "" {
			t.Errorf("expected empty service, got %s", service)
		}
		if method != "" {
			t.Errorf("expected empty method, got %s", method)
		}
	})
}

func TestNewContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	baseCtx := context.Background()

	ctx := newContext(baseCtx, w, req, info)

	// Verify all values are stored correctly
	if RequestFromContext(ctx) != req {
		t.Error("request not stored in context")
	}

	retrievedService, retrievedMethod, ok := MethodFromContext(ctx)
	if !ok {
		t.Error("RPC info not stored in context")
	}
	if retrievedService != "TestService" || retrievedMethod != "TestMethod" {
		t.Error("RPC info stored incorrectly")
	}

	// Verify SetHeader works
	SetHeader(ctx, "X-Test", "value")
	if w.Header().Get("X-Test") != "value" {
		t.Error("writer not stored in context")
	}
}
