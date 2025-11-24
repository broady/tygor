package tygor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestNewError(t *testing.T) {
	err := NewError(CodeNotFound, "resource not found")
	if err.Code != CodeNotFound {
		t.Errorf("expected code %s, got %s", CodeNotFound, err.Code)
	}
	if err.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got %s", err.Message)
	}
}

func TestErrorf(t *testing.T) {
	err := Errorf(CodeInvalidArgument, "invalid field: %s", "email")
	if err.Code != CodeInvalidArgument {
		t.Errorf("expected code %s, got %s", CodeInvalidArgument, err.Code)
	}
	if err.Message != "invalid field: email" {
		t.Errorf("expected formatted message, got %s", err.Message)
	}
}

func TestErrorError(t *testing.T) {
	err := NewError(CodeInternal, "something went wrong")
	expected := "internal: something went wrong"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestDefaultErrorTransformer(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		wantCode ErrorCode
		wantMsg  string
	}{
		{
			name:     "nil error",
			input:    nil,
			wantCode: "",
			wantMsg:  "",
		},
		{
			name:     "RPC error passthrough",
			input:    NewError(CodeNotFound, "not found"),
			wantCode: CodeNotFound,
			wantMsg:  "not found",
		},
		{
			name:     "context deadline exceeded",
			input:    context.DeadlineExceeded,
			wantCode: CodeUnavailable,
			wantMsg:  "context deadline exceeded",
		},
		{
			name:     "context canceled",
			input:    context.Canceled,
			wantCode: CodeCanceled,
			wantMsg:  "context canceled",
		},
		{
			name:     "generic error",
			input:    errors.New("something failed"),
			wantCode: CodeInternal,
			wantMsg:  "something failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultErrorTransformer(tt.input)
			if tt.input == nil {
				if result != nil {
					t.Errorf("expected nil for nil input, got %v", result)
				}
				return
			}
			if result.Code != tt.wantCode {
				t.Errorf("expected code %s, got %s", tt.wantCode, result.Code)
			}
			if result.Message != tt.wantMsg {
				t.Errorf("expected message %q, got %q", tt.wantMsg, result.Message)
			}
		})
	}
}

func TestDefaultErrorTransformer_ValidationErrors(t *testing.T) {
	type TestStruct struct {
		Email string `validate:"required,email"`
		Age   int    `validate:"gte=0,lte=120"`
	}

	validate := validator.New()
	s := TestStruct{Email: "invalid", Age: -1}
	err := validate.Struct(s)

	result := DefaultErrorTransformer(err)
	if result.Code != CodeInvalidArgument {
		t.Errorf("expected code %s, got %s", CodeInvalidArgument, result.Code)
	}
	if result.Message != "validation failed" {
		t.Errorf("expected message 'validation failed', got %s", result.Message)
	}
	if result.Details == nil {
		t.Fatal("expected details to be non-nil")
	}
	if _, ok := result.Details["Email"]; !ok {
		t.Error("expected Email field in details")
	}
	if _, ok := result.Details["Age"]; !ok {
		t.Error("expected Age field in details")
	}
}

func TestDefaultErrorTransformer_MultiError(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	multiErr := errors.Join(err1, err2)

	result := DefaultErrorTransformer(multiErr)
	if result.Code != CodeInternal {
		t.Errorf("expected code from first error %s, got %s", CodeInternal, result.Code)
	}
	// Message should contain both errors
	if result.Message != "error 1; error 2" {
		t.Errorf("expected combined message, got %q", result.Message)
	}
}

func TestHTTPStatusFromCode(t *testing.T) {
	tests := []struct {
		code       ErrorCode
		wantStatus int
	}{
		{CodeInvalidArgument, http.StatusBadRequest},
		{CodeUnauthenticated, http.StatusUnauthorized},
		{CodePermissionDenied, http.StatusForbidden},
		{CodeNotFound, http.StatusNotFound},
		{CodeUnavailable, http.StatusServiceUnavailable},
		{CodeCanceled, 499},
		{CodeInternal, http.StatusInternalServerError},
		{ErrorCode("unknown"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			status := HTTPStatusFromCode(tt.code)
			if status != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, status)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	rpcErr := NewError(CodeNotFound, "resource not found")
	w := httptest.NewRecorder()

	writeError(w, rpcErr)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	// Check response body contains error
	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty body")
	}
}

type failingWriter struct {
	headerWritten bool
}

func (fw *failingWriter) Header() http.Header {
	return http.Header{}
}

func (fw *failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func (fw *failingWriter) WriteHeader(statusCode int) {
	fw.headerWritten = true
}

func TestWriteError_EncodingFailure(t *testing.T) {
	rpcErr := NewError(CodeInternal, "test error")
	w := &failingWriter{}

	// Capture stderr to verify error logging
	oldStderr := os.Stderr
	r, fakeStderr, _ := os.Pipe()
	os.Stderr = fakeStderr

	writeError(w, rpcErr)

	// Restore stderr
	fakeStderr.Close()
	os.Stderr = oldStderr

	stderrOutput := make([]byte, 1024)
	n, _ := r.Read(stderrOutput)
	r.Close()

	if n > 0 && !strings.Contains(string(stderrOutput[:n]), "FATAL") {
		t.Logf("stderr output: %s", string(stderrOutput[:n]))
	}

	if !w.headerWritten {
		t.Error("expected WriteHeader to be called")
	}
}
