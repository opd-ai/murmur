// Package mechanics - Territory Drift implementation.
// Per ANONYMOUS_GAME_MECHANICS.md, Territory Drift is a persistent game
// where Specters claim and contest regions of the Pulse Map.
package territory

import (
	"math"
	"sync"
	"time"
)

// Territory constants per ANONYMOUS_GAME_MECHANICS.md.
const (
	// TerritoryMinResonance requires Resonance 25 (Shade milestone).
	TerritoryMinResonance = 25

	// TerritoryResetInterval is how often territories reset (weekly).
	TerritoryResetInterval = 7 * 24 * time.Hour

	// TerritoryActivityWindow is the lookback for influence (30 days).
	TerritoryActivityWindow = 30 * 24 * time.Hour

	// ContestThreshold is the percentage within which territories are contested.
	ContestThreshold = 0.20 // 20% per spec.
)

// TerritoryState represents the current state of a territory.
type TerritoryState uint8

const (
	TerritoryNeutral    TerritoryState = iota // No dominant controller.
	TerritoryControlled                       // Single controller.
	TerritoryContested                        // Two+ similar influence levels.
)

// InfluenceSource represents a source of territory influence.
type InfluenceSource uint8

const (
	InfluenceWaveAmplified InfluenceSource = iota // Waves amplified in territory.
	InfluenceConnection                           // Connections in territory.
	InfluenceMechanic                             // Mechanic participations.
)

// InfluenceEvent records a single influence-contributing event.
type InfluenceEvent struct {
	SpecterKey [32]byte        // Specter's public key.
	Source     InfluenceSource // What generated the influence.
	Amount     float64         // Contribution amount.
	Timestamp  time.Time       // When the event occurred.
}

// Territory represents a region of the Pulse Map.
type Territory struct {
	ID         string             // Unique territory identifier.
	CentroidX  float64            // X coordinate of territory center.
	CentroidY  float64            // Y coordinate of territory center.
	MemberKeys [][]byte           // Public keys of nodes in territory.
	State      TerritoryState     // Current state (neutral/controlled/contested).
	Controller *[32]byte          // Controller's Specter key (if controlled).
	Contenders [][32]byte         // Specters contesting (if contested).
	LastReset  time.Time          // When the territory was last reset.
	Influence  map[string]float64 // Influence by Specter key (hex).
	events     []InfluenceEvent   // Raw influence events.
	mu         sync.RWMutex
}

// NewTerritory creates a new territory.
func NewTerritory(id string, centroidX, centroidY float64) *Territory {
	return &Territory{
		ID:        id,
		CentroidX: centroidX,
		CentroidY: centroidY,
		State:     TerritoryNeutral,
		LastReset: time.Now(),
		Influence: make(map[string]float64),
	}
}

// AddInfluence records an influence event for a Specter.
// Per ANONYMOUS_GAME_MECHANICS.md, influence = 8 * ln(1 + activities).
func (t *Territory) AddInfluence(specterKey [32]byte, source InfluenceSource, amount float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	event := InfluenceEvent{
		SpecterKey: specterKey,
		Source:     source,
		Amount:     amount,
		Timestamp:  time.Now(),
	}

	t.events = append(t.events, event)
}

// ComputeInfluence calculates current influence for all Specters.
// Uses the formula from ANONYMOUS_GAME_MECHANICS.md:
// influence = 8 * ln(1 + waves_amplified + connections + mechanic_participations).
func (t *Territory) ComputeInfluence() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Clear current influence.
	t.Influence = make(map[string]float64)

	// Only count events within the activity window.
	cutoff := time.Now().Add(-TerritoryActivityWindow)

	// Aggregate raw activity counts per Specter.
	activity := make(map[string]float64)
	for _, event := range t.events {
		if event.Timestamp.After(cutoff) {
			hex := keyToHex(event.SpecterKey[:])
			activity[hex] += event.Amount
		}
	}

	// Apply the logarithmic influence formula.
	for hex, rawActivity := range activity {
		t.Influence[hex] = 8.0 * math.Log1p(rawActivity)
	}

	// Update territory state based on influence.
	t.updateState()
}

// updateState determines the territory's control state.
func (t *Territory) updateState() {
	if len(t.Influence) == 0 {
		t.setNeutralState()
		return
	}

	topKey, topInfluence := t.findTopInfluencer()
	if topInfluence == 0 {
		t.setNeutralState()
		return
	}

	contenders := t.findContenders(topInfluence)
	if len(contenders) > 1 {
		t.setContestedState(contenders)
	} else {
		t.setControlledState(topKey)
	}
}

// setNeutralState sets the territory to neutral with no controller.
func (t *Territory) setNeutralState() {
	t.State = TerritoryNeutral
	t.Controller = nil
	t.Contenders = nil
}

// findTopInfluencer returns the key and value of the highest influence holder.
func (t *Territory) findTopInfluencer() (string, float64) {
	var topKey string
	var topInfluence float64
	for key, influence := range t.Influence {
		if influence > topInfluence {
			topKey = key
			topInfluence = influence
		}
	}
	return topKey, topInfluence
}

// findContenders returns all specters within the contest threshold of the top.
func (t *Territory) findContenders(topInfluence float64) []string {
	threshold := topInfluence * (1 - ContestThreshold)
	var contenders []string
	for key, influence := range t.Influence {
		if influence >= threshold {
			contenders = append(contenders, key)
		}
	}
	return contenders
}

// setContestedState sets the territory as contested by multiple specters.
func (t *Territory) setContestedState(contenders []string) {
	t.State = TerritoryContested
	t.Controller = nil
	t.Contenders = make([][32]byte, 0, len(contenders))
	for _, hex := range contenders {
		var key [32]byte
		hexToKey(hex, key[:])
		t.Contenders = append(t.Contenders, key)
	}
}

// setControlledState sets the territory as controlled by a single specter.
func (t *Territory) setControlledState(topKey string) {
	t.State = TerritoryControlled
	var key [32]byte
	hexToKey(topKey, key[:])
	t.Controller = &key
	t.Contenders = nil
}

// hexToKey converts a hex string back to a byte array.
func hexToKey(hex string, dst []byte) {
	for i := 0; i < len(dst) && i*2+1 < len(hex); i++ {
		dst[i] = hexDigit(hex[i*2])<<4 | hexDigit(hex[i*2+1])
	}
}

func hexDigit(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}

// GetInfluence returns a Specter's current influence in this territory.
func (t *Territory) GetInfluence(specterKey [32]byte) float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	hex := keyToHex(specterKey[:])
	return t.Influence[hex]
}

// GetController returns the current controller, if any.
func (t *Territory) GetController() *[32]byte {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.Controller
}

// IsContested returns true if the territory is being contested.
func (t *Territory) IsContested() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.State == TerritoryContested
}

// ShouldReset returns true if the territory should reset.
func (t *Territory) ShouldReset() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return time.Since(t.LastReset) >= TerritoryResetInterval
}

// Reset clears territory influence for a new period.
func (t *Territory) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Influence = make(map[string]float64)
	t.events = nil
	t.State = TerritoryNeutral
	t.Controller = nil
	t.Contenders = nil
	t.LastReset = time.Now()
}

// GarbageCollect removes old events outside the activity window.
func (t *Territory) GarbageCollect() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-TerritoryActivityWindow)

	var kept []InfluenceEvent
	removed := 0

	for _, event := range t.events {
		if event.Timestamp.After(cutoff) {
			kept = append(kept, event)
		} else {
			removed++
		}
	}

	t.events = kept
	return removed
}

// TerritoryManager manages multiple territories.
type TerritoryManager struct {
	mu          sync.RWMutex
	territories map[string]*Territory
}

// NewTerritoryManager creates a new territory manager.
func NewTerritoryManager() *TerritoryManager {
	return &TerritoryManager{
		territories: make(map[string]*Territory),
	}
}

// AddTerritory adds a new territory.
func (m *TerritoryManager) AddTerritory(t *Territory) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.territories[t.ID] = t
}

// GetTerritory retrieves a territory by ID.
func (m *TerritoryManager) GetTerritory(id string) *Territory {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.territories[id]
}

// ListTerritories returns all territories.
func (m *TerritoryManager) ListTerritories() []*Territory {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Territory, 0, len(m.territories))
	for _, t := range m.territories {
		result = append(result, t)
	}
	return result
}

// GetControlledTerritories returns territories controlled by a Specter.
func (m *TerritoryManager) GetControlledTerritories(specterKey [32]byte) []*Territory {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Territory
	hex := keyToHex(specterKey[:])

	for _, t := range m.territories {
		t.mu.RLock()
		if t.Controller != nil {
			ctrlHex := keyToHex(t.Controller[:])
			if ctrlHex == hex {
				result = append(result, t)
			}
		}
		t.mu.RUnlock()
	}

	return result
}

// GetContestedTerritories returns territories the Specter is contesting.
func (m *TerritoryManager) GetContestedTerritories(specterKey [32]byte) []*Territory {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Territory
	hex := keyToHex(specterKey[:])

	for _, t := range m.territories {
		t.mu.RLock()
		for _, contender := range t.Contenders {
			if keyToHex(contender[:]) == hex {
				result = append(result, t)
				break
			}
		}
		t.mu.RUnlock()
	}

	return result
}

// ComputeAllInfluence recalculates influence for all territories.
func (m *TerritoryManager) ComputeAllInfluence() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.territories {
		t.ComputeInfluence()
	}
}

// ResetExpiredTerritories resets territories that have exceeded the reset interval.
func (m *TerritoryManager) ResetExpiredTerritories() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, t := range m.territories {
		if t.ShouldReset() {
			t.Reset()
			count++
		}
	}
	return count
}

// GarbageCollectAll runs garbage collection on all territories.
func (m *TerritoryManager) GarbageCollectAll() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0
	for _, t := range m.territories {
		total += t.GarbageCollect()
	}
	return total
}

// ComputeTerritoryScore calculates a Specter's territory Resonance contribution.
// Per ANONYMOUS_GAME_MECHANICS.md:
// territory_score = 3 * ln(1 + territories_controlled + 0.5 * territories_contested).
func (m *TerritoryManager) ComputeTerritoryScore(specterKey [32]byte) float64 {
	controlled := float64(len(m.GetControlledTerritories(specterKey)))
	contested := float64(len(m.GetContestedTerritories(specterKey)))

	return 3.0 * math.Log1p(controlled+0.5*contested)
}

// Count returns the total number of territories.
func (m *TerritoryManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.territories)
}

// RecordWaveAmplification records a Wave amplification event.
func (m *TerritoryManager) RecordWaveAmplification(territoryID string, specterKey [32]byte) {
	t := m.GetTerritory(territoryID)
	if t != nil {
		t.AddInfluence(specterKey, InfluenceWaveAmplified, 1.0)
	}
}

// RecordConnection records a connection event in a territory.
func (m *TerritoryManager) RecordConnection(territoryID string, specterKey [32]byte) {
	t := m.GetTerritory(territoryID)
	if t != nil {
		t.AddInfluence(specterKey, InfluenceConnection, 1.0)
	}
}

// RecordMechanicParticipation records mechanic participation in a territory.
func (m *TerritoryManager) RecordMechanicParticipation(territoryID string, specterKey [32]byte) {
	t := m.GetTerritory(territoryID)
	if t != nil {
		t.AddInfluence(specterKey, InfluenceMechanic, 1.0)
	}
}
