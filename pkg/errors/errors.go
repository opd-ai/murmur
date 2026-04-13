// Package errors provides structured error handling for MURMUR.
// Per ROADMAP.md Priority 12, this package implements user-friendly error
// messages, graceful degradation strategies, and automatic retry with
// exponential backoff.
package errors

import (
	"errors"
	"fmt"
	"time"
)

// Kind represents a category of error for consistent handling.
type Kind uint8

const (
	// KindOther represents unclassified errors.
	KindOther Kind = iota
	// KindNetwork represents network-related errors (connection, timeout).
	KindNetwork
	// KindCrypto represents cryptographic errors (key, signature).
	KindCrypto
	// KindStorage represents storage errors (database, file).
	KindStorage
	// KindValidation represents validation errors (input, state).
	KindValidation
	// KindResource represents resource errors (memory, connections).
	KindResource
	// KindAnonymous represents anonymous layer errors (Shroud, Specter).
	KindAnonymous
)

// String returns a human-readable name for the error kind.
func (k Kind) String() string {
	switch k {
	case KindNetwork:
		return "network"
	case KindCrypto:
		return "crypto"
	case KindStorage:
		return "storage"
	case KindValidation:
		return "validation"
	case KindResource:
		return "resource"
	case KindAnonymous:
		return "anonymous"
	default:
		return "other"
	}
}

// Error is a structured error with kind, operation, and optional cause.
type Error struct {
	Kind    Kind
	Op      string // Operation that failed (e.g., "keys.Generate").
	Message string // User-friendly message.
	Cause   error  // Underlying error.
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/As.
func (e *Error) Unwrap() error {
	return e.Cause
}

// UserMessage returns a user-friendly description of the error.
func (e *Error) UserMessage() string {
	switch e.Kind {
	case KindNetwork:
		return "Network connection issue. Check your internet connection and try again."
	case KindCrypto:
		return "Security verification failed. Your data remains protected."
	case KindStorage:
		return "Unable to save or load data. Check disk space and permissions."
	case KindValidation:
		return "Invalid input. Please check your entry and try again."
	case KindResource:
		return "System resources low. Try closing other applications."
	case KindAnonymous:
		return "Anonymous layer temporarily unavailable. Falling back to surface mode."
	default:
		return "An unexpected error occurred. Please try again."
	}
}

// Retryable returns true if the error can be retried.
func (e *Error) Retryable() bool {
	switch e.Kind {
	case KindNetwork, KindResource, KindAnonymous:
		return true
	default:
		return false
	}
}

// New creates a new structured error.
func New(kind Kind, op, message string, cause error) *Error {
	return &Error{
		Kind:    kind,
		Op:      op,
		Message: message,
		Cause:   cause,
	}
}

// Wrap wraps an error with additional context.
func Wrap(err error, op string) error {
	if err == nil {
		return nil
	}

	var e *Error
	if errors.As(err, &e) {
		return &Error{
			Kind:    e.Kind,
			Op:      op,
			Message: e.Message,
			Cause:   err,
		}
	}

	return &Error{
		Kind:    KindOther,
		Op:      op,
		Message: err.Error(),
		Cause:   err,
	}
}

// Is reports whether err matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// GetKind extracts the Kind from an error.
func GetKind(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindOther
}

// RetryConfig configures retry behavior.
type RetryConfig struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
	Multiplier  float64
}

// DefaultRetryConfig returns sensible retry defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     5 * time.Second,
		Multiplier:  2.0,
	}
}

// Retry executes fn with exponential backoff on retryable errors.
func Retry(fn func() error, cfg RetryConfig) error {
	var lastErr error
	wait := cfg.InitialWait

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if !isRetryable(lastErr) {
			return lastErr
		}

		if attempt < cfg.MaxAttempts {
			time.Sleep(wait)
			wait = computeNextWait(wait, cfg)
		}
	}

	return lastErr
}

// isRetryable checks if an error can be retried.
func isRetryable(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Retryable()
}

// computeNextWait calculates the next backoff wait time.
func computeNextWait(current time.Duration, cfg RetryConfig) time.Duration {
	next := time.Duration(float64(current) * cfg.Multiplier)
	if next > cfg.MaxWait {
		return cfg.MaxWait
	}
	return next
}

// FallbackStrategy defines how to handle failures.
type FallbackStrategy uint8

const (
	// FallbackNone indicates no fallback is available.
	FallbackNone FallbackStrategy = iota
	// FallbackHybridMode falls back to Hybrid privacy mode.
	FallbackHybridMode
	// FallbackSurfaceOnly falls back to Surface layer only.
	FallbackSurfaceOnly
	// FallbackOffline operates without network.
	FallbackOffline
)

// SuggestFallback returns an appropriate fallback strategy for an error.
func SuggestFallback(err error) FallbackStrategy {
	kind := GetKind(err)
	switch kind {
	case KindAnonymous:
		return FallbackHybridMode
	case KindNetwork:
		return FallbackOffline
	default:
		return FallbackNone
	}
}
