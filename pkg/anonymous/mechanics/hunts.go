// Package mechanics - Specter Hunts implementation.
// Per ANONYMOUS_GAME_MECHANICS.md, Specter Hunts are network-wide,
// time-limited scavenger hunts across the Pulse Map where Specters
// discover hidden fragments by exploring and decoding clues.
package mechanics

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

// Hunt constants per ANONYMOUS_GAME_MECHANICS.md.
const (
	// HuntMinResonance requires Resonance 75 (Shade-Wraith milestone) to initiate.
	HuntMinResonance = 75

	// Hunt durations (30, 60, or 120 minutes).
	HuntDuration30Min  = 30 * time.Minute
	HuntDuration60Min  = 60 * time.Minute
	HuntDuration120Min = 120 * time.Minute

	// Fragment count limits.
	HuntMinFragments = 5
	HuntMaxFragments = 20

	// Claim proximity (hops from target node).
	HuntClaimProximityHops = 3

	// Clue reveal interval.
	HuntClueRevealInterval = 10 * time.Minute

	// Resonance bonus decay period.
	HuntBonusDecayDays = 14
)

// HuntState represents the current state of a hunt.
type HuntState uint8

const (
	HuntPending   HuntState = iota // Not yet started.
	HuntActive                     // Hunt is active.
	HuntCompleted                  // All fragments claimed.
	HuntExpired                    // Time limit reached.
)

// Errors for hunt operations.
var (
	ErrHuntNotFound         = errors.New("hunt not found")
	ErrHuntExpired          = errors.New("hunt has expired")
	ErrHuntNotActive        = errors.New("hunt is not active")
	ErrFragmentNotFound     = errors.New("fragment not found")
	ErrFragmentClaimed      = errors.New("fragment already claimed")
	ErrInvalidHuntDuration  = errors.New("invalid hunt duration")
	ErrInvalidFragmentCount = errors.New("invalid fragment count")
	ErrHuntInsufficientRes  = errors.New("insufficient resonance to initiate hunt")
	ErrNotInProximity       = errors.New("not within claim proximity of fragment")
)

// Fragment represents a hidden token in the hunt.
type Fragment struct {
	Index         int       // Fragment index (0 to FragmentCount-1).
	LocationHash  [32]byte  // SHA-256(hunt_seed || fragment_index).
	TargetPeerID  string    // Peer ID nearest to this fragment.
	Claimed       bool      // Whether this fragment has been claimed.
	ClaimerKey    *[32]byte // Specter who claimed this fragment.
	ClaimedAt     *time.Time
	Clues         []string // Progressive clues revealed over time.
	CluesRevealed int      // Number of clues revealed so far.
}

// Hunt represents a Specter Hunt event.
type Hunt struct {
	ID            [32]byte      // Unique hunt ID.
	Theme         string        // Short description of the hunt.
	Seed          [32]byte      // Seed for deterministic fragment placement.
	InitiatorKey  [32]byte      // Specter who initiated the hunt.
	CreatedAt     time.Time     // When the hunt was announced.
	Duration      time.Duration // Time limit (30, 60, or 120 minutes).
	ExpiresAt     time.Time     // When the hunt expires.
	State         HuntState     // Current state.
	FragmentCount int           // Number of fragments to find.
	Fragments     []*Fragment   // The hidden fragments.

	mu sync.RWMutex
}

// NewHunt creates a new Specter Hunt with the given parameters.
func NewHunt(
	theme string,
	seed [32]byte,
	initiatorKey [32]byte,
	duration time.Duration,
	fragmentCount int,
	initiatorResonance int,
) (*Hunt, error) {
	// Validate initiator resonance per RESONANCE_SYSTEM.md.
	if initiatorResonance < HuntMinResonance {
		return nil, ErrHuntInsufficientRes
	}
	if err := validateHuntDuration(duration); err != nil {
		return nil, err
	}
	if err := validateFragmentCount(fragmentCount); err != nil {
		return nil, err
	}

	now := time.Now()
	huntID := generateHuntID(seed, initiatorKey, now)

	hunt := &Hunt{
		ID:            huntID,
		Theme:         theme,
		Seed:          seed,
		InitiatorKey:  initiatorKey,
		CreatedAt:     now,
		Duration:      duration,
		ExpiresAt:     now.Add(duration),
		State:         HuntActive,
		FragmentCount: fragmentCount,
	}

	hunt.Fragments = hunt.generateFragments()
	return hunt, nil
}

// validateHuntDuration checks if the duration is valid.
func validateHuntDuration(duration time.Duration) error {
	if duration != HuntDuration30Min &&
		duration != HuntDuration60Min &&
		duration != HuntDuration120Min {
		return ErrInvalidHuntDuration
	}
	return nil
}

// validateFragmentCount checks if the fragment count is valid.
func validateFragmentCount(count int) error {
	if count < HuntMinFragments || count > HuntMaxFragments {
		return ErrInvalidFragmentCount
	}
	return nil
}

// generateHuntID creates a deterministic hunt ID.
func generateHuntID(seed, initiatorKey [32]byte, createdAt time.Time) [32]byte {
	var huntID [32]byte
	h := blake3.New()
	h.Write(seed[:])
	h.Write(initiatorKey[:])
	binary.Write(h, binary.BigEndian, createdAt.Unix())
	copy(huntID[:], h.Sum(nil))
	return huntID
}

// generateFragments creates all fragments for this hunt.
func (h *Hunt) generateFragments() []*Fragment {
	fragments := make([]*Fragment, h.FragmentCount)
	for i := 0; i < h.FragmentCount; i++ {
		fragments[i] = h.generateFragment(i)
	}
	return fragments
}

// generateFragment creates a single fragment at the given index.
func (h *Hunt) generateFragment(index int) *Fragment {
	locationHash := computeFragmentLocation(h.Seed, index)
	clues := generateFragmentClues(locationHash, h.Theme)

	return &Fragment{
		Index:        index,
		LocationHash: locationHash,
		Clues:        clues,
	}
}

// computeFragmentLocation computes the location hash for a fragment.
// Per ANONYMOUS_GAME_MECHANICS.md: fragment_location = SHA-256(hunt_seed || fragment_index).
func computeFragmentLocation(seed [32]byte, index int) [32]byte {
	h := sha256.New()
	h.Write(seed[:])
	binary.Write(h, binary.BigEndian, uint32(index))
	var location [32]byte
	copy(location[:], h.Sum(nil))
	return location
}

// generateFragmentClues creates progressive clues for a fragment.
func generateFragmentClues(locationHash [32]byte, theme string) []string {
	clues := make([]string, 4)

	// Clue 1: Vague directional hint based on first bytes.
	clues[0] = generateDirectionalClue(locationHash)

	// Clue 2: XOR-distance range hint.
	clues[1] = generateDistanceClue(locationHash)

	// Clue 3: Partial hash reveal (first 4 hex chars).
	clues[2] = generatePartialHashClue(locationHash, 4)

	// Clue 4: More specific partial hash (first 8 hex chars).
	clues[3] = generatePartialHashClue(locationHash, 8)

	return clues
}

// generateDirectionalClue creates a vague directional hint.
func generateDirectionalClue(hash [32]byte) string {
	// Use first byte to determine quadrant.
	quadrant := hash[0] % 4
	directions := []string{
		"northeastern region",
		"southeastern region",
		"southwestern region",
		"northwestern region",
	}
	return "Fragment lies in the " + directions[quadrant]
}

// generateDistanceClue creates a distance range hint.
func generateDistanceClue(hash [32]byte) string {
	// Use second byte to estimate distance.
	distance := hash[1]
	if distance < 64 {
		return "Fragment is near the network core"
	} else if distance < 128 {
		return "Fragment is in the mid-range of the topology"
	} else if distance < 192 {
		return "Fragment is toward the network fringe"
	}
	return "Fragment is at the outer edge of the topology"
}

// generatePartialHashClue reveals partial hash bytes.
func generatePartialHashClue(hash [32]byte, hexChars int) string {
	bytes := hexChars / 2
	if bytes > len(hash) {
		bytes = len(hash)
	}
	return "Location prefix: " + keyToHex(hash[:bytes])
}

// IsExpired returns true if the hunt has passed its time limit.
func (h *Hunt) IsExpired() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return time.Now().After(h.ExpiresAt)
}

// IsCompleted returns true if all fragments have been claimed.
func (h *Hunt) IsCompleted() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.State == HuntCompleted
}

// IsActive returns true if the hunt is currently active.
func (h *Hunt) IsActive() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.State == HuntActive && !time.Now().After(h.ExpiresAt)
}

// GetFragment returns a fragment by index.
func (h *Hunt) GetFragment(index int) *Fragment {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if index < 0 || index >= len(h.Fragments) {
		return nil
	}
	return h.Fragments[index]
}

// GetUnclaimedFragments returns all unclaimed fragments.
func (h *Hunt) GetUnclaimedFragments() []*Fragment {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var unclaimed []*Fragment
	for _, f := range h.Fragments {
		if !f.Claimed {
			unclaimed = append(unclaimed, f)
		}
	}
	return unclaimed
}

// GetClaimedCount returns the number of claimed fragments.
func (h *Hunt) GetClaimedCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, f := range h.Fragments {
		if f.Claimed {
			count++
		}
	}
	return count
}

// RevealClues updates clue visibility based on elapsed time.
// New clues are revealed every HuntClueRevealInterval.
func (h *Hunt) RevealClues() {
	h.mu.Lock()
	defer h.mu.Unlock()

	elapsed := time.Since(h.CreatedAt)
	intervals := int(elapsed / HuntClueRevealInterval)

	for _, f := range h.Fragments {
		if intervals > f.CluesRevealed && f.CluesRevealed < len(f.Clues) {
			f.CluesRevealed = min(intervals, len(f.Clues))
		}
	}
}

// GetVisibleClues returns the currently visible clues for a fragment.
func (h *Hunt) GetVisibleClues(fragmentIndex int) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if fragmentIndex < 0 || fragmentIndex >= len(h.Fragments) {
		return nil
	}

	f := h.Fragments[fragmentIndex]
	if f.CluesRevealed == 0 {
		return []string{f.Clues[0]} // Always show at least the first clue.
	}
	return f.Clues[:min(f.CluesRevealed, len(f.Clues))]
}

// ClaimFragment attempts to claim a fragment.
// The claimer must be within HuntClaimProximityHops of the target node.
func (h *Hunt) ClaimFragment(
	fragmentIndex int,
	claimerKey [32]byte,
	proximityProof ProximityProof,
) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.checkHuntState(); err != nil {
		return err
	}
	if fragmentIndex < 0 || fragmentIndex >= len(h.Fragments) {
		return ErrFragmentNotFound
	}

	fragment := h.Fragments[fragmentIndex]
	if fragment.Claimed {
		return ErrFragmentClaimed
	}

	if !proximityProof.Verify(fragment.LocationHash, HuntClaimProximityHops) {
		return ErrNotInProximity
	}

	h.recordClaim(fragment, claimerKey)
	h.checkCompletion()
	return nil
}

// checkHuntState validates the hunt can accept claims.
func (h *Hunt) checkHuntState() error {
	if h.State == HuntExpired || time.Now().After(h.ExpiresAt) {
		h.State = HuntExpired
		return ErrHuntExpired
	}
	if h.State != HuntActive {
		return ErrHuntNotActive
	}
	return nil
}

// recordClaim records a successful fragment claim.
func (h *Hunt) recordClaim(fragment *Fragment, claimerKey [32]byte) {
	now := time.Now()
	fragment.Claimed = true
	fragment.ClaimerKey = &claimerKey
	fragment.ClaimedAt = &now
}

// checkCompletion updates hunt state if all fragments are claimed.
func (h *Hunt) checkCompletion() {
	for _, f := range h.Fragments {
		if !f.Claimed {
			return
		}
	}
	h.State = HuntCompleted
}

// ProximityProof proves the claimer is near a target location.
type ProximityProof struct {
	ClaimerPeerID  string   // Claimer's peer ID.
	ConnectedPeers []string // Peers the claimer is connected to.
	HopDistances   []int    // Hop distances from target.
}

// Verify checks if the proximity proof is valid.
func (p ProximityProof) Verify(targetHash [32]byte, maxHops int) bool {
	// Simplified verification: check if any connected peer
	// is within maxHops of the target based on XOR distance.
	// In production, this would use DHT routing to verify proximity.
	if len(p.HopDistances) == 0 {
		return false
	}
	for _, hops := range p.HopDistances {
		if hops <= maxHops {
			return true
		}
	}
	return false
}

// GetLeaderboard returns the hunt participants sorted by claims.
func (h *Hunt) GetLeaderboard() []HuntParticipant {
	h.mu.RLock()
	defer h.mu.RUnlock()

	scores := make(map[string]int)
	for _, f := range h.Fragments {
		if f.Claimed && f.ClaimerKey != nil {
			key := keyToHex(f.ClaimerKey[:])
			scores[key]++
		}
	}

	participants := make([]HuntParticipant, 0, len(scores))
	for key, claims := range scores {
		var k [32]byte
		hexToKey(key, k[:])
		participants = append(participants, HuntParticipant{
			SpecterKey: k,
			Claims:     claims,
		})
	}

	sortParticipants(participants)
	return participants
}

// HuntParticipant represents a participant in a hunt.
type HuntParticipant struct {
	SpecterKey [32]byte
	Claims     int
}

// sortParticipants sorts participants by claims (descending).
func sortParticipants(participants []HuntParticipant) {
	for i := 0; i < len(participants)-1; i++ {
		for j := i + 1; j < len(participants); j++ {
			if participants[j].Claims > participants[i].Claims {
				participants[i], participants[j] = participants[j], participants[i]
			}
		}
	}
}

// ComputeHuntBonus calculates the Resonance bonus for hunt participation.
// Based on ANONYMOUS_GAME_MECHANICS.md puzzle bonus formula adapted for hunts.
func ComputeHuntBonus(fragmentsClaimed, totalFragments int) float64 {
	if totalFragments == 0 {
		return 0
	}
	ratio := float64(fragmentsClaimed) / float64(totalFragments)
	return 5.0 * ratio * float64(fragmentsClaimed) // Base * ratio * claims.
}

// HuntStore manages active and historical hunts.
type HuntStore struct {
	mu      sync.RWMutex
	hunts   map[[32]byte]*Hunt // By hunt ID.
	active  []*Hunt            // Active hunts.
	history []*Hunt            // Completed/expired hunts.
}

// NewHuntStore creates a new hunt store.
func NewHuntStore() *HuntStore {
	return &HuntStore{
		hunts: make(map[[32]byte]*Hunt),
	}
}

// AddHunt adds a new hunt to the store.
func (s *HuntStore) AddHunt(h *Hunt) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.hunts[h.ID] = h
	s.active = append(s.active, h)
}

// GetHunt retrieves a hunt by ID.
func (s *HuntStore) GetHunt(id [32]byte) *Hunt {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.hunts[id]
}

// GetActiveHunts returns all active hunts.
func (s *HuntStore) GetActiveHunts() []*Hunt {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []*Hunt
	for _, h := range s.active {
		if h.IsActive() {
			active = append(active, h)
		}
	}
	return active
}

// UpdateHuntStates processes expired hunts and moves them to history.
func (s *HuntStore) UpdateHuntStates() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var stillActive []*Hunt
	now := time.Now()

	for _, h := range s.active {
		h.mu.Lock()
		if now.After(h.ExpiresAt) && h.State == HuntActive {
			h.State = HuntExpired
		}

		if h.State == HuntCompleted || h.State == HuntExpired {
			s.history = append(s.history, h)
		} else {
			stillActive = append(stillActive, h)
		}
		h.mu.Unlock()
	}

	s.active = stillActive
}

// RevealAllClues reveals clues for all active hunts.
func (s *HuntStore) RevealAllClues() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, h := range s.active {
		h.RevealClues()
	}
}

// GetHuntHistory returns completed/expired hunts.
func (s *HuntStore) GetHuntHistory(limit int) []*Hunt {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.history) {
		limit = len(s.history)
	}

	result := make([]*Hunt, limit)
	for i := 0; i < limit; i++ {
		result[i] = s.history[len(s.history)-1-i]
	}
	return result
}

// Count returns the number of active hunts.
func (s *HuntStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, h := range s.active {
		if h.IsActive() {
			count++
		}
	}
	return count
}

// GarbageCollect removes old history entries.
func (s *HuntStore) GarbageCollect(maxHistory int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	var removed int
	s.history, removed = GarbageCollectHistory(s.history, s.hunts, maxHistory, func(h *Hunt) [32]byte { return h.ID })
	return removed
}

// HuntStateString returns the human-readable name of a hunt state.
func HuntStateString(s HuntState) string {
	switch s {
	case HuntPending:
		return "Pending"
	case HuntActive:
		return "Active"
	case HuntCompleted:
		return "Completed"
	case HuntExpired:
		return "Expired"
	default:
		return "Unknown"
	}
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
