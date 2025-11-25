package tygor

import (
	"net/http"
	"net/http/httptest"

	"github.com/broady/tygor/internal/tygortest"
)

// TestRequestBuilder wraps tygortest.RequestBuilder for use in tygor package tests.
type TestRequestBuilder struct {
	*tygortest.RequestBuilder
}

// NewTestRequest creates a new test request builder for tygor handlers.
func NewTestRequest() *TestRequestBuilder {
	return &TestRequestBuilder{
		RequestBuilder: tygortest.NewRequest(tygortest.ContextSetup()),
	}
}

// POST sets the HTTP method to POST and returns the TestRequestBuilder for chaining.
func (tr *TestRequestBuilder) POST(path string) *TestRequestBuilder {
	tr.RequestBuilder.POST(path)
	return tr
}

// GET sets the HTTP method to GET and returns the TestRequestBuilder for chaining.
func (tr *TestRequestBuilder) GET(path string) *TestRequestBuilder {
	tr.RequestBuilder.GET(path)
	return tr
}

// WithJSON sets the request body as JSON and returns the TestRequestBuilder for chaining.
func (tr *TestRequestBuilder) WithJSON(v any) *TestRequestBuilder {
	tr.RequestBuilder.WithJSON(v)
	return tr
}

// WithBody sets the raw request body and returns the TestRequestBuilder for chaining.
func (tr *TestRequestBuilder) WithBody(body string) *TestRequestBuilder {
	tr.RequestBuilder.WithBody(body)
	return tr
}

// WithHeader adds a header to the request and returns the TestRequestBuilder for chaining.
func (tr *TestRequestBuilder) WithHeader(key, value string) *TestRequestBuilder {
	tr.RequestBuilder.WithHeader(key, value)
	return tr
}

// WithQuery adds a query parameter and returns the TestRequestBuilder for chaining.
func (tr *TestRequestBuilder) WithQuery(key, value string) *TestRequestBuilder {
	tr.RequestBuilder.WithQuery(key, value)
	return tr
}

// WithRPCInfo sets the service and method for RPC context and returns the TestRequestBuilder for chaining.
func (tr *TestRequestBuilder) WithRPCInfo(service, method string) *TestRequestBuilder {
	tr.RequestBuilder.WithRPCInfo(service, method)
	return tr
}

// Build creates the HTTP request with tygor RPC context.
func (tr *TestRequestBuilder) Build() (*http.Request, *httptest.ResponseRecorder) {
	return tr.RequestBuilder.Build()
}

// ServeHandler builds the request and serves it to a tygor handler.
func (tr *TestRequestBuilder) ServeHandler(handler RPCMethod, config HandlerConfig) *httptest.ResponseRecorder {
	req, w := tr.Build()
	handler.ServeHTTP(w, req, config)
	return w
}
