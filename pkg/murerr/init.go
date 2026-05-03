// Package murerr provides enhanced error formatting and user-friendly error messages.
// Per AUDIT.md remediation, this improves error feedback during initialization failures.
package murerr

import (
	"fmt"
	"strings"
)

// InitError represents an initialization failure with context and recovery suggestions.
type InitError struct {
	Subsystem string
	Cause     error
	Hint      string
}

// Error implements the error interface.
func (e *InitError) Error() string {
	return fmt.Sprintf("%s initialization failed: %v", e.Subsystem, e.Cause)
}

// Unwrap returns the underlying error for error wrapping chains.
func (e *InitError) Unwrap() error {
	return e.Cause
}

// Format returns a user-friendly multi-line error message with recovery hints.
func (e *InitError) Format() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("═══════════════════════════════════════════════════════════\n")
	b.WriteString(" MURMUR Initialization Error\n")
	b.WriteString("═══════════════════════════════════════════════════════════\n\n")

	b.WriteString(fmt.Sprintf("Failed to initialize: %s\n\n", e.Subsystem))
	b.WriteString(fmt.Sprintf("Error: %v\n\n", e.Cause))

	if e.Hint != "" {
		b.WriteString("Suggested action:\n")
		b.WriteString(fmt.Sprintf("  %s\n\n", e.Hint))
	}

	b.WriteString("═══════════════════════════════════════════════════════════\n\n")

	return b.String()
}

// WrapStorageError wraps a storage initialization error with helpful context.
func WrapStorageError(err error) *InitError {
	hint := "Check that ~/.murmur/ directory is writable and not corrupted.\n" +
		"  Try: rm ~/.murmur/murmur.db to reset the database."

	return &InitError{
		Subsystem: "Storage",
		Cause:     err,
		Hint:      hint,
	}
}

// WrapIdentityError wraps an identity initialization error with helpful context.
func WrapIdentityError(err error) *InitError {
	hint := "Check that your keypair file is valid and not corrupted.\n" +
		"  Try: rm ~/.murmur/identity.key to generate a new identity."

	return &InitError{
		Subsystem: "Identity",
		Cause:     err,
		Hint:      hint,
	}
}

// WrapNetworkError wraps a networking initialization error with helpful context.
func WrapNetworkError(err error) *InitError {
	hint := "Check that no other MURMUR instance is running.\n" +
		"  Check your firewall settings allow outbound connections.\n" +
		"  If behind a strict firewall, try --bootstrap with a known peer multiaddr."

	return &InitError{
		Subsystem: "Networking",
		Cause:     err,
		Hint:      hint,
	}
}

// WrapContentError wraps a content subsystem error with helpful context.
func WrapContentError(err error) *InitError {
	return &InitError{
		Subsystem: "Content",
		Cause:     err,
		Hint:      "Check that the Wave cache has sufficient disk space.",
	}
}

// WrapBeaconError wraps a Shroud beacon error with helpful context.
func WrapBeaconError(err error) *InitError {
	return &InitError{
		Subsystem: "Shroud Beacon",
		Cause:     err,
		Hint:      "Anonymous layer initialization failed. This is non-critical; Surface Layer still functional.",
	}
}
