// Package mechanics - Forge publisher tests.
// Per ROADMAP.md line 474, tests for forge events, entries, votes.
package mechanics

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

// TestNewForgePublisher tests ForgePublisher creation.
func TestNewForgePublisher(t *testing.T) {
	pub := NewForgePublisher(nil, nil)
	if pub == nil {
		t.Fatal("expected non-nil publisher")
	}
	if pub.topic != TopicAnonymousMechanics {
		t.Errorf("expected topic %s, got %s", TopicAnonymousMechanics, pub.topic)
	}
}

// TestForgePublisher_PublishForgeCreated tests forge creation publishing.
func TestForgePublisher_PublishForgeCreated(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewForgePublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	forge, err := NewSigilForge(ForgeMicroFiction, "Write a story about hope", initiatorKey, ForgeDuration30Min, ForgeMinResonance)
	if err != nil {
		t.Fatalf("failed to create forge: %v", err)
	}

	ctx := context.Background()
	err = pub.PublishForgeCreated(ctx, forge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.published[0].data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetForgeEvent()
	if event == nil {
		t.Fatal("expected forge event")
	}
	if event.EventType != pb.ForgeEventType_FORGE_EVENT_CREATED {
		t.Errorf("expected CREATED event type, got %v", event.EventType)
	}
	if event.Project == nil {
		t.Fatal("expected project in event")
	}
}

// TestForgePublisher_PublishForgeCreated_NoPublisher tests without publisher.
func TestForgePublisher_PublishForgeCreated_NoPublisher(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewForgePublisher(nil, privKey)

	var initiatorKey [32]byte
	forge, _ := NewSigilForge(ForgeSigilArt, "Test", initiatorKey, ForgeDuration30Min, ForgeMinResonance)

	err := pub.PublishForgeCreated(context.Background(), forge)
	if err != ErrPublisherNotSet {
		t.Errorf("expected ErrPublisherNotSet, got %v", err)
	}
}

// TestForgePublisher_PublishForgeCreated_NilForge tests with nil forge.
func TestForgePublisher_PublishForgeCreated_NilForge(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewForgePublisher(mockPub, privKey)

	err := pub.PublishForgeCreated(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil forge")
	}
}

// TestForgePublisher_PublishForgeCreated_NoPrivateKey tests without private key.
func TestForgePublisher_PublishForgeCreated_NoPrivateKey(t *testing.T) {
	mockPub := &mockPublisher{}
	pub := NewForgePublisher(mockPub, nil)

	var initiatorKey [32]byte
	forge, _ := NewSigilForge(ForgeSigilArt, "Test", initiatorKey, ForgeDuration30Min, ForgeMinResonance)

	err := pub.PublishForgeCreated(context.Background(), forge)
	if err != ErrMissingPrivateKey {
		t.Errorf("expected ErrMissingPrivateKey, got %v", err)
	}
}

// TestForgePublisher_PublishEntry tests entry submission publishing.
func TestForgePublisher_PublishEntry(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewForgePublisher(mockPub, privKey)

	var forgeID [32]byte
	forgeID[0] = 1

	var specterKey [32]byte
	copy(specterKey[:], privKey.Public().(ed25519.PublicKey))

	entry := &ForgeEntry{
		ID:           [32]byte{2},
		ForgeID:      forgeID,
		SpecterKey:   specterKey,
		Content:      []byte("A tale of two cities..."),
		SubmittedAt:  time.Now(),
		amplifierSet: make(map[string]bool),
	}

	ctx := context.Background()
	err := pub.PublishEntry(ctx, forgeID, entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.published[0].data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetForgeEvent()
	if event == nil {
		t.Fatal("expected forge event")
	}
	if event.EventType != pb.ForgeEventType_FORGE_EVENT_CONTRIBUTION {
		t.Errorf("expected CONTRIBUTION event type, got %v", event.EventType)
	}
	if event.Contribution == nil {
		t.Fatal("expected contribution in event")
	}
}

// TestForgePublisher_PublishEntry_NilEntry tests with nil entry.
func TestForgePublisher_PublishEntry_NilEntry(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewForgePublisher(mockPub, privKey)

	var forgeID [32]byte
	err := pub.PublishEntry(context.Background(), forgeID, nil)
	if err == nil {
		t.Error("expected error for nil entry")
	}
}

// TestForgePublisher_PublishAmplification tests amplification publishing.
func TestForgePublisher_PublishAmplification(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewForgePublisher(mockPub, privKey)

	var forgeID, entryID, amplifierKey [32]byte
	forgeID[0] = 1
	entryID[0] = 2
	copy(amplifierKey[:], privKey.Public().(ed25519.PublicKey))

	ctx := context.Background()
	err := pub.PublishAmplification(ctx, forgeID, entryID, amplifierKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.published[0].data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetForgeEvent()
	if event == nil {
		t.Fatal("expected forge event")
	}
	if event.EventType != pb.ForgeEventType_FORGE_EVENT_CONTRIBUTION {
		t.Errorf("expected CONTRIBUTION event type, got %v", event.EventType)
	}
}

// TestForgePublisher_PublishForgeFinalized tests finalization publishing.
func TestForgePublisher_PublishForgeFinalized(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewForgePublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	forge, _ := NewSigilForge(ForgeSigilArt, "Test prompt", initiatorKey, ForgeDuration30Min, ForgeMinResonance)
	winningSigil := []byte("winning sigil data")

	ctx := context.Background()
	err := pub.PublishForgeFinalized(ctx, forge, winningSigil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.published[0].data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetForgeEvent()
	if event == nil {
		t.Fatal("expected forge event")
	}
	if event.EventType != pb.ForgeEventType_FORGE_EVENT_FINALIZED {
		t.Errorf("expected FINALIZED event type, got %v", event.EventType)
	}
	if string(event.FinalSigil) != "winning sigil data" {
		t.Errorf("expected winning sigil data, got %s", event.FinalSigil)
	}
}

// TestForgePublisher_PublishForgeFinalized_NilForge tests with nil forge.
func TestForgePublisher_PublishForgeFinalized_NilForge(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewForgePublisher(mockPub, privKey)

	err := pub.PublishForgeFinalized(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error for nil forge")
	}
}

// TestForgePublisher_PublishForgeFailed tests failure publishing.
func TestForgePublisher_PublishForgeFailed(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewForgePublisher(mockPub, privKey)

	var forgeID [32]byte
	forgeID[0] = 1

	ctx := context.Background()
	err := pub.PublishForgeFailed(ctx, forgeID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.published))
	}

	// Verify message content.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.published[0].data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetForgeEvent()
	if event == nil {
		t.Fatal("expected forge event")
	}
	if event.EventType != pb.ForgeEventType_FORGE_EVENT_FAILED {
		t.Errorf("expected FAILED event type, got %v", event.EventType)
	}
}

// TestNewForgeReceiver tests ForgeReceiver creation.
func TestNewForgeReceiver(t *testing.T) {
	store := NewForgeStore()
	receiver := NewForgeReceiver(store)
	if receiver == nil {
		t.Fatal("expected non-nil receiver")
	}
	if receiver.forgeStore != store {
		t.Error("store not set correctly")
	}
}

// TestForgeReceiver_HandleForgeCreated tests forge creation handling.
func TestForgeReceiver_HandleForgeCreated(t *testing.T) {
	store := NewForgeStore()
	receiver := NewForgeReceiver(store)

	// Create and publish a forge.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewForgePublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	forge, _ := NewSigilForge(ForgeMicroFiction, "Test prompt", initiatorKey, ForgeDuration30Min, ForgeMinResonance)

	err := publisher.PublishForgeCreated(context.Background(), forge)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify forge was stored.
	stored := store.GetForge(forge.ID)
	if stored == nil {
		t.Fatal("expected forge to be stored")
	}
}

// TestForgeReceiver_HandleContribution tests entry handling.
func TestForgeReceiver_HandleContribution(t *testing.T) {
	store := NewForgeStore()
	receiver := NewForgeReceiver(store)

	// Create and store a forge first.
	_, forgePrivKey, _ := ed25519.GenerateKey(rand.Reader)
	var initiatorKey [32]byte
	copy(initiatorKey[:], forgePrivKey.Public().(ed25519.PublicKey))

	forge, _ := NewSigilForge(ForgeMicroFiction, "Test", initiatorKey, ForgeDuration30Min, ForgeMinResonance)
	store.AddForge(forge)

	// Create entry submitter keys (separate from forge initiator).
	_, entryPrivKey, _ := ed25519.GenerateKey(rand.Reader)
	var specterKey [32]byte
	copy(specterKey[:], entryPrivKey.Public().(ed25519.PublicKey))

	// Publish entry using entry submitter's key.
	mockPub := &mockPublisher{}
	publisher := NewForgePublisher(mockPub, entryPrivKey)

	entry := &ForgeEntry{
		ForgeID:      forge.ID,
		SpecterKey:   specterKey,
		Content:      []byte("My story entry"),
		SubmittedAt:  time.Now(),
		amplifierSet: make(map[string]bool),
	}
	entry.ID = computeEntryID(specterKey, entry.Content)

	err := publisher.PublishEntry(context.Background(), forge.ID, entry)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify entry was added.
	stored := store.GetForge(forge.ID)
	if stored == nil {
		t.Fatal("expected forge to be stored")
	}
	if len(stored.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(stored.Entries))
	}
}

// TestForgeReceiver_HandleForgeFinalized tests finalization handling.
func TestForgeReceiver_HandleForgeFinalized(t *testing.T) {
	store := NewForgeStore()
	receiver := NewForgeReceiver(store)

	// Create and store a forge.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	forge, _ := NewSigilForge(ForgeSigilArt, "Test", initiatorKey, ForgeDuration30Min, ForgeMinResonance)
	store.AddForge(forge)

	// Publish finalization.
	mockPub := &mockPublisher{}
	publisher := NewForgePublisher(mockPub, privKey)

	err := publisher.PublishForgeFinalized(context.Background(), forge, []byte("winner"))
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify forge state changed.
	stored := store.GetForge(forge.ID)
	if stored.State != ForgeCompleted {
		t.Errorf("expected state ForgeCompleted, got %v", stored.State)
	}
}

// TestForgeReceiver_HandleForgeFailed tests failure handling.
func TestForgeReceiver_HandleForgeFailed(t *testing.T) {
	store := NewForgeStore()
	receiver := NewForgeReceiver(store)

	// Create and store a forge.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	forge, _ := NewSigilForge(ForgeSigilArt, "Test", initiatorKey, ForgeDuration30Min, ForgeMinResonance)
	store.AddForge(forge)

	// Publish failure.
	mockPub := &mockPublisher{}
	publisher := NewForgePublisher(mockPub, privKey)

	err := publisher.PublishForgeFailed(context.Background(), forge.ID)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify forge state changed.
	stored := store.GetForge(forge.ID)
	if stored.State != ForgeExpired {
		t.Errorf("expected state ForgeExpired, got %v", stored.State)
	}
}

// TestForgeReceiver_HandleMessage_InvalidData tests with invalid data.
func TestForgeReceiver_HandleMessage_InvalidData(t *testing.T) {
	store := NewForgeStore()
	receiver := NewForgeReceiver(store)

	err := receiver.HandleMessage([]byte("invalid"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

// TestForgeReceiver_HandleMessage_NonForgeEvent tests with non-forge event.
func TestForgeReceiver_HandleMessage_NonForgeEvent(t *testing.T) {
	store := NewForgeStore()
	receiver := NewForgeReceiver(store)

	// Create a gossip message without a forge event.
	gossipMsg := &pb.GossipMessage{}
	data, _ := proto.Marshal(gossipMsg)

	err := receiver.HandleMessage(data)
	if err != nil {
		t.Errorf("expected nil for non-forge event, got %v", err)
	}
}

// TestForgeReceiver_HandleMessage_MissingSignature tests with missing signature.
func TestForgeReceiver_HandleMessage_MissingSignature(t *testing.T) {
	store := NewForgeStore()
	receiver := NewForgeReceiver(store)

	// Create an unsigned forge event.
	event := &pb.ForgeEvent{
		EventType: pb.ForgeEventType_FORGE_EVENT_CREATED,
		ProjectId: []byte("test"),
	}
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_ForgeEvent{
			ForgeEvent: event,
		},
	}
	data, _ := proto.Marshal(gossipMsg)

	err := receiver.HandleMessage(data)
	if err != ErrMissingSignature {
		t.Errorf("expected ErrMissingSignature, got %v", err)
	}
}

// TestForgeEventSignatureRoundTrip tests signature verification.
func TestForgeEventSignatureRoundTrip(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewForgePublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	forge, _ := NewSigilForge(ForgeMicroFiction, "Test", initiatorKey, ForgeDuration30Min, ForgeMinResonance)

	err := publisher.PublishForgeCreated(context.Background(), forge)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Unmarshal and verify signature.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.published[0].data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetForgeEvent()
	if event == nil {
		t.Fatal("expected forge event")
	}
	if len(event.Signature) == 0 {
		t.Fatal("expected signature")
	}

	// Verify the signature through receiver.
	store := NewForgeStore()
	receiver := NewForgeReceiver(store)
	err = receiver.verifyEventSignature(event)
	if err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
}

// BenchmarkForgePublisher_PublishForgeCreated benchmarks forge creation publishing.
func BenchmarkForgePublisher_PublishForgeCreated(b *testing.B) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewForgePublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	forge, _ := NewSigilForge(ForgeMicroFiction, "Benchmark prompt", initiatorKey, ForgeDuration30Min, ForgeMinResonance)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockPub.published = nil
		pub.PublishForgeCreated(ctx, forge)
	}
}

// BenchmarkForgeReceiver_HandleMessage benchmarks message handling.
func BenchmarkForgeReceiver_HandleMessage(b *testing.B) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewForgePublisher(mockPub, privKey)

	var initiatorKey [32]byte
	copy(initiatorKey[:], privKey.Public().(ed25519.PublicKey))

	forge, _ := NewSigilForge(ForgeMicroFiction, "Benchmark", initiatorKey, ForgeDuration30Min, ForgeMinResonance)

	publisher.PublishForgeCreated(context.Background(), forge)
	data := mockPub.published[0].data

	store := NewForgeStore()
	receiver := NewForgeReceiver(store)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		receiver.HandleMessage(data)
	}
}
