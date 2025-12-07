package telegramsender

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors for common error conditions
var (
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrCircuitBreakerOpen = errors.New("circuit breaker open")
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	ErrInvalidRequest     = errors.New("invalid request")
	ErrPathTraversal      = errors.New("path traversal detected")
	ErrResponseTooLarge   = errors.New("response exceeds maximum allowed size")
	ErrInvalidBaseURL     = errors.New("invalid base URL")
)

// TelegramError represents an error response from the Telegram API.
// Implements the error interface so it can be used with errors.As().
type TelegramError struct {
	Code        int
	Description string
	RetryAfter  time.Duration
}

// Error implements the error interface
func (e *TelegramError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("telegram API error %d: %s (retry after %s)", e.Code, e.Description, e.RetryAfter)
	}
	return fmt.Sprintf("telegram API error %d: %s", e.Code, e.Description)
}

// Is implements errors.Is for TelegramError
func (e *TelegramError) Is(target error) bool {
	t, ok := target.(*TelegramError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Unwrap returns nil as TelegramError doesn't wrap other errors
func (e *TelegramError) Unwrap() error {
	return nil
}

// IsRetryable returns true if the error indicates a temporary condition
// that may succeed on retry
func (e *TelegramError) IsRetryable() bool {
	// 429 - Too Many Requests
	// 500, 502, 503, 504 - Server errors
	return e.Code == 429 || (e.Code >= 500 && e.Code <= 504)
}

// NewTelegramError creates a new TelegramError
func NewTelegramError(code int, description string) *TelegramError {
	return &TelegramError{
		Code:        code,
		Description: description,
	}
}

// NewTelegramErrorWithRetry creates a new TelegramError with retry information
func NewTelegramErrorWithRetry(code int, description string, retryAfter time.Duration) *TelegramError {
	return &TelegramError{
		Code:        code,
		Description: description,
		RetryAfter:  retryAfter,
	}
}

// ValidationError represents a validation error with field information
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// Is implements errors.Is for ValidationError
func (e *ValidationError) Is(target error) bool {
	_, ok := target.(*ValidationError)
	return ok
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// ConfigError represents a configuration error
type ConfigError struct {
	Key     string
	Message string
}

// Error implements the error interface
func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error for %s: %s", e.Key, e.Message)
}

// Is implements errors.Is for ConfigError
func (e *ConfigError) Is(target error) bool {
	_, ok := target.(*ConfigError)
	return ok
}

// NewConfigError creates a new ConfigError
func NewConfigError(key, message string) *ConfigError {
	return &ConfigError{
		Key:     key,
		Message: message,
	}
}
