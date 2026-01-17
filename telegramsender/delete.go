package telegramsender

import (
	"context"
	"fmt"
	"strconv"
)

// Request types

// DeleteMessageRequest represents a request to delete a message.
type DeleteMessageRequest struct {
	ChatID    ChatID `json:"chat_id"`
	MessageID int    `json:"message_id"`
}

// ForwardMessageRequest represents a request to forward a message.
type ForwardMessageRequest struct {
	ChatID              ChatID `json:"chat_id"`
	FromChatID          ChatID `json:"from_chat_id"`
	MessageID           int    `json:"message_id"`
	MessageThreadID     int    `json:"message_thread_id,omitempty"`
	DisableNotification bool   `json:"disable_notification,omitempty"`
	ProtectContent      bool   `json:"protect_content,omitempty"`
}

// CopyMessageRequest represents a request to copy a message.
type CopyMessageRequest struct {
	ChatID                   ChatID                `json:"chat_id"`
	FromChatID               ChatID                `json:"from_chat_id"`
	MessageID                int                   `json:"message_id"`
	MessageThreadID          int                   `json:"message_thread_id,omitempty"`
	Caption                  string                `json:"caption,omitempty"`
	ParseMode                string                `json:"parse_mode,omitempty"`
	CaptionEntities          []MessageEntity       `json:"caption_entities,omitempty"`
	ShowCaptionAboveMedia    bool                  `json:"show_caption_above_media,omitempty"`
	DisableNotification      bool                  `json:"disable_notification,omitempty"`
	ProtectContent           bool                  `json:"protect_content,omitempty"`
	ReplyToMessageID         int                   `json:"reply_to_message_id,omitempty"`
	AllowSendingWithoutReply bool                  `json:"allow_sending_without_reply,omitempty"`
	ReplyMarkup              *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// Public methods

// DeleteMessage deletes a message.
// Messages can only be deleted if sent less than 48 hours ago.
func (t *TelegramAPI) DeleteMessage(ctx context.Context, req DeleteMessageRequest) (bool, error) {
	if err := ValidateConfig(t.config); err != nil {
		return false, fmt.Errorf("config: %w", err)
	}
	if err := ValidateDeleteMessageRequest(req); err != nil {
		return false, fmt.Errorf("validation: %w", err)
	}

	chatID := chatID64(req.ChatID)
	return withRetry(ctx, t, chatID, func() (bool, error) {
		return t.doDelete(ctx, req)
	})
}

// ForwardMessage forwards a message to another chat.
// Service messages and protected content can't be forwarded.
func (t *TelegramAPI) ForwardMessage(ctx context.Context, req ForwardMessageRequest) (*Message, error) {
	if err := ValidateConfig(t.config); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := ValidateForwardMessageRequest(req); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	chatID := chatID64(req.ChatID)
	return withRetry(ctx, t, chatID, func() (*Message, error) {
		return t.doForward(ctx, req)
	})
}

// CopyMessage copies a message without the "forwarded from" header.
// Returns the new message's ID.
func (t *TelegramAPI) CopyMessage(ctx context.Context, req CopyMessageRequest) (*MessageID, error) {
	if err := ValidateConfig(t.config); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := ValidateCopyMessageRequest(req); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	chatID := chatID64(req.ChatID)
	return withRetry(ctx, t, chatID, func() (*MessageID, error) {
		return t.doCopy(ctx, req)
	})
}

// Convenience methods using Editable interface

// Delete deletes a message using the Editable interface.
func (t *TelegramAPI) Delete(ctx context.Context, msg Editable) (bool, error) {
	if msg == nil {
		return false, NewValidationError("msg", "nil")
	}

	msgIDStr, chatID := msg.MessageSig()
	if chatID == 0 {
		return false, NewValidationError("msg", "cannot delete inline messages")
	}

	msgID, _ := strconv.Atoi(msgIDStr)
	return t.DeleteMessage(ctx, DeleteMessageRequest{
		ChatID:    chatID,
		MessageID: msgID,
	})
}

// Forward forwards a message using the Editable interface.
func (t *TelegramAPI) Forward(ctx context.Context, msg Editable, to ChatID, opts ...ForwardOption) (*Message, error) {
	if msg == nil {
		return nil, NewValidationError("msg", "nil")
	}

	msgIDStr, fromChatID := msg.MessageSig()
	if fromChatID == 0 {
		return nil, NewValidationError("msg", "cannot forward inline messages")
	}

	msgID, _ := strconv.Atoi(msgIDStr)
	req := ForwardMessageRequest{
		ChatID:     to,
		FromChatID: fromChatID,
		MessageID:  msgID,
	}

	for _, opt := range opts {
		opt(&req)
	}

	return t.ForwardMessage(ctx, req)
}

// Copy copies a message using the Editable interface.
func (t *TelegramAPI) Copy(ctx context.Context, msg Editable, to ChatID, opts ...CopyOption) (*MessageID, error) {
	if msg == nil {
		return nil, NewValidationError("msg", "nil")
	}

	msgIDStr, fromChatID := msg.MessageSig()
	if fromChatID == 0 {
		return nil, NewValidationError("msg", "cannot copy inline messages")
	}

	msgID, _ := strconv.Atoi(msgIDStr)
	req := CopyMessageRequest{
		ChatID:     to,
		FromChatID: fromChatID,
		MessageID:  msgID,
	}

	for _, opt := range opts {
		opt(&req)
	}

	return t.CopyMessage(ctx, req)
}

// Functional options

// ForwardOption configures ForwardMessageRequest.
type ForwardOption func(*ForwardMessageRequest)

// Silent disables notification for forwarded message.
func Silent() ForwardOption {
	return func(r *ForwardMessageRequest) { r.DisableNotification = true }
}

// Protected protects forwarded content from further forwarding/saving.
func Protected() ForwardOption {
	return func(r *ForwardMessageRequest) { r.ProtectContent = true }
}

// InThread sends to a specific forum thread.
func InThread(threadID int) ForwardOption {
	return func(r *ForwardMessageRequest) { r.MessageThreadID = threadID }
}

// CopyOption configures CopyMessageRequest.
type CopyOption func(*CopyMessageRequest)

// WithCopyCaption sets new caption for copied message.
func WithCopyCaption(caption string) CopyOption {
	return func(r *CopyMessageRequest) { r.Caption = caption }
}

// WithCopyParseMode sets parse mode for caption.
func WithCopyParseMode(mode string) CopyOption {
	return func(r *CopyMessageRequest) { r.ParseMode = mode }
}

// CopySilent disables notification for copied message.
func CopySilent() CopyOption {
	return func(r *CopyMessageRequest) { r.DisableNotification = true }
}

// CopyProtected protects copied content.
func CopyProtected() CopyOption {
	return func(r *CopyMessageRequest) { r.ProtectContent = true }
}

// ReplyTo makes copied message a reply to another message.
func ReplyTo(messageID int) CopyOption {
	return func(r *CopyMessageRequest) { r.ReplyToMessageID = messageID }
}

// WithCopyKeyboard sets inline keyboard for copied message.
func WithCopyKeyboard(kb *InlineKeyboardMarkup) CopyOption {
	return func(r *CopyMessageRequest) { r.ReplyMarkup = kb }
}

// CopyInThread sends copied message to a specific forum thread.
func CopyInThread(threadID int) CopyOption {
	return func(r *CopyMessageRequest) { r.MessageThreadID = threadID }
}

// Private implementation methods

func (t *TelegramAPI) doDelete(ctx context.Context, req DeleteMessageRequest) (bool, error) {
	chatID := chatID64(req.ChatID)

	if err := t.waitForChatRateLimit(ctx, chatID); err != nil {
		return false, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
	}

	if err := t.globalLimiter.Wait(ctx); err != nil {
		return false, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
	}

	resp, err := t.breaker.Execute(func() (*TelegramResponse, error) {
		return t.executeRequest(ctx, MethodDeleteMessage, req)
	})
	if err != nil {
		return false, wrapBreakerError(err)
	}

	return parseResponse[bool](resp)
}

func (t *TelegramAPI) doForward(ctx context.Context, req ForwardMessageRequest) (*Message, error) {
	chatID := chatID64(req.ChatID)

	if err := t.waitForChatRateLimit(ctx, chatID); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
	}

	if err := t.globalLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
	}

	resp, err := t.breaker.Execute(func() (*TelegramResponse, error) {
		return t.executeRequest(ctx, MethodForwardMessage, req)
	})
	if err != nil {
		return nil, wrapBreakerError(err)
	}

	return parseResponse[*Message](resp)
}

func (t *TelegramAPI) doCopy(ctx context.Context, req CopyMessageRequest) (*MessageID, error) {
	chatID := chatID64(req.ChatID)

	if err := t.waitForChatRateLimit(ctx, chatID); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
	}

	if err := t.globalLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRateLimitExceeded, err)
	}

	resp, err := t.breaker.Execute(func() (*TelegramResponse, error) {
		return t.executeRequest(ctx, MethodCopyMessage, req)
	})
	if err != nil {
		return nil, wrapBreakerError(err)
	}

	return parseResponse[*MessageID](resp)
}
