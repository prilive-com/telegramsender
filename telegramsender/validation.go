package telegramsender

import (
	"fmt"
	"unicode/utf8"
)

// Validation for edit requests

// ValidateEditMessageTextRequest validates EditMessageTextRequest.
func ValidateEditMessageTextRequest(req EditMessageTextRequest) error {
	if err := validateMessageTarget(req.ChatID, req.MessageID, req.InlineMessageID); err != nil {
		return err
	}

	if req.Text == "" {
		return NewValidationError("text", "required")
	}

	if n := utf8.RuneCountInString(req.Text); n > MaxMessageLength {
		return NewValidationError("text", fmt.Sprintf("too long: %d > %d", n, MaxMessageLength))
	}

	return validateParseMode(req.ParseMode)
}

// ValidateEditMessageCaptionRequest validates EditMessageCaptionRequest.
func ValidateEditMessageCaptionRequest(req EditMessageCaptionRequest) error {
	if err := validateMessageTarget(req.ChatID, req.MessageID, req.InlineMessageID); err != nil {
		return err
	}

	if req.Caption != "" {
		if n := utf8.RuneCountInString(req.Caption); n > MaxCaptionLengthDefault {
			return NewValidationError("caption", fmt.Sprintf("too long: %d > %d", n, MaxCaptionLengthDefault))
		}
	}

	return validateParseMode(req.ParseMode)
}

// ValidateEditMessageReplyMarkupRequest validates EditMessageReplyMarkupRequest.
func ValidateEditMessageReplyMarkupRequest(req EditMessageReplyMarkupRequest) error {
	return validateMessageTarget(req.ChatID, req.MessageID, req.InlineMessageID)
}

// Validation for message management requests

// ValidateDeleteMessageRequest validates DeleteMessageRequest.
func ValidateDeleteMessageRequest(req DeleteMessageRequest) error {
	if req.ChatID == nil {
		return NewValidationError("chat_id", "required")
	}
	if req.MessageID <= 0 {
		return NewValidationError("message_id", "must be positive")
	}
	return nil
}

// ValidateForwardMessageRequest validates ForwardMessageRequest.
func ValidateForwardMessageRequest(req ForwardMessageRequest) error {
	if req.ChatID == nil {
		return NewValidationError("chat_id", "required")
	}
	if req.FromChatID == nil {
		return NewValidationError("from_chat_id", "required")
	}
	if req.MessageID <= 0 {
		return NewValidationError("message_id", "must be positive")
	}
	return nil
}

// ValidateCopyMessageRequest validates CopyMessageRequest.
func ValidateCopyMessageRequest(req CopyMessageRequest) error {
	if req.ChatID == nil {
		return NewValidationError("chat_id", "required")
	}
	if req.FromChatID == nil {
		return NewValidationError("from_chat_id", "required")
	}
	if req.MessageID <= 0 {
		return NewValidationError("message_id", "must be positive")
	}

	if req.Caption != "" {
		if n := utf8.RuneCountInString(req.Caption); n > MaxCaptionLengthDefault {
			return NewValidationError("caption", fmt.Sprintf("too long: %d > %d", n, MaxCaptionLengthDefault))
		}
	}

	return validateParseMode(req.ParseMode)
}

// Validation for callback requests

// ValidateAnswerCallbackQueryRequest validates AnswerCallbackQueryRequest.
func ValidateAnswerCallbackQueryRequest(req AnswerCallbackQueryRequest) error {
	if req.CallbackQueryID == "" {
		return NewValidationError("callback_query_id", "required")
	}

	if req.Text != "" {
		if n := utf8.RuneCountInString(req.Text); n > MaxCallbackAnswerTextLength {
			return NewValidationError("text", fmt.Sprintf("too long: %d > %d", n, MaxCallbackAnswerTextLength))
		}
	}

	return nil
}

// Shared validation helpers

func validateMessageTarget(chatID ChatID, messageID int, inlineMessageID string) error {
	hasChat := chatID != nil && messageID > 0
	hasInline := inlineMessageID != ""

	switch {
	case !hasChat && !hasInline:
		return NewValidationError("target", "chat_id+message_id or inline_message_id required")
	case hasChat && hasInline:
		return NewValidationError("target", "cannot specify both chat_id and inline_message_id")
	}

	return nil
}

func validateParseMode(mode string) error {
	switch mode {
	case "", ParseModeHTML, ParseModeMarkdown, ParseModeMarkdownV2:
		return nil
	default:
		return NewValidationError("parse_mode", "must be HTML, Markdown, or MarkdownV2")
	}
}
