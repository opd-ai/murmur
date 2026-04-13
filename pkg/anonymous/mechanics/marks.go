// Package mechanics - Specter Marks implementation.
// Per ANONYMOUS_GAME_MECHANICS.md, marks are anonymous annotations
// on Surface nodes visible on the Pulse Map.
package mechanics

import (
	"crypto/ed25519"
	"errors"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

// Mark thresholds per ANONYMOUS_GAME_MECHANICS.md.
const (
	// MarkMinResonance requires Resonance 50 (Wraith milestone).
	MarkMinResonance = 50

	// MarkDuration is how long marks persist (30 days with decay).
	MarkDuration = 30 * 24 * time.Hour

	// MaxMarksPerTarget limits marks per target from the same Specter.
	MaxMarksPerTarget = 1
)

// MarkCategory identifies the type of mark.
type MarkCategory uint8

// Mark categories per ANONYMOUS_GAME_MECHANICS.md.
const (
	MarkWatcher MarkCategory = iota + 1 // Neutral observation.
	MarkAlly                            // Positive association.
	MarkRival                           // Competitive/adversarial.
)

// Errors for mark operations.
var (
	ErrMarkInsufficientResonance = errors.New("insufficient resonance for marks")
	ErrMarkAlreadyPlaced         = errors.New("mark already placed on this target")
	ErrMarkNotFound              = errors.New("mark not found")
	ErrInvalidMarkCategory       = errors.New("invalid mark category")
	ErrTargetRequired            = errors.New("target key required")
)

// Mark represents a Specter Mark placed on a Surface node.
// Marks are anonymous annotations visible on the Pulse Map.
type Mark struct {
	ID         [32]byte     // Unique mark ID (BLAKE3 hash).
	MarkerKey  [32]byte     // Specter's Curve25519 public key.
	TargetKey  []byte       // Target's Ed25519 public key.
	Category   MarkCategory // Watcher, Ally, or Rival.
	Note       string       // Optional short note (max 64 bytes).
	CreatedAt  time.Time    // When the mark was placed.
	ExpiresAt  time.Time    // When the mark decays (30 days).
	Visibility float64      // Visibility strength (1.0 to 0.0, decays over time).
	Signature  []byte       // Ed25519 signature for verification.
}

// IsExpired returns true if the mark has passed its expiration time.
func (m *Mark) IsExpired() bool {
	return time.Now().After(m.ExpiresAt)
}

// CurrentVisibility returns the mark's current visibility based on decay.
// Visibility decays linearly from 1.0 to 0.0 over the 30-day duration.
func (m *Mark) CurrentVisibility() float64 {
	if m.IsExpired() {
		return 0.0
	}

	elapsed := time.Since(m.CreatedAt)
	total := m.ExpiresAt.Sub(m.CreatedAt)

	if total == 0 {
		return 1.0
	}

	remaining := 1.0 - float64(elapsed)/float64(total)
	if remaining < 0 {
		return 0.0
	}
	return remaining
}

// CategoryString returns the human-readable name of a mark category.
func CategoryString(cat MarkCategory) string {
	switch cat {
	case MarkWatcher:
		return "Watcher"
	case MarkAlly:
		return "Ally"
	case MarkRival:
		return "Rival"
	default:
		return "Unknown"
	}
}

// CategoryDescription returns a description of the mark category.
func CategoryDescription(cat MarkCategory) string {
	switch cat {
	case MarkWatcher:
		return "Neutral observation - keeping an eye on this node"
	case MarkAlly:
		return "Positive association - a mark of support or respect"
	case MarkRival:
		return "Competitive relationship - a worthy opponent"
	default:
		return "Unknown mark type"
	}
}

// MarkStore manages Specter Mark storage.
type MarkStore struct {
	mu       sync.RWMutex
	marks    map[[32]byte]*Mark // By mark ID.
	byTarget map[string][]*Mark // By target key (hex).
	byMarker map[string][]*Mark // By marker key (hex).
	// Track which Specter has marked which target.
	markerTargets map[string]map[string]bool // marker -> target -> exists.
}

// NewMarkStore creates a new mark store.
func NewMarkStore() *MarkStore {
	return &MarkStore{
		marks:         make(map[[32]byte]*Mark),
		byTarget:      make(map[string][]*Mark),
		byMarker:      make(map[string][]*Mark),
		markerTargets: make(map[string]map[string]bool),
	}
}

// CanPlaceMark checks if a Specter can place a mark on a target.
func (s *MarkStore) CanPlaceMark(markerKey [32]byte, targetKey []byte, resonance int) error {
	if resonance < MarkMinResonance {
		return ErrMarkInsufficientResonance
	}

	if len(targetKey) == 0 {
		return ErrTargetRequired
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	markerHex := keyToHex(markerKey[:])
	targetHex := keyToHex(targetKey)

	if targets, ok := s.markerTargets[markerHex]; ok {
		if targets[targetHex] {
			return ErrMarkAlreadyPlaced
		}
	}

	return nil
}

// PlaceMark creates a new Specter Mark on a target.
func (s *MarkStore) PlaceMark(
	markerKey [32]byte,
	targetKey []byte,
	category MarkCategory,
	note string,
	resonance int,
	signingKey ed25519.PrivateKey,
) (*Mark, error) {
	// Validate inputs.
	if err := s.CanPlaceMark(markerKey, targetKey, resonance); err != nil {
		return nil, err
	}

	if category < MarkWatcher || category > MarkRival {
		return nil, ErrInvalidMarkCategory
	}

	// Truncate note to 64 bytes.
	if len(note) > 64 {
		note = note[:64]
	}

	now := time.Now()
	mark := &Mark{
		MarkerKey:  markerKey,
		TargetKey:  targetKey,
		Category:   category,
		Note:       note,
		CreatedAt:  now,
		ExpiresAt:  now.Add(MarkDuration),
		Visibility: 1.0,
	}

	// Generate mark ID as BLAKE3 hash.
	h := blake3.New()
	h.Write(markerKey[:])
	h.Write(targetKey)
	h.Write([]byte{byte(category)})
	var timestamp [8]byte
	now.UnmarshalBinary(timestamp[:])
	h.Write(timestamp[:])
	copy(mark.ID[:], h.Sum(nil))

	// Sign the mark.
	if signingKey != nil {
		signData := append(mark.ID[:], mark.TargetKey...)
		signData = append(signData, byte(category))
		signData = append(signData, []byte(note)...)
		mark.Signature = ed25519.Sign(signingKey, signData)
	}

	// Store the mark.
	s.mu.Lock()
	defer s.mu.Unlock()

	s.marks[mark.ID] = mark

	markerHex := keyToHex(markerKey[:])
	s.byMarker[markerHex] = append(s.byMarker[markerHex], mark)

	targetHex := keyToHex(targetKey)
	s.byTarget[targetHex] = append(s.byTarget[targetHex], mark)

	// Track marker-target relationship.
	if s.markerTargets[markerHex] == nil {
		s.markerTargets[markerHex] = make(map[string]bool)
	}
	s.markerTargets[markerHex][targetHex] = true

	return mark, nil
}

// GetMark retrieves a mark by ID.
func (s *MarkStore) GetMark(id [32]byte) (*Mark, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mark, ok := s.marks[id]
	if !ok {
		return nil, ErrMarkNotFound
	}

	if mark.IsExpired() {
		return nil, ErrMarkNotFound
	}

	return mark, nil
}

// GetMarksOnTarget returns all active marks on a target.
func (s *MarkStore) GetMarksOnTarget(targetKey []byte) []*Mark {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hex := keyToHex(targetKey)
	all := s.byTarget[hex]

	var active []*Mark
	for _, m := range all {
		if !m.IsExpired() {
			active = append(active, m)
		}
	}

	return active
}

// GetMarksByMarker returns all active marks placed by a Specter.
func (s *MarkStore) GetMarksByMarker(markerKey [32]byte) []*Mark {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hex := keyToHex(markerKey[:])
	all := s.byMarker[hex]

	var active []*Mark
	for _, m := range all {
		if !m.IsExpired() {
			active = append(active, m)
		}
	}

	return active
}

// RemoveMark removes a mark by ID.
func (s *MarkStore) RemoveMark(id [32]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mark, ok := s.marks[id]
	if !ok {
		return ErrMarkNotFound
	}

	delete(s.marks, id)

	// Update marker-target tracking.
	markerHex := keyToHex(mark.MarkerKey[:])
	targetHex := keyToHex(mark.TargetKey)

	if targets, ok := s.markerTargets[markerHex]; ok {
		delete(targets, targetHex)
		if len(targets) == 0 {
			delete(s.markerTargets, markerHex)
		}
	}

	return nil
}

// VerifyMark verifies a mark's signature.
func VerifyMark(mark *Mark, publicKey ed25519.PublicKey) bool {
	if mark == nil || len(mark.Signature) == 0 {
		return false
	}

	signData := append(mark.ID[:], mark.TargetKey...)
	signData = append(signData, byte(mark.Category))
	signData = append(signData, []byte(mark.Note)...)

	return ed25519.Verify(publicKey, signData, mark.Signature)
}

// GarbageCollect removes expired marks.
func (s *MarkStore) GarbageCollect() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := s.removeExpiredMarks()
	s.rebuildMarkIndexes()

	return removed
}

// removeExpiredMarks deletes expired marks and cleans up tracking.
func (s *MarkStore) removeExpiredMarks() int {
	removed := 0
	for id, mark := range s.marks {
		if mark.IsExpired() {
			s.cleanMarkerTargetTracking(mark)
			delete(s.marks, id)
			removed++
		}
	}
	return removed
}

// cleanMarkerTargetTracking removes a mark from the marker-target relationship.
func (s *MarkStore) cleanMarkerTargetTracking(mark *Mark) {
	markerHex := keyToHex(mark.MarkerKey[:])
	targetHex := keyToHex(mark.TargetKey)

	if targets, ok := s.markerTargets[markerHex]; ok {
		delete(targets, targetHex)
		if len(targets) == 0 {
			delete(s.markerTargets, markerHex)
		}
	}
}

// rebuildMarkIndexes rebuilds the target and marker indexes.
func (s *MarkStore) rebuildMarkIndexes() {
	s.byTarget = make(map[string][]*Mark)
	s.byMarker = make(map[string][]*Mark)

	for _, mark := range s.marks {
		markerHex := keyToHex(mark.MarkerKey[:])
		s.byMarker[markerHex] = append(s.byMarker[markerHex], mark)

		targetHex := keyToHex(mark.TargetKey)
		s.byTarget[targetHex] = append(s.byTarget[targetHex], mark)
	}
}

// Count returns the number of active (non-expired) marks.
func (s *MarkStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, mark := range s.marks {
		if !mark.IsExpired() {
			count++
		}
	}
	return count
}

// CountMarksByCategory returns counts of marks by category on a target.
func (s *MarkStore) CountMarksByCategory(targetKey []byte) map[MarkCategory]int {
	marks := s.GetMarksOnTarget(targetKey)

	counts := make(map[MarkCategory]int)
	for _, m := range marks {
		counts[m.Category]++
	}

	return counts
}

// GetDominantMark returns the most common mark category on a target.
// Returns MarkWatcher if no marks exist.
func (s *MarkStore) GetDominantMark(targetKey []byte) MarkCategory {
	counts := s.CountMarksByCategory(targetKey)

	if len(counts) == 0 {
		return MarkWatcher
	}

	dominant := MarkWatcher
	maxCount := 0

	for cat, count := range counts {
		if count > maxCount {
			dominant = cat
			maxCount = count
		}
	}

	return dominant
}
