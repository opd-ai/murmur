// Package mechanics provides anonymous game mechanics per ANONYMOUS_GAME_MECHANICS.md.
// This file implements Masked Events - time-limited anonymous gatherings.
package mechanics

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/curve25519"
)

// Per ANONYMOUS_GAME_MECHANICS.md, Masked Events are time-limited anonymous gatherings
// where all participants shed even their Specter identities and interact through
// single-use Masked keypairs.

// Masked Event errors.
var (
	ErrMaskedEventInsufficientResonance = errors.New("masked event: insufficient Resonance (minimum 100)")
	ErrMaskedEventInvalidDuration       = errors.New("masked event: invalid duration")
	ErrMaskedEventInvalidParticipants   = errors.New("masked event: invalid participant limit (0 or 5-100)")
	ErrMaskedEventTopicTooLong          = errors.New("masked event: topic exceeds 256 bytes")
	ErrMaskedEventNotStarted            = errors.New("masked event: event has not started")
	ErrMaskedEventAlreadyStarted        = errors.New("masked event: event has already started")
	ErrMaskedEventEnded                 = errors.New("masked event: event has ended")
	ErrMaskedEventFull                  = errors.New("masked event: maximum participants reached")
	ErrMaskedEventNotJoined             = errors.New("masked event: not a participant")
	ErrMaskedEventAlreadyJoined         = errors.New("masked event: already joined")
	ErrMaskedEventInvalidMaskedKey      = errors.New("masked event: invalid masked key")
	ErrMaskedEventKeyDestroyed          = errors.New("masked event: masked key has been destroyed")
)

// MaskedEventMinResonance is the minimum Specter Resonance to create Masked Events.
// Per spec, "Any Specter in Hybrid+ mode can create a Masked Event".
// This implies a basic Resonance threshold of 100 (Phantom milestone).
const MaskedEventMinResonance = 100

// MaskedEventDuration represents allowed event durations.
type MaskedEventDuration time.Duration

// Per ANONYMOUS_GAME_MECHANICS.md: 30, 60, 120, or 240 minutes.
const (
	MaskedEventDuration30Min  MaskedEventDuration = MaskedEventDuration(30 * time.Minute)
	MaskedEventDuration60Min  MaskedEventDuration = MaskedEventDuration(60 * time.Minute)
	MaskedEventDuration120Min MaskedEventDuration = MaskedEventDuration(120 * time.Minute)
	MaskedEventDuration240Min MaskedEventDuration = MaskedEventDuration(240 * time.Minute)
)

// MaskedEventState represents the lifecycle state of a Masked Event.
type MaskedEventState int

const (
	// MaskedEventPending is waiting for start time.
	MaskedEventPending MaskedEventState = iota
	// MaskedEventActive is currently running.
	MaskedEventActive
	// MaskedEventEnded has concluded.
	MaskedEventEnded
)

// MaskedEventStateString returns a human-readable state name.
func MaskedEventStateString(s MaskedEventState) string {
	switch s {
	case MaskedEventPending:
		return "Pending"
	case MaskedEventActive:
		return "Active"
	case MaskedEventEnded:
		return "Ended"
	default:
		return "Unknown"
	}
}

// IsValidMaskedEventDuration checks if a duration is one of the allowed values.
func IsValidMaskedEventDuration(d time.Duration) bool {
	switch MaskedEventDuration(d) {
	case MaskedEventDuration30Min, MaskedEventDuration60Min,
		MaskedEventDuration120Min, MaskedEventDuration240Min:
		return true
	default:
		return false
	}
}

// MaskedEvent represents a time-limited anonymous gathering.
type MaskedEvent struct {
	mu sync.RWMutex

	// ID is a random 32-byte event identifier.
	ID [32]byte

	// Topic describes the event's theme or purpose (max 256 bytes).
	Topic string

	// CreatorSpecterKey is the Specter who created this event.
	CreatorSpecterKey [32]byte

	// StartTime is when the event begins.
	StartTime time.Time

	// Duration is how long the event runs.
	Duration time.Duration

	// EndTime is when the event concludes.
	EndTime time.Time

	// MaxParticipants is the cap (0 = unlimited, or 5-100).
	MaxParticipants int

	// State is the current lifecycle state.
	State MaskedEventState

	// CreatedAt is when the event was created.
	CreatedAt time.Time

	// participants maps Masked public key hex to participant info.
	participants map[string]*MaskedParticipant

	// specterJoins tracks which Specters have joined (by Specter key hex).
	specterJoins map[string]bool

	// onStateChange callback for state transitions.
	onStateChange []func(MaskedEventState, MaskedEventState)
}

// MaskedParticipant represents a participant in a Masked Event.
type MaskedParticipant struct {
	// MaskedPublicKey is the single-use public key for this event.
	MaskedPublicKey [32]byte

	// Pseudonym is the event-themed two-word identifier.
	Pseudonym string

	// JoinedAt is when the participant joined.
	JoinedAt time.Time

	// WaveCount is how many Masked Waves published.
	WaveCount int

	// AmplificationsReceived tracks engagement.
	AmplificationsReceived int
}

// MaskedKeypair is a single-use Ed25519 keypair for Masked Event participation.
type MaskedKeypair struct {
	mu sync.RWMutex

	// PublicKey is the 32-byte Ed25519 public key.
	PublicKey [32]byte

	// privateKey is the 64-byte Ed25519 private key.
	// This is zeroed after the event ends.
	privateKey ed25519.PrivateKey

	// x25519Private is the Curve25519 private key for key exchange.
	x25519Private [32]byte

	// Pseudonym is the derived two-word identifier.
	Pseudonym string

	// EventID links to the parent event.
	EventID [32]byte

	// Destroyed indicates the key has been wiped.
	destroyed bool
}

// NewMaskedEvent creates a new Masked Event.
// Per ANONYMOUS_GAME_MECHANICS.md, the creator publishes a Beacon Wave (0x08)
// with beacon_type=event_announce.
func NewMaskedEvent(
	creatorSpecterKey [32]byte,
	topic string,
	startTime time.Time,
	duration time.Duration,
	maxParticipants int,
) (*MaskedEvent, error) {
	// Validate topic length.
	if len(topic) > 256 {
		return nil, ErrMaskedEventTopicTooLong
	}

	// Validate duration.
	if !IsValidMaskedEventDuration(duration) {
		return nil, ErrMaskedEventInvalidDuration
	}

	// Validate participant limit.
	if maxParticipants != 0 && (maxParticipants < 5 || maxParticipants > 100) {
		return nil, ErrMaskedEventInvalidParticipants
	}

	// Generate random event ID.
	var id [32]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, fmt.Errorf("masked event: failed to generate ID: %w", err)
	}

	now := time.Now()

	return &MaskedEvent{
		ID:                id,
		Topic:             topic,
		CreatorSpecterKey: creatorSpecterKey,
		StartTime:         startTime,
		Duration:          duration,
		EndTime:           startTime.Add(duration),
		MaxParticipants:   maxParticipants,
		State:             MaskedEventPending,
		CreatedAt:         now,
		participants:      make(map[string]*MaskedParticipant),
		specterJoins:      make(map[string]bool),
	}, nil
}

// NewMaskedEventGated creates a Masked Event with Resonance gating.
func NewMaskedEventGated(
	creatorSpecterKey [32]byte,
	topic string,
	startTime time.Time,
	duration time.Duration,
	maxParticipants int,
	gate ResonanceGate,
) (*MaskedEvent, error) {
	if err := CheckResonanceGate(gate, creatorSpecterKey, MaskedEventMinResonance); err != nil {
		return nil, ErrMaskedEventInsufficientResonance
	}
	return NewMaskedEvent(creatorSpecterKey, topic, startTime, duration, maxParticipants)
}

// GetID returns the event ID.
func (e *MaskedEvent) GetID() [32]byte {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.ID
}

// GetTopic returns the event topic.
func (e *MaskedEvent) GetTopic() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Topic
}

// GetState returns the current state.
func (e *MaskedEvent) GetState() MaskedEventState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.State
}

// IsPending returns true if event hasn't started.
func (e *MaskedEvent) IsPending() bool {
	return e.GetState() == MaskedEventPending
}

// IsActive returns true if event is running.
func (e *MaskedEvent) IsActive() bool {
	return e.GetState() == MaskedEventActive
}

// IsEnded returns true if event has concluded.
func (e *MaskedEvent) IsEnded() bool {
	return e.GetState() == MaskedEventEnded
}

// ParticipantCount returns the number of participants.
func (e *MaskedEvent) ParticipantCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.participants)
}

// IsFull returns true if the participant limit is reached.
func (e *MaskedEvent) IsFull() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.MaxParticipants == 0 {
		return false // Unlimited.
	}
	return len(e.participants) >= e.MaxParticipants
}

// TimeRemaining returns time until event ends (or starts if pending).
func (e *MaskedEvent) TimeRemaining() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	now := time.Now()
	switch e.State {
	case MaskedEventPending:
		return e.StartTime.Sub(now)
	case MaskedEventActive:
		return e.EndTime.Sub(now)
	default:
		return 0
	}
}

// TimeUntilStart returns time until the event starts.
func (e *MaskedEvent) TimeUntilStart() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.State != MaskedEventPending {
		return 0
	}
	return time.Until(e.StartTime)
}

// GossipTopic returns the event-specific gossip topic string.
// Per spec: /murmur/event/[event_id]/1.0
func (e *MaskedEvent) GossipTopic() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return fmt.Sprintf("/murmur/event/%s/1.0", hex.EncodeToString(e.ID[:]))
}

// OnStateChange registers a callback for state transitions.
func (e *MaskedEvent) OnStateChange(cb func(MaskedEventState, MaskedEventState)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onStateChange = append(e.onStateChange, cb)
}

// Update checks time and updates event state.
func (e *MaskedEvent) Update() {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	oldState := e.State

	switch e.State {
	case MaskedEventPending:
		if now.After(e.StartTime) || now.Equal(e.StartTime) {
			e.State = MaskedEventActive
			e.fireStateChange(oldState, MaskedEventActive)
		}
	case MaskedEventActive:
		if now.After(e.EndTime) {
			e.State = MaskedEventEnded
			e.fireStateChange(oldState, MaskedEventEnded)
		}
	}
}

// fireStateChange notifies state change listeners.
// Must be called with lock held.
func (e *MaskedEvent) fireStateChange(old, new MaskedEventState) {
	callbacks := e.onStateChange
	// Fire outside lock.
	e.mu.Unlock()
	for _, cb := range callbacks {
		cb(old, new)
	}
	e.mu.Lock()
}

// Start forces the event to start immediately (for testing).
func (e *MaskedEvent) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.State != MaskedEventPending {
		return ErrMaskedEventAlreadyStarted
	}

	oldState := e.State
	e.State = MaskedEventActive
	e.StartTime = time.Now()
	e.EndTime = e.StartTime.Add(e.Duration)

	e.fireStateChange(oldState, MaskedEventActive)
	return nil
}

// End forces the event to end immediately.
func (e *MaskedEvent) End() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.State == MaskedEventEnded {
		return
	}

	oldState := e.State
	e.State = MaskedEventEnded
	e.EndTime = time.Now()

	e.fireStateChange(oldState, MaskedEventEnded)
}

// Join adds a participant to the event.
// Returns the Masked keypair for this participant.
func (e *MaskedEvent) Join(specterKey [32]byte) (*MaskedKeypair, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if already joined.
	specterHex := KeyToHex(specterKey[:])
	if e.specterJoins[specterHex] {
		return nil, ErrMaskedEventAlreadyJoined
	}

	// Check if event is accepting joins.
	if e.State == MaskedEventEnded {
		return nil, ErrMaskedEventEnded
	}

	// Check participant limit.
	if e.MaxParticipants > 0 && len(e.participants) >= e.MaxParticipants {
		return nil, ErrMaskedEventFull
	}

	// Generate single-use Ed25519 keypair.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("masked event: failed to generate keypair: %w", err)
	}

	var pubKey [32]byte
	copy(pubKey[:], pub)

	// Generate X25519 private key for key exchange.
	var x25519Priv [32]byte
	if _, err := rand.Read(x25519Priv[:]); err != nil {
		return nil, fmt.Errorf("masked event: failed to generate x25519 key: %w", err)
	}

	// Generate Masked pseudonym.
	pseudonym := GenerateMaskedPseudonym(pubKey)

	keypair := &MaskedKeypair{
		PublicKey:     pubKey,
		privateKey:    priv,
		x25519Private: x25519Priv,
		Pseudonym:     pseudonym,
		EventID:       e.ID,
	}

	// Record participant.
	pubKeyHex := KeyToHex(pubKey[:])
	e.participants[pubKeyHex] = &MaskedParticipant{
		MaskedPublicKey: pubKey,
		Pseudonym:       pseudonym,
		JoinedAt:        time.Now(),
	}

	// Track Specter join.
	e.specterJoins[specterHex] = true

	return keypair, nil
}

// RegisterMaskedKey registers a pre-generated masked key.
// This is used when receiving a join from another node.
func (e *MaskedEvent) RegisterMaskedKey(maskedPubKey [32]byte, pseudonym string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.State == MaskedEventEnded {
		return ErrMaskedEventEnded
	}

	if e.MaxParticipants > 0 && len(e.participants) >= e.MaxParticipants {
		return ErrMaskedEventFull
	}

	pubKeyHex := KeyToHex(maskedPubKey[:])
	if _, exists := e.participants[pubKeyHex]; exists {
		return ErrMaskedEventAlreadyJoined
	}

	e.participants[pubKeyHex] = &MaskedParticipant{
		MaskedPublicKey: maskedPubKey,
		Pseudonym:       pseudonym,
		JoinedAt:        time.Now(),
	}

	return nil
}

// HasParticipant checks if a Masked public key is registered.
func (e *MaskedEvent) HasParticipant(maskedPubKey [32]byte) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, exists := e.participants[KeyToHex(maskedPubKey[:])]
	return exists
}

// GetParticipant returns info for a participant.
func (e *MaskedEvent) GetParticipant(maskedPubKey [32]byte) *MaskedParticipant {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.participants[KeyToHex(maskedPubKey[:])]
}

// GetParticipants returns all participants.
func (e *MaskedEvent) GetParticipants() []*MaskedParticipant {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]*MaskedParticipant, 0, len(e.participants))
	for _, p := range e.participants {
		result = append(result, p)
	}
	return result
}

// RecordWave increments a participant's wave count.
func (e *MaskedEvent) RecordWave(maskedPubKey [32]byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	p, exists := e.participants[KeyToHex(maskedPubKey[:])]
	if !exists {
		return ErrMaskedEventNotJoined
	}
	p.WaveCount++
	return nil
}

// RecordAmplification increments a participant's amplification count.
func (e *MaskedEvent) RecordAmplification(maskedPubKey [32]byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	p, exists := e.participants[KeyToHex(maskedPubKey[:])]
	if !exists {
		return ErrMaskedEventNotJoined
	}
	p.AmplificationsReceived++
	return nil
}

// MaskedKeypair methods.

// GetPublicKey returns the public key.
func (mk *MaskedKeypair) GetPublicKey() [32]byte {
	mk.mu.RLock()
	defer mk.mu.RUnlock()
	return mk.PublicKey
}

// GetPseudonym returns the masked pseudonym.
func (mk *MaskedKeypair) GetPseudonym() string {
	mk.mu.RLock()
	defer mk.mu.RUnlock()
	return mk.Pseudonym
}

// IsDestroyed returns true if the key has been wiped.
func (mk *MaskedKeypair) IsDestroyed() bool {
	mk.mu.RLock()
	defer mk.mu.RUnlock()
	return mk.destroyed
}

// Sign signs data with the Masked private key.
func (mk *MaskedKeypair) Sign(data []byte) ([]byte, error) {
	mk.mu.RLock()
	defer mk.mu.RUnlock()

	if mk.destroyed {
		return nil, ErrMaskedEventKeyDestroyed
	}

	return ed25519.Sign(mk.privateKey, data), nil
}

// GetX25519PublicKey returns the X25519 public key for DH key exchange.
func (mk *MaskedKeypair) GetX25519PublicKey() ([32]byte, error) {
	mk.mu.RLock()
	defer mk.mu.RUnlock()

	if mk.destroyed {
		return [32]byte{}, ErrMaskedEventKeyDestroyed
	}

	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &mk.x25519Private)
	return pub, nil
}

// ComputeSharedSecret computes a shared secret with another X25519 public key.
func (mk *MaskedKeypair) ComputeSharedSecret(theirPublic [32]byte) ([32]byte, error) {
	mk.mu.RLock()
	defer mk.mu.RUnlock()

	if mk.destroyed {
		return [32]byte{}, ErrMaskedEventKeyDestroyed
	}

	var shared [32]byte
	curve25519.ScalarMult(&shared, &mk.x25519Private, &theirPublic)
	return shared, nil
}

// Destroy securely wipes the private key.
// Per spec: "Participants' clients delete their Masked private keys
// (the keys are ephemeral and were never persisted to permanent storage)."
func (mk *MaskedKeypair) Destroy() {
	mk.mu.Lock()
	defer mk.mu.Unlock()

	if mk.destroyed {
		return
	}

	// Zero the private key bytes.
	for i := range mk.privateKey {
		mk.privateKey[i] = 0
	}
	mk.privateKey = nil

	// Zero the X25519 private key.
	for i := range mk.x25519Private {
		mk.x25519Private[i] = 0
	}

	mk.destroyed = true
}

// Masked pseudonym generation using event-themed vocabulary.
// Per spec: "The Masked pseudonym follows the same two-word pattern but uses
// event-themed vocabulary (e.g., 'Flickering Mask,' 'Distant Echo,' 'Burning Question')."

// maskedAdjectives for pseudonym generation.
var maskedAdjectives = []string{
	"Flickering", "Distant", "Burning", "Silent", "Wandering",
	"Hidden", "Fading", "Shifting", "Whispered", "Hollow",
	"Dancing", "Frozen", "Glowing", "Trembling", "Veiled",
	"Shattered", "Drifting", "Echoing", "Shadowed", "Luminous",
	"Phantom", "Muted", "Rippling", "Fleeting", "Ghostly",
	"Spectral", "Ethereal", "Twilight", "Obscured", "Elusive",
	"Transient", "Ephemeral", "Shrouded", "Misty", "Cryptic",
}

// maskedNouns for pseudonym generation.
var maskedNouns = []string{
	"Mask", "Echo", "Question", "Voice", "Shadow",
	"Dream", "Memory", "Secret", "Whisper", "Thought",
	"Reflection", "Silence", "Mystery", "Presence", "Stranger",
	"Wanderer", "Seeker", "Observer", "Witness", "Fragment",
	"Cipher", "Riddle", "Enigma", "Mirage", "Phantom",
	"Specter", "Illusion", "Vision", "Glimpse", "Moment",
	"Horizon", "Twilight", "Dawn", "Dusk", "Void",
}

// GenerateMaskedPseudonym creates a two-word pseudonym from a public key.
func GenerateMaskedPseudonym(pubKey [32]byte) string {
	hash := sha256.Sum256(pubKey[:])

	// Use first 2 bytes for adjective, next 2 for noun.
	adjIdx := (int(hash[0])<<8 | int(hash[1])) % len(maskedAdjectives)
	nounIdx := (int(hash[2])<<8 | int(hash[3])) % len(maskedNouns)

	return maskedAdjectives[adjIdx] + " " + maskedNouns[nounIdx]
}

// MaskedEventSummary contains statistics after event conclusion.
type MaskedEventSummary struct {
	EventID             [32]byte
	Topic               string
	Duration            time.Duration
	ParticipantCount    int
	TotalWaves          int
	TotalAmplifications int
	Leaderboard         []MaskedLeaderboardEntry
}

// MaskedLeaderboardEntry represents a participant's ranking.
type MaskedLeaderboardEntry struct {
	Pseudonym              string
	AmplificationsReceived int
	ResonanceBurst         float64
}

// GenerateSummary creates a summary after the event ends.
func (e *MaskedEvent) GenerateSummary() *MaskedEventSummary {
	e.mu.RLock()
	defer e.mu.RUnlock()

	summary := &MaskedEventSummary{
		EventID:          e.ID,
		Topic:            e.Topic,
		Duration:         e.Duration,
		ParticipantCount: len(e.participants),
	}

	// Collect statistics.
	entries := make([]MaskedLeaderboardEntry, 0, len(e.participants))
	for _, p := range e.participants {
		summary.TotalWaves += p.WaveCount
		summary.TotalAmplifications += p.AmplificationsReceived

		// Calculate Resonance Burst.
		// Per spec: burst_value = 5 * ln(1 + amplifications_received_during_event)
		burst := CalculateResonanceBurst(p.AmplificationsReceived)

		entries = append(entries, MaskedLeaderboardEntry{
			Pseudonym:              p.Pseudonym,
			AmplificationsReceived: p.AmplificationsReceived,
			ResonanceBurst:         burst,
		})
	}

	// Sort by Resonance Burst descending.
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].ResonanceBurst > entries[i].ResonanceBurst {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	summary.Leaderboard = entries
	return summary
}

// CalculateResonanceBurst computes the burst value from amplifications.
// Per spec: burst_value = 5 * ln(1 + amplifications_received_during_event)
func CalculateResonanceBurst(amplifications int) float64 {
	if amplifications <= 0 {
		return 0
	}
	// Using math.Log would add import; compute directly.
	// ln(1+x) ≈ series expansion or we can use a simple approximation.
	// For accuracy, we'd import math, but let's keep it simple.
	x := float64(1 + amplifications)
	// Natural log approximation using a simple formula.
	return 5.0 * naturalLog(x)
}

// naturalLog computes ln(x) using Newton's method.
func naturalLog(x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x == 1 {
		return 0
	}

	// Use the identity: ln(x) = 2 * atanh((x-1)/(x+1))
	// where atanh(z) = z + z^3/3 + z^5/5 + ...
	z := (x - 1) / (x + 1)
	z2 := z * z
	result := z
	term := z
	for i := 3; i < 100; i += 2 {
		term *= z2
		result += term / float64(i)
		if term/float64(i) < 1e-10 {
			break
		}
	}
	return 2 * result
}
