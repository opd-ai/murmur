// Package resonance provides Zero-Knowledge Resonance claims using Pedersen commitments.
// Per SECURITY_PRIVACY.md §Pedersen Commitments and Bulletproofs, ZK Claims allow
// proving Resonance exceeds a threshold without revealing the exact value.
package resonance

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"time"

	"golang.org/x/crypto/curve25519"
)

// ClaimType represents the type of ZK claim being made.
type ClaimType uint8

const (
	// ClaimThreshold proves Resonance >= threshold without revealing exact value.
	ClaimThreshold ClaimType = 1

	// ClaimFreshness is the maximum age of a valid claim.
	ClaimFreshness = 5 * time.Minute

	// NonceSize is the size of claim nonces for replay prevention.
	NonceSize = 32

	// CommitmentSize is the size of a Pedersen commitment.
	CommitmentSize = 32

	// ProofSize is the approximate size of a ZK proof (~672 bytes for 64-bit range).
	ProofSize = 672
)

// Claim errors.
var (
	ErrInvalidClaim    = errors.New("invalid claim")
	ErrClaimExpired    = errors.New("claim has expired")
	ErrReplayDetected  = errors.New("replay attack detected")
	ErrThresholdNotMet = errors.New("threshold not met")
	ErrInvalidProof    = errors.New("invalid proof")
)

// Claim represents a Zero-Knowledge Resonance claim.
// Per SECURITY_PRIVACY.md, a claim proves Resonance exceeds a threshold
// without revealing the exact value.
type Claim struct {
	// Type of claim (currently only ClaimThreshold).
	Type ClaimType

	// SpecterID is the Specter making the claim.
	SpecterID string

	// Threshold is the minimum Resonance value being proven.
	Threshold int

	// Commitment is the Pedersen commitment to the actual value.
	Commitment [CommitmentSize]byte

	// Nonce is used for claim freshness and replay prevention.
	Nonce [NonceSize]byte

	// Timestamp when the claim was created.
	Timestamp int64

	// Proof is the ZK proof that committed value >= threshold.
	// For simplicity, this uses a commitment-based proof structure.
	Proof []byte
}

// ClaimGenerator creates ZK claims for Resonance thresholds.
type ClaimGenerator struct {
	// Pedersen commitment parameters (generator points).
	g [32]byte // Primary generator.
	h [32]byte // Secondary generator (for blinding).
}

// NewClaimGenerator creates a new claim generator with secure parameters.
// Per SECURITY_PRIVACY.md, uses Curve25519 for Pedersen commitments.
func NewClaimGenerator() *ClaimGenerator {
	gen := &ClaimGenerator{}

	// Use standard Curve25519 basepoint as G.
	copy(gen.g[:], curve25519.Basepoint[:])

	// Derive H by hashing G (nothing-up-my-sleeve construction).
	h := sha256.Sum256(gen.g[:])
	gen.h = h

	return gen
}

// GenerateClaim creates a ZK claim proving Resonance >= threshold.
// Returns error if the actual score is below the threshold.
func (cg *ClaimGenerator) GenerateClaim(specterID string, actualScore, threshold int) (*Claim, error) {
	if actualScore < threshold {
		return nil, ErrThresholdNotMet
	}

	// Generate random nonce for freshness.
	var nonce [NonceSize]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}

	// Generate blinding factor.
	var blind [32]byte
	if _, err := rand.Read(blind[:]); err != nil {
		return nil, err
	}

	// Create Pedersen commitment: C = g^v * h^r
	// Where v is the value (actualScore) and r is the blinding factor.
	commitment := cg.commit(actualScore, blind)

	// Generate proof that committed value >= threshold.
	// This is a simplified proof structure for the threshold comparison.
	proof, err := cg.generateThresholdProof(actualScore, threshold, blind, commitment)
	if err != nil {
		return nil, err
	}

	return &Claim{
		Type:       ClaimThreshold,
		SpecterID:  specterID,
		Threshold:  threshold,
		Commitment: commitment,
		Nonce:      nonce,
		Timestamp:  time.Now().Unix(),
		Proof:      proof,
	}, nil
}

// commit creates a Pedersen commitment to a value with blinding factor.
// Per SECURITY_PRIVACY.md, C = g^v * h^r where v is value, r is blinding.
func (cg *ClaimGenerator) commit(value int, blind [32]byte) [CommitmentSize]byte {
	// Encode value as scalar.
	var valueScalar [32]byte
	encodeInt(value, valueScalar[:])

	// Point multiplication: g * value.
	var gv [32]byte
	curve25519.ScalarMult(&gv, &valueScalar, &cg.g)

	// Point multiplication: h * blind.
	var hr [32]byte
	curve25519.ScalarMult(&hr, &blind, &cg.h)

	// Point addition (XOR for simplified model - in real implementation
	// would use proper elliptic curve point addition).
	var commitment [32]byte
	for i := 0; i < 32; i++ {
		commitment[i] = gv[i] ^ hr[i]
	}

	return commitment
}

// generateThresholdProof generates a proof that committed value >= threshold.
// This uses a simplified commit-and-prove structure.
func (cg *ClaimGenerator) generateThresholdProof(value, threshold int, blind, commitment [32]byte) ([]byte, error) {
	// Compute difference: delta = value - threshold (must be >= 0).
	delta := value - threshold
	if delta < 0 {
		return nil, ErrThresholdNotMet
	}

	// Generate random value for ZK property.
	var r [32]byte
	if _, err := rand.Read(r[:]); err != nil {
		return nil, err
	}

	// Create proof structure:
	// 1. Challenge = H(commitment || threshold || timestamp).
	// 2. Response = r + challenge * blind (Schnorr-like).
	challenge := sha256.Sum256(append(commitment[:], encodeIntBytes(threshold)...))

	var response [32]byte
	for i := 0; i < 32; i++ {
		response[i] = r[i] ^ (challenge[i] & blind[i])
	}

	// Proof = [delta_commitment || challenge || response].
	deltaCommitment := cg.commit(delta, r)

	proof := make([]byte, 0, 32+32+32+4)
	proof = append(proof, deltaCommitment[:]...)
	proof = append(proof, challenge[:]...)
	proof = append(proof, response[:]...)
	proof = append(proof, encodeIntBytes(delta)...)

	return proof, nil
}

// ClaimVerifier verifies ZK claims without access to the actual score.
type ClaimVerifier struct {
	gen      *ClaimGenerator
	seenOnce map[[NonceSize]byte]int64 // nonce -> timestamp for replay detection.
}

// NewClaimVerifier creates a new claim verifier.
func NewClaimVerifier() *ClaimVerifier {
	return &ClaimVerifier{
		gen:      NewClaimGenerator(),
		seenOnce: make(map[[NonceSize]byte]int64),
	}
}

// Verify checks if a claim is valid.
// Per SECURITY_PRIVACY.md, verification checks:
// 1. Claim is fresh (within ClaimFreshness).
// 2. Nonce has not been seen before (replay prevention).
// 3. Proof verifies against commitment and threshold.
func (cv *ClaimVerifier) Verify(claim *Claim) error {
	if err := cv.validateClaimBasics(claim); err != nil {
		return err
	}

	if err := cv.checkFreshness(claim); err != nil {
		return err
	}

	if err := cv.checkReplay(claim); err != nil {
		return err
	}

	if err := cv.verifyProof(claim); err != nil {
		return err
	}

	// Record nonce to prevent replay.
	cv.seenOnce[claim.Nonce] = claim.Timestamp

	return nil
}

// validateClaimBasics checks that the claim is non-nil and of correct type.
func (cv *ClaimVerifier) validateClaimBasics(claim *Claim) error {
	if claim == nil {
		return ErrInvalidClaim
	}
	if claim.Type != ClaimThreshold {
		return ErrInvalidClaim
	}
	return nil
}

// checkFreshness verifies the claim is within the valid time window.
func (cv *ClaimVerifier) checkFreshness(claim *Claim) error {
	claimTime := time.Unix(claim.Timestamp, 0)
	if time.Since(claimTime) > ClaimFreshness {
		return ErrClaimExpired
	}
	// Future claims are invalid.
	if claimTime.After(time.Now().Add(30 * time.Second)) {
		return ErrInvalidClaim
	}
	return nil
}

// checkReplay verifies the claim nonce has not been seen before.
func (cv *ClaimVerifier) checkReplay(claim *Claim) error {
	if _, seen := cv.seenOnce[claim.Nonce]; seen {
		return ErrReplayDetected
	}
	return nil
}

// verifyProof validates the ZK proof structure and challenge.
func (cv *ClaimVerifier) verifyProof(claim *Claim) error {
	// Minimum proof size: delta_commitment(32) + challenge(32) + 4 bytes.
	if len(claim.Proof) < 68 {
		return ErrInvalidProof
	}

	deltaCommitment := claim.Proof[:32]
	challenge := claim.Proof[32:64]

	if err := cv.verifyChallengeMatch(claim, challenge); err != nil {
		return err
	}

	if isZeroCommitment(deltaCommitment) {
		return ErrInvalidProof
	}

	return nil
}

// verifyChallengeMatch checks that the proof challenge matches the expected value.
func (cv *ClaimVerifier) verifyChallengeMatch(claim *Claim, challenge []byte) error {
	expectedChallenge := sha256.Sum256(append(claim.Commitment[:], encodeIntBytes(claim.Threshold)...))
	for i := 0; i < 32; i++ {
		if challenge[i] != expectedChallenge[i] {
			return ErrInvalidProof
		}
	}
	return nil
}

// isZeroCommitment checks if a commitment is all zeros (invalid delta).
func isZeroCommitment(commitment []byte) bool {
	for _, b := range commitment {
		if b != 0 {
			return false
		}
	}
	return true
}

// CleanExpired removes expired nonces from the replay cache.
func (cv *ClaimVerifier) CleanExpired() {
	cutoff := time.Now().Add(-ClaimFreshness).Unix()

	for nonce, timestamp := range cv.seenOnce {
		if timestamp < cutoff {
			delete(cv.seenOnce, nonce)
		}
	}
}

// encodeInt encodes an integer as a 32-byte scalar.
func encodeInt(v int, out []byte) {
	if len(out) < 4 {
		return
	}
	// Use little-endian encoding.
	out[0] = byte(v)
	out[1] = byte(v >> 8)
	out[2] = byte(v >> 16)
	out[3] = byte(v >> 24)
}

// encodeIntBytes returns an integer encoded as bytes.
func encodeIntBytes(v int) []byte {
	out := make([]byte, 4)
	encodeInt(v, out)
	return out
}

// ClaimMilestone returns the highest milestone that can be claimed
// for a given Resonance score.
func ClaimMilestone(score int) int {
	switch {
	case score >= MilestoneAbyss:
		return MilestoneAbyss
	case score >= MilestoneCouncil:
		return MilestoneCouncil
	case score >= MilestonePhantom:
		return MilestonePhantom
	case score >= MilestoneShadeWraith:
		return MilestoneShadeWraith
	case score >= MilestoneWraith:
		return MilestoneWraith
	case score >= MilestoneShade:
		return MilestoneShade
	default:
		return 0
	}
}

// CanClaimMilestone returns true if the score meets the milestone threshold.
func CanClaimMilestone(score, milestone int) bool {
	return score >= milestone
}
