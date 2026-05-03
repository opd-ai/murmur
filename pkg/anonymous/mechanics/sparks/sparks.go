// Package mechanics - Surface Sparks implementation.
// Per ANONYMOUS_GAME_MECHANICS.md: "Surface Sparks are lightweight, Surface-Layer-exclusive
// challenge mechanics that give Open-mode users a taste of gamified interaction."
package sparks

import (
	"crypto/ed25519"
	"errors"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

// Spark constants per ANONYMOUS_GAME_MECHANICS.md.
const (
	// SparkDuration is how long a Spark challenge is active.
	SparkDuration = 5 * time.Minute

	// SparkMaxResponseTime is the window for participating in a Spark.
	SparkMaxResponseTime = 5 * time.Minute

	// SparkCrownDuration is how long the winner crown displays.
	SparkCrownDuration = 1 * time.Hour
)

// SparkType identifies the type of Surface Spark.
type SparkType uint8

// Spark types per ANONYMOUS_GAME_MECHANICS.md.
const (
	SparkWaveRelay SparkType = iota + 1 // Wave Relay challenge.
	SparkEchoRace                       // Echo Race - first amplifier wins.
)

// SparkState represents the lifecycle state of a Spark.
type SparkState uint8

// Spark states.
const (
	SparkActive    SparkState = iota + 1 // Spark is accepting responses.
	SparkCompleted                       // Spark has ended.
	SparkExpired                         // Spark timed out.
	SparkCancelled                       // Spark cancelled by initiator.
)

// Spark errors.
var (
	ErrSparkNotFound     = errors.New("spark not found")
	ErrSparkExpired      = errors.New("spark has expired")
	ErrSparkInvalidType  = errors.New("invalid spark type")
	ErrSparkAlreadyWon   = errors.New("spark already has a winner")
	ErrSparkSelfResponse = errors.New("cannot respond to own spark")
	ErrInvalidPrompt     = errors.New("invalid spark prompt")
)

// Spark represents a Surface Spark challenge event.
type Spark struct {
	ID          [32]byte   // Unique spark ID (BLAKE3 hash).
	Type        SparkType  // WaveRelay or EchoRace.
	InitiatorID []byte     // Ed25519 public key of spark creator.
	Prompt      string     // Creative constraint for WaveRelay (optional for EchoRace).
	CreatedAt   time.Time  // When the spark was initiated.
	ExpiresAt   time.Time  // When the spark challenge window closes.
	State       SparkState // Current state.
	WinnerID    []byte     // Ed25519 public key of winner (for EchoRace).
	WinnerTime  time.Time  // When the winner was determined.
	Signature   []byte     // Ed25519 signature from initiator.
}

// SparkResponse represents a response to a Spark challenge.
type SparkResponse struct {
	ID          [32]byte  // Unique response ID.
	SparkID     [32]byte  // ID of the spark being responded to.
	ResponderID []byte    // Ed25519 public key of responder.
	WaveID      [32]byte  // ID of the Wave containing the response.
	CreatedAt   time.Time // When the response was submitted.
	Signature   []byte    // Ed25519 signature.
}

// SparkResult holds the outcome of a Spark challenge.
type SparkResult struct {
	SparkID        [32]byte  // ID of the spark.
	Type           SparkType // Type of spark.
	WinnerID       []byte    // Winner's public key (nil if no winner).
	ResponseTime   time.Duration
	TotalResponses int
	CompletedAt    time.Time
}

// IsExpired returns true if the spark has passed its expiration time.
func (s *Spark) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsActive returns true if the spark is still accepting responses.
func (s *Spark) IsActive() bool {
	return s.State == SparkActive && !s.IsExpired()
}

// SparkTypeString returns the human-readable name of a spark type.
func SparkTypeString(t SparkType) string {
	switch t {
	case SparkWaveRelay:
		return "Wave Relay"
	case SparkEchoRace:
		return "Echo Race"
	default:
		return "Unknown"
	}
}

// SparkStore manages Surface Spark storage.
type SparkStore struct {
	mu           sync.RWMutex
	sparks       map[[32]byte]*Spark           // By spark ID.
	byInitiator  map[string][]*Spark           // By initiator key (hex).
	responses    map[[32]byte][]*SparkResponse // By spark ID.
	results      map[[32]byte]*SparkResult     // By spark ID.
	crownHolders map[string]time.Time          // Winner crowns by key (hex).
}

// NewSparkStore creates a new spark store.
func NewSparkStore() *SparkStore {
	return &SparkStore{
		sparks:       make(map[[32]byte]*Spark),
		byInitiator:  make(map[string][]*Spark),
		responses:    make(map[[32]byte][]*SparkResponse),
		results:      make(map[[32]byte]*SparkResult),
		crownHolders: make(map[string]time.Time),
	}
}

// CreateSpark initiates a new Surface Spark challenge.
func (s *SparkStore) CreateSpark(
	sparkType SparkType,
	initiatorID []byte,
	prompt string,
	privKey ed25519.PrivateKey,
) (*Spark, error) {
	if sparkType != SparkWaveRelay && sparkType != SparkEchoRace {
		return nil, ErrSparkInvalidType
	}

	// WaveRelay requires a prompt.
	if sparkType == SparkWaveRelay && len(prompt) == 0 {
		return nil, ErrInvalidPrompt
	}

	// Generate spark ID.
	h := blake3.New()
	h.Write(initiatorID)
	h.Write([]byte{byte(sparkType)})
	h.Write([]byte(prompt))
	now := time.Now()
	nowBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		nowBytes[i] = byte(now.UnixNano() >> (8 * i))
	}
	h.Write(nowBytes)
	var id [32]byte
	copy(id[:], h.Sum(nil))

	spark := &Spark{
		ID:          id,
		Type:        sparkType,
		InitiatorID: initiatorID,
		Prompt:      prompt,
		CreatedAt:   now,
		ExpiresAt:   now.Add(SparkDuration),
		State:       SparkActive,
	}

	// Sign the spark.
	if privKey != nil {
		signData := append(id[:], byte(sparkType))
		signData = append(signData, initiatorID...)
		signData = append(signData, []byte(prompt)...)
		spark.Signature = ed25519.Sign(privKey, signData)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sparks[id] = spark
	keyHex := keyToHex(initiatorID)
	s.byInitiator[keyHex] = append(s.byInitiator[keyHex], spark)
	s.responses[id] = make([]*SparkResponse, 0)

	return spark, nil
}

// AddSpark adds a pre-built spark received from the network.
// This is used by SparkReceiver for network-received sparks.
func (s *SparkStore) AddSpark(spark *Spark) error {
	if spark == nil {
		return ErrSparkInvalidType
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if spark already exists.
	if _, ok := s.sparks[spark.ID]; ok {
		return nil // Idempotent: already received this spark.
	}

	s.sparks[spark.ID] = spark
	keyHex := keyToHex(spark.InitiatorID)
	s.byInitiator[keyHex] = append(s.byInitiator[keyHex], spark)
	s.responses[spark.ID] = make([]*SparkResponse, 0)

	return nil
}

// RespondToSpark submits a response to a Spark challenge.
func (s *SparkStore) RespondToSpark(
	sparkID [32]byte,
	responderID []byte,
	waveID [32]byte,
	privKey ed25519.PrivateKey,
) (*SparkResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	spark, ok := s.sparks[sparkID]
	if !ok {
		return nil, ErrSparkNotFound
	}

	// Check spark is still active.
	if spark.IsExpired() {
		return nil, ErrSparkExpired
	}

	// For EchoRace, check if there's already a winner (comes before state check).
	if spark.Type == SparkEchoRace && spark.WinnerID != nil {
		return nil, ErrSparkAlreadyWon
	}

	// Check state (for WaveRelay, State should still be Active).
	if spark.State != SparkActive {
		return nil, ErrSparkExpired
	}

	// Cannot respond to own spark.
	if keyToHex(spark.InitiatorID) == keyToHex(responderID) {
		return nil, ErrSparkSelfResponse
	}

	// Generate response ID.
	h := blake3.New()
	h.Write(sparkID[:])
	h.Write(responderID)
	h.Write(waveID[:])
	now := time.Now()
	nowBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		nowBytes[i] = byte(now.UnixNano() >> (8 * i))
	}
	h.Write(nowBytes)
	var id [32]byte
	copy(id[:], h.Sum(nil))

	response := &SparkResponse{
		ID:          id,
		SparkID:     sparkID,
		ResponderID: responderID,
		WaveID:      waveID,
		CreatedAt:   now,
	}

	// Sign the response.
	if privKey != nil {
		signData := append(id[:], sparkID[:]...)
		signData = append(signData, responderID...)
		signData = append(signData, waveID[:]...)
		response.Signature = ed25519.Sign(privKey, signData)
	}

	s.responses[sparkID] = append(s.responses[sparkID], response)

	// For EchoRace, first response wins.
	if spark.Type == SparkEchoRace {
		spark.WinnerID = responderID
		spark.WinnerTime = now
		spark.State = SparkCompleted

		// Grant crown to winner.
		winnerHex := keyToHex(responderID)
		s.crownHolders[winnerHex] = now.Add(SparkCrownDuration)

		// Record result.
		s.results[sparkID] = &SparkResult{
			SparkID:        sparkID,
			Type:           spark.Type,
			WinnerID:       responderID,
			ResponseTime:   now.Sub(spark.CreatedAt),
			TotalResponses: 1,
			CompletedAt:    now,
		}
	}

	return response, nil
}

// GetSpark retrieves a spark by ID.
func (s *SparkStore) GetSpark(id [32]byte) (*Spark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	spark, ok := s.sparks[id]
	if !ok {
		return nil, ErrSparkNotFound
	}
	return spark, nil
}

// GetResponses returns all responses to a spark.
func (s *SparkStore) GetResponses(sparkID [32]byte) []*SparkResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.responses[sparkID]
}

// GetResult returns the result of a completed spark.
func (s *SparkStore) GetResult(sparkID [32]byte) *SparkResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.results[sparkID]
}

// GetActiveSparks returns all currently active sparks.
func (s *SparkStore) GetActiveSparks() []*Spark {
	s.mu.RLock()
	defer s.mu.RUnlock()

	active := make([]*Spark, 0)
	for _, spark := range s.sparks {
		if spark.IsActive() {
			active = append(active, spark)
		}
	}
	return active
}

// GetSparksByInitiator returns all sparks created by an initiator.
func (s *SparkStore) GetSparksByInitiator(initiatorID []byte) []*Spark {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keyHex := keyToHex(initiatorID)
	sparks := make([]*Spark, 0)
	for _, spark := range s.byInitiator[keyHex] {
		sparks = append(sparks, spark)
	}
	return sparks
}

// HasCrown returns true if a user currently holds a crown from an EchoRace win.
func (s *SparkStore) HasCrown(userID []byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keyHex := keyToHex(userID)
	expiry, ok := s.crownHolders[keyHex]
	if !ok {
		return false
	}
	return time.Now().Before(expiry)
}

// GetCrownExpiry returns when a user's crown expires (zero time if no crown).
func (s *SparkStore) GetCrownExpiry(userID []byte) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keyHex := keyToHex(userID)
	expiry, ok := s.crownHolders[keyHex]
	if !ok {
		return time.Time{}
	}
	if time.Now().After(expiry) {
		return time.Time{}
	}
	return expiry
}

// ExpireSparks marks expired sparks and generates results.
func (s *SparkStore) ExpireSparks() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	expired := 0
	now := time.Now()

	for id, spark := range s.sparks {
		if spark.State == SparkActive && spark.IsExpired() {
			spark.State = SparkExpired

			// Generate result for WaveRelay sparks.
			if spark.Type == SparkWaveRelay {
				responses := s.responses[id]
				s.results[id] = &SparkResult{
					SparkID:        id,
					Type:           spark.Type,
					WinnerID:       nil, // No winner for WaveRelay.
					TotalResponses: len(responses),
					CompletedAt:    now,
				}
			}
			expired++
		}
	}

	return expired
}

// PurgeExpiredCrowns removes expired crown holders.
func (s *SparkStore) PurgeExpiredCrowns() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	purged := 0
	for key, expiry := range s.crownHolders {
		if now.After(expiry) {
			delete(s.crownHolders, key)
			purged++
		}
	}
	return purged
}

// CountActiveSparks returns the number of active sparks.
func (s *SparkStore) CountActiveSparks() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, spark := range s.sparks {
		if spark.IsActive() {
			count++
		}
	}
	return count
}

// CountTotalSparks returns the total number of sparks (all states).
func (s *SparkStore) CountTotalSparks() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sparks)
}

// CountCrownHolders returns the number of users currently holding crowns.
func (s *SparkStore) CountCrownHolders() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	now := time.Now()
	for _, expiry := range s.crownHolders {
		if now.Before(expiry) {
			count++
		}
	}
	return count
}

// GetSparksByType returns all sparks of a given type.
func (s *SparkStore) GetSparksByType(sparkType SparkType) []*Spark {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sparks := make([]*Spark, 0)
	for _, spark := range s.sparks {
		if spark.Type == sparkType {
			sparks = append(sparks, spark)
		}
	}
	return sparks
}

// VerifySpark validates a spark's signature.
func VerifySpark(spark *Spark, pubKey ed25519.PublicKey) bool {
	if spark == nil || len(spark.Signature) == 0 {
		return false
	}

	signData := append(spark.ID[:], byte(spark.Type))
	signData = append(signData, spark.InitiatorID...)
	signData = append(signData, []byte(spark.Prompt)...)

	return ed25519.Verify(pubKey, signData, spark.Signature)
}

// VerifySparkResponse validates a spark response's signature.
func VerifySparkResponse(response *SparkResponse, pubKey ed25519.PublicKey) bool {
	if response == nil || len(response.Signature) == 0 {
		return false
	}

	signData := append(response.ID[:], response.SparkID[:]...)
	signData = append(signData, response.ResponderID...)
	signData = append(signData, response.WaveID[:]...)

	return ed25519.Verify(pubKey, signData, response.Signature)
}
