package errors

import (
	"fmt"
	"net/http"
	"time"
)

// ErrorCode represents a standardized error code
type ErrorCode string

// Standard error codes
const (
	ErrTemplateNotFound   ErrorCode = "TEMPLATE_NOT_FOUND"
	ErrInvalidParameters  ErrorCode = "INVALID_PARAMETERS"
	ErrCompilationFailed  ErrorCode = "COMPILATION_FAILED"
	ErrDeviceNotConnected ErrorCode = "DEVICE_NOT_CONNECTED"
	ErrFlashFailed        ErrorCode = "FLASH_FAILED"
	ErrSecretsNotFound    ErrorCode = "SECRETS_NOT_FOUND"
	ErrElectricalSafety   ErrorCode = "ELECTRICAL_SAFETY_VIOLATION"
	ErrUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrForbidden          ErrorCode = "FORBIDDEN"
	ErrInternalServer     ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrBadRequest         ErrorCode = "BAD_REQUEST"
	ErrNotFound           ErrorCode = "NOT_FOUND"
	ErrConflict           ErrorCode = "CONFLICT"
	ErrRateLimited        ErrorCode = "RATE_LIMITED"
)

// APIError represents a structured API error
type APIError struct {
	Code      ErrorCode         `json:"code"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	RequestID string            `json:"request_id,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// HTTPStatus returns the appropriate HTTP status code for the error
func (e *APIError) HTTPStatus() int {
	switch e.Code {
	case ErrBadRequest, ErrInvalidParameters, ErrElectricalSafety:
		return http.StatusBadRequest
	case ErrUnauthorized:
		return http.StatusUnauthorized
	case ErrForbidden:
		return http.StatusForbidden
	case ErrNotFound, ErrTemplateNotFound, ErrSecretsNotFound:
		return http.StatusNotFound
	case ErrConflict:
		return http.StatusConflict
	case ErrRateLimited:
		return http.StatusTooManyRequests
	case ErrServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// New creates a new APIError
func New(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// NewWithDetails creates a new APIError with additional details
func NewWithDetails(code ErrorCode, message string, details map[string]string) *APIError {
	return &APIError{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now().UTC(),
	}
}

// NewWithRequestID creates a new APIError with a request ID
func NewWithRequestID(code ErrorCode, message string, requestID string) *APIError {
	return &APIError{
		Code:      code,
		Message:   message,
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, code ErrorCode, message string) *APIError {
	if apiErr, ok := err.(*APIError); ok {
		// If it's already an APIError, preserve the original code and add context
		return &APIError{
			Code:      apiErr.Code,
			Message:   fmt.Sprintf("%s: %s", message, apiErr.Message),
			Details:   apiErr.Details,
			RequestID: apiErr.RequestID,
			Timestamp: time.Now().UTC(),
		}
	}

	return &APIError{
		Code:      code,
		Message:   fmt.Sprintf("%s: %v", message, err),
		Timestamp: time.Now().UTC(),
	}
}

// IsCode checks if an error has a specific error code
func IsCode(err error, code ErrorCode) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Code == code
	}
	return false
}

// GetCode returns the error code from an error, or empty string if not an APIError
func GetCode(err error) ErrorCode {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Code
	}
	return ""
}

// Common error constructors
func BadRequest(message string) *APIError {
	return New(ErrBadRequest, message)
}

func NotFound(message string) *APIError {
	return New(ErrNotFound, message)
}

func Unauthorized(message string) *APIError {
	return New(ErrUnauthorized, message)
}

func Forbidden(message string) *APIError {
	return New(ErrForbidden, message)
}

func InternalServer(message string) *APIError {
	return New(ErrInternalServer, message)
}

func ServiceUnavailable(message string) *APIError {
	return New(ErrServiceUnavailable, message)
}

func InvalidParameters(message string) *APIError {
	return New(ErrInvalidParameters, message)
}

func TemplateNotFound(templateID string) *APIError {
	return NewWithDetails(ErrTemplateNotFound, "Template not found", map[string]string{
		"template_id": templateID,
	})
}

func DeviceNotConnected(port string) *APIError {
	return NewWithDetails(ErrDeviceNotConnected, "Device not connected", map[string]string{
		"port": port,
	})
}

func CompilationFailed(details string) *APIError {
	return NewWithDetails(ErrCompilationFailed, "Compilation failed", map[string]string{
		"details": details,
	})
}

func ElectricalSafetyViolation(violation string) *APIError {
	return NewWithDetails(ErrElectricalSafety, "Electrical safety violation", map[string]string{
		"violation": violation,
	})
}
