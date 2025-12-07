package telegramsender

import (
	"context"
	"net/http"
)

// Sender defines the interface for sending Telegram messages.
// This interface allows for easy mocking in tests.
type Sender interface {
	// SendMessage sends a text message to the specified chat
	SendMessage(ctx context.Context, request MessageRequest) (*MessageResult, error)

	// SendPhoto sends a photo by URL or file_id to the specified chat
	SendPhoto(ctx context.Context, request PhotoRequest) (*MessageResult, error)

	// SendPhotoFile sends a local photo file to the specified chat
	SendPhotoFile(ctx context.Context, request PhotoFileRequest) (*MessageResult, error)
}

// Ensure TelegramAPI implements Sender at compile time
var _ Sender = (*TelegramAPI)(nil)

// HTTPClient is an interface for HTTP client operations.
// This allows for mocking HTTP calls in tests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Ensure http.Client implements HTTPClient
var _ HTTPClient = (*http.Client)(nil)

// RateLimiter is an interface for rate limiting operations.
type RateLimiter interface {
	Wait(ctx context.Context) error
}

// CircuitBreaker is an interface for circuit breaker operations.
type CircuitBreaker interface {
	Execute(req func() (interface{}, error)) (interface{}, error)
}
