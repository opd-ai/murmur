// Package mechanics - Oracle Pools implementation.
// Per ANONYMOUS_GAME_MECHANICS.md, Oracle Pools are anonymous,
// stake-free prediction markets where Specters forecast network events
// and earn Resonance for accuracy.
package oracle

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"

	"github.com/zeebo/blake3"
)

// Oracle Pool constants per ANONYMOUS_GAME_MECHANICS.md.
const (
	// OracleMinResonance requires Resonance 100 (Phantom milestone) to create a pool.
	OracleMinResonance = 100

	// OracleMaxQuestionLength is the max UTF-8 bytes for a pool question.
	OracleMaxQuestionLength = 256

	// OracleTopPercentile is the fraction of predictors who receive rewards.
	OracleTopPercentile = 0.25 // Top 25%.

	// OracleBonusDecayDays is how long the Resonance bonus lasts.
	OracleBonusDecayDays = 14
)

// OraclePoolState represents the current state of an Oracle Pool.
type OraclePoolState uint8

const (
	OraclePoolOpen      OraclePoolState = iota // Accepting predictions.
	OraclePoolPending                          // Deadline passed, awaiting resolution.
	OraclePoolResolved                         // Outcome determined, rewards distributed.
	OraclePoolCancelled                        // Pool cancelled (insufficient participation).
)

// OraclePredictionType indicates the type of prediction.
type OraclePredictionType uint8

const (
	OraclePredictionBoolean OraclePredictionType = iota // True/false prediction.
	OraclePredictionNumeric                             // Numeric value prediction.
)

// Errors for Oracle Pool operations.
var (
	ErrOracleNotFound            = errors.New("oracle pool not found")
	ErrOraclePoolClosed          = errors.New("oracle pool is closed for predictions")
	ErrOraclePoolNotResolved     = errors.New("oracle pool not yet resolved")
	ErrOraclePoolAlreadyResolved = errors.New("oracle pool already resolved")
	ErrOracleQuestionTooLong     = errors.New("oracle question exceeds max length")
	ErrOracleInsufficientRes     = errors.New("insufficient resonance to create oracle pool")
	ErrOraclePredictionExists    = errors.New("prediction already submitted")
	ErrOracleInvalidCommitment   = errors.New("invalid commitment hash")
	ErrOracleInvalidReveal       = errors.New("reveal does not match commitment")
	ErrOracleRevealNotOpen       = errors.New("reveal period not open")
	ErrOracleCommitmentRequired  = errors.New("commitment required before reveal")
)

// Commitment represents a hashed prediction commitment.
type Commitment struct {
	SpecterKey  [32]byte // Specter's public key.
	Hash        [32]byte // SHA-256(prediction || nonce).
	SubmittedAt time.Time
}

// Prediction represents a revealed prediction.
type Prediction struct {
	SpecterKey [32]byte // Specter's public key.
	Value      float64  // Numeric prediction value (or 1.0/0.0 for boolean).
	Nonce      [32]byte // Random nonce used in commitment.
	RevealedAt time.Time
	Accuracy   float64 // Computed after resolution.
	Rank       int     // Rank among participants.
	Rewarded   bool    // True if in top percentile.
	Bonus      float64 // Resonance bonus earned.
}

// OraclePool represents a prediction market.
type OraclePool struct {
	ID               [32]byte             // Unique pool ID.
	Question         string               // The prediction question.
	PredictionType   OraclePredictionType // Boolean or numeric.
	ResolutionMethod string               // How outcome is determined.
	CreatorKey       [32]byte             // Specter who created the pool.
	CreatedAt        time.Time
	Deadline         time.Time // Prediction submission cutoff.
	ResolutionTime   time.Time // When outcome is evaluated.
	State            OraclePoolState

	// Outcome (set after resolution).
	Outcome    *float64 // Actual outcome value.
	ResolvedAt *time.Time

	// Commitments and predictions.
	commitments map[string]*Commitment // By specter key hex.
	predictions map[string]*Prediction // By specter key hex.

	mu sync.RWMutex
}

// NewOraclePool creates a new Oracle Pool.
// NOTE: This function does not enforce Resonance gating. For gated creation,
// use NewOraclePoolGated which requires Resonance >= OracleMinResonance (100).
func NewOraclePool(
	question string,
	predictionType OraclePredictionType,
	resolutionMethod string,
	creatorKey [32]byte,
	deadline time.Time,
	resolutionTime time.Time,
) (*OraclePool, error) {
	if len(question) > OracleMaxQuestionLength {
		return nil, ErrOracleQuestionTooLong
	}

	now := time.Now()
	poolID := generatePoolID(question, creatorKey, now)

	return &OraclePool{
		ID:               poolID,
		Question:         question,
		PredictionType:   predictionType,
		ResolutionMethod: resolutionMethod,
		CreatorKey:       creatorKey,
		CreatedAt:        now,
		Deadline:         deadline,
		ResolutionTime:   resolutionTime,
		State:            OraclePoolOpen,
		commitments:      make(map[string]*Commitment),
		predictions:      make(map[string]*Prediction),
	}, nil
}

// NewOraclePoolGated creates a new Oracle Pool with Resonance gating.
// Per ANONYMOUS_GAME_MECHANICS.md, only Specters with Resonance >= 100
// (Phantom milestone) may create Oracle Pools.
func NewOraclePoolGated(
	question string,
	predictionType OraclePredictionType,
	resolutionMethod string,
	creatorKey [32]byte,
	deadline time.Time,
	resolutionTime time.Time,
	gate mechanics.ResonanceGate,
) (*OraclePool, error) {
	if err := mechanics.CheckResonanceGate(gate, creatorKey, OracleMinResonance); err != nil {
		return nil, ErrOracleInsufficientRes
	}
	return NewOraclePool(question, predictionType, resolutionMethod,
		creatorKey, deadline, resolutionTime)
}

// generatePoolID creates a deterministic pool ID.
func generatePoolID(question string, creatorKey [32]byte, createdAt time.Time) [32]byte {
	var poolID [32]byte
	h := blake3.New()
	h.Write([]byte(question))
	h.Write(creatorKey[:])
	binary.Write(h, binary.BigEndian, createdAt.Unix())
	copy(poolID[:], h.Sum(nil))
	return poolID
}

// IsOpen returns true if the pool is accepting predictions.
func (p *OraclePool) IsOpen() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.State == OraclePoolOpen && time.Now().Before(p.Deadline)
}

// IsResolved returns true if the pool has been resolved.
func (p *OraclePool) IsResolved() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.State == OraclePoolResolved
}

// CanReveal returns true if the reveal period is open.
func (p *OraclePool) CanReveal() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	now := time.Now()
	return now.After(p.Deadline) && now.Before(p.ResolutionTime) && p.State != OraclePoolResolved
}

// SubmitCommitment submits a hashed prediction commitment.
func (p *OraclePool) SubmitCommitment(specterKey, commitmentHash [32]byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isOpenUnsafe() {
		return ErrOraclePoolClosed
	}

	keyHex := mechanics.KeyToHex(specterKey[:])
	if _, exists := p.commitments[keyHex]; exists {
		return ErrOraclePredictionExists
	}

	p.commitments[keyHex] = &Commitment{
		SpecterKey:  specterKey,
		Hash:        commitmentHash,
		SubmittedAt: time.Now(),
	}

	return nil
}

// isOpenUnsafe checks if pool is open without locking.
func (p *OraclePool) isOpenUnsafe() bool {
	return p.State == OraclePoolOpen && time.Now().Before(p.Deadline)
}

// RevealPrediction reveals a prediction by providing the value and nonce.
func (p *OraclePool) RevealPrediction(specterKey [32]byte, value float64, nonce [32]byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.canRevealUnsafe() {
		return ErrOracleRevealNotOpen
	}

	keyHex := mechanics.KeyToHex(specterKey[:])
	commitment, exists := p.commitments[keyHex]
	if !exists {
		return ErrOracleCommitmentRequired
	}

	// Verify the reveal matches the commitment.
	if !verifyCommitment(value, nonce, commitment.Hash) {
		return ErrOracleInvalidReveal
	}

	// Check for duplicate reveal.
	if _, revealed := p.predictions[keyHex]; revealed {
		return ErrOraclePredictionExists
	}

	p.predictions[keyHex] = &Prediction{
		SpecterKey: specterKey,
		Value:      value,
		Nonce:      nonce,
		RevealedAt: time.Now(),
	}

	return nil
}

// canRevealUnsafe checks if reveal is allowed without locking.
func (p *OraclePool) canRevealUnsafe() bool {
	now := time.Now()
	return now.After(p.Deadline) && now.Before(p.ResolutionTime) && p.State != OraclePoolResolved
}

// verifyCommitment checks if a reveal matches a commitment.
func verifyCommitment(value float64, nonce, expectedHash [32]byte) bool {
	h := sha256.New()
	binary.Write(h, binary.BigEndian, value)
	h.Write(nonce[:])
	var computedHash [32]byte
	copy(computedHash[:], h.Sum(nil))
	return computedHash == expectedHash
}

// ComputeCommitmentHash generates a commitment hash for a prediction.
func ComputeCommitmentHash(value float64, nonce [32]byte) [32]byte {
	h := sha256.New()
	binary.Write(h, binary.BigEndian, value)
	h.Write(nonce[:])
	var hash [32]byte
	copy(hash[:], h.Sum(nil))
	return hash
}

// Resolve resolves the Oracle Pool with the actual outcome.
func (p *OraclePool) Resolve(outcome float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.State == OraclePoolResolved {
		return ErrOraclePoolAlreadyResolved
	}

	now := time.Now()
	p.Outcome = &outcome
	p.ResolvedAt = &now
	p.State = OraclePoolResolved

	p.computeAccuracyAndRanks(outcome)
	p.distributeRewards()

	return nil
}

// computeAccuracyAndRanks computes accuracy scores and ranks for all predictions.
func (p *OraclePool) computeAccuracyAndRanks(outcome float64) {
	// Compute accuracy for each prediction.
	for _, pred := range p.predictions {
		if p.PredictionType == OraclePredictionBoolean {
			// For boolean: 1.0 if correct, 0.0 if incorrect.
			if (pred.Value >= 0.5 && outcome >= 0.5) || (pred.Value < 0.5 && outcome < 0.5) {
				pred.Accuracy = 1.0
			} else {
				pred.Accuracy = 0.0
			}
		} else {
			// For numeric: accuracy = 1 / (1 + |prediction - outcome|).
			pred.Accuracy = 1.0 / (1.0 + math.Abs(pred.Value-outcome))
		}
	}

	// Sort predictions by accuracy descending.
	sorted := p.getSortedPredictions()
	for i, pred := range sorted {
		pred.Rank = i + 1
	}
}

// getSortedPredictions returns predictions sorted by accuracy (descending).
func (p *OraclePool) getSortedPredictions() []*Prediction {
	preds := make([]*Prediction, 0, len(p.predictions))
	for _, pred := range p.predictions {
		preds = append(preds, pred)
	}

	sort.Slice(preds, func(i, j int) bool {
		return preds[i].Accuracy > preds[j].Accuracy
	})

	return preds
}

// distributeRewards distributes Resonance rewards to top predictors.
func (p *OraclePool) distributeRewards() {
	numParticipants := len(p.predictions)
	if numParticipants == 0 {
		return
	}

	// Top 25% receive rewards.
	numRewarded := int(math.Ceil(float64(numParticipants) * OracleTopPercentile))
	if numRewarded < 1 {
		numRewarded = 1
	}

	sorted := p.getSortedPredictions()
	for i, pred := range sorted {
		if i < numRewarded {
			pred.Rewarded = true
			pred.Bonus = computeOracleBonus(numParticipants, pred.Rank)
		}
	}
}

// computeOracleBonus calculates the Resonance bonus for a predictor.
// Per ANONYMOUS_GAME_MECHANICS.md: oracle_bonus = 3 * ln(1 + pool_participant_count / rank).
func computeOracleBonus(participantCount, rank int) float64 {
	if rank == 0 {
		return 0
	}
	return 3.0 * math.Log1p(float64(participantCount)/float64(rank))
}

// ComputeOracleBonus exports the bonus calculation for external use.
func ComputeOracleBonus(participantCount, rank int) float64 {
	return computeOracleBonus(participantCount, rank)
}

// GetPrediction returns a prediction by specter key.
func (p *OraclePool) GetPrediction(specterKey [32]byte) *Prediction {
	p.mu.RLock()
	defer p.mu.RUnlock()

	keyHex := mechanics.KeyToHex(specterKey[:])
	return p.predictions[keyHex]
}

// GetCommitment returns a commitment by specter key.
func (p *OraclePool) GetCommitment(specterKey [32]byte) *Commitment {
	p.mu.RLock()
	defer p.mu.RUnlock()

	keyHex := mechanics.KeyToHex(specterKey[:])
	return p.commitments[keyHex]
}

// GetAllPredictions returns all revealed predictions.
func (p *OraclePool) GetAllPredictions() []*Prediction {
	p.mu.RLock()
	defer p.mu.RUnlock()

	preds := make([]*Prediction, 0, len(p.predictions))
	for _, pred := range p.predictions {
		preds = append(preds, pred)
	}
	return preds
}

// GetWinners returns the rewarded predictors.
func (p *OraclePool) GetWinners() []*Prediction {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var winners []*Prediction
	for _, pred := range p.predictions {
		if pred.Rewarded {
			winners = append(winners, pred)
		}
	}

	// Sort by rank.
	sort.Slice(winners, func(i, j int) bool {
		return winners[i].Rank < winners[j].Rank
	})

	return winners
}

// ParticipantCount returns the number of revealed predictions.
func (p *OraclePool) ParticipantCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.predictions)
}

// CommitmentCount returns the number of commitments.
func (p *OraclePool) CommitmentCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.commitments)
}

// UpdateState updates the pool state based on current time.
func (p *OraclePool) UpdateState() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	if p.State == OraclePoolOpen && now.After(p.Deadline) {
		p.State = OraclePoolPending
	}
}

// OraclePoolStore manages Oracle Pools.
type OraclePoolStore struct {
	mu      sync.RWMutex
	pools   map[[32]byte]*OraclePool // By pool ID.
	active  []*OraclePool            // Active pools.
	history []*OraclePool            // Resolved/cancelled pools.
}

// NewOraclePoolStore creates a new Oracle Pool store.
func NewOraclePoolStore() *OraclePoolStore {
	return &OraclePoolStore{
		pools: make(map[[32]byte]*OraclePool),
	}
}

// AddPool adds a new Oracle Pool to the store.
func (s *OraclePoolStore) AddPool(p *OraclePool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pools[p.ID] = p
	s.active = append(s.active, p)
}

// GetPool retrieves a pool by ID.
func (s *OraclePoolStore) GetPool(id [32]byte) *OraclePool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.pools[id]
}

// GetActivePools returns all active (non-resolved) pools.
func (s *OraclePoolStore) GetActivePools() []*OraclePool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []*OraclePool
	for _, p := range s.active {
		if p.State != OraclePoolResolved && p.State != OraclePoolCancelled {
			active = append(active, p)
		}
	}
	return active
}

// GetOpenPools returns pools currently accepting predictions.
func (s *OraclePoolStore) GetOpenPools() []*OraclePool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var open []*OraclePool
	for _, p := range s.active {
		if p.IsOpen() {
			open = append(open, p)
		}
	}
	return open
}

// GetPendingPools returns pools awaiting resolution.
func (s *OraclePoolStore) GetPendingPools() []*OraclePool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pending []*OraclePool
	now := time.Now()
	for _, p := range s.active {
		p.mu.RLock()
		if p.State == OraclePoolPending || (p.State == OraclePoolOpen && now.After(p.Deadline)) {
			pending = append(pending, p)
		}
		p.mu.RUnlock()
	}
	return pending
}

// UpdatePoolStates updates state for all pools.
func (s *OraclePoolStore) UpdatePoolStates() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var stillActive []*OraclePool

	for _, p := range s.active {
		p.UpdateState()

		if p.State == OraclePoolResolved || p.State == OraclePoolCancelled {
			s.history = append(s.history, p)
		} else {
			stillActive = append(stillActive, p)
		}
	}

	s.active = stillActive
}

// GetPoolHistory returns resolved/cancelled pools.
func (s *OraclePoolStore) GetPoolHistory(limit int) []*OraclePool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.history) {
		limit = len(s.history)
	}

	result := make([]*OraclePool, limit)
	for i := 0; i < limit; i++ {
		result[i] = s.history[len(s.history)-1-i]
	}
	return result
}

// Count returns the number of active pools.
func (s *OraclePoolStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.active)
}

// GarbageCollect removes old history entries.
func (s *OraclePoolStore) GarbageCollect(maxHistory int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	var removed int
	s.history, removed = mechanics.GarbageCollectHistory(s.history, s.pools, maxHistory, func(p *OraclePool) [32]byte { return p.ID })
	return removed
}

// OraclePoolStateString returns the human-readable name of a pool state.
func OraclePoolStateString(s OraclePoolState) string {
	switch s {
	case OraclePoolOpen:
		return "Open"
	case OraclePoolPending:
		return "Pending"
	case OraclePoolResolved:
		return "Resolved"
	case OraclePoolCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// OraclePredictionTypeString returns the human-readable name of a prediction type.
func OraclePredictionTypeString(t OraclePredictionType) string {
	switch t {
	case OraclePredictionBoolean:
		return "Boolean"
	case OraclePredictionNumeric:
		return "Numeric"
	default:
		return "Unknown"
	}
}
