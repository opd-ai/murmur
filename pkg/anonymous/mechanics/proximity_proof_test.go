package mechanics

import (
	"crypto/ed25519"
	"crypto/rand"
	"math/big"
	"testing"
	"time"
)

func TestNewProximityAttestation(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	distance := big.NewInt(12345)
	att := NewProximityAttestation(priv, "peer123", claimerKey, targetHash, distance)

	if att == nil {
		t.Fatal("expected attestation, got nil")
	}

	var expectedPub [32]byte
	copy(expectedPub[:], pub)
	if att.AttesterPubKey != expectedPub {
		t.Error("attester public key mismatch")
	}

	if att.AttesterPeerID != "peer123" {
		t.Errorf("expected peer123, got %s", att.AttesterPeerID)
	}

	if att.ClaimerPubKey != claimerKey {
		t.Error("claimer key mismatch")
	}

	if att.TargetHash != targetHash {
		t.Error("target hash mismatch")
	}

	if att.Timestamp == 0 {
		t.Error("timestamp should not be zero")
	}
}

func TestProximityAttestation_Verify(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)

	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	att := NewProximityAttestation(priv, "peer123", claimerKey, targetHash, big.NewInt(100))

	// Valid signature should verify.
	if !att.Verify() {
		t.Error("valid attestation should verify")
	}

	// Tampered data should not verify.
	attCopy := *att
	attCopy.ClaimerPubKey[0] ^= 0xFF
	if attCopy.Verify() {
		t.Error("tampered attestation should not verify")
	}
}

func TestProximityAttestation_IsExpired(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)

	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	att := NewProximityAttestation(priv, "peer123", claimerKey, targetHash, big.NewInt(100))

	// Fresh attestation should not be expired.
	if att.IsExpired() {
		t.Error("fresh attestation should not be expired")
	}

	// Attestation with old timestamp should be expired.
	att.Timestamp = time.Now().Add(-ProximityProofTTL - time.Minute).Unix()
	if !att.IsExpired() {
		t.Error("old attestation should be expired")
	}
}

func TestDHTProximityProof_Create(t *testing.T) {
	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	proof := NewDHTProximityProof(claimerKey, "claimer-peer", targetHash, 50)

	if proof == nil {
		t.Fatal("expected proof, got nil")
	}

	if proof.ClaimerPubKey != claimerKey {
		t.Error("claimer key mismatch")
	}

	if proof.ClaimerPeerID != "claimer-peer" {
		t.Error("claimer peer ID mismatch")
	}

	if proof.TargetHash != targetHash {
		t.Error("target hash mismatch")
	}

	if proof.RoutingTableSize != 50 {
		t.Errorf("expected routing table size 50, got %d", proof.RoutingTableSize)
	}

	if len(proof.Attestations) != 0 {
		t.Error("new proof should have no attestations")
	}
}

func TestDHTProximityProof_AddAttestation(t *testing.T) {
	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	proof := NewDHTProximityProof(claimerKey, "claimer-peer", targetHash, 50)

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	att := NewProximityAttestation(priv, "attester-peer", claimerKey, targetHash, big.NewInt(100))

	proof.AddAttestation(*att)

	if len(proof.Attestations) != 1 {
		t.Errorf("expected 1 attestation, got %d", len(proof.Attestations))
	}
}

func TestDHTProximityProof_Verify(t *testing.T) {
	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	proof := NewDHTProximityProof(claimerKey, "claimer-peer", targetHash, 50)

	// Create an attestation with very small XOR distance (close peer).
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	// Small distance = peer is very close to target.
	att := NewProximityAttestation(priv, "attester-peer", claimerKey, targetHash, big.NewInt(1))
	proof.AddAttestation(*att)

	// Should verify for 3 hops.
	if !proof.Verify(targetHash, 3) {
		t.Error("proof with close attester should verify")
	}

	// Wrong target should fail.
	var wrongTarget [32]byte
	rand.Read(wrongTarget[:])
	if proof.Verify(wrongTarget, 3) {
		t.Error("proof should not verify for wrong target")
	}
}

func TestDHTProximityProof_VerifyNoAttestations(t *testing.T) {
	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	proof := NewDHTProximityProof(claimerKey, "claimer-peer", targetHash, 50)

	if proof.Verify(targetHash, 3) {
		t.Error("proof without attestations should not verify")
	}
}

func TestDHTProximityProof_VerifySelfAttestation(t *testing.T) {
	// Generate a keypair for the claimer.
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	var claimerKey [32]byte
	copy(claimerKey[:], pub)

	var targetHash [32]byte
	rand.Read(targetHash[:])

	proof := NewDHTProximityProof(claimerKey, "claimer-peer", targetHash, 50)

	// Create self-attestation (claimer attests their own proximity).
	// Using the same private key means AttesterPubKey == ClaimerPubKey.
	att := NewProximityAttestation(priv, "claimer-peer", claimerKey, targetHash, big.NewInt(1))
	proof.AddAttestation(*att)

	// Self-attestations should be rejected.
	if proof.Verify(targetHash, 3) {
		t.Error("proof with only self-attestation should not verify")
	}
}

func TestDHTProximityProof_VerifyExpiredAttestation(t *testing.T) {
	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	proof := NewDHTProximityProof(claimerKey, "claimer-peer", targetHash, 50)

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	att := NewProximityAttestation(priv, "attester-peer", claimerKey, targetHash, big.NewInt(1))
	att.Timestamp = time.Now().Add(-ProximityProofTTL - time.Minute).Unix()
	proof.AddAttestation(*att)

	// Expired attestations should be ignored.
	if proof.Verify(targetHash, 3) {
		t.Error("proof with only expired attestation should not verify")
	}
}

func TestComputeXORDistance(t *testing.T) {
	// Same hashes should have zero distance.
	var a, b [32]byte
	copy(a[:], []byte("test-hash-12345678901234567890"))
	copy(b[:], a[:])

	dist := ComputeXORDistance(a, b)
	if dist.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("same hashes should have zero distance, got %s", dist.String())
	}

	// Different hashes should have non-zero distance.
	b[0] = a[0] ^ 0xFF
	dist = ComputeXORDistance(a, b)
	if dist.Cmp(big.NewInt(0)) == 0 {
		t.Error("different hashes should have non-zero distance")
	}
}

func TestCalculateXORThreshold(t *testing.T) {
	// Test threshold behavior: more hops = larger threshold (further reach).
	thresh1 := calculateXORThreshold(1)
	thresh2 := calculateXORThreshold(2)
	thresh3 := calculateXORThreshold(3)

	// More hops = larger threshold (can accept more distant attesters).
	if thresh1.Cmp(thresh2) >= 0 {
		t.Error("threshold for 1 hop should be less than 2 hops")
	}
	if thresh2.Cmp(thresh3) >= 0 {
		t.Error("threshold for 2 hops should be less than 3 hops")
	}

	// All thresholds should be positive.
	if thresh1.Sign() <= 0 || thresh2.Sign() <= 0 || thresh3.Sign() <= 0 {
		t.Error("thresholds should be positive")
	}
}

func TestPeerIDToHash(t *testing.T) {
	hash1 := PeerIDToHash("peer1")
	hash2 := PeerIDToHash("peer2")
	hash3 := PeerIDToHash("peer1")

	// Same peer ID should produce same hash.
	if hash1 != hash3 {
		t.Error("same peer ID should produce same hash")
	}

	// Different peer IDs should produce different hashes.
	if hash1 == hash2 {
		t.Error("different peer IDs should produce different hashes")
	}

	// Hash should not be all zeros.
	var zero [32]byte
	if hash1 == zero {
		t.Error("hash should not be all zeros")
	}
}

func TestProximityVerifier_Create(t *testing.T) {
	var localPubKey [32]byte
	rand.Read(localPubKey[:])

	routingFunc := func() []string {
		return []string{"peer1", "peer2", "peer3"}
	}

	verifier := NewProximityVerifier("local-peer", localPubKey, routingFunc)

	if verifier == nil {
		t.Fatal("expected verifier, got nil")
	}

	if verifier.localPeerID != "local-peer" {
		t.Error("local peer ID mismatch")
	}
}

func TestProximityVerifier_CreateAttestation(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	var localPubKey [32]byte
	copy(localPubKey[:], priv.Public().(ed25519.PublicKey))

	verifier := NewProximityVerifier("local-peer", localPubKey, nil)

	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	att := verifier.CreateAttestation(priv, claimerKey, targetHash)

	if att == nil {
		t.Fatal("expected attestation, got nil")
	}

	if !att.Verify() {
		t.Error("attestation should verify")
	}

	if att.ClaimerPubKey != claimerKey {
		t.Error("claimer key mismatch")
	}
}

func TestProximityVerifier_IsNearTarget(t *testing.T) {
	var localPubKey [32]byte
	rand.Read(localPubKey[:])

	verifier := NewProximityVerifier("local-peer", localPubKey, nil)

	// Test with peer whose hash is close to target.
	peerID := "test-peer-12345"
	peerHash := PeerIDToHash(peerID)

	// Use the peer's hash as target - should be 0 distance.
	if !verifier.IsNearTarget(peerID, peerHash, 3) {
		t.Error("peer should be near itself")
	}

	// Completely different target should likely be far.
	var farTarget [32]byte
	farTarget[0] = 0xFF
	farTarget[1] = 0xFF

	// This might still be close depending on hash, so just check it doesn't panic.
	_ = verifier.IsNearTarget(peerID, farTarget, 3)
}

func TestProximityVerifier_GetNearbyPeers(t *testing.T) {
	var localPubKey [32]byte
	rand.Read(localPubKey[:])

	peers := []string{"peer1", "peer2", "peer3", "peer4", "peer5"}
	routingFunc := func() []string {
		return peers
	}

	verifier := NewProximityVerifier("local-peer", localPubKey, routingFunc)

	// Use a target that one of the peers is close to.
	targetHash := PeerIDToHash("peer3")

	nearby := verifier.GetNearbyPeers(targetHash, 3)

	// At least peer3 should be in the list (0 distance from itself).
	found := false
	for _, p := range nearby {
		if p == "peer3" {
			found = true
			break
		}
	}
	if !found {
		t.Error("peer3 should be in nearby list when target is peer3's hash")
	}
}

func TestProximityVerifier_VerifyProof(t *testing.T) {
	var localPubKey [32]byte
	rand.Read(localPubKey[:])

	verifier := NewProximityVerifier("local-peer", localPubKey, nil)

	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	proof := NewDHTProximityProof(claimerKey, "claimer-peer", targetHash, 50)

	// Without attestations should fail.
	err := verifier.VerifyProof(proof, 3)
	if err != ErrNoAttestations {
		t.Errorf("expected ErrNoAttestations, got %v", err)
	}

	// Add valid attestation.
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	att := NewProximityAttestation(priv, "attester-peer", claimerKey, targetHash, big.NewInt(1))
	proof.AddAttestation(*att)

	// Should verify now.
	err = verifier.VerifyProof(proof, 3)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestDHTProximityProof_ToLegacyProof(t *testing.T) {
	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	proof := NewDHTProximityProof(claimerKey, "claimer-peer", targetHash, 50)

	// Add attestation.
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	att := NewProximityAttestation(priv, "attester-peer", claimerKey, targetHash, big.NewInt(1))
	proof.AddAttestation(*att)

	legacy := proof.ToLegacyProof()

	if legacy.ClaimerPeerID != "claimer-peer" {
		t.Error("claimer peer ID mismatch")
	}

	if len(legacy.ConnectedPeers) != 1 {
		t.Errorf("expected 1 connected peer, got %d", len(legacy.ConnectedPeers))
	}

	if legacy.ConnectedPeers[0] != "attester-peer" {
		t.Errorf("expected attester-peer, got %s", legacy.ConnectedPeers[0])
	}

	// Legacy proof should also verify.
	if !legacy.Verify(targetHash, 3) {
		t.Error("legacy proof should verify")
	}
}

func TestEstimateHopsFromXOR(t *testing.T) {
	// Zero distance = same node = 0 hops.
	hops := estimateHopsFromXOR(big.NewInt(0))
	if hops != 0 {
		t.Errorf("zero distance should be 0 hops, got %d", hops)
	}

	// Small distance should be few hops.
	hopsSmall := estimateHopsFromXOR(big.NewInt(1))
	if hopsSmall != 0 {
		t.Errorf("distance of 1 should be 0 hops, got %d", hopsSmall)
	}

	// Large distance should be more hops.
	largeDistance := new(big.Int).Lsh(big.NewInt(1), 200) // 2^200.
	hopsLarge := estimateHopsFromXOR(largeDistance)

	// Larger distance = more bits = more hops.
	if hopsSmall > hopsLarge {
		t.Errorf("larger XOR distance should result in more hops, small=%d large=%d", hopsSmall, hopsLarge)
	}
}

func TestCreateDHTProofFromRoutingTable(t *testing.T) {
	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	// Create attesters.
	attesters := make(map[string]ed25519.PrivateKey)
	routingTable := []string{}

	for i := 0; i < 10; i++ {
		peerID := "peer-" + string(rune('A'+i))
		_, priv, _ := ed25519.GenerateKey(rand.Reader)
		attesters[peerID] = priv
		routingTable = append(routingTable, peerID)
	}

	// Use a target hash from one of the peers.
	targetHash := PeerIDToHash("peer-A")

	proof := CreateDHTProofFromRoutingTable(
		claimerKey,
		"claimer-peer",
		targetHash,
		routingTable,
		attesters,
	)

	if proof == nil {
		t.Fatal("expected proof, got nil")
	}

	// Should have at least one attestation (peer-A is at distance 0).
	if len(proof.Attestations) == 0 {
		t.Error("expected at least one attestation")
	}

	// Proof should verify.
	if !proof.Verify(targetHash, 3) {
		t.Error("proof should verify")
	}
}

func BenchmarkComputeXORDistance(b *testing.B) {
	var a, c [32]byte
	rand.Read(a[:])
	rand.Read(c[:])

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ComputeXORDistance(a, c)
	}
}

func BenchmarkProximityAttestation_Verify(b *testing.B) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)

	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	att := NewProximityAttestation(priv, "peer123", claimerKey, targetHash, big.NewInt(100))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		att.Verify()
	}
}

func BenchmarkDHTProximityProof_Verify(b *testing.B) {
	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	var targetHash [32]byte
	rand.Read(targetHash[:])

	proof := NewDHTProximityProof(claimerKey, "claimer-peer", targetHash, 50)

	// Add 5 attestations.
	for i := 0; i < 5; i++ {
		_, priv, _ := ed25519.GenerateKey(rand.Reader)
		att := NewProximityAttestation(priv, "attester-"+string(rune('A'+i)), claimerKey, targetHash, big.NewInt(int64(i+1)))
		proof.AddAttestation(*att)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proof.Verify(targetHash, 3)
	}
}
