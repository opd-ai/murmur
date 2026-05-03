// Package mechanics - Oracle Pool outcome verification.
// Per ANONYMOUS_GAME_MECHANICS.md: "Resolution is deterministic — any node can
// independently verify the outcome by observing the specified metric."
// Per ROADMAP.md line 463: "Outcome verification — network-observable event
// confirmation protocol".
package mechanics

import (
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

// Observable metric types for Oracle Pool outcomes.
// These are network-observable events that any node can independently verify.
type ObservableMetricType uint8

const (
	// MetricGossipVolume tracks message count on a GossipSub topic over a period.
	// Example: "Will daily gossip volume on /murmur/waves/1 exceed 10,000?"
	MetricGossipVolume ObservableMetricType = iota

	// MetricTerritoryCount tracks the number of territories in a region.
	// Example: "Will a new territory form in the eastern cluster within 7 days?"
	MetricTerritoryCount

	// MetricEventParticipation tracks participation count in Masked Events.
	// Example: "Will Masked Event participation exceed 50 this week?"
	MetricEventParticipation

	// MetricNodeCount tracks the number of active nodes in the network.
	// Example: "Will node count exceed 1000 by end of month?"
	MetricNodeCount

	// MetricWaveCount tracks the number of Waves published in a period.
	// Example: "Will Wave publications exceed 5000 this week?"
	MetricWaveCount

	// MetricSpecterCount tracks the number of active Specters.
	// Example: "Will Specter count double in the next month?"
	MetricSpecterCount

	// MetricHuntSuccess tracks Specter Hunt completion rate.
	// Example: "Will more than 50% of hunts succeed this week?"
	MetricHuntSuccess

	// MetricCustom allows user-defined metrics with explicit verification.
	MetricCustom
)

// Errors for outcome verification.
var (
	ErrVerificationNotStarted   = errors.New("verification not yet started")
	ErrVerificationNotComplete  = errors.New("verification not yet complete")
	ErrVerificationFailed       = errors.New("verification failed")
	ErrInvalidMetricType        = errors.New("invalid metric type")
	ErrMissingObservableData    = errors.New("missing observable data")
	ErrOutcomeConflict          = errors.New("outcome conflict between nodes")
	ErrInsufficientConfirmation = errors.New("insufficient confirmation votes")
)

// MetricObservation represents a single node's observation of a metric.
type MetricObservation struct {
	ObserverKey  [32]byte             // Observer's public key.
	PoolID       [32]byte             // Pool being verified.
	MetricType   ObservableMetricType // Type of metric observed.
	Value        float64              // Observed value.
	Timestamp    time.Time            // When observation was made.
	Signature    [64]byte             // Ed25519 signature over observation data.
	ObserverHash [32]byte             // BLAKE3 hash of observer + pool + value.
}

// OutcomeVote represents a node's vote on a pool outcome.
type OutcomeVote struct {
	VoterKey  [32]byte  // Voter's public key.
	PoolID    [32]byte  // Pool being verified.
	Outcome   float64   // Voted outcome value.
	Timestamp time.Time // When vote was cast.
	Signature [64]byte  // Ed25519 signature.
}

// VerificationState represents the state of outcome verification.
type VerificationState uint8

const (
	VerificationPending   VerificationState = iota // Awaiting observations.
	VerificationCollect                            // Collecting observations.
	VerificationConsensus                          // Reaching consensus.
	VerificationConfirmed                          // Outcome confirmed.
	VerificationDisputed                           // Outcome disputed.
	VerificationFailed                             // Verification failed.
)

// VerificationResult contains the final verification outcome.
type VerificationResult struct {
	PoolID           [32]byte
	ConfirmedOutcome float64
	Confirmations    int       // Number of confirming nodes.
	Disputes         int       // Number of disputing nodes.
	ConfirmedAt      time.Time // When consensus was reached.
	State            VerificationState
}

// OutcomeVerifier manages the verification of Oracle Pool outcomes.
type OutcomeVerifier struct {
	mu sync.RWMutex

	// Pending verifications by pool ID.
	pending map[[32]byte]*PoolVerification

	// Completed verifications.
	completed map[[32]byte]*VerificationResult

	// Configuration.
	config VerificationConfig

	// Metric observers (injected dependencies).
	observers map[ObservableMetricType]MetricObserver
}

// VerificationConfig holds configuration for outcome verification.
type VerificationConfig struct {
	// MinConfirmations required to confirm an outcome.
	MinConfirmations int

	// ConsensusThreshold is the percentage of agreement needed (0.0-1.0).
	ConsensusThreshold float64

	// ObservationWindow is how long to collect observations.
	ObservationWindow time.Duration

	// MaxValueDelta is the maximum difference allowed between observations.
	MaxValueDelta float64
}

// DefaultVerificationConfig returns sensible defaults.
func DefaultVerificationConfig() VerificationConfig {
	return VerificationConfig{
		MinConfirmations:   3,
		ConsensusThreshold: 0.66, // 2/3 majority.
		ObservationWindow:  5 * time.Minute,
		MaxValueDelta:      0.01, // 1% tolerance for numeric values.
	}
}

// MetricObserver is an interface for observing network metrics.
// Implementations query the network state to provide metric values.
type MetricObserver interface {
	// Observe returns the current value of the metric.
	Observe(params MetricParams) (float64, error)

	// MetricType returns the type this observer handles.
	MetricType() ObservableMetricType
}

// MetricParams contains parameters for metric observation.
type MetricParams struct {
	Topic      string    // GossipSub topic (for volume metrics).
	StartTime  time.Time // Period start.
	EndTime    time.Time // Period end.
	Region     string    // Spatial region (for territory metrics).
	Threshold  float64   // Threshold value (for comparison queries).
	CustomData []byte    // Additional data for custom metrics.
}

// PoolVerification tracks the verification state for a single pool.
type PoolVerification struct {
	PoolID       [32]byte
	MetricType   ObservableMetricType
	MetricParams MetricParams
	State        VerificationState
	StartedAt    time.Time

	// Observations from different nodes.
	observations map[string]*MetricObservation // By observer key hex.

	// Votes on the outcome.
	votes map[string]*OutcomeVote // By voter key hex.

	// Computed consensus value.
	consensusValue   *float64
	consensusReached bool
}

// NewOutcomeVerifier creates a new outcome verifier.
func NewOutcomeVerifier(config VerificationConfig) *OutcomeVerifier {
	return &OutcomeVerifier{
		pending:   make(map[[32]byte]*PoolVerification),
		completed: make(map[[32]byte]*VerificationResult),
		config:    config,
		observers: make(map[ObservableMetricType]MetricObserver),
	}
}

// RegisterObserver registers a metric observer.
func (v *OutcomeVerifier) RegisterObserver(obs MetricObserver) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.observers[obs.MetricType()] = obs
}

// StartVerification initiates verification for an Oracle Pool.
func (v *OutcomeVerifier) StartVerification(
	poolID [32]byte,
	metricType ObservableMetricType,
	params MetricParams,
) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, exists := v.pending[poolID]; exists {
		return nil // Already started.
	}

	if _, exists := v.completed[poolID]; exists {
		return nil // Already completed.
	}

	pv := &PoolVerification{
		PoolID:       poolID,
		MetricType:   metricType,
		MetricParams: params,
		State:        VerificationCollect,
		StartedAt:    time.Now(),
		observations: make(map[string]*MetricObservation),
		votes:        make(map[string]*OutcomeVote),
	}

	v.pending[poolID] = pv
	return nil
}

// SubmitObservation records an observation from a node.
func (v *OutcomeVerifier) SubmitObservation(obs *MetricObservation) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	pv, exists := v.pending[obs.PoolID]
	if !exists {
		return ErrVerificationNotStarted
	}

	// Accept observations during Collect, Consensus, or Confirmed phases.
	// Once Disputed or Failed, no more observations are accepted.
	switch pv.State {
	case VerificationCollect, VerificationConsensus, VerificationConfirmed:
		// OK to continue.
	default:
		return ErrVerificationNotStarted
	}

	// Verify the observation hash.
	expectedHash := computeObservationHash(obs.ObserverKey, obs.PoolID, obs.Value)
	if expectedHash != obs.ObserverHash {
		return ErrVerificationFailed
	}

	keyHex := keyToHex(obs.ObserverKey[:])
	pv.observations[keyHex] = obs

	// Check if we have enough observations to proceed to consensus.
	if len(pv.observations) >= v.config.MinConfirmations && pv.State == VerificationCollect {
		pv.State = VerificationConsensus
	}

	// Recompute consensus if not yet confirmed.
	if pv.State == VerificationConsensus {
		v.computeConsensus(pv)
	}

	return nil
}

// SubmitVote records a vote on the outcome.
func (v *OutcomeVerifier) SubmitVote(vote *OutcomeVote) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	pv, exists := v.pending[vote.PoolID]
	if !exists {
		return ErrVerificationNotStarted
	}

	keyHex := keyToHex(vote.VoterKey[:])
	pv.votes[keyHex] = vote

	// Update consensus if in consensus phase.
	if pv.State == VerificationConsensus {
		v.computeConsensus(pv)
	}

	return nil
}

// computeConsensus determines the consensus outcome.
func (v *OutcomeVerifier) computeConsensus(pv *PoolVerification) {
	if len(pv.observations) < v.config.MinConfirmations {
		return
	}

	// Collect all observed values.
	var values []float64
	for _, obs := range pv.observations {
		values = append(values, obs.Value)
	}

	// Check for consensus (values within tolerance).
	consensusValue, agreed := findConsensusValue(values, v.config.MaxValueDelta)
	if !agreed {
		pv.State = VerificationDisputed
		return
	}

	// Count confirmations.
	confirmations := countConfirmations(values, consensusValue, v.config.MaxValueDelta)
	total := len(values)

	ratio := float64(confirmations) / float64(total)
	if ratio >= v.config.ConsensusThreshold {
		pv.consensusValue = &consensusValue
		pv.consensusReached = true
		pv.State = VerificationConfirmed
	}
}

// findConsensusValue finds a value that most observations agree on.
func findConsensusValue(values []float64, delta float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}

	// For boolean predictions (0 or 1), use majority.
	allBoolean := true
	for _, v := range values {
		if v != 0 && v != 1 {
			allBoolean = false
			break
		}
	}

	if allBoolean {
		var ones int
		for _, v := range values {
			if v == 1 {
				ones++
			}
		}
		if ones > len(values)/2 {
			return 1, true
		}
		return 0, true
	}

	// For numeric predictions, find the median.
	median := computeMedian(values)

	// Check if most values are within delta of median.
	within := 0
	for _, v := range values {
		diff := v - median
		if diff < 0 {
			diff = -diff
		}
		if diff <= delta*median {
			within++
		}
	}

	if within > len(values)/2 {
		return median, true
	}

	return 0, false
}

// computeMedian calculates the median of a slice.
func computeMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Copy to avoid modifying original.
	sorted := make([]float64, len(values))
	copy(sorted, values)

	// Simple bubble sort for small slices.
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// countConfirmations counts how many values agree with the consensus.
func countConfirmations(values []float64, consensus, delta float64) int {
	count := 0
	for _, v := range values {
		diff := v - consensus
		if diff < 0 {
			diff = -diff
		}
		if diff <= delta*consensus || (consensus == 0 && v == 0) {
			count++
		}
	}
	return count
}

// FinalizeVerification completes the verification and returns the result.
func (v *OutcomeVerifier) FinalizeVerification(poolID [32]byte) (*VerificationResult, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	pv, exists := v.pending[poolID]
	if !exists {
		// Check if already completed.
		if result, ok := v.completed[poolID]; ok {
			return result, nil
		}
		return nil, ErrVerificationNotStarted
	}

	if !pv.consensusReached {
		// Check if observation window has passed.
		if time.Since(pv.StartedAt) < v.config.ObservationWindow {
			return nil, ErrVerificationNotComplete
		}
		// Window passed without consensus.
		pv.State = VerificationFailed
	}

	result := &VerificationResult{
		PoolID:      poolID,
		State:       pv.State,
		ConfirmedAt: time.Now(),
	}

	if pv.consensusReached && pv.consensusValue != nil {
		result.ConfirmedOutcome = *pv.consensusValue
		result.Confirmations = countConfirmations(
			v.getObservationValues(pv),
			*pv.consensusValue,
			v.config.MaxValueDelta,
		)
		result.Disputes = len(pv.observations) - result.Confirmations
	}

	// Move to completed.
	delete(v.pending, poolID)
	v.completed[poolID] = result

	return result, nil
}

// getObservationValues extracts values from observations.
func (v *OutcomeVerifier) getObservationValues(pv *PoolVerification) []float64 {
	var values []float64
	for _, obs := range pv.observations {
		values = append(values, obs.Value)
	}
	return values
}

// GetVerificationState returns the current state for a pool.
func (v *OutcomeVerifier) GetVerificationState(poolID [32]byte) (VerificationState, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if pv, exists := v.pending[poolID]; exists {
		return pv.State, true
	}
	if result, exists := v.completed[poolID]; exists {
		return result.State, true
	}
	return VerificationPending, false
}

// GetResult returns the verification result if available.
func (v *OutcomeVerifier) GetResult(poolID [32]byte) (*VerificationResult, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	result, exists := v.completed[poolID]
	return result, exists
}

// ObserveAndSubmit uses a registered observer to make an observation.
func (v *OutcomeVerifier) ObserveAndSubmit(
	poolID [32]byte,
	observerKey [32]byte,
	sign func([]byte) [64]byte,
) error {
	v.mu.RLock()
	pv, exists := v.pending[poolID]
	if !exists {
		v.mu.RUnlock()
		return ErrVerificationNotStarted
	}
	metricType := pv.MetricType
	params := pv.MetricParams
	v.mu.RUnlock()

	observer, ok := v.observers[metricType]
	if !ok {
		return ErrInvalidMetricType
	}

	value, err := observer.Observe(params)
	if err != nil {
		return err
	}

	obs := &MetricObservation{
		ObserverKey:  observerKey,
		PoolID:       poolID,
		MetricType:   metricType,
		Value:        value,
		Timestamp:    time.Now(),
		ObserverHash: computeObservationHash(observerKey, poolID, value),
	}

	// Sign the observation.
	obs.Signature = sign(obs.ObserverHash[:])

	return v.SubmitObservation(obs)
}

// computeObservationHash computes a hash for an observation.
func computeObservationHash(observerKey, poolID [32]byte, value float64) [32]byte {
	h := blake3.New()
	h.Write(observerKey[:])
	h.Write(poolID[:])
	binary.Write(h, binary.BigEndian, value)
	var hash [32]byte
	copy(hash[:], h.Sum(nil))
	return hash
}

// GossipVolumeObserver observes GossipSub message volume.
