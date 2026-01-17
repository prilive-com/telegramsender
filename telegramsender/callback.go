package telegramsender

import (
	"context"
	"fmt"
)

// AnswerCallbackQueryRequest represents a request to answer a callback query.
type AnswerCallbackQueryRequest struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
	URL             string `json:"url,omitempty"`
	CacheTime       int    `json:"cache_time,omitempty"`
}

// AnswerCallbackQuery answers a callback query from an inline keyboard.
//
// IMPORTANT: Always call this, even without parameters.
// Not answering causes infinite loading indicator on the button.
func (t *TelegramAPI) AnswerCallbackQuery(ctx context.Context, req AnswerCallbackQueryRequest) (bool, error) {
	if err := ValidateConfig(t.config); err != nil {
		return false, fmt.Errorf("config: %w", err)
	}
	if err := ValidateAnswerCallbackQueryRequest(req); err != nil {
		return false, fmt.Errorf("validation: %w", err)
	}

	// No retry for callbacks - they have a time limit
	return t.doAnswerCallback(ctx, req)
}

// Convenience methods

// Answer answers a callback query with optional configuration.
//
// Example usage:
//
//	t.Answer(ctx, callback)                              // Just acknowledge
//	t.Answer(ctx, callback, AnswerText("Done!"))         // With notification
//	t.Answer(ctx, callback, AnswerText("Error!"), Alert) // With alert
func (t *TelegramAPI) Answer(ctx context.Context, cb *CallbackQuery, opts ...AnswerOption) (bool, error) {
	if cb == nil {
		return false, NewValidationError("callback", "nil")
	}

	req := AnswerCallbackQueryRequest{CallbackQueryID: cb.ID}
	for _, opt := range opts {
		opt(&req)
	}

	return t.AnswerCallbackQuery(ctx, req)
}

// Respond is an alias for Answer (telebot-style API).
func (t *TelegramAPI) Respond(ctx context.Context, cb *CallbackQuery, opts ...AnswerOption) (bool, error) {
	return t.Answer(ctx, cb, opts...)
}

// Acknowledge answers a callback without showing any notification.
func (t *TelegramAPI) Acknowledge(ctx context.Context, cb *CallbackQuery) (bool, error) {
	return t.Answer(ctx, cb)
}

// AlertText answers with an alert dialog.
func (t *TelegramAPI) AlertText(ctx context.Context, cb *CallbackQuery, text string) (bool, error) {
	return t.Answer(ctx, cb, AnswerText(text), Alert)
}

// NotifyText answers with a toast notification.
func (t *TelegramAPI) NotifyText(ctx context.Context, cb *CallbackQuery, text string) (bool, error) {
	return t.Answer(ctx, cb, AnswerText(text))
}

// Functional options

// AnswerOption configures AnswerCallbackQueryRequest.
type AnswerOption func(*AnswerCallbackQueryRequest)

// AnswerText sets notification text (max 200 chars).
func AnswerText(text string) AnswerOption {
	return func(r *AnswerCallbackQueryRequest) { r.Text = text }
}

// Alert shows an alert dialog instead of a toast notification.
var Alert AnswerOption = func(r *AnswerCallbackQueryRequest) { r.ShowAlert = true }

// AnswerURL opens a URL (for games or t.me links).
func AnswerURL(url string) AnswerOption {
	return func(r *AnswerCallbackQueryRequest) { r.URL = url }
}

// CacheFor sets how long the result can be cached (in seconds).
func CacheFor(seconds int) AnswerOption {
	return func(r *AnswerCallbackQueryRequest) { r.CacheTime = seconds }
}

// Private implementation

func (t *TelegramAPI) doAnswerCallback(ctx context.Context, req AnswerCallbackQueryRequest) (bool, error) {
	// Global rate limit only - callbacks don't have per-chat limits
	if err := t.globalLimiter.Wait(ctx); err != nil {
		return false, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
	}

	resp, err := t.breaker.Execute(func() (*TelegramResponse, error) {
		return t.executeRequest(ctx, MethodAnswerCallbackQuery, req)
	})
	if err != nil {
		return false, wrapBreakerError(err)
	}

	return parseResponse[bool](resp)
}
