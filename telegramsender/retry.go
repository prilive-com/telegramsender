package telegramsender

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// withRetry executes an operation with retry logic and exponential backoff.
// Uses Go generics to handle any return type.
func withRetry[T any](ctx context.Context, t *TelegramAPI, chatID int64, op func() (T, error)) (T, error) {
	var result T
	var err error

	for attempt := range t.config.MaxRetries + 1 {
		result, err = op()
		if err == nil {
			return result, nil
		}

		// Last attempt - don't retry
		if attempt == t.config.MaxRetries {
			break
		}

		// Non-retryable error
		if !t.isRetryable(err) {
			t.logger.Error("non-retryable error", "error", err, "attempt", attempt)
			return result, err
		}

		// Calculate backoff
		backoff := t.backoffDuration(attempt+1, err)

		t.logger.Info("retrying", "attempt", attempt+1, "backoff", backoff, "chat_id", chatID)

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(backoff):
		}
	}

	return result, fmt.Errorf("%w: %w", ErrMaxRetriesExceeded, err)
}

// backoffDuration calculates the backoff duration, respecting Retry-After header.
func (t *TelegramAPI) backoffDuration(attempt int, err error) time.Duration {
	var telegramErr *TelegramError
	if errors.As(err, &telegramErr) && telegramErr.RetryAfter > 0 {
		return telegramErr.RetryAfter
	}
	return t.calculateBackoff(attempt)
}

// Response parsing helpers

// parseResponse parses TelegramResponse into the target type.
func parseResponse[T any](resp *TelegramResponse) (T, error) {
	var result T

	if !resp.OK {
		return result, NewTelegramError(resp.ErrorCode, resp.Description)
	}

	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return result, fmt.Errorf("parse response: %w", err)
	}

	return result, nil
}

// parseMessageOrTrue parses response that returns either Message or true (for inline messages).
func parseMessageOrTrue(resp *TelegramResponse) (*Message, error) {
	if !resp.OK {
		return nil, NewTelegramError(resp.ErrorCode, resp.Description)
	}

	// Inline messages return true instead of Message
	if bytes.Equal(resp.Result, []byte("true")) {
		return nil, nil
	}

	var msg Message
	if err := json.Unmarshal(resp.Result, &msg); err != nil {
		return nil, fmt.Errorf("parse message: %w", err)
	}

	return &msg, nil
}

// ChatID helpers

// chatID64 extracts int64 from ChatID for rate limiting.
func chatID64(id ChatID) int64 {
	switch v := id.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// rateLimitChatID returns the chat ID for rate limiting purposes.
// Returns 0 for inline messages (no per-chat rate limit).
func rateLimitChatID(chatID ChatID, messageID int) int64 {
	if messageID == 0 {
		return 0 // Inline message
	}
	return chatID64(chatID)
}
