package resonance

import (
	"testing"
	"time"
)

func TestNewBulletproofRangeProver(t *testing.T) {
	prover, err := NewBulletproofRangeProver()
	if err != nil {
		t.Fatalf("NewBulletproofRangeProver failed: %v", err)
	}
	if prover == nil {
		t.Fatal("Prover is nil")
	}
	if prover.params == nil {
		t.Fatal("Params is nil")
	}
	if prover.prover == nil {
		t.Fatal("Internal prover is nil")
	}
}

func TestNewBulletproofRangeVerifier(t *testing.T) {
	verifier, err := NewBulletproofRangeVerifier()
	if err != nil {
		t.Fatalf("NewBulletproofRangeVerifier failed: %v", err)
	}
	if verifier == nil {
		t.Fatal("Verifier is nil")
	}
}

func TestBulletproofRangeProofRoundTrip(t *testing.T) {
	prover, err := NewBulletproofRangeProver()
	if err != nil {
		t.Fatalf("NewBulletproofRangeProver failed: %v", err)
	}

	verifier, err := NewBulletproofRangeVerifier()
	if err != nil {
		t.Fatalf("NewBulletproofRangeVerifier failed: %v", err)
	}

	// Test with a small value.
	value := uint64(42)
	proof, gamma, err := prover.GenerateRangeProof(value)
	if err != nil {
		t.Fatalf("GenerateRangeProof failed: %v", err)
	}
	if proof == nil {
		t.Fatal("Proof is nil")
	}
	if gamma == nil {
		t.Fatal("Gamma is nil")
	}
	if len(proof.Proof) == 0 {
		t.Error("Proof bytes are empty")
	}
	if len(proof.CapV) == 0 {
		t.Error("CapV is empty")
	}

	// Verify the proof.
	if err := verifier.Verify(proof); err != nil {
		t.Errorf("Verify failed: %v", err)
	}
}

func TestBulletproofRangeProofLargeValue(t *testing.T) {
	prover, err := NewBulletproofRangeProver()
	if err != nil {
		t.Fatalf("NewBulletproofRangeProver failed: %v", err)
	}

	verifier, err := NewBulletproofRangeVerifier()
	if err != nil {
		t.Fatalf("NewBulletproofRangeVerifier failed: %v", err)
	}

	// Test with a large value (close to max int).
	value := uint64(1000000000)
	proof, _, err := prover.GenerateRangeProof(value)
	if err != nil {
		t.Fatalf("GenerateRangeProof failed for large value: %v", err)
	}

	if err := verifier.Verify(proof); err != nil {
		t.Errorf("Verify failed for large value: %v", err)
	}
}

func TestBulletproofRangeProofZeroValue(t *testing.T) {
	prover, err := NewBulletproofRangeProver()
	if err != nil {
		t.Fatalf("NewBulletproofRangeProver failed: %v", err)
	}

	verifier, err := NewBulletproofRangeVerifier()
	if err != nil {
		t.Fatalf("NewBulletproofRangeVerifier failed: %v", err)
	}

	// Test with zero value.
	proof, _, err := prover.GenerateRangeProof(0)
	if err != nil {
		t.Fatalf("GenerateRangeProof failed for zero: %v", err)
	}

	if err := verifier.Verify(proof); err != nil {
		t.Errorf("Verify failed for zero value: %v", err)
	}
}

func TestBulletproofRangeProofReplayPrevention(t *testing.T) {
	prover, err := NewBulletproofRangeProver()
	if err != nil {
		t.Fatalf("NewBulletproofRangeProver failed: %v", err)
	}

	verifier, err := NewBulletproofRangeVerifier()
	if err != nil {
		t.Fatalf("NewBulletproofRangeVerifier failed: %v", err)
	}

	proof, _, err := prover.GenerateRangeProof(100)
	if err != nil {
		t.Fatalf("GenerateRangeProof failed: %v", err)
	}

	// First verification should succeed.
	if err := verifier.Verify(proof); err != nil {
		t.Errorf("First verify failed: %v", err)
	}

	// Second verification with same nonce should fail.
	if err := verifier.Verify(proof); err != ErrReplayDetected {
		t.Errorf("Expected ErrReplayDetected, got: %v", err)
	}
}

func TestBulletproofRangeProofExpired(t *testing.T) {
	prover, err := NewBulletproofRangeProver()
	if err != nil {
		t.Fatalf("NewBulletproofRangeProver failed: %v", err)
	}

	verifier, err := NewBulletproofRangeVerifier()
	if err != nil {
		t.Fatalf("NewBulletproofRangeVerifier failed: %v", err)
	}

	proof, _, err := prover.GenerateRangeProof(100)
	if err != nil {
		t.Fatalf("GenerateRangeProof failed: %v", err)
	}

	// Backdate the proof beyond freshness window.
	proof.Timestamp = time.Now().Add(-ClaimFreshness - time.Minute).Unix()

	if err := verifier.Verify(proof); err != ErrClaimExpired {
		t.Errorf("Expected ErrClaimExpired, got: %v", err)
	}
}

func TestBulletproofRangeProofFutureTimestamp(t *testing.T) {
	prover, err := NewBulletproofRangeProver()
	if err != nil {
		t.Fatalf("NewBulletproofRangeProver failed: %v", err)
	}

	verifier, err := NewBulletproofRangeVerifier()
	if err != nil {
		t.Fatalf("NewBulletproofRangeVerifier failed: %v", err)
	}

	proof, _, err := prover.GenerateRangeProof(100)
	if err != nil {
		t.Fatalf("GenerateRangeProof failed: %v", err)
	}

	// Set timestamp in the future.
	proof.Timestamp = time.Now().Add(5 * time.Minute).Unix()

	if err := verifier.Verify(proof); err != ErrInvalidClaim {
		t.Errorf("Expected ErrInvalidClaim, got: %v", err)
	}
}

func TestBulletproofThresholdProverSuccess(t *testing.T) {
	prover, err := NewBulletproofThresholdProver()
	if err != nil {
		t.Fatalf("NewBulletproofThresholdProver failed: %v", err)
	}

	verifier, err := NewBulletproofThresholdVerifier()
	if err != nil {
		t.Fatalf("NewBulletproofThresholdVerifier failed: %v", err)
	}

	// Value exceeds threshold.
	proof, err := prover.GenerateThresholdProof(150, 100)
	if err != nil {
		t.Fatalf("GenerateThresholdProof failed: %v", err)
	}
	if proof == nil {
		t.Fatal("Proof is nil")
	}
	if proof.Threshold != 100 {
		t.Errorf("Threshold = %d, want 100", proof.Threshold)
	}

	if err := verifier.VerifyThresholdProof(proof); err != nil {
		t.Errorf("VerifyThresholdProof failed: %v", err)
	}
}

func TestBulletproofThresholdProverExactThreshold(t *testing.T) {
	prover, err := NewBulletproofThresholdProver()
	if err != nil {
		t.Fatalf("NewBulletproofThresholdProver failed: %v", err)
	}

	verifier, err := NewBulletproofThresholdVerifier()
	if err != nil {
		t.Fatalf("NewBulletproofThresholdVerifier failed: %v", err)
	}

	// Value equals threshold (delta = 0).
	proof, err := prover.GenerateThresholdProof(100, 100)
	if err != nil {
		t.Fatalf("GenerateThresholdProof failed for exact threshold: %v", err)
	}

	if err := verifier.VerifyThresholdProof(proof); err != nil {
		t.Errorf("VerifyThresholdProof failed for exact threshold: %v", err)
	}
}

func TestBulletproofThresholdProverBelowThreshold(t *testing.T) {
	prover, err := NewBulletproofThresholdProver()
	if err != nil {
		t.Fatalf("NewBulletproofThresholdProver failed: %v", err)
	}

	// Value below threshold should fail.
	_, err = prover.GenerateThresholdProof(50, 100)
	if err != ErrThresholdNotMet {
		t.Errorf("Expected ErrThresholdNotMet, got: %v", err)
	}
}

func TestBulletproofThresholdProofSerialization(t *testing.T) {
	prover, err := NewBulletproofThresholdProver()
	if err != nil {
		t.Fatalf("NewBulletproofThresholdProver failed: %v", err)
	}

	proof, err := prover.GenerateThresholdProof(200, 100)
	if err != nil {
		t.Fatalf("GenerateThresholdProof failed: %v", err)
	}

	// Serialize.
	data := proof.Bytes()
	if len(data) == 0 {
		t.Fatal("Serialized proof is empty")
	}

	// Deserialize.
	proof2 := &BulletproofThresholdProof{}
	if err := proof2.SetBytes(data); err != nil {
		t.Fatalf("SetBytes failed: %v", err)
	}

	// Compare.
	if proof2.Threshold != proof.Threshold {
		t.Errorf("Threshold mismatch: %d vs %d", proof2.Threshold, proof.Threshold)
	}
	if proof2.DeltaRangeProof.Timestamp != proof.DeltaRangeProof.Timestamp {
		t.Errorf("Timestamp mismatch")
	}
	if proof2.DeltaRangeProof.N != proof.DeltaRangeProof.N {
		t.Errorf("N mismatch: %d vs %d", proof2.DeltaRangeProof.N, proof.DeltaRangeProof.N)
	}
}

func TestBulletproofVerifierCleanExpiredNonces(t *testing.T) {
	verifier, err := NewBulletproofRangeVerifier()
	if err != nil {
		t.Fatalf("NewBulletproofRangeVerifier failed: %v", err)
	}

	// Manually add some nonces (simulating old verifications).
	verifier.mu.Lock()
	oldTime := time.Now().Add(-ClaimFreshness - time.Hour).Unix()
	newTime := time.Now().Unix()
	verifier.seenOnce[[32]byte{1}] = oldTime
	verifier.seenOnce[[32]byte{2}] = newTime
	verifier.mu.Unlock()

	// Clean expired.
	verifier.CleanExpiredNonces()

	// Old nonce should be removed.
	verifier.mu.RLock()
	defer verifier.mu.RUnlock()
	if _, ok := verifier.seenOnce[[32]byte{1}]; ok {
		t.Error("Old nonce should have been cleaned")
	}
	if _, ok := verifier.seenOnce[[32]byte{2}]; !ok {
		t.Error("New nonce should still exist")
	}
}

func TestDefaultBulletproofParams(t *testing.T) {
	params := DefaultBulletproofParams()

	if params.Curve == nil {
		t.Error("Curve is nil")
	}
	if len(params.RangeDomain) == 0 {
		t.Error("RangeDomain is empty")
	}
	if len(params.IppDomain) == 0 {
		t.Error("IppDomain is empty")
	}
	if params.BitLength != 64 {
		t.Errorf("BitLength = %d, want 64", params.BitLength)
	}
}

func BenchmarkBulletproofRangeProofGeneration(b *testing.B) {
	prover, err := NewBulletproofRangeProver()
	if err != nil {
		b.Fatalf("NewBulletproofRangeProver failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := prover.GenerateRangeProof(uint64(i))
		if err != nil {
			b.Fatalf("GenerateRangeProof failed: %v", err)
		}
	}
}

func BenchmarkBulletproofRangeProofVerification(b *testing.B) {
	prover, err := NewBulletproofRangeProver()
	if err != nil {
		b.Fatalf("NewBulletproofRangeProver failed: %v", err)
	}

	// Generate proofs for verification.
	proofs := make([]*BulletproofRangeProof, b.N)
	for i := 0; i < b.N; i++ {
		proof, _, err := prover.GenerateRangeProof(uint64(i))
		if err != nil {
			b.Fatalf("GenerateRangeProof failed: %v", err)
		}
		proofs[i] = proof
	}

	verifier, err := NewBulletproofRangeVerifier()
	if err != nil {
		b.Fatalf("NewBulletproofRangeVerifier failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := verifier.Verify(proofs[i]); err != nil {
			b.Fatalf("Verify failed: %v", err)
		}
	}
}
