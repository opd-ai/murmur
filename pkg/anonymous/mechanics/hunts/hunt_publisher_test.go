package hunts

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"math/big"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

func TestNewHuntPublisher(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	_ = pub

	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	if hp == nil {
		t.Fatal("expected HuntPublisher, got nil")
	}
	if hp.topic != mechanics.TopicAnonymousMechanics {
		t.Errorf("expected topic %s, got %s", mechanics.TopicAnonymousMechanics, hp.topic)
	}
}

func TestHuntPublisher_PublishHuntCreated(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, err := NewHunt("Test Hunt", seed, initiatorKey, HuntDuration30Min, 5, 100)
	if err != nil {
		t.Fatalf("failed to create hunt: %v", err)
	}

	err = hp.PublishHuntCreated(context.Background(), hunt)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Errorf("expected 1 publish call, got %d", len(mockPub.Published))
	}
}

func TestHuntPublisher_PublishFragmentClaim(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	var huntID, claimerKey [32]byte
	rand.Read(huntID[:])
	rand.Read(claimerKey[:])

	err := hp.PublishFragmentClaim(context.Background(), huntID, 0, claimerKey, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Errorf("expected 1 publish call, got %d", len(mockPub.Published))
	}
}

func TestHuntPublisher_PublishClueReveal(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	var huntID [32]byte
	rand.Read(huntID[:])

	err := hp.PublishClueReveal(context.Background(), huntID, 0, 1, "Test clue")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Errorf("expected 1 publish call, got %d", len(mockPub.Published))
	}
}

func TestHuntPublisher_PublishHuntCompleted(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiatorKey, HuntDuration30Min, 5, 100)

	err := hp.PublishHuntCompleted(context.Background(), hunt)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Errorf("expected 1 publish call, got %d", len(mockPub.Published))
	}
}

func TestHuntPublisher_PublishHuntExpired(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	var huntID [32]byte
	rand.Read(huntID[:])

	err := hp.PublishHuntExpired(context.Background(), huntID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Errorf("expected 1 publish call, got %d", len(mockPub.Published))
	}
}

func TestHuntPublisher_NoPublisher(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	hp := NewHuntPublisher(nil, priv)

	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiatorKey, HuntDuration30Min, 5, 100)

	err := hp.PublishHuntCreated(context.Background(), hunt)
	if err != mechanics.ErrPublisherNotSet {
		t.Errorf("expected mechanics.ErrPublisherNotSet, got %v", err)
	}
}

func TestHuntPublisher_NoPrivateKey(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, nil)

	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiatorKey, HuntDuration30Min, 5, 100)

	err := hp.PublishHuntCreated(context.Background(), hunt)
	if err != mechanics.ErrMissingPrivateKey {
		t.Errorf("expected mechanics.ErrMissingPrivateKey, got %v", err)
	}
}

func TestNewHuntReceiver(t *testing.T) {
	store := NewHuntStore()
	receiver := NewHuntReceiver(store)

	if receiver == nil {
		t.Fatal("expected HuntReceiver, got nil")
	}
	if receiver.huntStore != store {
		t.Error("hunt store not set correctly")
	}
}

func TestHuntReceiver_HandleHuntCreated(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	store := NewHuntStore()
	receiver := NewHuntReceiver(store)

	// Create and publish a hunt.
	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiatorKey, HuntDuration30Min, 5, 100)
	hp.PublishHuntCreated(context.Background(), hunt)

	// Receive the message.
	err := receiver.HandleMessage(mockPub.Published[len(mockPub.Published)-1].Data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify hunt was added.
	storedHunt := store.GetHunt(hunt.ID)
	if storedHunt == nil {
		t.Error("hunt was not added to store")
	}
}

func TestHuntReceiver_HandleFragmentClaim(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	store := NewHuntStore()
	receiver := NewHuntReceiver(store)

	// Create and add a hunt.
	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiatorKey, HuntDuration30Min, 5, 100)
	store.AddHunt(hunt)

	// Create a proximity proof with valid attestation.
	// Use a small XOR distance to ensure it's within mechanics.HuntClaimProximityHops (3).
	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	_, attesterPriv, _ := ed25519.GenerateKey(rand.Reader)
	// Small XOR distance = attester is close to target location = few hops.
	smallDistance := big.NewInt(1) // BitLen()=1, hops = 1/42 = 0
	att := mechanics.NewProximityAttestation(
		attesterPriv,
		"attester-peer",
		claimerKey,
		hunt.Fragments[0].LocationHash,
		smallDistance,
	)

	proof := mechanics.NewDHTProximityProof(claimerKey, "claimer-peer", hunt.Fragments[0].LocationHash, 50)
	proof.AddAttestation(*att)

	// Publish claim.
	hp.PublishFragmentClaim(context.Background(), hunt.ID, 0, claimerKey, proof)

	// Receive the message.
	err := receiver.HandleMessage(mockPub.Published[len(mockPub.Published)-1].Data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify fragment was claimed.
	if !hunt.Fragments[0].Claimed {
		t.Error("fragment was not claimed")
	}
}

func TestHuntReceiver_HandleClueReveal(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	store := NewHuntStore()
	receiver := NewHuntReceiver(store)

	// Create and add a hunt.
	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiatorKey, HuntDuration30Min, 5, 100)
	store.AddHunt(hunt)

	// Publish clue reveal.
	hp.PublishClueReveal(context.Background(), hunt.ID, 0, 1, "New clue")

	// Receive the message.
	err := receiver.HandleMessage(mockPub.Published[len(mockPub.Published)-1].Data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHuntReceiver_HandleHuntCompleted(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	store := NewHuntStore()
	receiver := NewHuntReceiver(store)

	// Create and add a hunt.
	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiatorKey, HuntDuration30Min, 5, 100)
	store.AddHunt(hunt)

	// Publish completion.
	hp.PublishHuntCompleted(context.Background(), hunt)

	// Receive the message.
	err := receiver.HandleMessage(mockPub.Published[len(mockPub.Published)-1].Data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify state changed.
	if hunt.State != HuntCompleted {
		t.Errorf("expected HuntCompleted state, got %v", hunt.State)
	}
}

func TestHuntReceiver_HandleHuntExpired(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	store := NewHuntStore()
	receiver := NewHuntReceiver(store)

	// Create and add a hunt.
	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiatorKey, HuntDuration30Min, 5, 100)
	store.AddHunt(hunt)

	// Publish expiration.
	hp.PublishHuntExpired(context.Background(), hunt.ID)

	// Receive the message.
	err := receiver.HandleMessage(mockPub.Published[len(mockPub.Published)-1].Data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify state changed.
	if hunt.State != HuntExpired {
		t.Errorf("expected HuntExpired state, got %v", hunt.State)
	}
}

func TestHuntReceiver_InvalidSignature(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	store := NewHuntStore()
	receiver := NewHuntReceiver(store)

	// Create and publish a hunt.
	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiatorKey, HuntDuration30Min, 5, 100)
	hp.PublishHuntCreated(context.Background(), hunt)

	// Corrupt the signature.
	data := mockPub.Published[len(mockPub.Published)-1].Data
	if len(data) > 10 {
		data[len(data)-5] ^= 0xFF
	}

	// Should fail verification.
	err := receiver.HandleMessage(data)
	if err == nil {
		t.Error("expected error for corrupted data")
	}
}

func TestHuntReceiver_HuntNotFound(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	store := NewHuntStore()
	receiver := NewHuntReceiver(store)

	var huntID [32]byte
	rand.Read(huntID[:])

	// Publish expiration for non-existent hunt.
	hp.PublishHuntExpired(context.Background(), huntID)

	// Should fail because hunt doesn't exist.
	err := receiver.HandleMessage(mockPub.Published[len(mockPub.Published)-1].Data)
	if err != ErrHuntNotFound {
		t.Errorf("expected ErrHuntNotFound, got %v", err)
	}
}

func TestHuntToNetworkProtoRoundTrip(t *testing.T) {
	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiatorKey, HuntDuration60Min, 10, 100)

	// Convert to proto and back.
	pbHunt := huntToNetworkProto(hunt)
	restored := networkProtoToHunt(pbHunt)

	if restored.ID != hunt.ID {
		t.Error("ID mismatch")
	}
	if restored.Theme != hunt.Theme {
		t.Error("Theme mismatch")
	}
	if restored.Seed != hunt.Seed {
		t.Error("Seed mismatch")
	}
	if restored.InitiatorKey != hunt.InitiatorKey {
		t.Error("InitiatorKey mismatch")
	}
	if len(restored.Fragments) != len(hunt.Fragments) {
		t.Errorf("Fragment count mismatch: got %d, want %d", len(restored.Fragments), len(hunt.Fragments))
	}
}

func TestFragmentToNetworkProtoRoundTrip(t *testing.T) {
	var locationHash [32]byte
	rand.Read(locationHash[:])

	claimedAt := time.Now()
	var claimerKey [32]byte
	rand.Read(claimerKey[:])

	frag := &Fragment{
		Index:         5,
		LocationHash:  locationHash,
		TargetPeerID:  "peer-123",
		Claimed:       true,
		ClaimerKey:    &claimerKey,
		ClaimedAt:     &claimedAt,
		Clues:         []string{"Clue 1", "Clue 2"},
		CluesRevealed: 1,
	}

	// Convert to proto and back.
	pbFrag := fragmentToNetworkProto(frag)
	restored := networkProtoToFragment(pbFrag)

	if restored.Index != frag.Index {
		t.Error("Index mismatch")
	}
	if restored.LocationHash != frag.LocationHash {
		t.Error("LocationHash mismatch")
	}
	if restored.TargetPeerID != frag.TargetPeerID {
		t.Error("TargetPeerID mismatch")
	}
	if restored.Claimed != frag.Claimed {
		t.Error("Claimed mismatch")
	}
	if *restored.ClaimerKey != *frag.ClaimerKey {
		t.Error("ClaimerKey mismatch")
	}
	if len(restored.Clues) != len(frag.Clues) {
		t.Error("Clues length mismatch")
	}
}

func TestProximityProofRoundTrip(t *testing.T) {
	var claimerKey, targetHash [32]byte
	rand.Read(claimerKey[:])
	rand.Read(targetHash[:])

	proof := mechanics.NewDHTProximityProof(claimerKey, "claimer-peer", targetHash, 50)

	// Add attestation.
	_, attesterPriv, _ := ed25519.GenerateKey(rand.Reader)
	att := mechanics.NewProximityAttestation(
		attesterPriv,
		"attester-peer",
		claimerKey,
		targetHash,
		mechanics.ComputeXORDistance(mechanics.PeerIDToHash("attester-peer"), targetHash),
	)
	proof.AddAttestation(*att)

	// Convert to proto and back.
	pbProof := proximityProofToProto(proof)
	restored := protoToProximityProof(pbProof, claimerKey)

	if restored.ClaimerPubKey != proof.ClaimerPubKey {
		t.Error("ClaimerPubKey mismatch")
	}
	if restored.ClaimerPeerID != proof.ClaimerPeerID {
		t.Error("ClaimerPeerID mismatch")
	}
	if restored.TargetHash != proof.TargetHash {
		t.Error("TargetHash mismatch")
	}
	if len(restored.Attestations) != len(proof.Attestations) {
		t.Errorf("Attestation count mismatch: got %d, want %d", len(restored.Attestations), len(proof.Attestations))
	}
}

func BenchmarkHuntPublisher_PublishHuntCreated(b *testing.B) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, _ := NewHunt("Benchmark Hunt", seed, initiatorKey, HuntDuration30Min, 5, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hp.PublishHuntCreated(context.Background(), hunt)
	}
}

func BenchmarkHuntReceiver_HandleMessage(b *testing.B) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mechanics.MockPublisher{}
	hp := NewHuntPublisher(mockPub, priv)

	store := NewHuntStore()
	receiver := NewHuntReceiver(store)

	var seed, initiatorKey [32]byte
	rand.Read(seed[:])
	rand.Read(initiatorKey[:])

	hunt, _ := NewHunt("Benchmark Hunt", seed, initiatorKey, HuntDuration30Min, 5, 100)
	hp.PublishHuntCreated(context.Background(), hunt)
	data := mockPub.Published[len(mockPub.Published)-1].Data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		receiver.HandleMessage(data)
	}
}
