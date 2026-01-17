package telegramsender

import (
	"context"
	"net/http"
)

// Sender sends messages (original interface for backward compatibility).
type Sender interface {
	SendMessage(ctx context.Context, req MessageRequest) (*MessageResult, error)
	SendPhoto(ctx context.Context, req PhotoRequest) (*MessageResult, error)
	SendPhotoFile(ctx context.Context, req PhotoFileRequest) (*MessageResult, error)
}

// Editor edits messages.
type Editor interface {
	EditMessageText(ctx context.Context, req EditMessageTextRequest) (*Message, error)
	EditMessageCaption(ctx context.Context, req EditMessageCaptionRequest) (*Message, error)
	EditMessageReplyMarkup(ctx context.Context, req EditMessageReplyMarkupRequest) (*Message, error)

	// Convenience methods
	Edit(ctx context.Context, msg Editable, text string, opts ...EditOption) (*Message, error)
	EditCaption(ctx context.Context, msg Editable, caption string, opts ...EditCaptionOption) (*Message, error)
	EditReplyMarkup(ctx context.Context, msg Editable, markup *InlineKeyboardMarkup) (*Message, error)
}

// Manager manages messages (delete, forward, copy).
type Manager interface {
	DeleteMessage(ctx context.Context, req DeleteMessageRequest) (bool, error)
	ForwardMessage(ctx context.Context, req ForwardMessageRequest) (*Message, error)
	CopyMessage(ctx context.Context, req CopyMessageRequest) (*MessageID, error)

	// Convenience methods
	Delete(ctx context.Context, msg Editable) (bool, error)
	Forward(ctx context.Context, msg Editable, to ChatID, opts ...ForwardOption) (*Message, error)
	Copy(ctx context.Context, msg Editable, to ChatID, opts ...CopyOption) (*MessageID, error)
}

// Responder answers callback queries.
type Responder interface {
	AnswerCallbackQuery(ctx context.Context, req AnswerCallbackQueryRequest) (bool, error)

	// Convenience methods
	Answer(ctx context.Context, cb *CallbackQuery, opts ...AnswerOption) (bool, error)
	Respond(ctx context.Context, cb *CallbackQuery, opts ...AnswerOption) (bool, error)
	Acknowledge(ctx context.Context, cb *CallbackQuery) (bool, error)
}

// Bot combines all Telegram Bot API interfaces.
type Bot interface {
	Sender
	Editor
	Manager
	Responder
}

// Compile-time interface checks
var (
	_ Sender    = (*TelegramAPI)(nil)
	_ Editor    = (*TelegramAPI)(nil)
	_ Manager   = (*TelegramAPI)(nil)
	_ Responder = (*TelegramAPI)(nil)
	_ Bot       = (*TelegramAPI)(nil)
)

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
