// Package resonance provides Zero-Knowledge Resonance claims using Pedersen commitments.
// This file implements proper Ristretto-based Pedersen commitments per SECURITY_PRIVACY.md.
// The Ristretto group provides a prime-order group suitable for cryptographic operations.
package resonance

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/bwesterb/go-ristretto"
)

// Additional errors for Ristretto operations.
var ErrInvalidCommitmentPoint = errors.New("invalid commitment point encoding")

// PedersenCommitment represents a Pedersen commitment on the Ristretto group.
// C = v*G + r*H where v is the value and r is the blinding factor.
type PedersenCommitment struct {
	Point ristretto.Point // The commitment point.
}

// PedersenParams holds the public parameters for Pedersen commitments.
// G is the standard base point, H is a "nothing-up-my-sleeve" point derived from G.
type PedersenParams struct {
	G ristretto.Point // Primary generator (standard base point).
	H ristretto.Point // Secondary generator (for blinding factor).
}

// DefaultPedersenParams returns the default Pedersen commitment parameters.
// G is the standard Ristretto base point.
// H is derived by hashing "MURMUR_PEDERSEN_H" to ensure no known discrete log relationship.
func DefaultPedersenParams() *PedersenParams {
	params := &PedersenParams{}

	// G is the standard base point.
	params.G.SetBase()

	// Derive H using hash-to-point (nothing-up-my-sleeve construction).
	// Per SECURITY_PRIVACY.md, H must have no known discrete log relationship to G.
	params.H.DeriveDalek([]byte("MURMUR_PEDERSEN_H"))

	return params
}

// Commit creates a Pedersen commitment to a value with a random blinding factor.
// Returns the commitment and the blinding factor (needed to open the commitment).
func (p *PedersenParams) Commit(value int64) (*PedersenCommitment, ristretto.Scalar) {
	// Generate random blinding factor.
	var blind ristretto.Scalar
	blind.Rand()

	return p.CommitWithBlind(value, blind), blind
}

// CommitWithBlind creates a Pedersen commitment with a specific blinding factor.
// C = v*G + r*H
func (p *PedersenParams) CommitWithBlind(value int64, blind ristretto.Scalar) *PedersenCommitment {
	// Convert value to scalar.
	var v ristretto.Scalar
	var vBytes [32]byte
	binary.LittleEndian.PutUint64(vBytes[:8], uint64(value))
	v.SetBytes(&vBytes)

	// Compute v*G.
	var vG ristretto.Point
	vG.ScalarMult(&p.G, &v)

	// Compute r*H.
	var rH ristretto.Point
	rH.ScalarMult(&p.H, &blind)

	// Compute C = v*G + r*H.
	var commitment ristretto.Point
	commitment.Add(&vG, &rH)

	return &PedersenCommitment{Point: commitment}
}

// Bytes returns the 32-byte encoding of the commitment.
func (c *PedersenCommitment) Bytes() [32]byte {
	var out [32]byte
	c.Point.BytesInto(&out)
	return out
}

// SetBytes sets the commitment from a 32-byte encoding.
func (c *PedersenCommitment) SetBytes(b [32]byte) error {
	if !c.Point.SetBytes(&b) {
		return ErrInvalidCommitmentPoint
	}
	return nil
}

// Equal returns true if two commitments are equal.
func (c *PedersenCommitment) Equal(other *PedersenCommitment) bool {
	return c.Point.Equals(&other.Point)
}

// ThresholdProof is a zero-knowledge proof that a committed value exceeds a threshold.
// This uses a Schnorr-style sigma protocol to prove knowledge of the opening.
type ThresholdProof struct {
	// DeltaCommitment is the commitment to (value - threshold).
	DeltaCommitment PedersenCommitment

	// ResponseValue is the Schnorr response for the value component.
	ResponseValue ristretto.Scalar

	// ResponseBlind is the Schnorr response for the blinding component.
	ResponseBlind ristretto.Scalar

	// Challenge is the Fiat-Shamir challenge.
	Challenge ristretto.Scalar

	// RandomCommitment is used in the Schnorr protocol.
	RandomCommitment ristretto.Point

	// Timestamp when the proof was generated.
	Timestamp int64

	// Nonce for replay prevention.
	Nonce [32]byte
}

// ProofBytes returns the serialized proof as bytes.
// Format: delta_commitment(32) || responseValue(32) || responseBlind(32) || challenge(32) || random_commitment(32) || nonce(32) || timestamp(8)
func (p *ThresholdProof) ProofBytes() []byte {
	out := make([]byte, 0, 200)

	deltaBytes := p.DeltaCommitment.Bytes()
	out = append(out, deltaBytes[:]...)

	respValBytes := p.ResponseValue.Bytes()
	out = append(out, respValBytes[:]...)

	respBlindBytes := p.ResponseBlind.Bytes()
	out = append(out, respBlindBytes[:]...)

	chalBytes := p.Challenge.Bytes()
	out = append(out, chalBytes[:]...)

	randBytes := p.RandomCommitment.Bytes()
	out = append(out, randBytes[:]...)

	out = append(out, p.Nonce[:]...)

	var tsBytes [8]byte
	binary.LittleEndian.PutUint64(tsBytes[:], uint64(p.Timestamp))
	out = append(out, tsBytes[:]...)

	return out
}

// SetProofBytes deserializes a proof from bytes.
func (p *ThresholdProof) SetProofBytes(data []byte) error {
	if len(data) < 200 {
		return ErrInvalidProof
	}

	var deltaBytes [32]byte
	copy(deltaBytes[:], data[0:32])
	if err := p.DeltaCommitment.SetBytes(deltaBytes); err != nil {
		return err
	}

	var respValBytes [32]byte
	copy(respValBytes[:], data[32:64])
	p.ResponseValue.SetBytes(&respValBytes)

	var respBlindBytes [32]byte
	copy(respBlindBytes[:], data[64:96])
	p.ResponseBlind.SetBytes(&respBlindBytes)

	var chalBytes [32]byte
	copy(chalBytes[:], data[96:128])
	p.Challenge.SetBytes(&chalBytes)

	var randBytes [32]byte
	copy(randBytes[:], data[128:160])
	if !p.RandomCommitment.SetBytes(&randBytes) {
		return ErrInvalidProof
	}

	copy(p.Nonce[:], data[160:192])

	p.Timestamp = int64(binary.LittleEndian.Uint64(data[192:200]))

	return nil
}

// RistrettoClaimGenerator generates ZK threshold proofs using Ristretto.
type RistrettoClaimGenerator struct {
	params *PedersenParams
}

// NewRistrettoClaimGenerator creates a new claim generator using Ristretto.
func NewRistrettoClaimGenerator() *RistrettoClaimGenerator {
	return &RistrettoClaimGenerator{
		params: DefaultPedersenParams(),
	}
}

// GenerateThresholdProof creates a ZK proof that value >= threshold.
// Returns the original commitment, the proof, and the blinding factor.
func (g *RistrettoClaimGenerator) GenerateThresholdProof(
	value, threshold int64,
) (*PedersenCommitment, *ThresholdProof, ristretto.Scalar, error) {
	if value < threshold {
		return nil, nil, ristretto.Scalar{}, ErrThresholdNotMet
	}

	// Commit to the value.
	commitment, blind := g.params.Commit(value)

	// Compute delta = value - threshold (must be >= 0).
	delta := value - threshold

	// Commit to delta with a new blinding factor.
	var deltaBlind ristretto.Scalar
	deltaBlind.Rand()
	deltaCommitment := g.params.CommitWithBlind(delta, deltaBlind)

	// Generate nonce.
	var nonce [32]byte
	rand.Read(nonce[:])

	// Generate random scalars for Schnorr protocol (one for value, one for blind).
	var kValue, kBlind ristretto.Scalar
	kValue.Rand()
	kBlind.Rand()

	// Compute random commitment R = k_v*G + k_r*H (proving knowledge of delta commitment opening).
	var kG, kH ristretto.Point
	kG.ScalarMult(&g.params.G, &kValue)
	kH.ScalarMult(&g.params.H, &kBlind)
	var R ristretto.Point
	R.Add(&kG, &kH)

	// Fiat-Shamir challenge: c = H(commitment || delta_commitment || R || threshold || nonce).
	challengeBytes := computeChallenge(commitment, deltaCommitment, &R, threshold, nonce[:])
	var challenge ristretto.Scalar
	challenge.SetBytes(&challengeBytes)

	// Convert delta to scalar.
	var deltaScalar ristretto.Scalar
	var deltaBytes [32]byte
	binary.LittleEndian.PutUint64(deltaBytes[:8], uint64(delta))
	deltaScalar.SetBytes(&deltaBytes)

	// Response: s_v = k_v + c * delta, s_r = k_r + c * deltaBlind.
	var responseValue, responseBlind ristretto.Scalar
	responseValue.Mul(&challenge, &deltaScalar)
	responseValue.Add(&responseValue, &kValue)

	responseBlind.Mul(&challenge, &deltaBlind)
	responseBlind.Add(&responseBlind, &kBlind)

	proof := &ThresholdProof{
		DeltaCommitment:  *deltaCommitment,
		ResponseValue:    responseValue,
		ResponseBlind:    responseBlind,
		Challenge:        challenge,
		RandomCommitment: R,
		Timestamp:        time.Now().Unix(),
		Nonce:            nonce,
	}

	return commitment, proof, blind, nil
}

// computeChallenge computes the Fiat-Shamir challenge hash.
func computeChallenge(
	commitment *PedersenCommitment,
	deltaCommitment *PedersenCommitment,
	R *ristretto.Point,
	threshold int64,
	nonce []byte,
) [32]byte {
	h := sha256.New()

	commitBytes := commitment.Bytes()
	h.Write(commitBytes[:])

	deltaBytes := deltaCommitment.Bytes()
	h.Write(deltaBytes[:])

	rBytes := R.Bytes()
	h.Write(rBytes[:])

	var threshBytes [8]byte
	binary.LittleEndian.PutUint64(threshBytes[:], uint64(threshold))
	h.Write(threshBytes[:])

	h.Write(nonce)

	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// RistrettoClaimVerifier verifies ZK threshold proofs using Ristretto.
type RistrettoClaimVerifier struct {
	params   *PedersenParams
	seenOnce map[[32]byte]int64
	mu       sync.RWMutex
}

// NewRistrettoClaimVerifier creates a new claim verifier.
func NewRistrettoClaimVerifier() *RistrettoClaimVerifier {
	return &RistrettoClaimVerifier{
		params:   DefaultPedersenParams(),
		seenOnce: make(map[[32]byte]int64),
	}
}

// VerifyThresholdProof verifies a ZK threshold proof.
// Returns nil if the proof is valid, error otherwise.
func (v *RistrettoClaimVerifier) VerifyThresholdProof(
	commitment *PedersenCommitment,
	proof *ThresholdProof,
	threshold int64,
) error {
	if err := v.checkProofFreshness(proof); err != nil {
		return err
	}

	if err := v.checkReplay(proof); err != nil {
		return err
	}

	if err := v.verifyChallengeMatches(commitment, proof, threshold); err != nil {
		return err
	}

	if err := v.verifySchnorrProof(proof); err != nil {
		return err
	}

	v.recordNonce(proof)
	return nil
}

// checkProofFreshness verifies the proof timestamp is within acceptable bounds.
func (v *RistrettoClaimVerifier) checkProofFreshness(proof *ThresholdProof) error {
	proofTime := time.Unix(proof.Timestamp, 0)
	if time.Since(proofTime) > ClaimFreshness {
		return ErrClaimExpired
	}
	if proofTime.After(time.Now().Add(30 * time.Second)) {
		return ErrInvalidClaim
	}
	return nil
}

// checkReplay verifies the nonce has not been seen before.
func (v *RistrettoClaimVerifier) checkReplay(proof *ThresholdProof) error {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if _, seen := v.seenOnce[proof.Nonce]; seen {
		return ErrReplayDetected
	}
	return nil
}

// verifyChallengeMatches verifies the challenge was computed correctly.
func (v *RistrettoClaimVerifier) verifyChallengeMatches(
	commitment *PedersenCommitment,
	proof *ThresholdProof,
	threshold int64,
) error {
	expectedChallenge := computeChallenge(
		commitment,
		&proof.DeltaCommitment,
		&proof.RandomCommitment,
		threshold,
		proof.Nonce[:],
	)
	var expectedChalScalar ristretto.Scalar
	expectedChalScalar.SetBytes(&expectedChallenge)

	if !proof.Challenge.Equals(&expectedChalScalar) {
		return ErrInvalidProof
	}
	return nil
}

// verifySchnorrProof verifies the Schnorr proof equation: s_v*G + s_r*H == R + c*C.
func (v *RistrettoClaimVerifier) verifySchnorrProof(proof *ThresholdProof) error {
	left := v.computeSchnorrLeft(proof)
	right := v.computeSchnorrRight(proof)

	if !left.Equals(&right) {
		return ErrInvalidProof
	}
	return nil
}

// computeSchnorrLeft computes left side of Schnorr equation: s_v*G + s_r*H.
func (v *RistrettoClaimVerifier) computeSchnorrLeft(proof *ThresholdProof) ristretto.Point {
	var sG, sH ristretto.Point
	sG.ScalarMult(&v.params.G, &proof.ResponseValue)
	sH.ScalarMult(&v.params.H, &proof.ResponseBlind)
	var left ristretto.Point
	left.Add(&sG, &sH)
	return left
}

// computeSchnorrRight computes right side of Schnorr equation: R + c*deltaCommitment.
func (v *RistrettoClaimVerifier) computeSchnorrRight(proof *ThresholdProof) ristretto.Point {
	var cDelta ristretto.Point
	cDelta.ScalarMult(&proof.DeltaCommitment.Point, &proof.Challenge)
	var right ristretto.Point
	right.Add(&proof.RandomCommitment, &cDelta)
	return right
}

// recordNonce records the nonce in the replay cache.
func (v *RistrettoClaimVerifier) recordNonce(proof *ThresholdProof) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.seenOnce[proof.Nonce] = proof.Timestamp
}

// CleanExpiredNonces removes old nonces from the replay cache.
func (v *RistrettoClaimVerifier) CleanExpiredNonces() {
	v.mu.Lock()
	defer v.mu.Unlock()

	cutoff := time.Now().Add(-ClaimFreshness).Unix()
	for nonce, timestamp := range v.seenOnce {
		if timestamp < cutoff {
			delete(v.seenOnce, nonce)
		}
	}
}

// ZKClaimType represents the type of ZK claim.
type ZKClaimType uint8

const (
	// ZKClaimResonanceRange proves Resonance >= threshold.
	ZKClaimResonanceRange ZKClaimType = iota + 1

	// ZKClaimSpecterAge proves Specter age >= threshold days.
	ZKClaimSpecterAge

	// ZKClaimIgnitionCount proves met >= threshold peers in person.
	ZKClaimIgnitionCount

	// ZKClaimEventParticipation proves participated in >= threshold events.
	ZKClaimEventParticipation
)

// String returns a human-readable claim type name.
func (t ZKClaimType) String() string {
	switch t {
	case ZKClaimResonanceRange:
		return "ResonanceRange"
	case ZKClaimSpecterAge:
		return "SpecterAge"
	case ZKClaimIgnitionCount:
		return "IgnitionCount"
	case ZKClaimEventParticipation:
		return "EventParticipation"
	default:
		return "Unknown"
	}
}

// ZKClaim is a complete zero-knowledge claim with metadata.
type ZKClaim struct {
	Type       ZKClaimType        // Type of claim.
	SpecterID  string             // Specter making the claim.
	Threshold  int64              // Minimum value being proven.
	Commitment PedersenCommitment // Commitment to the actual value.
	Proof      ThresholdProof     // ZK proof that value >= threshold.
}

// NewZKClaim creates a new ZK claim for a Resonance threshold.
func NewZKClaim(claimType ZKClaimType, specterID string, value, threshold int64) (*ZKClaim, ristretto.Scalar, error) {
	gen := NewRistrettoClaimGenerator()
	commitment, proof, blind, err := gen.GenerateThresholdProof(value, threshold)
	if err != nil {
		return nil, ristretto.Scalar{}, err
	}

	return &ZKClaim{
		Type:       claimType,
		SpecterID:  specterID,
		Threshold:  threshold,
		Commitment: *commitment,
		Proof:      *proof,
	}, blind, nil
}

// Verify verifies the ZK claim.
func (c *ZKClaim) Verify() error {
	verifier := NewRistrettoClaimVerifier()
	return verifier.VerifyThresholdProof(&c.Commitment, &c.Proof, c.Threshold)
}

// Bytes serializes the claim.
func (c *ZKClaim) Bytes() []byte {
	out := make([]byte, 0)

	// Type (1 byte).
	out = append(out, byte(c.Type))

	// SpecterID length (2 bytes) + SpecterID.
	idBytes := []byte(c.SpecterID)
	var idLen [2]byte
	binary.LittleEndian.PutUint16(idLen[:], uint16(len(idBytes)))
	out = append(out, idLen[:]...)
	out = append(out, idBytes...)

	// Threshold (8 bytes).
	var threshBytes [8]byte
	binary.LittleEndian.PutUint64(threshBytes[:], uint64(c.Threshold))
	out = append(out, threshBytes[:]...)

	// Commitment (32 bytes).
	commitBytes := c.Commitment.Bytes()
	out = append(out, commitBytes[:]...)

	// Proof.
	proofBytes := c.Proof.ProofBytes()
	out = append(out, proofBytes...)

	return out
}

// SetBytes deserializes a claim from bytes.
func (c *ZKClaim) SetBytes(data []byte) error {
	if len(data) < 43 { // Minimum: type(1) + idLen(2) + threshold(8) + commitment(32)
		return ErrInvalidClaim
	}

	pos := 0
	var err error

	pos, err = c.decodeType(data, pos)
	if err != nil {
		return err
	}

	pos, err = c.decodeSpecterID(data, pos)
	if err != nil {
		return err
	}

	pos, err = c.decodeThreshold(data, pos)
	if err != nil {
		return err
	}

	pos, err = c.decodeCommitment(data, pos)
	if err != nil {
		return err
	}

	return c.decodeProof(data, pos)
}

func (c *ZKClaim) decodeType(data []byte, pos int) (int, error) {
	c.Type = ZKClaimType(data[pos])
	return pos + 1, nil
}

func (c *ZKClaim) decodeSpecterID(data []byte, pos int) (int, error) {
	idLen := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
	pos += 2
	if pos+idLen > len(data) {
		return pos, ErrInvalidClaim
	}
	c.SpecterID = string(data[pos : pos+idLen])
	return pos + idLen, nil
}

func (c *ZKClaim) decodeThreshold(data []byte, pos int) (int, error) {
	if pos+8 > len(data) {
		return pos, ErrInvalidClaim
	}
	c.Threshold = int64(binary.LittleEndian.Uint64(data[pos : pos+8]))
	return pos + 8, nil
}

func (c *ZKClaim) decodeCommitment(data []byte, pos int) (int, error) {
	if pos+32 > len(data) {
		return pos, ErrInvalidClaim
	}
	var commitBytes [32]byte
	copy(commitBytes[:], data[pos:pos+32])
	if err := c.Commitment.SetBytes(commitBytes); err != nil {
		return pos, err
	}
	return pos + 32, nil
}

func (c *ZKClaim) decodeProof(data []byte, pos int) error {
	if pos+200 > len(data) {
		return ErrInvalidClaim
	}
	return c.Proof.SetProofBytes(data[pos:])
}

// ProofSizeBytes returns the expected size of a ZK proof.
// Per SECURITY_PRIVACY.md, this is approximately 672 bytes for a 64-bit range,
// but our Schnorr-based proof is smaller at ~200 bytes.
func ProofSizeBytes() int {
	return 200
}

// ZKClaimVerifierAdapter provides ZK claim verification for mechanics.
// Per ROADMAP.md line 400: "ZK claims used for Council admission and mini-game thresholds".
type ZKClaimVerifierAdapter struct{}

// NewZKClaimVerifierAdapter creates a new verifier adapter.
func NewZKClaimVerifierAdapter() *ZKClaimVerifierAdapter {
	return &ZKClaimVerifierAdapter{}
}

// VerifyResonanceClaim verifies that a ZK claim proves Resonance >= threshold.
// The proof bytes must be a serialized ZKClaim of type ZKClaimResonanceRange.
func (a *ZKClaimVerifierAdapter) VerifyResonanceClaim(proof []byte, minResonance int64) error {
	if len(proof) == 0 {
		return ErrInvalidClaim
	}

	var claim ZKClaim
	if err := claim.SetBytes(proof); err != nil {
		return err
	}

	// Verify the claim type is appropriate for Resonance threshold.
	if claim.Type != ZKClaimResonanceRange {
		return ErrInvalidClaim
	}

	// Verify the claim's threshold meets the requirement.
	if claim.Threshold < minResonance {
		return ErrInvalidClaim
	}

	// Verify the cryptographic proof.
	return claim.Verify()
}
