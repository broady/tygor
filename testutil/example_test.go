package testutil_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/broady/tygor"
	"github.com/broady/tygor/testutil"
)

// Example types for testing
type ExampleRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type ExampleResponse struct {
	Message string `json:"message"`
	ID      int    `json:"id"`
}

type GetParams struct {
	Query string `schema:"query"`
	Limit int    `schema:"limit"`
}

// Example handler function
func exampleHandler(ctx context.Context, req *ExampleRequest) (*ExampleResponse, error) {
	return &ExampleResponse{
		Message: "Hello, " + req.Name,
		ID:      123,
	}, nil
}

// TestRequestBuilder demonstrates the fluent API for building requests
func TestRequestBuilder(t *testing.T) {
	handler := tygor.Unary(exampleHandler)

	// Build request with tygor context
	req, w := testutil.NewRequest(tygor.TestContextSetup()).
		POST("/test").
		WithJSON(&ExampleRequest{Name: "Alice", Email: "alice@example.com"}).
		Build()

	// Serve the handler
	handler.ServeHTTP(w, req, tygor.HandlerConfig{})

	// Use assertion helpers
	testutil.AssertStatus(t, w, http.StatusOK)
	testutil.AssertJSONResponse(t, w, &ExampleResponse{
		Message: "Hello, Alice",
		ID:      123,
	})
}

// TestRequestBuilder_Validation demonstrates validation error handling
func TestRequestBuilder_Validation(t *testing.T) {
	handler := tygor.Unary(exampleHandler)

	req, w := testutil.NewRequest(tygor.TestContextSetup()).
		POST("/test").
		WithJSON(&ExampleRequest{Name: "Alice", Email: "invalid-email"}).
		Build()

	handler.ServeHTTP(w, req, tygor.HandlerConfig{})

	testutil.AssertStatus(t, w, http.StatusBadRequest)
	errResp := testutil.AssertJSONError(t, w, string(tygor.CodeInvalidArgument))

	if errResp.Message != "validation failed" {
		t.Errorf("expected validation error message, got %s", errResp.Message)
	}
}

// TestRequestBuilder_GET demonstrates GET request with query parameters
func TestRequestBuilder_GET(t *testing.T) {
	type GetParams struct {
		Query string `schema:"query"`
		Limit int    `schema:"limit"`
	}

	getHandler := func(ctx context.Context, req *GetParams) (*ExampleResponse, error) {
		return &ExampleResponse{
			Message: "Search: " + req.Query,
			ID:      req.Limit,
		}, nil
	}

	handler := tygor.UnaryGet(getHandler)

	req, w := testutil.NewRequest(tygor.TestContextSetup()).
		GET("/search").
		WithQuery("query", "golang").
		WithQuery("limit", "10").
		Build()

	handler.ServeHTTP(w, req, tygor.HandlerConfig{})

	testutil.AssertStatus(t, w, http.StatusOK)
	testutil.AssertJSONResponse(t, w, &ExampleResponse{
		Message: "Search: golang",
		ID:      10,
	})
}

// TestRequestBuilder_CustomHeader demonstrates custom headers
func TestRequestBuilder_CustomHeader(t *testing.T) {
	authHandler := func(ctx context.Context, req *ExampleRequest) (*ExampleResponse, error) {
		httpReq := tygor.RequestFromContext(ctx)
		token := httpReq.Header.Get("X-API-Key")
		if token != "secret" {
			return nil, tygor.NewError(tygor.CodeUnauthenticated, "invalid api key")
		}
		return &ExampleResponse{Message: "authenticated"}, nil
	}

	handler := tygor.Unary(authHandler)

	req, w := testutil.NewRequest(tygor.TestContextSetup()).
		POST("/test").
		WithJSON(&ExampleRequest{Name: "Alice", Email: "alice@example.com"}).
		WithHeader("X-API-Key", "secret").
		Build()

	handler.ServeHTTP(w, req, tygor.HandlerConfig{})

	testutil.AssertStatus(t, w, http.StatusOK)
}

// TestAssertHeader demonstrates header assertions
func TestAssertHeader(t *testing.T) {
	getHandler := func(ctx context.Context, req *GetParams) (*ExampleResponse, error) {
		return &ExampleResponse{Message: "cached response"}, nil
	}
	handler := tygor.UnaryGet(getHandler).Cache(60 * time.Second)

	req, w := testutil.NewRequest(tygor.TestContextSetup()).
		GET("/test").
		WithQuery("query", "test").
		WithQuery("limit", "10").
		Build()

	handler.ServeHTTP(w, req, tygor.HandlerConfig{})

	testutil.AssertHeader(t, w, "Cache-Control", "private, max-age=60")
}

// Example showing the before/after comparison
func ExampleRequestBuilder_comparison() {
	// BEFORE (manual setup - verbose):
	// reqBody := `{"name":"Alice","email":"alice@example.com"}`
	// req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
	// req.Header.Set("Content-Type", "application/json")
	// info := &tygor.RPCInfo{Service: "TestService", Method: "TestMethod"}
	// w := httptest.NewRecorder()
	// ctx := tygor.NewTestContext(req.Context(), w, req, info)
	// req = req.WithContext(ctx)
	// config := tygor.HandlerConfig{}
	// handler.ServeHTTP(w, req, config)

	// AFTER (using testutil - more concise):
	handler := tygor.Unary(exampleHandler)
	req, w := testutil.NewRequest(tygor.TestContextSetup()).
		POST("/test").
		WithJSON(&ExampleRequest{Name: "Alice", Email: "alice@example.com"}).
		Build()

	handler.ServeHTTP(w, req, tygor.HandlerConfig{})
	testutil.AssertStatus(nil, w, http.StatusOK)
}
