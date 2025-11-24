package tygor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ErrorCode represents a machine-readable error code.
type ErrorCode string

const (
	CodeInvalidArgument  ErrorCode = "invalid_argument"
	CodeUnauthenticated  ErrorCode = "unauthenticated"
	CodePermissionDenied ErrorCode = "permission_denied"
	CodeNotFound         ErrorCode = "not_found"
	CodeUnavailable      ErrorCode = "unavailable"
	CodeInternal         ErrorCode = "internal"
	CodeCanceled         ErrorCode = "canceled"
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
		return NewError(CodeUnavailable, "context deadline exceeded")
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

// HTTPStatusFromCode maps an ErrorCode to an HTTP status code.
func HTTPStatusFromCode(code ErrorCode) int {
	switch code {
	case CodeInvalidArgument:
		return http.StatusBadRequest
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodePermissionDenied:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeUnavailable:
		return http.StatusServiceUnavailable
	case CodeCanceled:
		return 499 // Client Closed Request (Nginx standard)
	case CodeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func writeError(w http.ResponseWriter, rpcErr *Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(HTTPStatusFromCode(rpcErr.Code))
	json.NewEncoder(w).Encode(rpcErr)
}
