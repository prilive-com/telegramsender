# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.3.0] - 2026-01-01

### Added

- **v3 API** - Simplified configuration with modern Go library patterns
  - `New(token, opts...)` - Simple programmatic constructor
  - `NewFromConfig(path, opts...)` - Multi-source config (file + env + options)
  - `Client` type with `SendMessage()`, `SendPhoto()`, `SendPhotoFile()`, `Close()`, `Config()`, `API()`
  - Interface-based `Option` pattern for type-safe configuration
  - Client implements `Sender` interface

- **Configuration Options** (v3)
  - `WithBaseURLOption(url)` - Custom base URL
  - `WithRequestTimeoutOption(d)` - HTTP request timeout
  - `WithKeepAliveOption(d)` - HTTP keep-alive duration
  - `WithConnectionPool(maxIdle, timeout)` - Connection pool settings
  - `WithRateLimitOption(rps, burst)` - Rate limiting settings
  - `WithBreakerConfig(maxReq, interval, timeout)` - Circuit breaker
  - `WithRetryOption(max, initial, maxBackoff, factor)` - Retry settings
  - `WithMaxRetriesOption(n)` - Max retries shorthand
  - `WithContentLimitsOption(caption, fileSize)` - Content limits
  - `WithAllowedPhotoDirsOption(dirs...)` - Security restrictions
  - `WithLogger(logger)` - Custom slog.Logger
  - `WithLogFile(path)` - Log file path
  - `WithHTTPClientOption(client)` - Custom HTTP client for testing

- **Presets** (v3)
  - `ProductionPreset()` - Production-optimized settings
  - `DevelopmentPreset()` - Development-friendly settings
  - `HighThroughputPreset()` - High-throughput scenarios

- **Multi-source Configuration**
  - koanf-based configuration loading
  - Precedence: defaults → config file → env vars (TELEGRAM_*) → programmatic options
  - YAML config file support

- **Validation**
  - go-playground/validator integration
  - Public `ValidateBotToken()` function
  - Actionable error messages with remediation hints

- **New Files**
  - `client.go` - v3 Client type and constructors
  - `options.go` - Option interface and With* functions
  - `client_test.go` - Tests for v3 API (with .env file support)
  - `example/v3/main.go` - v3 API example
  - `example/v3/config.yaml` - Example config file

- **Dependencies**
  - `github.com/knadh/koanf/v2` - Multi-source configuration
  - `github.com/go-playground/validator/v10` - Struct validation
  - `github.com/joho/godotenv` - Load .env files in tests

### Deprecated

- `NewConfig()` - Use `New()` instead
- `LoadConfig()` - Use `NewFromConfig()` instead
- `NewTelegramAPI()` - Use `New()` instead

All deprecated functions will be removed in v4.

## [2.0.0] - 2025-12-07

### Breaking Changes

- **Module path changed** to `github.com/prilive-com/telegramsender/v2`
- **Logger type changed**: `NewLogger()` now returns `*Logger` (wrapper struct) instead of `*slog.Logger`
  - Callers MUST call `logger.Close()` to release file handles
- **Go version**: Requires Go 1.24.3+

### Added

- **Security**
  - Path traversal protection for `SendPhotoFile` with configurable allowed directories
  - Base URL validation with whitelist (only `api.telegram.org` allowed by default)
  - Log file path validation to prevent writes to sensitive directories
  - TLS 1.2 minimum version enforcement
  - Response size limits (10MB) to prevent memory exhaustion DoS
  - Cryptographically secure random jitter using `crypto/rand`

- **Interfaces for testability**
  - `Sender` interface for mocking `TelegramAPI` in tests
  - `HTTPClient`, `RateLimiter`, `CircuitBreaker` interfaces

- **Typed errors**
  - `TelegramError` - implements `error` interface, supports `errors.As()` and `errors.Is()`
  - `ValidationError` - for request validation failures
  - `ConfigError` - for configuration errors
  - Sentinel errors: `ErrInvalidConfig`, `ErrRateLimitExceeded`, `ErrCircuitBreakerOpen`, `ErrMaxRetriesExceeded`, `ErrPathTraversal`, `ErrResponseTooLarge`, `ErrInvalidBaseURL`

- **Configuration improvements**
  - `NewConfig()` constructor with functional options pattern
  - `WithBaseURL()`, `WithRequestTimeout()`, `WithRateLimit()`, `WithRetry()`, `WithAllowedPhotoDirs()`, etc.
  - `AllowedPhotoDirs` config option for restricting photo upload paths

- **Validation**
  - Message text length validation (4096 UTF-8 characters max)
  - `ValidateMessageRequest()`, `ValidatePhotoRequest()`, `ValidatePhotoFileRequest()`
  - `ValidateBaseURL()`, `ValidatePhotoPath()`

- **Per-chat rate limiting** - Telegram rate limits are per-chat, not global

- **Constants file** - All magic numbers extracted to named constants

- **Comprehensive test suite** with environment-based configuration

### Changed

- **Retry logic refactored** - Extracted to `withRetry()` helper, eliminating code duplication
- **Error handling** - `TelegramError` now properly implements `error` interface, fixing `errors.As()` for `RetryAfter`
- **Jitter calculation** - Now uses cryptographically random values instead of deterministic alternation
- **Directory permissions** - Log directories created with `0700` instead of `0755`

### Fixed

- **CRITICAL**: Logger file handle leak - file was never closed on success
- **CRITICAL**: `RetryAfter` from Telegram 429 responses was never used (error type assertion always failed)
- **CRITICAL**: Path traversal vulnerability in `SendPhotoFile` allowed reading arbitrary files
- **CRITICAL**: Unbounded response body reading could cause memory exhaustion
- **HIGH**: Base URL could be set to malicious hosts
- **HIGH**: Log file path could write to sensitive system directories
- **MEDIUM**: Deterministic jitter caused thundering herd problems
- **MEDIUM**: Global rate limiter instead of per-chat

### Removed

- Direct return of `*slog.Logger` from `NewLogger()` (now wrapped in `*Logger`)

## [1.x.x] - Previous versions

See git history for changes prior to v2.0.0.

**Note**: v1.x is now deprecated. Please migrate to v2.0.0.
