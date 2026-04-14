// Package mechanics - Echo Chains implementation.
// Per ANONYMOUS_GAME_MECHANICS.md: "Echo Chains incentivize meaningful amplification
// by creating visible, rewarded chains of re-broadcast."
package mechanics

import (
	"errors"
	"math"
	"sync"
	"time"
)

// Echo Chain constants per ANONYMOUS_GAME_MECHANICS.md.
const (
	// EchoChainMinLength is the minimum amplifiers needed to form a chain.
	EchoChainMinLength = 3

	// EchoChainDuration is how long chains persist on the Pulse Map.
	EchoChainDuration = 1 * time.Hour

	// EchoChainShimmerThreshold is the length at which shimmer effect activates.
	EchoChainShimmerThreshold = 5
)

// EchoChainLayer indicates which layer the chain belongs to.
type EchoChainLayer uint8

// Chain layers for rendering.
const (
	EchoChainSurface   EchoChainLayer = iota + 1 // Golden arcs.
	EchoChainAnonymous                           // Silver arcs.
)

// EchoChain errors.
var (
	ErrChainNotFound     = errors.New("echo chain not found")
	ErrChainTooShort     = errors.New("chain too short to form")
	ErrChainDuplicate    = errors.New("node already in chain")
	ErrInvalidChainLayer = errors.New("invalid chain layer")
)

// EchoChainNode represents a node in an echo chain.
type EchoChainNode struct {
	NodeID      []byte    // Ed25519 public key of the amplifier.
	WaveID      [32]byte  // ID of the Wave at this point in the chain.
	AmplifiedAt time.Time // When this amplification occurred.
	Position    int       // Position in chain (0 = original author).
}

// EchoChain represents a chain of amplifications.
type EchoChain struct {
	ID         [32]byte         // Unique chain ID (derived from original Wave).
	OriginalID [32]byte         // ID of the original Wave being amplified.
	Nodes      []*EchoChainNode // Ordered list of chain participants.
	Layer      EchoChainLayer   // Surface or Anonymous.
	FormedAt   time.Time        // When chain reached minimum length.
	ExpiresAt  time.Time        // When chain visual expires.
	HasShimmer bool             // True if chain length >= 5.
	TotalBonus float64          // Sum of all Resonance bonuses awarded.
}

// Length returns the number of nodes in the chain.
func (c *EchoChain) Length() int {
	return len(c.Nodes)
}

// IsExpired returns true if the chain has expired.
func (c *EchoChain) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// CalculateBonus computes the Resonance bonus for a chain of length N.
// Per spec: echo_chain_bonus = 1 * ln(N)
func CalculateEchoChainBonus(length int) float64 {
	if length < EchoChainMinLength {
		return 0.0
	}
	return math.Log(float64(length))
}

// EchoChainLayerString returns the display name for a layer.
func EchoChainLayerString(layer EchoChainLayer) string {
	switch layer {
	case EchoChainSurface:
		return "Surface"
	case EchoChainAnonymous:
		return "Anonymous"
	default:
		return "Unknown"
	}
}

// EchoChainStore manages echo chain storage and tracking.
type EchoChainStore struct {
	mu            sync.RWMutex
	chains        map[[32]byte]*EchoChain // By chain ID.
	byOriginal    map[[32]byte]*EchoChain // By original Wave ID.
	pendingChains map[[32]byte]*EchoChain // Chains building but not yet formed.
	nodeBonus     map[string]float64      // Accumulated bonus by node key (hex).
	activeChains  map[[32]byte]bool       // Currently visible chains.
}

// NewEchoChainStore creates a new echo chain store.
func NewEchoChainStore() *EchoChainStore {
	return &EchoChainStore{
		chains:        make(map[[32]byte]*EchoChain),
		byOriginal:    make(map[[32]byte]*EchoChain),
		pendingChains: make(map[[32]byte]*EchoChain),
		nodeBonus:     make(map[string]float64),
		activeChains:  make(map[[32]byte]bool),
	}
}

// RecordAmplification records an amplification event and potentially forms/extends a chain.
func (s *EchoChainStore) RecordAmplification(
	originalWaveID [32]byte,
	amplifierID []byte,
	amplificationWaveID [32]byte,
	layer EchoChainLayer,
) (*EchoChain, error) {
	if layer != EchoChainSurface && layer != EchoChainAnonymous {
		return nil, ErrInvalidChainLayer
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if chain already exists or is pending.
	chain, exists := s.pendingChains[originalWaveID]
	if !exists {
		chain, exists = s.chains[originalWaveID]
	}

	if !exists {
		// Create new pending chain.
		chain = &EchoChain{
			ID:         originalWaveID, // Chain ID = original Wave ID.
			OriginalID: originalWaveID,
			Nodes:      make([]*EchoChainNode, 0),
			Layer:      layer,
		}
		s.pendingChains[originalWaveID] = chain
	}

	// Check for duplicate amplifier.
	amplifierHex := keyToHex(amplifierID)
	for _, node := range chain.Nodes {
		if keyToHex(node.NodeID) == amplifierHex {
			return nil, ErrChainDuplicate
		}
	}

	// Add new node to chain.
	now := time.Now()
	node := &EchoChainNode{
		NodeID:      amplifierID,
		WaveID:      amplificationWaveID,
		AmplifiedAt: now,
		Position:    len(chain.Nodes),
	}
	chain.Nodes = append(chain.Nodes, node)

	// Check if chain just reached minimum length.
	if len(chain.Nodes) == EchoChainMinLength {
		chain.FormedAt = now
		chain.ExpiresAt = now.Add(EchoChainDuration)
		chain.HasShimmer = false

		// Move from pending to active.
		delete(s.pendingChains, originalWaveID)
		s.chains[originalWaveID] = chain
		s.byOriginal[originalWaveID] = chain
		s.activeChains[chain.ID] = true

		// Award bonuses to all participants.
		s.awardBonusesLocked(chain)
	} else if len(chain.Nodes) > EchoChainMinLength {
		// Chain extended - update shimmer and award incremental bonus.
		if len(chain.Nodes) >= EchoChainShimmerThreshold {
			chain.HasShimmer = true
		}
		// Award bonus to new participant.
		bonus := CalculateEchoChainBonus(len(chain.Nodes))
		s.nodeBonus[amplifierHex] += bonus
		chain.TotalBonus += bonus
	}

	return chain, nil
}

// awardBonusesLocked awards Resonance bonuses to all chain participants.
// Must be called with lock held.
func (s *EchoChainStore) awardBonusesLocked(chain *EchoChain) {
	bonus := CalculateEchoChainBonus(len(chain.Nodes))
	for _, node := range chain.Nodes {
		nodeHex := keyToHex(node.NodeID)
		s.nodeBonus[nodeHex] += bonus
		chain.TotalBonus += bonus
	}
}

// GetChain retrieves a chain by ID.
func (s *EchoChainStore) GetChain(id [32]byte) (*EchoChain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chain, ok := s.chains[id]
	if !ok {
		return nil, ErrChainNotFound
	}
	return chain, nil
}

// GetChainByOriginal retrieves a chain by the original Wave ID.
func (s *EchoChainStore) GetChainByOriginal(originalWaveID [32]byte) (*EchoChain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chain, ok := s.byOriginal[originalWaveID]
	if !ok {
		// Check pending chains.
		chain, ok = s.pendingChains[originalWaveID]
		if !ok {
			return nil, ErrChainNotFound
		}
	}
	return chain, nil
}

// GetActiveChains returns all non-expired chains.
func (s *EchoChainStore) GetActiveChains() []*EchoChain {
	s.mu.RLock()
	defer s.mu.RUnlock()

	active := make([]*EchoChain, 0)
	for _, chain := range s.chains {
		if !chain.IsExpired() {
			active = append(active, chain)
		}
	}
	return active
}

// GetNodeBonus returns the accumulated echo chain bonus for a node.
func (s *EchoChainStore) GetNodeBonus(nodeID []byte) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodeHex := keyToHex(nodeID)
	return s.nodeBonus[nodeHex]
}

// GetChainParticipants returns the node IDs in a chain.
func (s *EchoChainStore) GetChainParticipants(chainID [32]byte) [][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chain, ok := s.chains[chainID]
	if !ok {
		return nil
	}

	participants := make([][]byte, len(chain.Nodes))
	for i, node := range chain.Nodes {
		participants[i] = node.NodeID
	}
	return participants
}

// IsNodeInChain checks if a node is part of a specific chain.
func (s *EchoChainStore) IsNodeInChain(chainID [32]byte, nodeID []byte) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chain, ok := s.chains[chainID]
	if !ok {
		return false
	}

	nodeHex := keyToHex(nodeID)
	for _, node := range chain.Nodes {
		if keyToHex(node.NodeID) == nodeHex {
			return true
		}
	}
	return false
}

// GetChainsByLayer returns all chains of a specific layer.
func (s *EchoChainStore) GetChainsByLayer(layer EchoChainLayer) []*EchoChain {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chains := make([]*EchoChain, 0)
	for _, chain := range s.chains {
		if chain.Layer == layer && !chain.IsExpired() {
			chains = append(chains, chain)
		}
	}
	return chains
}

// GetShimmeringChains returns all chains with shimmer effect (5+ nodes).
func (s *EchoChainStore) GetShimmeringChains() []*EchoChain {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chains := make([]*EchoChain, 0)
	for _, chain := range s.chains {
		if chain.HasShimmer && !chain.IsExpired() {
			chains = append(chains, chain)
		}
	}
	return chains
}

// ExpireChains marks expired chains and removes them from active set.
func (s *EchoChainStore) ExpireChains() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	expired := 0
	for id, chain := range s.chains {
		if chain.IsExpired() {
			delete(s.activeChains, id)
			expired++
		}
	}
	return expired
}

// PurgePendingChains removes pending chains that are too old.
func (s *EchoChainStore) PurgePendingChains(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	purged := 0

	for id, chain := range s.pendingChains {
		if len(chain.Nodes) > 0 {
			firstAmplification := chain.Nodes[0].AmplifiedAt
			if now.Sub(firstAmplification) > maxAge {
				delete(s.pendingChains, id)
				purged++
			}
		}
	}

	return purged
}

// CountActiveChains returns the number of non-expired chains.
func (s *EchoChainStore) CountActiveChains() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, chain := range s.chains {
		if !chain.IsExpired() {
			count++
		}
	}
	return count
}

// CountPendingChains returns the number of chains still forming.
func (s *EchoChainStore) CountPendingChains() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.pendingChains)
}

// GetLongestChain returns the chain with the most nodes.
func (s *EchoChainStore) GetLongestChain() *EchoChain {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var longest *EchoChain
	maxLen := 0

	for _, chain := range s.chains {
		if len(chain.Nodes) > maxLen && !chain.IsExpired() {
			longest = chain
			maxLen = len(chain.Nodes)
		}
	}

	return longest
}

// GetChainsContainingNode returns all chains that include a specific node.
func (s *EchoChainStore) GetChainsContainingNode(nodeID []byte) []*EchoChain {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodeHex := keyToHex(nodeID)
	chains := make([]*EchoChain, 0)

	for _, chain := range s.chains {
		for _, node := range chain.Nodes {
			if keyToHex(node.NodeID) == nodeHex {
				chains = append(chains, chain)
				break
			}
		}
	}

	return chains
}

// GetTotalBonusAwarded returns the total Resonance bonus awarded across all chains.
func (s *EchoChainStore) GetTotalBonusAwarded() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0.0
	for _, bonus := range s.nodeBonus {
		total += bonus
	}
	return total
}
