package telegramsender

import "log/slog"

// SecretToken is a string type that redacts itself in logs and string output.
// Use this for sensitive values like API tokens or secrets.
type SecretToken string

// LogValue implements slog.LogValuer to redact sensitive tokens in logs.
func (SecretToken) LogValue() slog.Value {
	return slog.StringValue("[REDACTED]")
}

// String returns "[REDACTED]" to prevent accidental exposure.
func (SecretToken) String() string {
	return "[REDACTED]"
}

// Value returns the actual secret value. Use sparingly and never log the result.
func (t SecretToken) Value() string {
	return string(t)
}
