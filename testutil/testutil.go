// Package testutil provides testing helpers for HTTP handlers and tygor RPC handlers.
// This package is designed to be import-cycle safe and can be used from any package.
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// RequestBuilder helps construct test HTTP requests with fluent API.
type RequestBuilder struct {
	method       string
	path         string
	body         []byte
	headers      map[string]string
	queryParams  map[string]string
	service      string
	rpcMethod    string
	contextSetup ContextSetupFunc
}

// NewRequest creates a new request builder.
// Optionally accepts a ContextSetupFunc to configure the request context.
func NewRequest(contextSetup ...ContextSetupFunc) *RequestBuilder {
	var setup ContextSetupFunc
	if len(contextSetup) > 0 {
		setup = contextSetup[0]
	}
	return &RequestBuilder{
		method:       "GET",
		path:         "/",
		headers:      make(map[string]string),
		queryParams:  make(map[string]string),
		service:      "TestService",
		rpcMethod:    "TestMethod",
		contextSetup: setup,
	}
}

// GET sets the HTTP method to GET.
func (b *RequestBuilder) GET(path string) *RequestBuilder {
	b.method = "GET"
	b.path = path
	return b
}

// POST sets the HTTP method to POST.
func (b *RequestBuilder) POST(path string) *RequestBuilder {
	b.method = "POST"
	b.path = path
	return b
}

// WithJSON sets the request body as JSON.
func (b *RequestBuilder) WithJSON(v any) *RequestBuilder {
	data, _ := json.Marshal(v)
	b.body = data
	b.headers["Content-Type"] = "application/json"
	return b
}

// WithBody sets the raw request body.
func (b *RequestBuilder) WithBody(body string) *RequestBuilder {
	b.body = []byte(body)
	return b
}

// WithHeader adds a header to the request.
func (b *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	b.headers[key] = value
	return b
}

// WithQuery adds a query parameter.
func (b *RequestBuilder) WithQuery(key, value string) *RequestBuilder {
	b.queryParams[key] = value
	return b
}

// WithRPCInfo sets the service and method for RPC context.
func (b *RequestBuilder) WithRPCInfo(service, method string) *RequestBuilder {
	b.service = service
	b.rpcMethod = method
	return b
}

// ContextSetupFunc is a function that sets up the request context.
// It receives the current context, response writer, and request, and returns a new context.
type ContextSetupFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request, service, method string) context.Context

// Build creates the HTTP request and ResponseRecorder.
// Uses the contextSetup provided to NewRequest().
func (b *RequestBuilder) Build() (*http.Request, *httptest.ResponseRecorder) {
	path := b.path
	if len(b.queryParams) > 0 {
		params := []string{}
		for k, v := range b.queryParams {
			params = append(params, k+"="+v)
		}
		path += "?" + strings.Join(params, "&")
	}

	var bodyReader *bytes.Reader
	if len(b.body) > 0 {
		bodyReader = bytes.NewReader(b.body)
	}

	var req *http.Request
	if bodyReader != nil {
		req = httptest.NewRequest(b.method, path, bodyReader)
	} else {
		req = httptest.NewRequest(b.method, path, nil)
	}

	for k, v := range b.headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()

	// Set up RPC context if provided
	if b.contextSetup != nil {
		ctx := b.contextSetup(req.Context(), w, req, b.service, b.rpcMethod)
		req = req.WithContext(ctx)
	}

	return req, w
}

// AssertStatus checks that the response has the expected status code.
func AssertStatus(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()
	if w.Code != expectedStatus {
		t.Errorf("expected status %d, got %d\nBody: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// AssertJSONResponse decodes the response body and compares it with expected value.
func AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expected any) {
	t.Helper()

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type to contain application/json, got %s", contentType)
	}

	// Use reflection to create the same type as expected
	expectedJSON, _ := json.Marshal(expected)
	actualJSON := w.Body.Bytes()

	// Compare as JSON to ignore formatting differences
	var expectedData, actualData any
	json.Unmarshal(expectedJSON, &expectedData)
	json.Unmarshal(actualJSON, &actualData)

	expectedStr, _ := json.MarshalIndent(expectedData, "", "  ")
	actualStr, _ := json.MarshalIndent(actualData, "", "  ")

	if string(expectedStr) != string(actualStr) {
		t.Errorf("response mismatch:\nExpected:\n%s\nActual:\n%s", expectedStr, actualStr)
	}
}

// ErrorResponse represents a generic error response with code and message.
type ErrorResponse struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// AssertJSONError checks that the response contains an error with the expected code.
func AssertJSONError(t *testing.T, w *httptest.ResponseRecorder, expectedCode string) *ErrorResponse {
	t.Helper()

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v\nBody: %s", err, w.Body.String())
	}

	if errResp.Code != expectedCode {
		t.Errorf("expected error code %s, got %s (message: %s)", expectedCode, errResp.Code, errResp.Message)
	}

	return &errResp
}

// AssertHeader checks that a response header has the expected value.
func AssertHeader(t *testing.T, w *httptest.ResponseRecorder, key, expectedValue string) {
	t.Helper()
	actual := w.Header().Get(key)
	if actual != expectedValue {
		t.Errorf("expected header %s=%s, got %s", key, expectedValue, actual)
	}
}

// DecodeJSON decodes the response body into the provided value.
func DecodeJSON(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(w.Body).Decode(v); err != nil {
		t.Fatalf("failed to decode response: %v\nBody: %s", err, w.Body.String())
	}
}
