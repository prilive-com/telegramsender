# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
