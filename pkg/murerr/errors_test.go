package murerr

import (
	"errors"
	"testing"
	"time"
)

func TestKindString(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindOther, "other"},
		{KindNetwork, "network"},
		{KindCrypto, "crypto"},
		{KindStorage, "storage"},
		{KindValidation, "validation"},
		{KindResource, "resource"},
		{KindAnonymous, "anonymous"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("Kind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorMessage(t *testing.T) {
	err := New(KindNetwork, "transport.Connect", "connection refused", nil)

	if err.Error() != "transport.Connect: connection refused" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	cause := errors.New("underlying error")
	errWithCause := New(KindStorage, "store.Open", "database locked", cause)

	if errWithCause.Error() != "store.Open: database locked: underlying error" {
		t.Errorf("unexpected error message with cause: %s", errWithCause.Error())
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := New(KindCrypto, "keys.Sign", "signature failed", cause)

	if !errors.Is(err, cause) {
		t.Error("Unwrap should allow errors.Is to match cause")
	}
}

func TestUserMessage(t *testing.T) {
	kinds := []Kind{
		KindNetwork, KindCrypto, KindStorage,
		KindValidation, KindResource, KindAnonymous, KindOther,
	}

	for _, kind := range kinds {
		err := New(kind, "test.Op", "internal", nil)
		msg := err.UserMessage()
		if msg == "" {
			t.Errorf("UserMessage for %v should not be empty", kind)
		}
	}
}

func TestRetryable(t *testing.T) {
	retryableKinds := []Kind{KindNetwork, KindResource, KindAnonymous}
	nonRetryableKinds := []Kind{KindOther, KindCrypto, KindStorage, KindValidation}

	for _, kind := range retryableKinds {
		err := New(kind, "test.Op", "test", nil)
		if !err.Retryable() {
			t.Errorf("%v should be retryable", kind)
		}
	}

	for _, kind := range nonRetryableKinds {
		err := New(kind, "test.Op", "test", nil)
		if err.Retryable() {
			t.Errorf("%v should not be retryable", kind)
		}
	}
}

func TestWrap(t *testing.T) {
	// Wrap nil returns nil.
	if Wrap(nil, "op") != nil {
		t.Error("Wrap(nil) should return nil")
	}

	// Wrap structured error preserves kind.
	original := New(KindNetwork, "inner.Op", "inner message", nil)
	wrapped := Wrap(original, "outer.Op")

	var e *Error
	if !errors.As(wrapped, &e) {
		t.Fatal("wrapped error should be *Error")
	}
	if e.Kind != KindNetwork {
		t.Errorf("wrapped error kind = %v, want %v", e.Kind, KindNetwork)
	}
	if e.Op != "outer.Op" {
		t.Errorf("wrapped error op = %v, want outer.Op", e.Op)
	}

	// Wrap standard error creates KindOther.
	stdErr := errors.New("standard error")
	wrappedStd := Wrap(stdErr, "std.Op")

	if !errors.As(wrappedStd, &e) {
		t.Fatal("wrapped std error should be *Error")
	}
	if e.Kind != KindOther {
		t.Errorf("wrapped std error kind = %v, want %v", e.Kind, KindOther)
	}
}

func TestGetKind(t *testing.T) {
	err := New(KindStorage, "db.Query", "not found", nil)
	if GetKind(err) != KindStorage {
		t.Errorf("GetKind = %v, want %v", GetKind(err), KindStorage)
	}

	stdErr := errors.New("standard")
	if GetKind(stdErr) != KindOther {
		t.Errorf("GetKind(std) = %v, want %v", GetKind(stdErr), KindOther)
	}
}

func TestRetry(t *testing.T) {
	// Success on first try.
	attempts := 0
	err := Retry(func() error {
		attempts++
		return nil
	}, RetryConfig{MaxAttempts: 3, InitialWait: time.Millisecond})
	if err != nil {
		t.Errorf("Retry should succeed: %v", err)
	}
	if attempts != 1 {
		t.Errorf("should have 1 attempt, got %d", attempts)
	}

	// Non-retryable error stops immediately.
	attempts = 0
	nonRetryable := New(KindCrypto, "test.Op", "invalid key", nil)
	err = Retry(func() error {
		attempts++
		return nonRetryable
	}, RetryConfig{MaxAttempts: 3, InitialWait: time.Millisecond})

	if !errors.Is(err, nonRetryable) {
		t.Errorf("should return non-retryable error")
	}
	if attempts != 1 {
		t.Errorf("non-retryable should stop after 1 attempt, got %d", attempts)
	}

	// Retryable error retries.
	attempts = 0
	retryable := New(KindNetwork, "test.Op", "timeout", nil)
	err = Retry(func() error {
		attempts++
		if attempts < 3 {
			return retryable
		}
		return nil
	}, RetryConfig{MaxAttempts: 5, InitialWait: time.Millisecond, Multiplier: 1.0})
	if err != nil {
		t.Errorf("Retry should eventually succeed: %v", err)
	}
	if attempts != 3 {
		t.Errorf("should have 3 attempts, got %d", attempts)
	}
}

func TestSuggestFallback(t *testing.T) {
	tests := []struct {
		kind Kind
		want FallbackStrategy
	}{
		{KindAnonymous, FallbackHybridMode},
		{KindNetwork, FallbackOffline},
		{KindCrypto, FallbackNone},
		{KindStorage, FallbackNone},
	}

	for _, tt := range tests {
		err := New(tt.kind, "test.Op", "test", nil)
		if got := SuggestFallback(err); got != tt.want {
			t.Errorf("SuggestFallback(%v) = %v, want %v", tt.kind, got, tt.want)
		}
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", cfg.MaxAttempts)
	}
	if cfg.InitialWait != 100*time.Millisecond {
		t.Errorf("InitialWait = %v, want 100ms", cfg.InitialWait)
	}
	if cfg.MaxWait != 5*time.Second {
		t.Errorf("MaxWait = %v, want 5s", cfg.MaxWait)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", cfg.Multiplier)
	}
}
