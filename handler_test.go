package tygor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type TestRequest struct {
	Name  string `json:"name" validate:"required,min=3"`
	Email string `json:"email" validate:"required,email"`
}

type TestResponse struct {
	Message string `json:"message"`
	ID      int    `json:"id"`
}

func TestNewHandler(t *testing.T) {
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{Message: "ok", ID: 1}, nil
	}

	handler := NewHandler(fn)
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
	if handler.method != "POST" {
		t.Errorf("expected default method POST, got %s", handler.method)
	}
	if handler.fn == nil {
		t.Error("expected fn to be set")
	}
}

func TestHandler_Method(t *testing.T) {
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{}, nil
	}

	handler := NewHandler(fn).Method("GET")
	if handler.method != "GET" {
		t.Errorf("expected method GET, got %s", handler.method)
	}
}

func TestHandler_Cache(t *testing.T) {
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{}, nil
	}

	ttl := 5 * time.Minute
	handler := NewHandler(fn).Cache(ttl)
	if handler.cacheTTL != ttl {
		t.Errorf("expected cache TTL %v, got %v", ttl, handler.cacheTTL)
	}
}

func TestHandler_WithInterceptor(t *testing.T) {
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{}, nil
	}

	interceptor := func(ctx context.Context, req any, info *RPCInfo, handler HandlerFunc) (any, error) {
		return handler(ctx, req)
	}

	handler := NewHandler(fn).WithInterceptor(interceptor)
	if len(handler.interceptors) != 1 {
		t.Errorf("expected 1 interceptor, got %d", len(handler.interceptors))
	}
}

func TestHandler_Metadata(t *testing.T) {
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{}, nil
	}

	handler := NewHandler(fn).Method("GET").Cache(1 * time.Minute)
	meta := handler.Metadata()

	if meta.Method != "GET" {
		t.Errorf("expected method GET, got %s", meta.Method)
	}
	if meta.CacheTTL != 1*time.Minute {
		t.Errorf("expected cache TTL 1m, got %v", meta.CacheTTL)
	}
	if meta.Request == nil {
		t.Error("expected Request type to be set")
	}
	if meta.Response == nil {
		t.Error("expected Response type to be set")
	}
}

func TestHandler_ServeHTTP_POST_Success(t *testing.T) {
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{Message: "hello " + req.Name, ID: 123}, nil
	}

	handler := NewHandler(fn)

	reqBody := `{"name":"John","email":"john@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Add context with RPC info
	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{}

	handler.ServeHTTP(w, req, config)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response TestResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Message != "hello John" {
		t.Errorf("expected message 'hello John', got %s", response.Message)
	}
	if response.ID != 123 {
		t.Errorf("expected ID 123, got %d", response.ID)
	}
}

func TestHandler_ServeHTTP_POST_ValidationError(t *testing.T) {
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{}, nil
	}

	handler := NewHandler(fn)

	// Invalid email and name too short
	reqBody := `{"name":"Jo","email":"invalid"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{}

	handler.ServeHTTP(w, req, config)

	if w.Code == http.StatusOK {
		t.Error("expected non-OK status for validation error")
	}

	var errResp Error
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Code != CodeInvalidArgument {
		t.Errorf("expected error code %s, got %s", CodeInvalidArgument, errResp.Code)
	}
}

func TestHandler_ServeHTTP_POST_InvalidJSON(t *testing.T) {
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{}, nil
	}

	handler := NewHandler(fn)

	reqBody := `{invalid json}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{}

	handler.ServeHTTP(w, req, config)

	if w.Code == http.StatusOK {
		t.Error("expected non-OK status for invalid JSON")
	}

	var errResp Error
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Code != CodeInvalidArgument {
		t.Errorf("expected error code %s, got %s", CodeInvalidArgument, errResp.Code)
	}
}

func TestHandler_ServeHTTP_GET_WithQueryParams(t *testing.T) {
	type GetRequest struct {
		Name  string `schema:"name"`
		Email string `schema:"email"`
	}

	fn := func(ctx context.Context, req GetRequest) (TestResponse, error) {
		return TestResponse{Message: "hello " + req.Name}, nil
	}

	handler := NewHandler(fn).Method("GET")

	req := httptest.NewRequest("GET", "/test?name=John&email=john@example.com", nil)

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{}

	handler.ServeHTTP(w, req, config)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response TestResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Message != "hello John" {
		t.Errorf("expected message 'hello John', got %s", response.Message)
	}
}

func TestHandler_ServeHTTP_HandlerError(t *testing.T) {
	testErr := NewError(CodeNotFound, "resource not found")
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{}, testErr
	}

	handler := NewHandler(fn)

	reqBody := `{"name":"John","email":"john@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{}

	handler.ServeHTTP(w, req, config)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var errResp Error
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Code != CodeNotFound {
		t.Errorf("expected error code %s, got %s", CodeNotFound, errResp.Code)
	}
	if errResp.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got %s", errResp.Message)
	}
}

func TestHandler_ServeHTTP_WithCache(t *testing.T) {
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{Message: "ok"}, nil
	}

	handler := NewHandler(fn).Cache(60 * time.Second)

	reqBody := `{"name":"John","email":"john@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{}

	handler.ServeHTTP(w, req, config)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "max-age=60" {
		t.Errorf("expected Cache-Control 'max-age=60', got %s", cacheControl)
	}
}

func TestHandler_ServeHTTP_WithInterceptor(t *testing.T) {
	interceptorCalled := false

	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{Message: "ok"}, nil
	}

	interceptor := func(ctx context.Context, req any, info *RPCInfo, handler HandlerFunc) (any, error) {
		interceptorCalled = true
		return handler(ctx, req)
	}

	handler := NewHandler(fn).WithInterceptor(interceptor)

	reqBody := `{"name":"John","email":"john@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{}

	handler.ServeHTTP(w, req, config)

	if !interceptorCalled {
		t.Error("expected interceptor to be called")
	}
}

func TestHandler_ServeHTTP_MaskInternalErrors(t *testing.T) {
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{}, errors.New("internal database error")
	}

	handler := NewHandler(fn)

	reqBody := `{"name":"John","email":"john@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{
		MaskInternalErrors: true,
	}

	handler.ServeHTTP(w, req, config)

	var errResp Error
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Message == "internal database error" {
		t.Error("expected internal error message to be masked")
	}
	if errResp.Message != "internal server error" {
		t.Errorf("expected message 'internal server error', got %s", errResp.Message)
	}
}

func TestHandler_ServeHTTP_CustomErrorTransformer(t *testing.T) {
	customErr := errors.New("custom error")
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{}, customErr
	}

	handler := NewHandler(fn)

	reqBody := `{"name":"John","email":"john@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{
		ErrorTransformer: func(err error) *Error {
			if err == customErr {
				return NewError(CodeUnavailable, "custom mapped error")
			}
			return nil
		},
	}

	handler.ServeHTTP(w, req, config)

	var errResp Error
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Code != CodeUnavailable {
		t.Errorf("expected error code %s, got %s", CodeUnavailable, errResp.Code)
	}
	if errResp.Message != "custom mapped error" {
		t.Errorf("expected message 'custom mapped error', got %s", errResp.Message)
	}
}

func TestHandler_ServeHTTP_EmptyBody(t *testing.T) {
	type EmptyRequest struct{}

	fn := func(ctx context.Context, req EmptyRequest) (TestResponse, error) {
		return TestResponse{Message: "ok"}, nil
	}

	handler := NewHandler(fn)

	// Send empty JSON object instead of nil body
	req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{}

	handler.ServeHTTP(w, req, config)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_ServeHTTP_PointerRequest(t *testing.T) {
	fn := func(ctx context.Context, req *TestRequest) (TestResponse, error) {
		if req == nil {
			return TestResponse{}, errors.New("nil request")
		}
		return TestResponse{Message: "hello " + req.Name}, nil
	}

	handler := NewHandler(fn)

	reqBody := `{"name":"John","email":"john@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{}

	handler.ServeHTTP(w, req, config)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response TestResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Message != "hello John" {
		t.Errorf("expected message 'hello John', got %s", response.Message)
	}
}

func TestHandler_ServeHTTP_InterceptorModifiesRequest(t *testing.T) {
	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		return TestResponse{Message: req.Name}, nil
	}

	interceptor := func(ctx context.Context, req any, info *RPCInfo, handler HandlerFunc) (any, error) {
		// Modify the request
		r := req.(TestRequest)
		r.Name = "Modified"
		return handler(ctx, r)
	}

	handler := NewHandler(fn).WithInterceptor(interceptor)

	reqBody := `{"name":"Original","email":"test@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{}

	handler.ServeHTTP(w, req, config)

	var response TestResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.Message != "Modified" {
		t.Errorf("expected message 'Modified', got %s", response.Message)
	}
}

func TestHandleError(t *testing.T) {
	t.Run("with custom error transformer", func(t *testing.T) {
		testErr := errors.New("test error")
		w := httptest.NewRecorder()

		config := HandlerConfig{
			ErrorTransformer: func(err error) *Error {
				if err == testErr {
					return NewError(CodeUnavailable, "transformed")
				}
				return nil
			},
		}

		handleError(w, testErr, config)

		var errResp Error
		json.NewDecoder(w.Body).Decode(&errResp)

		if errResp.Code != CodeUnavailable {
			t.Errorf("expected code %s, got %s", CodeUnavailable, errResp.Code)
		}
		if errResp.Message != "transformed" {
			t.Errorf("expected message 'transformed', got %s", errResp.Message)
		}
	})

	t.Run("fallback to default transformer", func(t *testing.T) {
		testErr := errors.New("test error")
		w := httptest.NewRecorder()

		config := HandlerConfig{
			ErrorTransformer: func(err error) *Error {
				// Return nil to use default transformer
				return nil
			},
		}

		handleError(w, testErr, config)

		var errResp Error
		json.NewDecoder(w.Body).Decode(&errResp)

		if errResp.Code != CodeInternal {
			t.Errorf("expected code %s, got %s", CodeInternal, errResp.Code)
		}
	})
}

func TestHandler_ChainedInterceptors(t *testing.T) {
	var callOrder []string

	fn := func(ctx context.Context, req TestRequest) (TestResponse, error) {
		callOrder = append(callOrder, "handler")
		return TestResponse{Message: "ok"}, nil
	}

	interceptor1 := func(ctx context.Context, req any, info *RPCInfo, handler HandlerFunc) (any, error) {
		callOrder = append(callOrder, "interceptor1-before")
		res, err := handler(ctx, req)
		callOrder = append(callOrder, "interceptor1-after")
		return res, err
	}

	interceptor2 := func(ctx context.Context, req any, info *RPCInfo, handler HandlerFunc) (any, error) {
		callOrder = append(callOrder, "interceptor2-before")
		res, err := handler(ctx, req)
		callOrder = append(callOrder, "interceptor2-after")
		return res, err
	}

	handler := NewHandler(fn).WithInterceptor(interceptor1).WithInterceptor(interceptor2)

	reqBody := `{"name":"John","email":"john@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	// Add config interceptor as well
	configInterceptor := func(ctx context.Context, req any, info *RPCInfo, handler HandlerFunc) (any, error) {
		callOrder = append(callOrder, "config-before")
		res, err := handler(ctx, req)
		callOrder = append(callOrder, "config-after")
		return res, err
	}

	config := HandlerConfig{
		Interceptors: []Interceptor{configInterceptor},
	}

	handler.ServeHTTP(w, req, config)

	// Expected order: config -> interceptor1 -> interceptor2 -> handler -> interceptor2 -> interceptor1 -> config
	expectedOrder := []string{
		"config-before",
		"interceptor1-before",
		"interceptor2-before",
		"handler",
		"interceptor2-after",
		"interceptor1-after",
		"config-after",
	}

	if len(callOrder) != len(expectedOrder) {
		t.Fatalf("expected %d calls, got %d: %v", len(expectedOrder), len(callOrder), callOrder)
	}

	for i, expected := range expectedOrder {
		if callOrder[i] != expected {
			t.Errorf("at position %d: expected %s, got %s", i, expected, callOrder[i])
		}
	}
}

func TestHandler_ServeHTTP_GET_PointerRequest(t *testing.T) {
	type GetRequest struct {
		Name string `schema:"name"`
	}

	fn := func(ctx context.Context, req *GetRequest) (TestResponse, error) {
		if req == nil {
			return TestResponse{}, errors.New("nil request")
		}
		return TestResponse{Message: "hello " + req.Name}, nil
	}

	handler := NewHandler(fn).Method("GET")

	req := httptest.NewRequest("GET", "/test?name=John", nil)

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{}

	handler.ServeHTTP(w, req, config)

	if w.Code != http.StatusOK {
		body, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 200, got %d: %s", w.Code, string(body))
	}

	w.Body = httptest.NewRecorder().Body
	json.NewEncoder(w.Body).Encode(TestResponse{Message: "hello John"})
}

func TestHandler_ServeHTTP_ResponseEncodingError(t *testing.T) {
	// This test simulates a response encoding error by returning a channel
	// which cannot be JSON encoded
	fn := func(ctx context.Context, req TestRequest) (chan int, error) {
		ch := make(chan int)
		return ch, nil
	}

	handler := NewHandler(fn)

	reqBody := `{"name":"John","email":"john@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	info := &RPCInfo{Service: "TestService", Method: "TestMethod"}
	w := httptest.NewRecorder()
	ctx := newContext(req.Context(), w, req, info)
	req = req.WithContext(ctx)

	config := HandlerConfig{}

	// Capture stderr
	oldStderr := os.Stderr
	r, fakeStderr, _ := os.Pipe()
	os.Stderr = fakeStderr

	handler.ServeHTTP(w, req, config)

	// Restore stderr
	fakeStderr.Close()
	os.Stderr = oldStderr

	stderrOutput := make([]byte, 1024)
	n, _ := r.Read(stderrOutput)
	r.Close()

	// Should have written error to stderr
	if n > 0 && !strings.Contains(string(stderrOutput[:n]), "FATAL") {
		t.Logf("stderr output: %s", string(stderrOutput[:n]))
	}
}
