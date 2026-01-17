package telegramsender

import "time"

// Telegram API limits
const (
	// MaxMessageLength is the maximum length of a text message in UTF-8 characters
	MaxMessageLength = 4096

	// MaxCaptionLengthDefault is the default maximum length of a photo caption in UTF-8 characters
	MaxCaptionLengthDefault = 1024

	// MaxFileSizeDefault is the default maximum file size in bytes (10MB)
	MaxFileSizeDefault = 10 * 1024 * 1024

	// MaxResponseSize is the maximum size of an API response we'll read (10MB)
	MaxResponseSize = 10 * 1024 * 1024

	// MaxCallbackDataLength is the maximum length of callback_data in bytes (Telegram limit)
	MaxCallbackDataLength = 64

	// MaxCallbackAnswerTextLength is the maximum length of callback answer text (Telegram limit)
	MaxCallbackAnswerTextLength = 200

	// MaxInlineKeyboardRows is the maximum number of rows in inline keyboard (Telegram limit)
	MaxInlineKeyboardRows = 100

	// MaxInlineKeyboardButtonsPerRow is the maximum buttons per row (Telegram limit)
	MaxInlineKeyboardButtonsPerRow = 8
)

// HTTP client defaults
const (
	// DefaultDialTimeout is the timeout for establishing a connection
	DefaultDialTimeout = 30 * time.Second

	// DefaultTLSHandshakeTimeout is the timeout for TLS handshake
	DefaultTLSHandshakeTimeout = 10 * time.Second

	// DefaultRequestTimeout is the default timeout for HTTP requests
	DefaultRequestTimeout = 10 * time.Second

	// DefaultKeepAlive is the default HTTP keep-alive duration
	DefaultKeepAlive = 30 * time.Second

	// DefaultMaxIdleConns is the default maximum idle connections
	DefaultMaxIdleConns = 10

	// DefaultIdleConnTimeout is the default idle connection timeout
	DefaultIdleConnTimeout = 90 * time.Second
)

// Rate limiting defaults
const (
	// DefaultRateLimitRequests is the default requests per second
	DefaultRateLimitRequests = 10.0

	// DefaultRateLimitBurst is the default burst size
	DefaultRateLimitBurst = 20

	// PerChatRateLimit is the rate limit per chat (1 message per second)
	PerChatRateLimit = 1.0

	// PerChatBurst is the burst size per chat
	PerChatBurst = 3
)

// Circuit breaker defaults
const (
	// DefaultBreakerMaxRequests is requests allowed in half-open state
	DefaultBreakerMaxRequests = 5

	// DefaultBreakerInterval is the window that resets failure counters
	DefaultBreakerInterval = 2 * time.Minute

	// DefaultBreakerTimeout is how long the breaker stays open
	DefaultBreakerTimeout = 60 * time.Second

	// BreakerMinRequests is minimum requests before evaluating error rate
	BreakerMinRequests = 5

	// BreakerErrorThreshold is the error rate threshold to trip the breaker (50%)
	BreakerErrorThreshold = 0.5
)

// Retry defaults
const (
	// DefaultMaxRetries is the default maximum number of retries
	DefaultMaxRetries = 3

	// DefaultRetryInitialBackoff is the initial backoff duration
	DefaultRetryInitialBackoff = 100 * time.Millisecond

	// DefaultRetryMaxBackoff is the maximum backoff duration
	DefaultRetryMaxBackoff = 10 * time.Second

	// DefaultRetryBackoffFactor is the multiplier for exponential backoff
	DefaultRetryBackoffFactor = 2.0

	// JitterMin is the minimum jitter factor (80%)
	JitterMin = 0.8

	// JitterMax is the maximum jitter factor (120%)
	JitterMax = 1.2
)

// File handling
const (
	// FileReadBufferSize is the buffer size for reading files (32KB)
	FileReadBufferSize = 32 * 1024
)

// Telegram API endpoints
const (
	// DefaultBaseURL is the default Telegram Bot API base URL
	DefaultBaseURL = "https://api.telegram.org"
)

// AllowedBaseURLHosts is the whitelist of allowed hosts for the Telegram API
var AllowedBaseURLHosts = []string{
	"api.telegram.org",
}

// Telegram API method names
const (
	// Send methods
	MethodSendMessage = "sendMessage"
	MethodSendPhoto   = "sendPhoto"

	// Edit methods
	MethodEditMessageText        = "editMessageText"
	MethodEditMessageCaption     = "editMessageCaption"
	MethodEditMessageMedia       = "editMessageMedia"
	MethodEditMessageReplyMarkup = "editMessageReplyMarkup"

	// Message management
	MethodDeleteMessage  = "deleteMessage"
	MethodForwardMessage = "forwardMessage"
	MethodCopyMessage    = "copyMessage"

	// Callback
	MethodAnswerCallbackQuery = "answerCallbackQuery"
)

// ParseMode constants for message formatting
const (
	ParseModeHTML       = "HTML"
	ParseModeMarkdown   = "Markdown"
	ParseModeMarkdownV2 = "MarkdownV2"
)

// ChatType constants
const (
	ChatTypePrivate    = "private"
	ChatTypeGroup      = "group"
	ChatTypeSupergroup = "supergroup"
	ChatTypeChannel    = "channel"
)
