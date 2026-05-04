// Package ignition provides the mutual confirmation protocol for Proximity Ignition.
// Per VIRAL_GROWTH_AND_ONBOARDING.md §Proximity Ignition:
// "Both devices display confirmation (each other's sigil preview)" and
// "Both parties accept, creating a Connection edge visible on both Pulse Maps."
// The protocol ensures both parties explicitly confirm the connection before
// it's established, preventing one-sided or accidental connections.

package ignition

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"sync"
	"time"
)

// Mutual confirmation protocol constants.
const (
	// ConfirmationTimeout is the maximum time to wait for mutual confirmation.
	// Per VIRAL_GROWTH_AND_ONBOARDING.md: "5-minute expiry window"
	ConfirmationTimeout = 5 * time.Minute

	// ChallengeSize is the size of the random challenge in bytes.
	ChallengeSize = 32

	// NonceSize is the size of the nonce for confirmation messages.
	NonceSize = 16

	// ConfirmationStateSize is the minimum size of a confirmation state message.
	ConfirmationStateSize = 32 + 16 + 64 // pubkey + nonce + signature minimum

	// ConfirmationMessageSize is the size of unsigned confirmation message (session_id + challenge + nonce + timestamp).
	ConfirmationMessageSize = 88
)

// ConfirmationState represents the current state of mutual confirmation.
type ConfirmationState int

const (
	// StateInitiator means we initiated the ignition flow.
	StateInitiator ConfirmationState = iota

	// StateResponder means we received an ignition request.
	StateResponder

	// StateChallengeSent means we've sent our challenge and await response.
	StateChallengeSent

	// StateChallengeReceived means we've received their challenge.
	StateChallengeReceived

	// StateConfirmed means both parties have confirmed.
	StateConfirmed

	// StateRejected means one party rejected the connection.
	StateRejected

	// StateTimeout means the confirmation timed out.
	StateTimeout
)

// String returns a human-readable state name.
func (s ConfirmationState) String() string {
	switch s {
	case StateInitiator:
		return "initiator"
	case StateResponder:
		return "responder"
	case StateChallengeSent:
		return "challenge_sent"
	case StateChallengeReceived:
		return "challenge_received"
	case StateConfirmed:
		return "confirmed"
	case StateRejected:
		return "rejected"
	case StateTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

// ConfirmationError represents errors during the confirmation protocol.
var (
	ErrConfirmationTimeout   = errors.New("confirmation timed out")
	ErrConfirmationRejected  = errors.New("confirmation rejected by peer")
	ErrInvalidChallenge      = errors.New("invalid challenge response")
	ErrProtocolViolation     = errors.New("confirmation protocol violation")
	ErrAlreadyConfirmed      = errors.New("already confirmed")
	ErrSessionNotFound       = errors.New("confirmation session not found")
	ErrInvalidNonce          = errors.New("invalid nonce")
	ErrChallengeResponseSize = errors.New("invalid challenge response size")
)

// ConfirmationSession tracks the state of a mutual confirmation.
// The protocol ensures both parties explicitly confirm before establishing a connection.
type ConfirmationSession struct {
	// ID is the unique session identifier.
	ID [32]byte

	// LocalKey is our private key for signing.
	LocalKey ed25519.PrivateKey

	// RemoteKey is the peer's public key (from IgnitionData).
	RemoteKey ed25519.PublicKey

	// State is the current confirmation state.
	State ConfirmationState

	// LocalChallenge is the challenge we sent to the peer.
	LocalChallenge [ChallengeSize]byte

	// RemoteChallenge is the challenge we received from the peer.
	RemoteChallenge [ChallengeSize]byte

	// LocalNonce prevents replay of our confirmation messages.
	LocalNonce [NonceSize]byte

	// RemoteNonce prevents replay of their confirmation messages.
	RemoteNonce [NonceSize]byte

	// CreatedAt is when the session was created.
	CreatedAt time.Time

	// ConfirmedAt is when mutual confirmation completed (if successful).
	ConfirmedAt time.Time

	// LocalConfirmed indicates we have confirmed locally.
	LocalConfirmed bool

	// RemoteConfirmed indicates peer has confirmed.
	RemoteConfirmed bool

	mu sync.RWMutex
}

// NewConfirmationSession creates a new confirmation session.
// The session ID is derived from both public keys to ensure uniqueness.
func NewConfirmationSession(localKey ed25519.PrivateKey, remoteKey ed25519.PublicKey, isInitiator bool) (*ConfirmationSession, error) {
	session := &ConfirmationSession{
		LocalKey:  localKey,
		RemoteKey: remoteKey,
		CreatedAt: time.Now(),
	}

	// Generate session ID from both keys (order-independent).
	localPub := localKey.Public().(ed25519.PublicKey)
	session.ID = deriveSessionID(localPub, remoteKey)

	// Set initial state based on role.
	if isInitiator {
		session.State = StateInitiator
	} else {
		session.State = StateResponder
	}

	// Generate local challenge and nonce.
	if _, err := rand.Read(session.LocalChallenge[:]); err != nil {
		return nil, err
	}
	if _, err := rand.Read(session.LocalNonce[:]); err != nil {
		return nil, err
	}

	return session, nil
}

// deriveSessionID creates a deterministic session ID from two public keys.
// The ID is the same regardless of which key is "local" vs "remote".
func deriveSessionID(key1, key2 ed25519.PublicKey) [32]byte {
	var id [32]byte
	// XOR the keys to get an order-independent ID.
	for i := 0; i < 32; i++ {
		id[i] = key1[i] ^ key2[i]
	}
	return id
}

// signConfirmationMessage signs an 88-byte confirmation message and returns msg+signature.
// Consolidates the duplicate pattern used in ChallengeMessage and ConfirmationMessage.
func signConfirmationMessage(key ed25519.PrivateKey, msg []byte) []byte {
	sig := ed25519.Sign(key, msg)
	result := make([]byte, len(msg)+len(sig))
	copy(result, msg)
	copy(result[len(msg):], sig)
	return result
}

// ChallengeMessage creates a signed challenge to send to the peer.
// Per VIRAL_GROWTH_AND_ONBOARDING.md: "Both parties accept" - the challenge
// proves we control our key and want to connect.
func (s *ConfirmationSession) ChallengeMessage() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Build message: session_id (32) + challenge (32) + nonce (16) + timestamp (8)
	msg := make([]byte, ConfirmationMessageSize)
	copy(msg[0:32], s.ID[:])
	copy(msg[32:64], s.LocalChallenge[:])
	copy(msg[64:80], s.LocalNonce[:])
	binary.BigEndian.PutUint64(msg[80:88], uint64(time.Now().Unix()))

	result := signConfirmationMessage(s.LocalKey, msg)

	s.State = StateChallengeSent
	return result, nil
}

// ProcessChallenge handles a received challenge from the peer.
// Returns true if the challenge is valid and we should proceed.
func (s *ConfirmationSession) ProcessChallenge(msg []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Minimum message size check.
	if len(msg) < 88+64 {
		return ErrChallengeResponseSize
	}

	// Extract components.
	sessionID := msg[0:32]
	challenge := msg[32:64]
	nonce := msg[64:80]
	timestamp := binary.BigEndian.Uint64(msg[80:88])
	signature := msg[88:152]

	// Verify session ID matches.
	var expectedID [32]byte
	copy(expectedID[:], sessionID)
	if expectedID != s.ID {
		return ErrProtocolViolation
	}

	// Verify timestamp is recent (within ConfirmationTimeout).
	msgTime := time.Unix(int64(timestamp), 0)
	if time.Since(msgTime) > ConfirmationTimeout {
		s.State = StateTimeout
		return ErrConfirmationTimeout
	}

	// Verify signature.
	if !ed25519.Verify(s.RemoteKey, msg[:88], signature) {
		return ErrInvalidSignature
	}

	// Store remote challenge and nonce.
	copy(s.RemoteChallenge[:], challenge)
	copy(s.RemoteNonce[:], nonce)
	s.State = StateChallengeReceived

	return nil
}

// ConfirmationMessage creates a signed confirmation response.
// This proves we received their challenge and agree to connect.
func (s *ConfirmationSession) ConfirmationMessage() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State == StateConfirmed {
		return nil, ErrAlreadyConfirmed
	}

	// Build message: session_id (32) + their_challenge (32) + our_nonce (16) + timestamp (8)
	msg := make([]byte, ConfirmationMessageSize)
	copy(msg[0:32], s.ID[:])
	copy(msg[32:64], s.RemoteChallenge[:])
	copy(msg[64:80], s.LocalNonce[:])
	binary.BigEndian.PutUint64(msg[80:88], uint64(time.Now().Unix()))

	result := signConfirmationMessage(s.LocalKey, msg)

	s.LocalConfirmed = true
	if s.RemoteConfirmed {
		s.State = StateConfirmed
		s.ConfirmedAt = time.Now()
	}

	return result, nil
}

// ProcessConfirmation handles a received confirmation from the peer.
// Returns true if both parties have now confirmed.
func (s *ConfirmationSession) ProcessConfirmation(msg []byte) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State == StateConfirmed {
		return true, nil
	}

	if len(msg) < 88+64 {
		return false, ErrChallengeResponseSize
	}

	confirmation := parseConfirmationMessage(msg)
	if err := s.validateConfirmation(confirmation, msg); err != nil {
		return false, err
	}

	s.RemoteConfirmed = true
	if s.LocalConfirmed {
		s.State = StateConfirmed
		s.ConfirmedAt = time.Now()
		return true, nil
	}

	return false, nil
}

// confirmationMessage holds parsed confirmation message components.
type confirmationMessage struct {
	sessionID         [32]byte
	challengeResponse [ChallengeSize]byte
	nonce             [NonceSize]byte
	timestamp         uint64
	signature         []byte
}

// parseConfirmationMessage extracts components from a confirmation message.
func parseConfirmationMessage(msg []byte) *confirmationMessage {
	var cm confirmationMessage
	copy(cm.sessionID[:], msg[0:32])
	copy(cm.challengeResponse[:], msg[32:64])
	copy(cm.nonce[:], msg[64:80])
	cm.timestamp = binary.BigEndian.Uint64(msg[80:88])
	cm.signature = msg[88:152]
	return &cm
}

// validateConfirmation verifies all confirmation message fields.
func (s *ConfirmationSession) validateConfirmation(cm *confirmationMessage, msg []byte) error {
	if cm.sessionID != s.ID {
		return ErrProtocolViolation
	}

	if cm.challengeResponse != s.LocalChallenge {
		return ErrInvalidChallenge
	}

	msgTime := time.Unix(int64(cm.timestamp), 0)
	if time.Since(msgTime) > ConfirmationTimeout {
		s.State = StateTimeout
		return ErrConfirmationTimeout
	}

	if s.RemoteNonce != [NonceSize]byte{} && cm.nonce != s.RemoteNonce {
		return ErrInvalidNonce
	}

	if !ed25519.Verify(s.RemoteKey, msg[:88], cm.signature) {
		return ErrInvalidSignature
	}

	return nil
}

// Reject marks the session as rejected by us.
func (s *ConfirmationSession) Reject() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State = StateRejected
}

// IsExpired returns true if the session has exceeded the timeout.
func (s *ConfirmationSession) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.CreatedAt) > ConfirmationTimeout
}

// IsConfirmed returns true if both parties have confirmed.
func (s *ConfirmationSession) IsConfirmed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State == StateConfirmed
}

// GetState returns the current state.
func (s *ConfirmationSession) GetState() ConfirmationState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.State
}

// ConfirmationManager manages multiple confirmation sessions.
// It handles cleanup of expired sessions and provides session lookup.
type ConfirmationManager struct {
	sessions map[[32]byte]*ConfirmationSession
	mu       sync.RWMutex
	localKey ed25519.PrivateKey
}

// NewConfirmationManager creates a new confirmation manager.
func NewConfirmationManager(localKey ed25519.PrivateKey) *ConfirmationManager {
	return &ConfirmationManager{
		sessions: make(map[[32]byte]*ConfirmationSession),
		localKey: localKey,
	}
}

// CreateSession creates a new confirmation session with a peer.
func (m *ConfirmationManager) CreateSession(remoteKey ed25519.PublicKey, isInitiator bool) (*ConfirmationSession, error) {
	session, err := NewConfirmationSession(m.localKey, remoteKey, isInitiator)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID.
func (m *ConfirmationManager) GetSession(id [32]byte) (*ConfirmationSession, error) {
	m.mu.RLock()
	session, exists := m.sessions[id]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// GetSessionByRemoteKey retrieves a session by the peer's public key.
func (m *ConfirmationManager) GetSessionByRemoteKey(remoteKey ed25519.PublicKey) (*ConfirmationSession, error) {
	localPub := m.localKey.Public().(ed25519.PublicKey)
	id := deriveSessionID(localPub, remoteKey)
	return m.GetSession(id)
}

// RemoveSession removes a session.
func (m *ConfirmationManager) RemoveSession(id [32]byte) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
}

// CleanupExpired removes all expired sessions.
func (m *ConfirmationManager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for id, session := range m.sessions {
		if session.IsExpired() {
			delete(m.sessions, id)
			count++
		}
	}
	return count
}

// StartCleanupLoop starts a background goroutine that periodically cleans up expired sessions.
func (m *ConfirmationManager) StartCleanupLoop(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.CleanupExpired()
			}
		}
	}()
}

// ActiveSessions returns the count of active (non-expired) sessions.
func (m *ConfirmationManager) ActiveSessions() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, session := range m.sessions {
		if !session.IsExpired() {
			count++
		}
	}
	return count
}
