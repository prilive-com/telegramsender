package telegramsender

import (
	"errors"
	"fmt"
	"strings"
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

// Sentinel errors for message operations
var (
	ErrMessageNotFound      = errors.New("message not found")
	ErrMessageNotModified   = errors.New("message is not modified")
	ErrMessageCantBeEdited  = errors.New("message can't be edited")
	ErrMessageCantBeDeleted = errors.New("message can't be deleted")
	ErrMessageTooOld        = errors.New("message is too old")
)

// Sentinel errors for callback operations
var (
	ErrInvalidCallbackData  = errors.New("invalid callback data")
	ErrCallbackQueryExpired = errors.New("callback query expired")
)

// Sentinel errors for chat/user operations
var (
	ErrChatNotFound    = errors.New("chat not found")
	ErrBotKicked       = errors.New("bot was kicked from chat")
	ErrBotBlocked      = errors.New("bot was blocked by user")
	ErrUserDeactivated = errors.New("user is deactivated")
	ErrNoRights        = errors.New("not enough rights")
)

// TelegramError represents an error response from the Telegram API.
type TelegramError struct {
	Code        int
	Description string
	RetryAfter  time.Duration
	Cause       error // Underlying sentinel error for errors.Is() support
}

func (e *TelegramError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("telegram: %s (code=%d, retry_after=%s)", e.Description, e.Code, e.RetryAfter)
	}
	return fmt.Sprintf("telegram: %s (code=%d)", e.Description, e.Code)
}

// Unwrap returns the underlying sentinel error, enabling errors.Is() matching.
func (e *TelegramError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns true if the error is temporary and may succeed on retry.
func (e *TelegramError) IsRetryable() bool {
	switch {
	case e.Code == 429: // Too Many Requests
		return true
	case e.Code >= 500 && e.Code <= 504: // Server errors
		return true
	default:
		return false
	}
}

// NewTelegramError creates a TelegramError with automatic sentinel detection.
func NewTelegramError(code int, description string) *TelegramError {
	return &TelegramError{
		Code:        code,
		Description: description,
		Cause:       detectSentinel(description),
	}
}

// NewTelegramErrorWithRetry creates a TelegramError with retry information.
func NewTelegramErrorWithRetry(code int, description string, retryAfter time.Duration) *TelegramError {
	return &TelegramError{
		Code:        code,
		Description: description,
		RetryAfter:  retryAfter,
		Cause:       detectSentinel(description),
	}
}

// detectSentinel maps Telegram API error descriptions to sentinel errors.
func detectSentinel(description string) error {
	desc := strings.ToLower(description)

	// Message errors
	switch {
	case strings.Contains(desc, "message to edit not found"),
		strings.Contains(desc, "message to delete not found"),
		strings.Contains(desc, "message not found"):
		return ErrMessageNotFound

	case strings.Contains(desc, "message is not modified"):
		return ErrMessageNotModified

	case strings.Contains(desc, "message can't be edited"):
		return ErrMessageCantBeEdited

	case strings.Contains(desc, "message can't be deleted"):
		return ErrMessageCantBeDeleted

	case strings.Contains(desc, "message is too old"):
		return ErrMessageTooOld

	// Callback errors
	case strings.Contains(desc, "button_data_invalid"):
		return ErrInvalidCallbackData

	case strings.Contains(desc, "query is too old"):
		return ErrCallbackQueryExpired

	// Chat/user errors
	case strings.Contains(desc, "chat not found"):
		return ErrChatNotFound

	case strings.Contains(desc, "bot was kicked"):
		return ErrBotKicked

	case strings.Contains(desc, "bot was blocked"):
		return ErrBotBlocked

	case strings.Contains(desc, "user is deactivated"):
		return ErrUserDeactivated

	case strings.Contains(desc, "not enough rights"):
		return ErrNoRights
	}

	return nil
}

// ValidationError represents a request validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s - %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Key     string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config: %s - %s", e.Key, e.Message)
}

// NewConfigError creates a new ConfigError.
func NewConfigError(key, message string) *ConfigError {
	return &ConfigError{Key: key, Message: message}
}
