package resonance

import (
	"testing"
	"time"

	"github.com/bwesterb/go-ristretto"
)

func TestDefaultPedersenParams(t *testing.T) {
	params := DefaultPedersenParams()

	// G should not be zero.
	var zero ristretto.Point
	zero.SetZero()
	if params.G.Equals(&zero) {
		t.Error("G should not be zero")
	}

	// H should not be zero.
	if params.H.Equals(&zero) {
		t.Error("H should not be zero")
	}

	// G and H should be different.
	if params.G.Equals(&params.H) {
		t.Error("G and H should be different")
	}
}

func TestPedersenCommit(t *testing.T) {
	params := DefaultPedersenParams()

	// Test basic commitment.
	commitment, blind := params.Commit(100)
	if commitment == nil {
		t.Fatal("Commit returned nil")
	}

	var zero ristretto.Point
	zero.SetZero()
	if commitment.Point.Equals(&zero) {
		t.Error("Commitment should not be zero")
	}

	// Same value, same blind should give same commitment.
	commitment2 := params.CommitWithBlind(100, blind)
	if !commitment.Equal(commitment2) {
		t.Error("Same value and blind should produce same commitment")
	}

	// Different blind should give different commitment.
	var differentBlind ristretto.Scalar
	differentBlind.Rand()
	commitment3 := params.CommitWithBlind(100, differentBlind)
	if commitment.Equal(commitment3) {
		t.Error("Different blind should produce different commitment")
	}

	// Different value should give different commitment.
	commitment4 := params.CommitWithBlind(200, blind)
	if commitment.Equal(commitment4) {
		t.Error("Different value should produce different commitment")
	}
}

func TestPedersenCommitmentBytes(t *testing.T) {
	params := DefaultPedersenParams()
	commitment, _ := params.Commit(42)

	// Get bytes.
	bytes := commitment.Bytes()
	if len(bytes) != 32 {
		t.Errorf("Commitment bytes length = %d, want 32", len(bytes))
	}

	// Round-trip.
	var commitment2 PedersenCommitment
	err := commitment2.SetBytes(bytes)
	if err != nil {
		t.Fatalf("SetBytes failed: %v", err)
	}

	if !commitment.Equal(&commitment2) {
		t.Error("Round-trip commitment should be equal")
	}
}

func TestRistrettoClaimGenerator(t *testing.T) {
	gen := NewRistrettoClaimGenerator()
	if gen == nil {
		t.Fatal("NewRistrettoClaimGenerator returned nil")
	}
	if gen.params == nil {
		t.Fatal("params should not be nil")
	}
}

func TestGenerateThresholdProof(t *testing.T) {
	gen := NewRistrettoClaimGenerator()

	// Test valid proof (value > threshold).
	commitment, proof, blind, err := gen.GenerateThresholdProof(100, 50)
	if err != nil {
		t.Fatalf("GenerateThresholdProof failed: %v", err)
	}
	if commitment == nil {
		t.Error("commitment should not be nil")
	}
	if proof == nil {
		t.Error("proof should not be nil")
	}

	var zeroScalar ristretto.Scalar
	if blind.Equals(&zeroScalar) {
		t.Error("blind should not be zero")
	}

	// Test value == threshold (boundary).
	_, _, _, err = gen.GenerateThresholdProof(100, 100)
	if err != nil {
		t.Errorf("Value == threshold should succeed: %v", err)
	}

	// Test value < threshold (should fail).
	_, _, _, err = gen.GenerateThresholdProof(50, 100)
	if err != ErrThresholdNotMet {
		t.Errorf("Value < threshold should return ErrThresholdNotMet, got %v", err)
	}
}

func TestVerifyThresholdProof(t *testing.T) {
	gen := NewRistrettoClaimGenerator()
	verifier := NewRistrettoClaimVerifier()

	// Generate a valid proof.
	value := int64(150)
	threshold := int64(100)
	commitment, proof, _, err := gen.GenerateThresholdProof(value, threshold)
	if err != nil {
		t.Fatalf("GenerateThresholdProof failed: %v", err)
	}

	// Verify the proof.
	err = verifier.VerifyThresholdProof(commitment, proof, threshold)
	if err != nil {
		t.Errorf("Valid proof should verify: %v", err)
	}
}

func TestVerifyThresholdProofWrongThreshold(t *testing.T) {
	gen := NewRistrettoClaimGenerator()
	verifier := NewRistrettoClaimVerifier()

	// Generate a proof for threshold 100.
	commitment, proof, _, err := gen.GenerateThresholdProof(150, 100)
	if err != nil {
		t.Fatalf("GenerateThresholdProof failed: %v", err)
	}

	// Try to verify with a different threshold.
	err = verifier.VerifyThresholdProof(commitment, proof, 50)
	if err != ErrInvalidProof {
		t.Errorf("Wrong threshold should fail with ErrInvalidProof, got %v", err)
	}
}

func TestVerifyThresholdProofReplay(t *testing.T) {
	gen := NewRistrettoClaimGenerator()
	verifier := NewRistrettoClaimVerifier()

	commitment, proof, _, err := gen.GenerateThresholdProof(150, 100)
	if err != nil {
		t.Fatalf("GenerateThresholdProof failed: %v", err)
	}

	// First verification should succeed.
	err = verifier.VerifyThresholdProof(commitment, proof, 100)
	if err != nil {
		t.Errorf("First verification should succeed: %v", err)
	}

	// Second verification (replay) should fail.
	err = verifier.VerifyThresholdProof(commitment, proof, 100)
	if err != ErrReplayDetected {
		t.Errorf("Replay should fail with ErrReplayDetected, got %v", err)
	}
}

func TestVerifyThresholdProofExpired(t *testing.T) {
	gen := NewRistrettoClaimGenerator()
	verifier := NewRistrettoClaimVerifier()

	commitment, proof, _, err := gen.GenerateThresholdProof(150, 100)
	if err != nil {
		t.Fatalf("GenerateThresholdProof failed: %v", err)
	}

	// Set timestamp to expired.
	proof.Timestamp = time.Now().Add(-ClaimFreshness - time.Minute).Unix()

	err = verifier.VerifyThresholdProof(commitment, proof, 100)
	if err != ErrClaimExpired {
		t.Errorf("Expired proof should fail with ErrClaimExpired, got %v", err)
	}
}

func TestVerifyThresholdProofFuture(t *testing.T) {
	gen := NewRistrettoClaimGenerator()
	verifier := NewRistrettoClaimVerifier()

	commitment, proof, _, err := gen.GenerateThresholdProof(150, 100)
	if err != nil {
		t.Fatalf("GenerateThresholdProof failed: %v", err)
	}

	// Set timestamp to far future.
	proof.Timestamp = time.Now().Add(5 * time.Minute).Unix()

	err = verifier.VerifyThresholdProof(commitment, proof, 100)
	if err != ErrInvalidClaim {
		t.Errorf("Future proof should fail with ErrInvalidClaim, got %v", err)
	}
}

func TestCleanExpiredNonces(t *testing.T) {
	verifier := NewRistrettoClaimVerifier()

	// Add some old nonces.
	oldTime := time.Now().Add(-ClaimFreshness - time.Minute).Unix()
	var nonce1, nonce2, nonce3 [32]byte
	nonce1[0] = 1
	nonce2[0] = 2
	nonce3[0] = 3

	verifier.mu.Lock()
	verifier.seenOnce[nonce1] = oldTime
	verifier.seenOnce[nonce2] = oldTime
	verifier.seenOnce[nonce3] = time.Now().Unix() // Fresh.
	verifier.mu.Unlock()

	verifier.CleanExpiredNonces()

	verifier.mu.RLock()
	count := len(verifier.seenOnce)
	verifier.mu.RUnlock()

	if count != 1 {
		t.Errorf("After cleanup, count = %d, want 1", count)
	}
}

func TestZKClaimType(t *testing.T) {
	tests := []struct {
		claimType ZKClaimType
		want      string
	}{
		{ZKClaimResonanceRange, "ResonanceRange"},
		{ZKClaimSpecterAge, "SpecterAge"},
		{ZKClaimIgnitionCount, "IgnitionCount"},
		{ZKClaimEventParticipation, "EventParticipation"},
		{ZKClaimType(255), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.claimType.String()
		if got != tt.want {
			t.Errorf("%d.String() = %s, want %s", tt.claimType, got, tt.want)
		}
	}
}

func TestNewZKClaim(t *testing.T) {
	claim, blind, err := NewZKClaim(ZKClaimResonanceRange, "specter123", 200, 100)
	if err != nil {
		t.Fatalf("NewZKClaim failed: %v", err)
	}
	if claim == nil {
		t.Fatal("claim should not be nil")
	}

	var zeroScalar ristretto.Scalar
	if blind.Equals(&zeroScalar) {
		t.Error("blind should not be zero")
	}

	if claim.Type != ZKClaimResonanceRange {
		t.Errorf("Type = %v, want %v", claim.Type, ZKClaimResonanceRange)
	}
	if claim.SpecterID != "specter123" {
		t.Errorf("SpecterID = %s, want specter123", claim.SpecterID)
	}
	if claim.Threshold != 100 {
		t.Errorf("Threshold = %d, want 100", claim.Threshold)
	}
}

func TestZKClaimVerify(t *testing.T) {
	claim, _, err := NewZKClaim(ZKClaimResonanceRange, "specter123", 200, 100)
	if err != nil {
		t.Fatalf("NewZKClaim failed: %v", err)
	}

	err = claim.Verify()
	if err != nil {
		t.Errorf("Verify should succeed: %v", err)
	}
}

func TestZKClaimBytes(t *testing.T) {
	claim, _, err := NewZKClaim(ZKClaimSpecterAge, "test_specter_id", 365, 90)
	if err != nil {
		t.Fatalf("NewZKClaim failed: %v", err)
	}

	// Serialize.
	data := claim.Bytes()
	if len(data) == 0 {
		t.Error("Bytes should not be empty")
	}

	// Deserialize.
	var claim2 ZKClaim
	err = claim2.SetBytes(data)
	if err != nil {
		t.Fatalf("SetBytes failed: %v", err)
	}

	if claim2.Type != claim.Type {
		t.Errorf("Type = %v, want %v", claim2.Type, claim.Type)
	}
	if claim2.SpecterID != claim.SpecterID {
		t.Errorf("SpecterID = %s, want %s", claim2.SpecterID, claim.SpecterID)
	}
	if claim2.Threshold != claim.Threshold {
		t.Errorf("Threshold = %d, want %d", claim2.Threshold, claim.Threshold)
	}
	if !claim2.Commitment.Equal(&claim.Commitment) {
		t.Error("Commitment mismatch")
	}
}

func TestThresholdProofBytes(t *testing.T) {
	gen := NewRistrettoClaimGenerator()
	_, proof, _, err := gen.GenerateThresholdProof(100, 50)
	if err != nil {
		t.Fatalf("GenerateThresholdProof failed: %v", err)
	}

	// Serialize.
	data := proof.ProofBytes()
	if len(data) != 200 {
		t.Errorf("ProofBytes length = %d, want 200", len(data))
	}

	// Deserialize.
	var proof2 ThresholdProof
	err = proof2.SetProofBytes(data)
	if err != nil {
		t.Fatalf("SetProofBytes failed: %v", err)
	}

	if !proof2.DeltaCommitment.Equal(&proof.DeltaCommitment) {
		t.Error("DeltaCommitment mismatch")
	}
	if !proof2.ResponseValue.Equals(&proof.ResponseValue) {
		t.Error("ResponseValue mismatch")
	}
	if !proof2.ResponseBlind.Equals(&proof.ResponseBlind) {
		t.Error("ResponseBlind mismatch")
	}
	if !proof2.Challenge.Equals(&proof.Challenge) {
		t.Error("Challenge mismatch")
	}
	if proof2.Nonce != proof.Nonce {
		t.Error("Nonce mismatch")
	}
	if proof2.Timestamp != proof.Timestamp {
		t.Error("Timestamp mismatch")
	}
}

func TestProofSizeBytes(t *testing.T) {
	size := ProofSizeBytes()
	if size != 200 {
		t.Errorf("ProofSizeBytes() = %d, want 200", size)
	}
}

func TestZKClaimBelowThreshold(t *testing.T) {
	_, _, err := NewZKClaim(ZKClaimResonanceRange, "specter123", 50, 100)
	if err != ErrThresholdNotMet {
		t.Errorf("Below threshold should return ErrThresholdNotMet, got %v", err)
	}
}

func TestPedersenCommitHomomorphic(t *testing.T) {
	// Pedersen commitments should be additively homomorphic:
	// C(a) + C(b) should equal C(a+b) with combined blinds.
	params := DefaultPedersenParams()

	var blind1, blind2 ristretto.Scalar
	blind1.Rand()
	blind2.Rand()

	c1 := params.CommitWithBlind(10, blind1)
	c2 := params.CommitWithBlind(20, blind2)

	// Compute C(a+b) with blind1+blind2.
	var combinedBlind ristretto.Scalar
	combinedBlind.Add(&blind1, &blind2)
	cSum := params.CommitWithBlind(30, combinedBlind)

	// Add the two commitments.
	var cAdded PedersenCommitment
	cAdded.Point.Add(&c1.Point, &c2.Point)

	if !cAdded.Equal(cSum) {
		t.Error("Pedersen commitments should be additively homomorphic")
	}
}

// TestZKClaimVerifierAdapter tests the adapter for mechanics integration.
func TestZKClaimVerifierAdapter(t *testing.T) {
	adapter := NewZKClaimVerifierAdapter()

	// Create a valid ZK claim for Resonance >= 200.
	claim, _, err := NewZKClaim(ZKClaimResonanceRange, "test-specter", 250, 200)
	if err != nil {
		t.Fatalf("NewZKClaim failed: %v", err)
	}

	proofBytes := claim.Bytes()

	// Verify with matching threshold.
	err = adapter.VerifyResonanceClaim(proofBytes, 200)
	if err != nil {
		t.Errorf("VerifyResonanceClaim should succeed: %v", err)
	}

	// Verify with lower threshold (should still pass).
	err = adapter.VerifyResonanceClaim(proofBytes, 100)
	if err != nil {
		t.Errorf("VerifyResonanceClaim with lower threshold should succeed: %v", err)
	}

	// Verify with higher threshold (should fail - claim proves >= 200, not >= 300).
	err = adapter.VerifyResonanceClaim(proofBytes, 300)
	if err == nil {
		t.Error("VerifyResonanceClaim with higher threshold should fail")
	}
}

// TestZKClaimVerifierAdapterInvalidProof tests adapter with invalid proofs.
func TestZKClaimVerifierAdapterInvalidProof(t *testing.T) {
	adapter := NewZKClaimVerifierAdapter()

	// Empty proof.
	err := adapter.VerifyResonanceClaim(nil, 100)
	if err == nil {
		t.Error("Should fail with nil proof")
	}

	err = adapter.VerifyResonanceClaim([]byte{}, 100)
	if err == nil {
		t.Error("Should fail with empty proof")
	}

	// Garbage bytes.
	err = adapter.VerifyResonanceClaim([]byte("garbage data"), 100)
	if err == nil {
		t.Error("Should fail with garbage proof")
	}
}

// TestZKClaimVerifierAdapterWrongClaimType tests adapter rejects wrong claim types.
func TestZKClaimVerifierAdapterWrongClaimType(t *testing.T) {
	adapter := NewZKClaimVerifierAdapter()

	// Create a ZK claim with wrong type (Ignition count instead of Resonance).
	claim, _, err := NewZKClaim(ZKClaimIgnitionCount, "test-specter", 10, 5)
	if err != nil {
		t.Fatalf("NewZKClaim failed: %v", err)
	}

	proofBytes := claim.Bytes()

	// Should fail because claim type is not ResonanceRange.
	err = adapter.VerifyResonanceClaim(proofBytes, 5)
	if err == nil {
		t.Error("Should fail with wrong claim type")
	}
}
