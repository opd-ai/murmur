// Package waves provides Wave creation, signing, and validation.
// This file implements Sigil Wave — Surface Layer Waves with embedded
// Specter sigil for anonymous-layer participation signaling.
// Per WAVES.md, Sigil Waves (type 0x05) contain a random Specter sigil
// without revealing the author's own Specter identity.
package waves

import (
	"crypto/rand"
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	pb "github.com/opd-ai/murmur/proto"
	"github.com/zeebo/blake3"
)

// Sigil Wave constants.
const (
	// SigilHashKey is the metadata key for the embedded sigil hash.
	SigilHashKey = "sigil_hash"

	// SigilSourceKey indicates how the sigil was selected.
	SigilSourceKey = "sigil_source"

	// SigilSourceRandom indicates random selection from visible Specters.
	SigilSourceRandom = "random"

	// SigilHashSize is the size of the sigil hash (32 bytes SHA-256).
	SigilHashSize = 32
)

// SigilSelector is an interface for selecting random Specter sigils.
type SigilSelector interface {
	// SelectRandomSigil returns a random Specter public key hash.
	// Returns nil if no Specters are available in the local topology.
	SelectRandomSigil() []byte
}

// SigilOptions configures Sigil Wave creation.
type SigilOptions struct {
	// TTL is the time-to-live for the Wave.
	TTL time.Duration

	// Difficulty is the PoW difficulty.
	Difficulty uint8

	// CustomSigilHash allows specifying a specific sigil hash.
	// If nil, a random sigil is selected via the selector.
	CustomSigilHash []byte
}

// DefaultSigilOptions returns default options for Sigil Wave creation.
func DefaultSigilOptions() SigilOptions {
	return SigilOptions{
		TTL:        DefaultTTL,
		Difficulty: DefaultDifficulty,
	}
}

// CreateSigil creates a Sigil Wave with an embedded Specter sigil.
// The sigil is either randomly selected or custom-provided.
// Per WAVES.md, Sigil Waves are a social signaling mechanic indicating
// anonymous layer participation without revealing which Specter.
func CreateSigil(content []byte, kp *keys.KeyPair, selector SigilSelector, opts SigilOptions) (*pb.Wave, error) {
	if kp == nil {
		return nil, ErrNilKeyPair
	}
	if len(content) > MaxContentSize {
		return nil, ErrContentTooLarge
	}
	if opts.TTL <= 0 {
		return nil, ErrInvalidTTL
	}
	if opts.TTL > MaxTTL {
		return nil, ErrTTLTooLong
	}

	// Determine the sigil hash.
	sigilHash, err := determineSigilHash(selector, opts)
	if err != nil {
		return nil, err
	}

	// Build the Sigil Wave.
	wave := buildSigilWave(content, kp, sigilHash, opts)

	// Sign and compute PoW.
	if err := signAndComputeWavePoW(wave, kp, opts.Difficulty); err != nil {
		return nil, err
	}

	return wave, nil
}

// determineSigilHash determines which sigil hash to use.
func determineSigilHash(selector SigilSelector, opts SigilOptions) ([]byte, error) {
	// Use custom sigil hash if provided.
	if len(opts.CustomSigilHash) == SigilHashSize {
		return opts.CustomSigilHash, nil
	}

	// Select a random sigil from the network.
	if selector != nil {
		if hash := selector.SelectRandomSigil(); len(hash) == SigilHashSize {
			return hash, nil
		}
	}

	// Fall back to a random hash (no Specters available).
	return generateRandomSigilHash()
}

// generateRandomSigilHash generates a random sigil hash.
// Used when no Specters are available in the local topology.
func generateRandomSigilHash() ([]byte, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, err
	}

	h := blake3.New()
	h.Write([]byte("murmur-random-sigil-v1"))
	h.Write(randomBytes)
	return h.Sum(nil)[:SigilHashSize], nil
}

// buildSigilWave constructs the Sigil Wave structure.
func buildSigilWave(content []byte, kp *keys.KeyPair, sigilHash []byte, opts SigilOptions) *pb.Wave {
	now := time.Now()

	// Initialize metadata with sigil information.
	metadata := map[string][]byte{
		SigilHashKey:   sigilHash,
		SigilSourceKey: []byte(SigilSourceRandom),
	}

	wave := &pb.Wave{
		WaveType:     pb.WaveType(TypeSigil),
		Content:      content,
		AuthorPubkey: kp.PublicKey,
		CreatedAt:    now.Unix(),
		TtlSeconds:   int64(opts.TTL.Seconds()),
		HopCount:     0,
		Metadata:     metadata,
	}

	wave.WaveId = computeWaveID(wave)
	return wave
}

// IsSigil checks if a Wave is a Sigil Wave.
func IsSigil(wave *pb.Wave) bool {
	if wave == nil {
		return false
	}
	if wave.WaveType != pb.WaveType(TypeSigil) {
		return false
	}
	_, ok := wave.Metadata[SigilHashKey]
	return ok
}

// GetSigilHash extracts the embedded sigil hash from a Sigil Wave.
func GetSigilHash(wave *pb.Wave) []byte {
	if wave == nil || wave.Metadata == nil {
		return nil
	}
	return wave.Metadata[SigilHashKey]
}

// ValidateSigil validates a Sigil Wave.
func ValidateSigil(wave *pb.Wave, difficulty uint8) error {
	if wave == nil {
		return ErrNilKeyPair // reusing error for nil check
	}
	if wave.WaveType != pb.WaveType(TypeSigil) {
		return ErrInvalidSig
	}

	// Check sigil hash is present.
	sigilHash := GetSigilHash(wave)
	if len(sigilHash) != SigilHashSize {
		return ErrInvalidSig
	}

	// Standard validation.
	return Validate(wave, difficulty)
}

// RandomSigilSelector implements SigilSelector using a list of known Specters.
type RandomSigilSelector struct {
	specterPubKeys [][]byte
}

// NewRandomSigilSelector creates a new selector with known Specter public keys.
func NewRandomSigilSelector(specterPubKeys [][]byte) *RandomSigilSelector {
	return &RandomSigilSelector{specterPubKeys: specterPubKeys}
}

// SelectRandomSigil selects a random Specter and returns its public key hash.
func (s *RandomSigilSelector) SelectRandomSigil() []byte {
	if len(s.specterPubKeys) == 0 {
		return nil
	}

	// Select a random index.
	indexBytes := make([]byte, 8)
	if _, err := rand.Read(indexBytes); err != nil {
		return nil
	}
	var index uint64
	for i, b := range indexBytes {
		index |= uint64(b) << (i * 8)
	}
	selected := s.specterPubKeys[index%uint64(len(s.specterPubKeys))]

	// Hash the public key to get the sigil hash.
	h := blake3.New()
	h.Write(selected)
	return h.Sum(nil)[:SigilHashSize]
}
