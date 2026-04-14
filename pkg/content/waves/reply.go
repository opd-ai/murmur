// Package waves provides Wave creation, signing, and validation.
// This file implements parent chain validation for Reply Waves per WAVES.md.
package waves

import (
	"errors"

	pb "github.com/opd-ai/murmur/proto"
)

// MaxParentChainDepth is the maximum depth for parent chain validation.
const MaxParentChainDepth = 100

// Errors for parent chain validation.
var (
	ErrParentNotFound   = errors.New("parent wave not found")
	ErrParentExpired    = errors.New("parent wave has expired")
	ErrParentChainCycle = errors.New("cyclic parent chain detected")
	ErrParentTooDeep    = errors.New("parent chain exceeds maximum depth")
	ErrParentInvalid    = errors.New("parent wave is invalid")
	ErrNotReplyWave     = errors.New("wave is not a reply")
	ErrMissingParent    = errors.New("reply wave missing parent hash")
)

// WaveProvider retrieves Waves by their ID.
// Implementations may fetch from local storage or network peers.
type WaveProvider interface {
	// GetWave retrieves a Wave by its ID. Returns nil, nil if not found.
	GetWave(waveID []byte) (*pb.Wave, error)
}

// ParentChainValidator validates the integrity of Reply Wave parent chains.
type ParentChainValidator struct {
	provider   WaveProvider
	difficulty uint8
}

// NewParentChainValidator creates a new validator with the given provider.
func NewParentChainValidator(provider WaveProvider, difficulty uint8) *ParentChainValidator {
	return &ParentChainValidator{
		provider:   provider,
		difficulty: difficulty,
	}
}

// ValidateParentChain validates a Reply Wave's parent chain.
// Returns the chain from the root to this wave (inclusive).
func (v *ParentChainValidator) ValidateParentChain(wave *pb.Wave) ([]*pb.Wave, error) {
	if wave == nil {
		return nil, errors.New("wave is nil")
	}

	// Only Reply Waves have parent chains to validate.
	if wave.WaveType != pb.WaveType(TypeReply) {
		return nil, ErrNotReplyWave
	}

	if len(wave.ParentHash) == 0 {
		return nil, ErrMissingParent
	}

	return v.traceChain(wave)
}

// traceChain traces the parent chain from a wave back to the root.
func (v *ParentChainValidator) traceChain(wave *pb.Wave) ([]*pb.Wave, error) {
	chain := []*pb.Wave{wave}
	seen := make(map[string]bool)
	seen[string(wave.WaveId)] = true

	current := wave
	for depth := 0; depth < MaxParentChainDepth; depth++ {
		if len(current.ParentHash) == 0 {
			// Reached root (non-reply wave).
			break
		}

		parentID := string(current.ParentHash)
		if seen[parentID] {
			return nil, ErrParentChainCycle
		}

		parent, err := v.fetchAndValidateParent(current.ParentHash)
		if err != nil {
			return nil, err
		}

		seen[parentID] = true
		chain = append([]*pb.Wave{parent}, chain...)
		current = parent
	}

	if len(chain) > MaxParentChainDepth {
		return nil, ErrParentTooDeep
	}

	return chain, nil
}

// fetchAndValidateParent retrieves and validates a parent wave.
func (v *ParentChainValidator) fetchAndValidateParent(parentHash []byte) (*pb.Wave, error) {
	parent, err := v.provider.GetWave(parentHash)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, ErrParentNotFound
	}

	// Validate the parent wave.
	if err := v.validateParent(parent); err != nil {
		return nil, err
	}

	return parent, nil
}

// validateParent validates a parent wave.
func (v *ParentChainValidator) validateParent(parent *pb.Wave) error {
	// Check if parent is expired.
	if IsExpired(parent) {
		return ErrParentExpired
	}

	// Validate the parent's signature and PoW.
	// Skip validation for Beacon waves (no signature).
	if parent.WaveType == pb.WaveType(TypeBeacon) {
		return ValidateBeacon(parent, v.difficulty)
	}

	return Validate(parent, v.difficulty)
}

// ParentChainResult contains the result of parent chain validation.
type ParentChainResult struct {
	Chain      []*pb.Wave
	RootWave   *pb.Wave
	Depth      int
	AllValid   bool
	MissingIDs [][]byte
}

// ValidateParentChainWithResult validates and returns detailed results.
func (v *ParentChainValidator) ValidateParentChainWithResult(wave *pb.Wave) (*ParentChainResult, error) {
	result := &ParentChainResult{
		AllValid:   true,
		MissingIDs: make([][]byte, 0),
	}

	chain, err := v.ValidateParentChain(wave)
	if err != nil {
		if err == ErrParentNotFound {
			result.AllValid = false
			result.MissingIDs = append(result.MissingIDs, wave.ParentHash)
		}
		return result, err
	}

	result.Chain = chain
	result.Depth = len(chain) - 1 // Root is depth 0
	if len(chain) > 0 {
		result.RootWave = chain[0]
	}

	return result, nil
}

// IsReply checks if a wave is a Reply Wave.
func IsReply(wave *pb.Wave) bool {
	if wave == nil {
		return false
	}
	return wave.WaveType == pb.WaveType(TypeReply) && len(wave.ParentHash) > 0
}

// ValidateReply validates a Reply Wave including its parent reference.
// Does not validate the full parent chain, just the immediate structure.
func ValidateReply(wave *pb.Wave, difficulty uint8) error {
	if wave == nil {
		return errors.New("wave is nil")
	}

	if wave.WaveType != pb.WaveType(TypeReply) {
		return ErrNotReplyWave
	}

	if len(wave.ParentHash) == 0 {
		return ErrMissingParent
	}

	// Validate basic wave properties.
	return Validate(wave, difficulty)
}

// GetParentHash returns the parent hash of a wave, or nil if not a reply.
func GetParentHash(wave *pb.Wave) []byte {
	if wave == nil || len(wave.ParentHash) == 0 {
		return nil
	}
	return wave.ParentHash
}

// GetReplyDepth calculates the depth of a reply in its thread.
// Uses the validator to trace the parent chain.
func GetReplyDepth(wave *pb.Wave, v *ParentChainValidator) (int, error) {
	if !IsReply(wave) {
		return 0, nil
	}

	chain, err := v.ValidateParentChain(wave)
	if err != nil {
		return 0, err
	}

	return len(chain) - 1, nil
}

// InMemoryWaveProvider is a simple in-memory WaveProvider for testing.
type InMemoryWaveProvider struct {
	waves map[string]*pb.Wave
}

// NewInMemoryWaveProvider creates a new in-memory provider.
func NewInMemoryWaveProvider() *InMemoryWaveProvider {
	return &InMemoryWaveProvider{
		waves: make(map[string]*pb.Wave),
	}
}

// Add adds a wave to the provider.
func (p *InMemoryWaveProvider) Add(wave *pb.Wave) {
	if wave != nil && len(wave.WaveId) > 0 {
		p.waves[string(wave.WaveId)] = wave
	}
}

// GetWave retrieves a wave by ID.
func (p *InMemoryWaveProvider) GetWave(waveID []byte) (*pb.Wave, error) {
	if wave, ok := p.waves[string(waveID)]; ok {
		return wave, nil
	}
	return nil, nil
}

// Remove removes a wave from the provider.
func (p *InMemoryWaveProvider) Remove(waveID []byte) {
	delete(p.waves, string(waveID))
}
