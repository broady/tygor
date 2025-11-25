package tygor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ErrorCode represents a machine-readable error code.
type ErrorCode string

const (
	CodeInvalidArgument   ErrorCode = "invalid_argument"
	CodeUnauthenticated   ErrorCode = "unauthenticated"
	CodePermissionDenied  ErrorCode = "permission_denied"
	CodeNotFound          ErrorCode = "not_found"
	CodeMethodNotAllowed  ErrorCode = "method_not_allowed"
	CodeConflict          ErrorCode = "conflict"
	CodeAlreadyExists     ErrorCode = "already_exists" // Alias for conflict, used when resource already exists
	CodeGone              ErrorCode = "gone"
	CodeResourceExhausted ErrorCode = "resource_exhausted"
	CodeCanceled          ErrorCode = "canceled"
	CodeInternal          ErrorCode = "internal"
	CodeNotImplemented    ErrorCode = "not_implemented"
	CodeUnavailable       ErrorCode = "unavailable"
	CodeDeadlineExceeded  ErrorCode = "deadline_exceeded"
)

// Error is the standard JSON error envelope.
type Error struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewError creates a new RPC error.
func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Errorf creates a new RPC error with a formatted message.
func Errorf(code ErrorCode, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Convenience constructors for common error codes.
// Each accepts an optional details map as the last argument.

// InvalidArgument creates an invalid_argument error (400).
func InvalidArgument(message string, details ...map[string]any) *Error {
	return newErrorWithDetails(CodeInvalidArgument, message, details)
}

// Unauthenticated creates an unauthenticated error (401).
func Unauthenticated(message string, details ...map[string]any) *Error {
	return newErrorWithDetails(CodeUnauthenticated, message, details)
}

// PermissionDenied creates a permission_denied error (403).
func PermissionDenied(message string, details ...map[string]any) *Error {
	return newErrorWithDetails(CodePermissionDenied, message, details)
}

// NotFound creates a not_found error (404).
func NotFound(message string, details ...map[string]any) *Error {
	return newErrorWithDetails(CodeNotFound, message, details)
}

// Conflict creates a conflict error (409).
func Conflict(message string, details ...map[string]any) *Error {
	return newErrorWithDetails(CodeConflict, message, details)
}

// AlreadyExists creates an already_exists error (409).
func AlreadyExists(message string, details ...map[string]any) *Error {
	return newErrorWithDetails(CodeAlreadyExists, message, details)
}

// Gone creates a gone error (410).
func Gone(message string, details ...map[string]any) *Error {
	return newErrorWithDetails(CodeGone, message, details)
}

// ResourceExhausted creates a resource_exhausted error (429).
func ResourceExhausted(message string, details ...map[string]any) *Error {
	return newErrorWithDetails(CodeResourceExhausted, message, details)
}

// Internal creates an internal error (500).
func Internal(message string, details ...map[string]any) *Error {
	return newErrorWithDetails(CodeInternal, message, details)
}

// NotImplemented creates a not_implemented error (501).
func NotImplemented(message string, details ...map[string]any) *Error {
	return newErrorWithDetails(CodeNotImplemented, message, details)
}

// Unavailable creates an unavailable error (503).
func Unavailable(message string, details ...map[string]any) *Error {
	return newErrorWithDetails(CodeUnavailable, message, details)
}

// DeadlineExceeded creates a deadline_exceeded error (504).
func DeadlineExceeded(message string, details ...map[string]any) *Error {
	return newErrorWithDetails(CodeDeadlineExceeded, message, details)
}

func newErrorWithDetails(code ErrorCode, message string, details []map[string]any) *Error {
	err := &Error{Code: code, Message: message}
	if len(details) > 0 && details[0] != nil {
		err.Details = details[0]
	}
	return err
}

// ErrorTransformer is a function that maps an application error to an RPC error.
// If it returns nil, the default transformer logic should be applied.
type ErrorTransformer func(error) *Error

// DefaultErrorTransformer maps standard Go errors to RPC errors.
func DefaultErrorTransformer(err error) *Error {
	if err == nil {
		return nil
	}

	var rpcErr *Error
	if errors.As(err, &rpcErr) {
		return rpcErr
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return NewError(CodeDeadlineExceeded, "request timeout")
	}

	if errors.Is(err, context.Canceled) {
		return NewError(CodeCanceled, "context canceled")
	}

	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		details := make(map[string]any)
		for _, ve := range valErrs {
			details[ve.Field()] = ve.Tag()
		}
		return &Error{
			Code:    CodeInvalidArgument,
			Message: "validation failed",
			Details: details,
		}
	}

	// Handle multi-errors (errors.Join)
	if u, ok := err.(interface{ Unwrap() []error }); ok {
		errs := u.Unwrap()
		if len(errs) > 0 {
			// Use the first error to determine code? Or just use Unknown/Internal?
			// For now, let's try to map the first error, but keep all messages.
			firstMapped := DefaultErrorTransformer(errs[0])
			msgs := make([]string, len(errs))
			for i, e := range errs {
				msgs[i] = e.Error()
			}
			return &Error{
				Code:    firstMapped.Code,
				Message: strings.Join(msgs, "; "),
				Details: firstMapped.Details,
			}
		}
	}

	// Fallback to internal error.
	// In a real production system, we might want to log the original error
	// and return a generic "internal server error" message to avoid leaking details.
	// For this implementation, we'll just return the error message.
	return NewError(CodeInternal, err.Error())
}

// HTTPStatus maps an ErrorCode to an HTTP status code.
func (c ErrorCode) HTTPStatus() int {
	switch c {
	case CodeInvalidArgument:
		return http.StatusBadRequest
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodePermissionDenied:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeMethodNotAllowed:
		return http.StatusMethodNotAllowed
	case CodeConflict, CodeAlreadyExists:
		return http.StatusConflict
	case CodeGone:
		return http.StatusGone
	case CodeResourceExhausted:
		return http.StatusTooManyRequests
	case CodeCanceled:
		return 499 // Client Closed Request (Nginx standard)
	case CodeInternal:
		return http.StatusInternalServerError
	case CodeNotImplemented:
		return http.StatusNotImplemented
	case CodeUnavailable:
		return http.StatusServiceUnavailable
	case CodeDeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

func writeError(w http.ResponseWriter, rpcErr *Error, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(rpcErr.Code.HTTPStatus())
	if err := encodeErrorResponse(w, rpcErr); err != nil {
		// Headers already sent, nothing we can do. Log for debugging.
		logger.Error("failed to encode error response",
			slog.String("code", string(rpcErr.Code)),
			slog.String("message", rpcErr.Message),
			slog.Any("error", err))
	}
}
