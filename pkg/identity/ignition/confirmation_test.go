package ignition

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestNewConfirmationSession(t *testing.T) {
	_, privA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	pubB, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	session, err := NewConfirmationSession(privA, pubB, true)
	if err != nil {
		t.Fatalf("NewConfirmationSession failed: %v", err)
	}

	if session.State != StateInitiator {
		t.Errorf("state = %v, want StateInitiator", session.State)
	}
	if session.LocalKey == nil {
		t.Error("local key should not be nil")
	}
	if session.RemoteKey == nil {
		t.Error("remote key should not be nil")
	}
}

func TestConfirmationSession_ChallengeMessage(t *testing.T) {
	_, privA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	pubB, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	session, err := NewConfirmationSession(privA, pubB, true)
	if err != nil {
		t.Fatalf("NewConfirmationSession failed: %v", err)
	}

	msg, err := session.ChallengeMessage()
	if err != nil {
		t.Fatalf("ChallengeMessage failed: %v", err)
	}

	// Message should be 88 bytes + 64 byte signature = 152 bytes.
	if len(msg) != 152 {
		t.Errorf("message length = %d, want 152", len(msg))
	}

	if session.State != StateChallengeSent {
		t.Errorf("state = %v, want StateChallengeSent", session.State)
	}
}

func TestConfirmationSession_FullProtocol(t *testing.T) {
	// Generate keys for Alice and Bob.
	pubA, privA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	pubB, privB, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	// Alice initiates.
	alice, err := NewConfirmationSession(privA, pubB, true)
	if err != nil {
		t.Fatalf("NewConfirmationSession (Alice) failed: %v", err)
	}

	// Bob responds.
	bob, err := NewConfirmationSession(privB, pubA, false)
	if err != nil {
		t.Fatalf("NewConfirmationSession (Bob) failed: %v", err)
	}

	// Verify session IDs match (order-independent).
	if alice.ID != bob.ID {
		t.Error("session IDs should match")
	}

	// Alice sends challenge.
	aliceChallenge, err := alice.ChallengeMessage()
	if err != nil {
		t.Fatalf("Alice challenge failed: %v", err)
	}

	// Bob processes Alice's challenge.
	if err := bob.ProcessChallenge(aliceChallenge); err != nil {
		t.Fatalf("Bob process challenge failed: %v", err)
	}

	// Bob sends challenge.
	bobChallenge, err := bob.ChallengeMessage()
	if err != nil {
		t.Fatalf("Bob challenge failed: %v", err)
	}

	// Alice processes Bob's challenge.
	if err := alice.ProcessChallenge(bobChallenge); err != nil {
		t.Fatalf("Alice process challenge failed: %v", err)
	}

	// Alice sends confirmation.
	aliceConfirm, err := alice.ConfirmationMessage()
	if err != nil {
		t.Fatalf("Alice confirmation failed: %v", err)
	}

	// Bob processes Alice's confirmation.
	confirmed, err := bob.ProcessConfirmation(aliceConfirm)
	if err != nil {
		t.Fatalf("Bob process confirmation failed: %v", err)
	}
	if confirmed {
		t.Error("should not be confirmed yet (Bob hasn't confirmed)")
	}

	// Bob sends confirmation.
	bobConfirm, err := bob.ConfirmationMessage()
	if err != nil {
		t.Fatalf("Bob confirmation failed: %v", err)
	}

	// Alice processes Bob's confirmation.
	confirmed, err = alice.ProcessConfirmation(bobConfirm)
	if err != nil {
		t.Fatalf("Alice process confirmation failed: %v", err)
	}
	if !confirmed {
		t.Error("should be confirmed after both parties confirm")
	}

	// Verify final states.
	if !alice.IsConfirmed() {
		t.Error("Alice should be confirmed")
	}
	if alice.State != StateConfirmed {
		t.Errorf("Alice state = %v, want StateConfirmed", alice.State)
	}

	// Note: Bob needs to process something to update his state,
	// but he already sent confirmation. In practice, you'd
	// re-send confirmations until both are StateConfirmed.
}

func TestConfirmationSession_InvalidChallenge(t *testing.T) {
	pubA, privA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	pubB, privB, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	_ = privA

	bob, err := NewConfirmationSession(privB, pubA, false)
	if err != nil {
		t.Fatalf("NewConfirmationSession (Bob) failed: %v", err)
	}

	_ = pubB

	// Create a challenge with wrong session ID.
	badChallenge := make([]byte, 152)
	rand.Read(badChallenge)

	err = bob.ProcessChallenge(badChallenge)
	if err == nil {
		t.Error("should reject invalid challenge")
	}
}

func TestConfirmationSession_TooShortChallenge(t *testing.T) {
	pubA, privA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	pubB, privB, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	_ = privA
	bob, err := NewConfirmationSession(privB, pubA, false)
	if err != nil {
		t.Fatalf("NewConfirmationSession failed: %v", err)
	}
	_ = pubB

	// Challenge too short.
	shortMsg := make([]byte, 50)
	err = bob.ProcessChallenge(shortMsg)
	if err != ErrChallengeResponseSize {
		t.Errorf("error = %v, want ErrChallengeResponseSize", err)
	}
}

func TestConfirmationSession_Reject(t *testing.T) {
	_, privA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	pubB, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	session, err := NewConfirmationSession(privA, pubB, true)
	if err != nil {
		t.Fatalf("NewConfirmationSession failed: %v", err)
	}

	session.Reject()
	if session.GetState() != StateRejected {
		t.Errorf("state = %v, want StateRejected", session.GetState())
	}
}

func TestConfirmationSession_IsExpired(t *testing.T) {
	_, privA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	pubB, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	session, err := NewConfirmationSession(privA, pubB, true)
	if err != nil {
		t.Fatalf("NewConfirmationSession failed: %v", err)
	}

	// Should not be expired immediately.
	if session.IsExpired() {
		t.Error("should not be expired immediately")
	}

	// Manually set creation time to the past.
	session.mu.Lock()
	session.CreatedAt = time.Now().Add(-ConfirmationTimeout - time.Second)
	session.mu.Unlock()

	if !session.IsExpired() {
		t.Error("should be expired after timeout")
	}
}

func TestConfirmationManager_CreateSession(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	remotePub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	manager := NewConfirmationManager(priv)

	session, err := manager.CreateSession(remotePub, true)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session == nil {
		t.Fatal("session should not be nil")
	}

	if manager.ActiveSessions() != 1 {
		t.Errorf("active sessions = %d, want 1", manager.ActiveSessions())
	}
}

func TestConfirmationManager_GetSession(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	remotePub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	manager := NewConfirmationManager(priv)

	session, err := manager.CreateSession(remotePub, true)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Get by ID.
	retrieved, err := manager.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if retrieved != session {
		t.Error("should retrieve same session")
	}

	// Get by remote key.
	retrieved, err = manager.GetSessionByRemoteKey(remotePub)
	if err != nil {
		t.Fatalf("GetSessionByRemoteKey failed: %v", err)
	}
	if retrieved != session {
		t.Error("should retrieve same session by remote key")
	}
}

func TestConfirmationManager_GetSessionNotFound(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	manager := NewConfirmationManager(priv)

	var fakeID [32]byte
	_, err = manager.GetSession(fakeID)
	if err != ErrSessionNotFound {
		t.Errorf("error = %v, want ErrSessionNotFound", err)
	}
}

func TestConfirmationManager_RemoveSession(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	remotePub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	manager := NewConfirmationManager(priv)

	session, err := manager.CreateSession(remotePub, true)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	manager.RemoveSession(session.ID)

	if manager.ActiveSessions() != 0 {
		t.Errorf("active sessions = %d, want 0", manager.ActiveSessions())
	}
}

func TestConfirmationManager_CleanupExpired(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	manager := NewConfirmationManager(priv)

	// Create several sessions.
	for i := 0; i < 5; i++ {
		remotePub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("key generation failed: %v", err)
		}
		session, err := manager.CreateSession(remotePub, true)
		if err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}

		// Expire some sessions.
		if i < 3 {
			session.mu.Lock()
			session.CreatedAt = time.Now().Add(-ConfirmationTimeout - time.Second)
			session.mu.Unlock()
		}
	}

	if manager.ActiveSessions() != 2 {
		t.Errorf("active sessions before cleanup = %d, want 2", manager.ActiveSessions())
	}

	removed := manager.CleanupExpired()
	if removed != 3 {
		t.Errorf("removed = %d, want 3", removed)
	}

	if manager.ActiveSessions() != 2 {
		t.Errorf("active sessions after cleanup = %d, want 2", manager.ActiveSessions())
	}
}

func TestConfirmationManager_StartCleanupLoop(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	manager := NewConfirmationManager(priv)

	// Create an expired session.
	remotePub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	session, err := manager.CreateSession(remotePub, true)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	session.mu.Lock()
	session.CreatedAt = time.Now().Add(-ConfirmationTimeout - time.Second)
	session.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cleanup loop with short interval.
	manager.StartCleanupLoop(ctx, 50*time.Millisecond)

	// Wait for cleanup to run.
	time.Sleep(100 * time.Millisecond)

	// Check the manager's internal sessions map directly since ActiveSessions
	// only counts non-expired (which the session already is).
	manager.mu.RLock()
	sessionCount := len(manager.sessions)
	manager.mu.RUnlock()

	if sessionCount != 0 {
		t.Errorf("sessions after cleanup = %d, want 0", sessionCount)
	}
}

func TestConfirmationState_String(t *testing.T) {
	tests := []struct {
		state ConfirmationState
		want  string
	}{
		{StateInitiator, "initiator"},
		{StateResponder, "responder"},
		{StateChallengeSent, "challenge_sent"},
		{StateChallengeReceived, "challenge_received"},
		{StateConfirmed, "confirmed"},
		{StateRejected, "rejected"},
		{StateTimeout, "timeout"},
		{ConfirmationState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeriveSessionID_OrderIndependent(t *testing.T) {
	pub1, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	pub2, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	id1 := deriveSessionID(pub1, pub2)
	id2 := deriveSessionID(pub2, pub1)

	if id1 != id2 {
		t.Error("session IDs should be order-independent")
	}
}

func TestConfirmationSession_AlreadyConfirmed(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}
	remotePub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	session, err := NewConfirmationSession(priv, remotePub, true)
	if err != nil {
		t.Fatalf("NewConfirmationSession failed: %v", err)
	}

	// Manually set to confirmed.
	session.mu.Lock()
	session.State = StateConfirmed
	session.mu.Unlock()

	_, err = session.ConfirmationMessage()
	if err != ErrAlreadyConfirmed {
		t.Errorf("error = %v, want ErrAlreadyConfirmed", err)
	}
}

func TestConfirmationConstants(t *testing.T) {
	// Per VIRAL_GROWTH_AND_ONBOARDING.md: "5-minute expiry window".
	if ConfirmationTimeout != 5*time.Minute {
		t.Errorf("ConfirmationTimeout = %v, want 5m", ConfirmationTimeout)
	}

	if ChallengeSize != 32 {
		t.Errorf("ChallengeSize = %d, want 32", ChallengeSize)
	}

	if NonceSize != 16 {
		t.Errorf("NonceSize = %d, want 16", NonceSize)
	}
}

// BenchmarkConfirmationSession benchmarks the full protocol.
func BenchmarkConfirmationSession(b *testing.B) {
	pubA, privA, _ := ed25519.GenerateKey(rand.Reader)
	pubB, privB, _ := ed25519.GenerateKey(rand.Reader)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alice, _ := NewConfirmationSession(privA, pubB, true)
		bob, _ := NewConfirmationSession(privB, pubA, false)

		aliceChallenge, _ := alice.ChallengeMessage()
		_ = bob.ProcessChallenge(aliceChallenge)

		bobChallenge, _ := bob.ChallengeMessage()
		_ = alice.ProcessChallenge(bobChallenge)

		aliceConfirm, _ := alice.ConfirmationMessage()
		_, _ = bob.ProcessConfirmation(aliceConfirm)

		bobConfirm, _ := bob.ConfirmationMessage()
		_, _ = alice.ProcessConfirmation(bobConfirm)
	}
}
