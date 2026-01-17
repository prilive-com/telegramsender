package telegramsender

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/sony/gobreaker/v2"
)

// Request types for edit operations

// EditMessageTextRequest represents a request to edit message text.
type EditMessageTextRequest struct {
	ChatID             ChatID                `json:"chat_id,omitempty"`
	MessageID          int                   `json:"message_id,omitempty"`
	InlineMessageID    string                `json:"inline_message_id,omitempty"`
	Text               string                `json:"text"`
	ParseMode          string                `json:"parse_mode,omitempty"`
	Entities           []MessageEntity       `json:"entities,omitempty"`
	LinkPreviewOptions *LinkPreviewOptions   `json:"link_preview_options,omitempty"`
	ReplyMarkup        *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// LinkPreviewOptions describes link preview generation options.
type LinkPreviewOptions struct {
	IsDisabled       bool   `json:"is_disabled,omitempty"`
	URL              string `json:"url,omitempty"`
	PreferSmallMedia bool   `json:"prefer_small_media,omitempty"`
	PreferLargeMedia bool   `json:"prefer_large_media,omitempty"`
	ShowAboveText    bool   `json:"show_above_text,omitempty"`
}

// EditMessageCaptionRequest represents a request to edit message caption.
type EditMessageCaptionRequest struct {
	ChatID                ChatID                `json:"chat_id,omitempty"`
	MessageID             int                   `json:"message_id,omitempty"`
	InlineMessageID       string                `json:"inline_message_id,omitempty"`
	Caption               string                `json:"caption,omitempty"`
	ParseMode             string                `json:"parse_mode,omitempty"`
	CaptionEntities       []MessageEntity       `json:"caption_entities,omitempty"`
	ShowCaptionAboveMedia bool                  `json:"show_caption_above_media,omitempty"`
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// EditMessageReplyMarkupRequest represents a request to edit reply markup only.
type EditMessageReplyMarkupRequest struct {
	ChatID          ChatID                `json:"chat_id,omitempty"`
	MessageID       int                   `json:"message_id,omitempty"`
	InlineMessageID string                `json:"inline_message_id,omitempty"`
	ReplyMarkup     *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// Public methods

// EditMessageText edits text and optionally the keyboard of a message.
// Returns edited Message, or nil for inline messages.
func (t *TelegramAPI) EditMessageText(ctx context.Context, req EditMessageTextRequest) (*Message, error) {
	if err := ValidateConfig(t.config); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := ValidateEditMessageTextRequest(req); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	chatID := rateLimitChatID(req.ChatID, req.MessageID)
	return withRetry(ctx, t, chatID, func() (*Message, error) {
		return t.doEditText(ctx, req)
	})
}

// EditMessageCaption edits the caption of a message.
// Returns edited Message, or nil for inline messages.
func (t *TelegramAPI) EditMessageCaption(ctx context.Context, req EditMessageCaptionRequest) (*Message, error) {
	if err := ValidateConfig(t.config); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := ValidateEditMessageCaptionRequest(req); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	chatID := rateLimitChatID(req.ChatID, req.MessageID)
	return withRetry(ctx, t, chatID, func() (*Message, error) {
		return t.doEditCaption(ctx, req)
	})
}

// EditMessageReplyMarkup edits only the reply markup of a message.
// Returns edited Message, or nil for inline messages.
func (t *TelegramAPI) EditMessageReplyMarkup(ctx context.Context, req EditMessageReplyMarkupRequest) (*Message, error) {
	if err := ValidateConfig(t.config); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := ValidateEditMessageReplyMarkupRequest(req); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	chatID := rateLimitChatID(req.ChatID, req.MessageID)
	return withRetry(ctx, t, chatID, func() (*Message, error) {
		return t.doEditReplyMarkup(ctx, req)
	})
}

// Convenience methods using Editable interface

// Edit edits a message using the Editable interface.
func (t *TelegramAPI) Edit(ctx context.Context, msg Editable, text string, opts ...EditOption) (*Message, error) {
	if msg == nil {
		return nil, NewValidationError("msg", "nil")
	}

	msgID, chatID := msg.MessageSig()
	req := EditMessageTextRequest{Text: text}

	if chatID == 0 {
		req.InlineMessageID = msgID
	} else {
		req.ChatID = chatID
		req.MessageID, _ = strconv.Atoi(msgID)
	}

	for _, opt := range opts {
		opt(&req)
	}

	return t.EditMessageText(ctx, req)
}

// EditCaption edits a message caption using the Editable interface.
func (t *TelegramAPI) EditCaption(ctx context.Context, msg Editable, caption string, opts ...EditCaptionOption) (*Message, error) {
	if msg == nil {
		return nil, NewValidationError("msg", "nil")
	}

	msgID, chatID := msg.MessageSig()
	req := EditMessageCaptionRequest{Caption: caption}

	if chatID == 0 {
		req.InlineMessageID = msgID
	} else {
		req.ChatID = chatID
		req.MessageID, _ = strconv.Atoi(msgID)
	}

	for _, opt := range opts {
		opt(&req)
	}

	return t.EditMessageCaption(ctx, req)
}

// EditReplyMarkup edits only the reply markup using the Editable interface.
func (t *TelegramAPI) EditReplyMarkup(ctx context.Context, msg Editable, markup *InlineKeyboardMarkup) (*Message, error) {
	if msg == nil {
		return nil, NewValidationError("msg", "nil")
	}

	msgID, chatID := msg.MessageSig()
	req := EditMessageReplyMarkupRequest{ReplyMarkup: markup}

	if chatID == 0 {
		req.InlineMessageID = msgID
	} else {
		req.ChatID = chatID
		req.MessageID, _ = strconv.Atoi(msgID)
	}

	return t.EditMessageReplyMarkup(ctx, req)
}

// Functional options

// EditOption configures EditMessageTextRequest.
type EditOption func(*EditMessageTextRequest)

// WithEditParseMode sets parse mode.
func WithEditParseMode(mode string) EditOption {
	return func(r *EditMessageTextRequest) { r.ParseMode = mode }
}

// WithEditKeyboard sets inline keyboard.
func WithEditKeyboard(kb *InlineKeyboardMarkup) EditOption {
	return func(r *EditMessageTextRequest) { r.ReplyMarkup = kb }
}

// WithLinkPreview configures link preview options.
func WithLinkPreview(opts LinkPreviewOptions) EditOption {
	return func(r *EditMessageTextRequest) { r.LinkPreviewOptions = &opts }
}

// WithoutLinkPreview disables link preview.
func WithoutLinkPreview() EditOption {
	return func(r *EditMessageTextRequest) {
		r.LinkPreviewOptions = &LinkPreviewOptions{IsDisabled: true}
	}
}

// WithEditEntities sets message entities.
func WithEditEntities(entities []MessageEntity) EditOption {
	return func(r *EditMessageTextRequest) { r.Entities = entities }
}

// EditCaptionOption configures EditMessageCaptionRequest.
type EditCaptionOption func(*EditMessageCaptionRequest)

// WithCaptionParseMode sets parse mode for caption.
func WithCaptionParseMode(mode string) EditCaptionOption {
	return func(r *EditMessageCaptionRequest) { r.ParseMode = mode }
}

// WithCaptionKeyboard sets inline keyboard for caption edit.
func WithCaptionKeyboard(kb *InlineKeyboardMarkup) EditCaptionOption {
	return func(r *EditMessageCaptionRequest) { r.ReplyMarkup = kb }
}

// WithCaptionEntities sets caption entities.
func WithCaptionEntities(entities []MessageEntity) EditCaptionOption {
	return func(r *EditMessageCaptionRequest) { r.CaptionEntities = entities }
}

// Private implementation methods

func (t *TelegramAPI) doEditText(ctx context.Context, req EditMessageTextRequest) (*Message, error) {
	chatID := rateLimitChatID(req.ChatID, req.MessageID)

	if chatID != 0 {
		if err := t.waitForChatRateLimit(ctx, chatID); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
		}
	}

	if err := t.globalLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
	}

	resp, err := t.breaker.Execute(func() (*TelegramResponse, error) {
		return t.executeRequest(ctx, MethodEditMessageText, req)
	})
	if err != nil {
		return nil, wrapBreakerError(err)
	}

	return parseMessageOrTrue(resp)
}

func (t *TelegramAPI) doEditCaption(ctx context.Context, req EditMessageCaptionRequest) (*Message, error) {
	chatID := rateLimitChatID(req.ChatID, req.MessageID)

	if chatID != 0 {
		if err := t.waitForChatRateLimit(ctx, chatID); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
		}
	}

	if err := t.globalLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
	}

	resp, err := t.breaker.Execute(func() (*TelegramResponse, error) {
		return t.executeRequest(ctx, MethodEditMessageCaption, req)
	})
	if err != nil {
		return nil, wrapBreakerError(err)
	}

	return parseMessageOrTrue(resp)
}

func (t *TelegramAPI) doEditReplyMarkup(ctx context.Context, req EditMessageReplyMarkupRequest) (*Message, error) {
	chatID := rateLimitChatID(req.ChatID, req.MessageID)

	if chatID != 0 {
		if err := t.waitForChatRateLimit(ctx, chatID); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
		}
	}

	if err := t.globalLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
	}

	resp, err := t.breaker.Execute(func() (*TelegramResponse, error) {
		return t.executeRequest(ctx, MethodEditMessageReplyMarkup, req)
	})
	if err != nil {
		return nil, wrapBreakerError(err)
	}

	return parseMessageOrTrue(resp)
}

func wrapBreakerError(err error) error {
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		return fmt.Errorf("%w: %w", ErrCircuitBreakerOpen, err)
	}
	return err
}
