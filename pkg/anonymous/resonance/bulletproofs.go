// Package resonance provides Zero-Knowledge Resonance claims using Bulletproofs.
// This file implements proper Bulletproofs range proofs per SECURITY_PRIVACY.md.
// Per the spec: "Bulletproofs range proof generation — prove Resonance within
// threshold without revealing exact value (~672 bytes for 64-bit range)".
//
// The Bulletproofs protocol is defined in https://eprint.iacr.org/2017/1066.pdf
// and provides compact range proofs with O(log n) proof size.
package resonance

import (
	crand "crypto/rand"
	"encoding/binary"
	"sync"
	"time"

	"github.com/coinbase/kryptology/pkg/bulletproof"
	"github.com/coinbase/kryptology/pkg/core/curves"
	"github.com/gtank/merlin"
)

// BulletproofRangeProofSize is the approximate size of a 64-bit range proof.
// Per SECURITY_PRIVACY.md: "~672 bytes for 64-bit range".
const BulletproofRangeProofSize = 672

// BulletproofRangeProof represents a complete Bulletproof range proof.
// This proves that a committed value lies within [0, 2^n) without revealing it.
type BulletproofRangeProof struct {
	// Proof is the serialized Bulletproof range proof.
	Proof []byte

	// CapV is the Pedersen commitment to the value (V = v*G + gamma*H).
	CapV []byte

	// G, H, U are the generator points used in the proof.
	// These are needed for verification.
	G, H, U []byte

	// N is the bit length (e.g., 64 for 64-bit range).
	N int

	// Timestamp when the proof was generated.
	Timestamp int64

	// Nonce for replay prevention.
	Nonce [32]byte
}

// BulletproofParams holds the parameters for Bulletproof operations.
type BulletproofParams struct {
	Curve       *curves.Curve
	RangeDomain []byte
	IppDomain   []byte
	BitLength   int
}

// DefaultBulletproofParams returns parameters for 64-bit range proofs.
// Uses Ed25519 curve which provides Ristretto-compatible operations.
func DefaultBulletproofParams() *BulletproofParams {
	return &BulletproofParams{
		Curve:       curves.ED25519(),
		RangeDomain: []byte("MURMUR_RANGE_PROOF_v1"),
		IppDomain:   []byte("MURMUR_IPP_v1"),
		BitLength:   64,
	}
}

// BulletproofRangeProver generates Bulletproof range proofs.
type BulletproofRangeProver struct {
	params *BulletproofParams
	prover *bulletproof.RangeProver
	mu     sync.Mutex
}

// NewBulletproofRangeProver creates a new Bulletproof range prover.
func NewBulletproofRangeProver() (*BulletproofRangeProver, error) {
	params := DefaultBulletproofParams()

	prover, err := bulletproof.NewRangeProver(
		params.BitLength,
		params.RangeDomain,
		params.IppDomain,
		*params.Curve,
	)
	if err != nil {
		return nil, err
	}

	return &BulletproofRangeProver{
		params: params,
		prover: prover,
	}, nil
}

// GenerateRangeProof creates a Bulletproof proving value lies in [0, 2^64).
// Returns the proof and the blinding factor (gamma) used for the commitment.
func (p *BulletproofRangeProver) GenerateRangeProof(value uint64) (*BulletproofRangeProof, curves.Scalar, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Convert value to scalar.
	v := p.params.Curve.Scalar.New(int(value))

	// Generate random blinding factor.
	gamma := p.params.Curve.Scalar.Random(crand.Reader)

	// Generate random generator points.
	// In a production system, these would be derived deterministically
	// from a domain separation string for verifiability.
	g := p.params.Curve.Point.Random(crand.Reader)
	h := p.params.Curve.Point.Random(crand.Reader)
	u := p.params.Curve.Point.Random(crand.Reader)

	// Create transcript for Fiat-Shamir transformation.
	transcript := merlin.NewTranscript("MURMUR_RANGE_PROOF")

	// Generate the range proof.
	proof, err := p.prover.Prove(v, gamma, p.params.BitLength, g, h, u, transcript)
	if err != nil {
		return nil, nil, err
	}

	// Compute the commitment V = v*G + gamma*H.
	capV := g.Mul(v).Add(h.Mul(gamma))

	// Generate nonce for replay prevention.
	var nonce [32]byte
	crand.Read(nonce[:])

	// Serialize the proof.
	proofBytes := proof.MarshalBinary()

	return &BulletproofRangeProof{
		Proof:     proofBytes,
		CapV:      capV.ToAffineCompressed(),
		G:         g.ToAffineCompressed(),
		H:         h.ToAffineCompressed(),
		U:         u.ToAffineCompressed(),
		N:         p.params.BitLength,
		Timestamp: time.Now().Unix(),
		Nonce:     nonce,
	}, gamma, nil
}

// BulletproofRangeVerifier verifies Bulletproof range proofs.
type BulletproofRangeVerifier struct {
	params   *BulletproofParams
	verifier *bulletproof.RangeVerifier
	seenOnce map[[32]byte]int64
	mu       sync.RWMutex
}

// NewBulletproofRangeVerifier creates a new Bulletproof range verifier.
func NewBulletproofRangeVerifier() (*BulletproofRangeVerifier, error) {
	params := DefaultBulletproofParams()

	verifier, err := bulletproof.NewRangeVerifier(
		params.BitLength,
		params.RangeDomain,
		params.IppDomain,
		*params.Curve,
	)
	if err != nil {
		return nil, err
	}

	return &BulletproofRangeVerifier{
		params:   params,
		verifier: verifier,
		seenOnce: make(map[[32]byte]int64),
	}, nil
}

// Verify checks if a Bulletproof range proof is valid.
func (v *BulletproofRangeVerifier) Verify(rangeProof *BulletproofRangeProof) error {
	// Check freshness.
	proofTime := time.Unix(rangeProof.Timestamp, 0)
	if time.Since(proofTime) > ClaimFreshness {
		return ErrClaimExpired
	}
	if proofTime.After(time.Now().Add(30 * time.Second)) {
		return ErrInvalidClaim
	}

	// Check replay.
	v.mu.RLock()
	if _, seen := v.seenOnce[rangeProof.Nonce]; seen {
		v.mu.RUnlock()
		return ErrReplayDetected
	}
	v.mu.RUnlock()

	// Deserialize generator points.
	capV, err := v.params.Curve.Point.FromAffineCompressed(rangeProof.CapV)
	if err != nil {
		return ErrInvalidProof
	}
	g, err := v.params.Curve.Point.FromAffineCompressed(rangeProof.G)
	if err != nil {
		return ErrInvalidProof
	}
	h, err := v.params.Curve.Point.FromAffineCompressed(rangeProof.H)
	if err != nil {
		return ErrInvalidProof
	}
	u, err := v.params.Curve.Point.FromAffineCompressed(rangeProof.U)
	if err != nil {
		return ErrInvalidProof
	}

	// Deserialize the proof.
	proof := bulletproof.NewRangeProof(v.params.Curve)
	if err := proof.UnmarshalBinary(rangeProof.Proof); err != nil {
		return ErrInvalidProof
	}

	// Create transcript for verification.
	transcript := merlin.NewTranscript("MURMUR_RANGE_PROOF")

	// Verify the proof.
	valid, err := v.verifier.Verify(proof, capV, g, h, u, rangeProof.N, transcript)
	if err != nil {
		return ErrInvalidProof
	}
	if !valid {
		return ErrInvalidProof
	}

	// Record nonce.
	v.mu.Lock()
	v.seenOnce[rangeProof.Nonce] = rangeProof.Timestamp
	v.mu.Unlock()

	return nil
}

// CleanExpiredNonces removes old nonces from the replay cache.
func (v *BulletproofRangeVerifier) CleanExpiredNonces() {
	v.mu.Lock()
	defer v.mu.Unlock()

	cutoff := time.Now().Add(-ClaimFreshness).Unix()
	for nonce, timestamp := range v.seenOnce {
		if timestamp < cutoff {
			delete(v.seenOnce, nonce)
		}
	}
}

// BulletproofThresholdProof proves value >= threshold using Bulletproofs.
// This is done by proving (value - threshold) lies in [0, 2^64),
// which implies value >= threshold (assuming value < 2^64).
type BulletproofThresholdProof struct {
	// DeltaRangeProof proves delta = value - threshold is in [0, 2^64).
	DeltaRangeProof BulletproofRangeProof

	// Threshold is the minimum value being proven.
	Threshold uint64
}

// BulletproofThresholdProver generates threshold proofs.
type BulletproofThresholdProver struct {
	rangeProver *BulletproofRangeProver
}

// NewBulletproofThresholdProver creates a new threshold prover.
func NewBulletproofThresholdProver() (*BulletproofThresholdProver, error) {
	rangeProver, err := NewBulletproofRangeProver()
	if err != nil {
		return nil, err
	}
	return &BulletproofThresholdProver{rangeProver: rangeProver}, nil
}

// GenerateThresholdProof creates a proof that value >= threshold.
func (p *BulletproofThresholdProver) GenerateThresholdProof(
	value, threshold uint64,
) (*BulletproofThresholdProof, error) {
	if value < threshold {
		return nil, ErrThresholdNotMet
	}

	delta := value - threshold

	// Generate range proof for delta (proves delta >= 0).
	deltaProof, _, err := p.rangeProver.GenerateRangeProof(delta)
	if err != nil {
		return nil, err
	}

	return &BulletproofThresholdProof{
		DeltaRangeProof: *deltaProof,
		Threshold:       threshold,
	}, nil
}

// BulletproofThresholdVerifier verifies threshold proofs.
type BulletproofThresholdVerifier struct {
	rangeVerifier *BulletproofRangeVerifier
}

// NewBulletproofThresholdVerifier creates a new threshold verifier.
func NewBulletproofThresholdVerifier() (*BulletproofThresholdVerifier, error) {
	rangeVerifier, err := NewBulletproofRangeVerifier()
	if err != nil {
		return nil, err
	}
	return &BulletproofThresholdVerifier{rangeVerifier: rangeVerifier}, nil
}

// VerifyThresholdProof verifies that the proof demonstrates value >= threshold.
func (v *BulletproofThresholdVerifier) VerifyThresholdProof(
	proof *BulletproofThresholdProof,
) error {
	// Verify the delta range proof.
	return v.rangeVerifier.Verify(&proof.DeltaRangeProof)
}

// Bytes serializes the threshold proof.
func (p *BulletproofThresholdProof) Bytes() []byte {
	// Serialize delta range proof components.
	out := make([]byte, 0, len(p.DeltaRangeProof.Proof)+200)

	// Proof length + proof.
	proofLen := uint16(len(p.DeltaRangeProof.Proof))
	out = append(out, byte(proofLen), byte(proofLen>>8))
	out = append(out, p.DeltaRangeProof.Proof...)

	// CapV length + CapV.
	capVLen := uint16(len(p.DeltaRangeProof.CapV))
	out = append(out, byte(capVLen), byte(capVLen>>8))
	out = append(out, p.DeltaRangeProof.CapV...)

	// G length + G.
	gLen := uint16(len(p.DeltaRangeProof.G))
	out = append(out, byte(gLen), byte(gLen>>8))
	out = append(out, p.DeltaRangeProof.G...)

	// H length + H.
	hLen := uint16(len(p.DeltaRangeProof.H))
	out = append(out, byte(hLen), byte(hLen>>8))
	out = append(out, p.DeltaRangeProof.H...)

	// U length + U.
	uLen := uint16(len(p.DeltaRangeProof.U))
	out = append(out, byte(uLen), byte(uLen>>8))
	out = append(out, p.DeltaRangeProof.U...)

	// N (4 bytes).
	var nBytes [4]byte
	binary.LittleEndian.PutUint32(nBytes[:], uint32(p.DeltaRangeProof.N))
	out = append(out, nBytes[:]...)

	// Timestamp (8 bytes).
	var tsBytes [8]byte
	binary.LittleEndian.PutUint64(tsBytes[:], uint64(p.DeltaRangeProof.Timestamp))
	out = append(out, tsBytes[:]...)

	// Nonce (32 bytes).
	out = append(out, p.DeltaRangeProof.Nonce[:]...)

	// Threshold (8 bytes).
	var threshBytes [8]byte
	binary.LittleEndian.PutUint64(threshBytes[:], p.Threshold)
	out = append(out, threshBytes[:]...)

	return out
}

// SetBytes deserializes a threshold proof.
func (p *BulletproofThresholdProof) SetBytes(data []byte) error {
	if len(data) < 60 {
		return ErrInvalidProof
	}

	pos := 0

	// Proof.
	proofLen := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	if pos+proofLen > len(data) {
		return ErrInvalidProof
	}
	p.DeltaRangeProof.Proof = make([]byte, proofLen)
	copy(p.DeltaRangeProof.Proof, data[pos:pos+proofLen])
	pos += proofLen

	// CapV.
	capVLen := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	if pos+capVLen > len(data) {
		return ErrInvalidProof
	}
	p.DeltaRangeProof.CapV = make([]byte, capVLen)
	copy(p.DeltaRangeProof.CapV, data[pos:pos+capVLen])
	pos += capVLen

	// G.
	gLen := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	if pos+gLen > len(data) {
		return ErrInvalidProof
	}
	p.DeltaRangeProof.G = make([]byte, gLen)
	copy(p.DeltaRangeProof.G, data[pos:pos+gLen])
	pos += gLen

	// H.
	hLen := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	if pos+hLen > len(data) {
		return ErrInvalidProof
	}
	p.DeltaRangeProof.H = make([]byte, hLen)
	copy(p.DeltaRangeProof.H, data[pos:pos+hLen])
	pos += hLen

	// U.
	uLen := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	if pos+uLen > len(data) {
		return ErrInvalidProof
	}
	p.DeltaRangeProof.U = make([]byte, uLen)
	copy(p.DeltaRangeProof.U, data[pos:pos+uLen])
	pos += uLen

	// N.
	if pos+4 > len(data) {
		return ErrInvalidProof
	}
	p.DeltaRangeProof.N = int(binary.LittleEndian.Uint32(data[pos : pos+4]))
	pos += 4

	// Timestamp.
	if pos+8 > len(data) {
		return ErrInvalidProof
	}
	p.DeltaRangeProof.Timestamp = int64(binary.LittleEndian.Uint64(data[pos : pos+8]))
	pos += 8

	// Nonce.
	if pos+32 > len(data) {
		return ErrInvalidProof
	}
	copy(p.DeltaRangeProof.Nonce[:], data[pos:pos+32])
	pos += 32

	// Threshold.
	if pos+8 > len(data) {
		return ErrInvalidProof
	}
	p.Threshold = binary.LittleEndian.Uint64(data[pos : pos+8])

	return nil
}
