package middleware

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/broady/tygor"
)

func TestLoggingInterceptor_Success(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	interceptor := LoggingInterceptor(logger)

	ctx := tygor.NewContext(context.Background(), "TestService", "TestMethod")

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	result, err := interceptor(ctx, "request", handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != "response" {
		t.Errorf("expected response, got %v", result)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "RPC started") {
		t.Error("expected 'RPC started' in log output")
	}
	if !strings.Contains(logOutput, "RPC completed") {
		t.Error("expected 'RPC completed' in log output")
	}
	if !strings.Contains(logOutput, "TestService") {
		t.Error("expected service name in log output")
	}
	if !strings.Contains(logOutput, "TestMethod") {
		t.Error("expected method name in log output")
	}
}

func TestLoggingInterceptor_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	interceptor := LoggingInterceptor(logger)

	ctx := tygor.NewContext(context.Background(), "TestService", "TestMethod")

	testErr := errors.New("test error")
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, testErr
	}

	result, err := interceptor(ctx, "request", handler)

	if err != testErr {
		t.Errorf("expected test error, got %v", err)
	}

	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "RPC started") {
		t.Error("expected 'RPC started' in log output")
	}
	if !strings.Contains(logOutput, "RPC failed") {
		t.Error("expected 'RPC failed' in log output")
	}
	if !strings.Contains(logOutput, "test error") {
		t.Error("expected error message in log output")
	}
}

func TestLoggingInterceptor_NilLogger(t *testing.T) {
	// Should not panic with nil logger, should use default
	interceptor := LoggingInterceptor(nil)

	ctx := tygor.NewContext(context.Background(), "TestService", "TestMethod")

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	result, err := interceptor(ctx, "request", handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != "response" {
		t.Errorf("expected response, got %v", result)
	}
}

func TestLoggingInterceptor_LogsDuration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	interceptor := LoggingInterceptor(logger)

	ctx := tygor.NewContext(context.Background(), "TestService", "TestMethod")

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	_, err := interceptor(ctx, "request", handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "duration") {
		t.Error("expected 'duration' in log output")
	}
}

func TestLoggingInterceptor_PropagatesContext(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	interceptor := LoggingInterceptor(logger)

	type ctxKey string
	key := ctxKey("test-key")
	baseCtx := context.WithValue(context.Background(), key, "test-value")
	ctx := tygor.NewContext(baseCtx, "TestService", "TestMethod")

	handler := func(ctx context.Context, req any) (any, error) {
		val := ctx.Value(key)
		if val != "test-value" {
			t.Error("expected context value to be propagated")
		}
		return "response", nil
	}

	_, err := interceptor(ctx, "request", handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoggingInterceptor_ServiceAndMethodInLogs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	interceptor := LoggingInterceptor(logger)

	tests := []struct {
		service string
		method  string
	}{
		{"Users", "Create"},
		{"Posts", "List"},
		{"Comments", "Delete"},
	}

	for _, tt := range tests {
		t.Run(tt.service+"."+tt.method, func(t *testing.T) {
			buf.Reset()

			ctx := tygor.NewContext(context.Background(), tt.service, tt.method)

			handler := func(ctx context.Context, req any) (any, error) {
				return nil, nil
			}

			_, _ = interceptor(ctx, nil, handler)

			logOutput := buf.String()
			if !strings.Contains(logOutput, tt.service) {
				t.Errorf("expected service %s in log output", tt.service)
			}
			if !strings.Contains(logOutput, tt.method) {
				t.Errorf("expected method %s in log output", tt.method)
			}
		})
	}
}

func TestLoggingInterceptor_ErrorDetails(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	interceptor := LoggingInterceptor(logger)

	ctx := tygor.NewContext(context.Background(), "TestService", "TestMethod")

	customErr := tygor.NewError(tygor.CodeNotFound, "resource not found")
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, customErr
	}

	_, err := interceptor(ctx, "request", handler)

	if err != customErr {
		t.Errorf("expected custom error, got %v", err)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "RPC failed") {
		t.Error("expected 'RPC failed' in log output")
	}
	// Error should be logged with details
	if !strings.Contains(logOutput, "not_found") || !strings.Contains(logOutput, "resource not found") {
		t.Error("expected error details in log output")
	}
}

func TestLoggingInterceptor_PassthroughRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	interceptor := LoggingInterceptor(logger)

	ctx := tygor.NewContext(context.Background(), "TestService", "TestMethod")

	type testReq struct {
		Key string
	}
	expectedReq := testReq{Key: "value"}
	handler := func(ctx context.Context, req any) (any, error) {
		if req != expectedReq {
			t.Error("expected request to be passed through")
		}
		return "response", nil
	}

	_, err := interceptor(ctx, expectedReq, handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
