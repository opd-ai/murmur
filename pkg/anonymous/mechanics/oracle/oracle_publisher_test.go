// Package mechanics - Oracle publisher tests.
// Per ROADMAP.md line 459, tests for pool creation, commitments, reveals, outcomes.
package oracle

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

// TestNewOraclePublisher tests OraclePublisher creation.
func TestNewOraclePublisher(t *testing.T) {
	pub := NewOraclePublisher(nil, nil)
	if pub == nil {
		t.Fatal("expected non-nil publisher")
	}
	if pub.topic != TopicAnonymousMechanics {
		t.Errorf("expected topic %s, got %s", TopicAnonymousMechanics, pub.topic)
	}
}

// TestOraclePublisher_PublishPoolCreated tests pool creation publishing.
func TestOraclePublisher_PublishPoolCreated(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewOraclePublisher(mockPub, privKey)

	var creatorKey [32]byte
	copy(creatorKey[:], privKey.Public().(ed25519.PublicKey))

	pool, err := NewOraclePool(
		"Will it rain tomorrow?",
		OraclePredictionBoolean,
		"weather API",
		creatorKey,
		time.Now().Add(24*time.Hour),
		time.Now().Add(48*time.Hour),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx := context.Background()
	err = pub.PublishPoolCreated(ctx, pool)
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

	event := gossipMsg.GetOracleEvent()
	if event == nil {
		t.Fatal("expected oracle event")
	}
	if event.EventType != pb.OracleEventType_ORACLE_EVENT_CREATED {
		t.Errorf("expected CREATED event type, got %v", event.EventType)
	}
	if event.Pool == nil {
		t.Fatal("expected pool in event")
	}
	if event.Pool.Question != "Will it rain tomorrow?" {
		t.Errorf("expected question 'Will it rain tomorrow?', got '%s'", event.Pool.Question)
	}
}

// TestOraclePublisher_PublishPoolCreated_NoPublisher tests without publisher.
func TestOraclePublisher_PublishPoolCreated_NoPublisher(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewOraclePublisher(nil, privKey)

	var creatorKey [32]byte
	pool, _ := NewOraclePool("Test?", OraclePredictionBoolean, "manual", creatorKey,
		time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))

	err := pub.PublishPoolCreated(context.Background(), pool)
	if err != ErrPublisherNotSet {
		t.Errorf("expected ErrPublisherNotSet, got %v", err)
	}
}

// TestOraclePublisher_PublishPoolCreated_NilPool tests with nil pool.
func TestOraclePublisher_PublishPoolCreated_NilPool(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewOraclePublisher(mockPub, privKey)

	err := pub.PublishPoolCreated(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil pool")
	}
}

// TestOraclePublisher_PublishPoolCreated_NoPrivateKey tests without private key.
func TestOraclePublisher_PublishPoolCreated_NoPrivateKey(t *testing.T) {
	mockPub := &mockPublisher{}
	pub := NewOraclePublisher(mockPub, nil)

	var creatorKey [32]byte
	pool, _ := NewOraclePool("Test?", OraclePredictionBoolean, "manual", creatorKey,
		time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))

	err := pub.PublishPoolCreated(context.Background(), pool)
	if err != ErrMissingPrivateKey {
		t.Errorf("expected ErrMissingPrivateKey, got %v", err)
	}
}

// TestOraclePublisher_PublishCommitment tests commitment publishing.
func TestOraclePublisher_PublishCommitment(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewOraclePublisher(mockPub, privKey)

	var poolID, specterKey, commitmentHash [32]byte
	poolID[0] = 1
	copy(specterKey[:], privKey.Public().(ed25519.PublicKey))
	commitmentHash[0] = 42

	ctx := context.Background()
	err := pub.PublishCommitment(ctx, poolID, specterKey, commitmentHash)
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

	event := gossipMsg.GetOracleEvent()
	if event == nil {
		t.Fatal("expected oracle event")
	}
	if event.EventType != pb.OracleEventType_ORACLE_EVENT_PREDICTION {
		t.Errorf("expected PREDICTION event type, got %v", event.EventType)
	}
}

// TestOraclePublisher_PublishReveal tests reveal publishing.
func TestOraclePublisher_PublishReveal(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewOraclePublisher(mockPub, privKey)

	var poolID, specterKey, nonce [32]byte
	poolID[0] = 1
	copy(specterKey[:], privKey.Public().(ed25519.PublicKey))

	ctx := context.Background()
	err := pub.PublishReveal(ctx, poolID, specterKey, 1.0, nonce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockPub.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(mockPub.published))
	}
}

// TestOraclePublisher_PublishPoolClosed tests pool closed publishing.
func TestOraclePublisher_PublishPoolClosed(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewOraclePublisher(mockPub, privKey)

	var poolID [32]byte
	poolID[0] = 1

	ctx := context.Background()
	err := pub.PublishPoolClosed(ctx, poolID)
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

	event := gossipMsg.GetOracleEvent()
	if event == nil {
		t.Fatal("expected oracle event")
	}
	if event.EventType != pb.OracleEventType_ORACLE_EVENT_CLOSED {
		t.Errorf("expected CLOSED event type, got %v", event.EventType)
	}
}

// TestOraclePublisher_PublishOutcome tests outcome publishing.
func TestOraclePublisher_PublishOutcome(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewOraclePublisher(mockPub, privKey)

	var creatorKey [32]byte
	copy(creatorKey[:], privKey.Public().(ed25519.PublicKey))

	pool, err := NewOraclePool(
		"Will it rain?",
		OraclePredictionBoolean,
		"manual",
		creatorKey,
		time.Now().Add(time.Hour),
		time.Now().Add(2*time.Hour),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx := context.Background()
	err = pub.PublishOutcome(ctx, pool, 1.0)
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

	event := gossipMsg.GetOracleEvent()
	if event == nil {
		t.Fatal("expected oracle event")
	}
	if event.EventType != pb.OracleEventType_ORACLE_EVENT_RESOLVED {
		t.Errorf("expected RESOLVED event type, got %v", event.EventType)
	}
	if event.WinningOption != 1 {
		t.Errorf("expected winning option 1, got %d", event.WinningOption)
	}
}

// TestOraclePublisher_PublishOutcome_NilPool tests with nil pool.
func TestOraclePublisher_PublishOutcome_NilPool(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewOraclePublisher(mockPub, privKey)

	err := pub.PublishOutcome(context.Background(), nil, 1.0)
	if err == nil {
		t.Error("expected error for nil pool")
	}
}

// TestNewOracleReceiver tests OracleReceiver creation.
func TestNewOracleReceiver(t *testing.T) {
	store := NewOraclePoolStore()
	receiver := NewOracleReceiver(store)
	if receiver == nil {
		t.Fatal("expected non-nil receiver")
	}
	if receiver.poolStore != store {
		t.Error("store not set correctly")
	}
}

// TestOracleReceiver_HandlePoolCreated tests pool creation handling.
func TestOracleReceiver_HandlePoolCreated(t *testing.T) {
	store := NewOraclePoolStore()
	receiver := NewOracleReceiver(store)

	// Create and publish a pool.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewOraclePublisher(mockPub, privKey)

	var creatorKey [32]byte
	copy(creatorKey[:], privKey.Public().(ed25519.PublicKey))

	pool, _ := NewOraclePool("Test question?", OraclePredictionBoolean, "manual", creatorKey,
		time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))

	err := publisher.PublishPoolCreated(context.Background(), pool)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify pool was stored.
	stored := store.GetPool(pool.ID)
	if stored == nil {
		t.Fatal("expected pool to be stored")
	}
	if stored.Question != "Test question?" {
		t.Errorf("expected 'Test question?', got '%s'", stored.Question)
	}
}

// TestOracleReceiver_HandlePoolClosed tests pool closed handling.
func TestOracleReceiver_HandlePoolClosed(t *testing.T) {
	store := NewOraclePoolStore()
	receiver := NewOracleReceiver(store)

	// Create and store a pool first.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	var creatorKey [32]byte
	copy(creatorKey[:], privKey.Public().(ed25519.PublicKey))

	pool, _ := NewOraclePool("Test?", OraclePredictionBoolean, "manual", creatorKey,
		time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))
	store.AddPool(pool)

	// Publish pool closed event.
	mockPub := &mockPublisher{}
	publisher := NewOraclePublisher(mockPub, privKey)

	err := publisher.PublishPoolClosed(context.Background(), pool.ID)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify pool state changed.
	stored := store.GetPool(pool.ID)
	if stored.State != OraclePoolPending {
		t.Errorf("expected state OraclePoolPending, got %v", stored.State)
	}
}

// TestOracleReceiver_HandleOutcome tests outcome handling.
func TestOracleReceiver_HandleOutcome(t *testing.T) {
	store := NewOraclePoolStore()
	receiver := NewOracleReceiver(store)

	// Create and store a pool first.
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	var creatorKey [32]byte
	copy(creatorKey[:], privKey.Public().(ed25519.PublicKey))

	pool, _ := NewOraclePool("Test?", OraclePredictionBoolean, "manual", creatorKey,
		time.Now().Add(-time.Hour), time.Now())
	pool.State = OraclePoolPending
	store.AddPool(pool)

	// Publish outcome event.
	mockPub := &mockPublisher{}
	publisher := NewOraclePublisher(mockPub, privKey)

	err := publisher.PublishOutcome(context.Background(), pool, 1.0)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle the published message.
	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != nil {
		t.Fatalf("failed to handle message: %v", err)
	}

	// Verify pool was resolved.
	stored := store.GetPool(pool.ID)
	if stored.State != OraclePoolResolved {
		t.Errorf("expected state OraclePoolResolved, got %v", stored.State)
	}
	if stored.Outcome == nil {
		t.Fatal("expected outcome to be set")
	}
	if *stored.Outcome != 1.0 {
		t.Errorf("expected outcome 1.0, got %f", *stored.Outcome)
	}
}

// TestOracleReceiver_HandleMessage_InvalidData tests with invalid data.
func TestOracleReceiver_HandleMessage_InvalidData(t *testing.T) {
	store := NewOraclePoolStore()
	receiver := NewOracleReceiver(store)

	err := receiver.HandleMessage([]byte("invalid"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

// TestOracleReceiver_HandleMessage_NonOracleEvent tests with non-oracle event.
func TestOracleReceiver_HandleMessage_NonOracleEvent(t *testing.T) {
	store := NewOraclePoolStore()
	receiver := NewOracleReceiver(store)

	// Create a gossip message without an oracle event.
	gossipMsg := &pb.GossipMessage{}
	data, _ := proto.Marshal(gossipMsg)

	err := receiver.HandleMessage(data)
	if err != nil {
		t.Errorf("expected nil for non-oracle event, got %v", err)
	}
}

// TestOracleReceiver_HandleMessage_MissingSignature tests with missing signature.
func TestOracleReceiver_HandleMessage_MissingSignature(t *testing.T) {
	store := NewOraclePoolStore()
	receiver := NewOracleReceiver(store)

	// Create an unsigned oracle event.
	event := &pb.OracleEvent{
		EventType: pb.OracleEventType_ORACLE_EVENT_CREATED,
		PoolId:    []byte("test"),
	}
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_OracleEvent{
			OracleEvent: event,
		},
	}
	data, _ := proto.Marshal(gossipMsg)

	err := receiver.HandleMessage(data)
	if err != ErrMissingSignature {
		t.Errorf("expected ErrMissingSignature, got %v", err)
	}
}

// TestOracleReceiver_HandleOutcome_PoolNotFound tests with pool not found.
func TestOracleReceiver_HandleOutcome_PoolNotFound(t *testing.T) {
	store := NewOraclePoolStore()
	receiver := NewOracleReceiver(store)

	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewOraclePublisher(mockPub, privKey)

	var poolID [32]byte
	poolID[0] = 99 // Non-existent pool.

	// Create a minimal pool to publish outcome (will be created on receive).
	var creatorKey [32]byte
	copy(creatorKey[:], privKey.Public().(ed25519.PublicKey))
	pool, _ := NewOraclePool("Test?", OraclePredictionBoolean, "manual", creatorKey,
		time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))

	err := publisher.PublishOutcome(context.Background(), pool, 1.0)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Handle should create pool from event data.
	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify pool was created and resolved.
	stored := store.GetPool(pool.ID)
	if stored == nil {
		t.Fatal("expected pool to be created from event")
	}
}

// TestOracleReceiver_HandleOutcome_AlreadyResolved tests double resolution.
func TestOracleReceiver_HandleOutcome_AlreadyResolved(t *testing.T) {
	store := NewOraclePoolStore()
	receiver := NewOracleReceiver(store)

	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	var creatorKey [32]byte
	copy(creatorKey[:], privKey.Public().(ed25519.PublicKey))

	pool, _ := NewOraclePool("Test?", OraclePredictionBoolean, "manual", creatorKey,
		time.Now().Add(-time.Hour), time.Now())
	outcome := 1.0
	pool.Outcome = &outcome
	pool.State = OraclePoolResolved
	store.AddPool(pool)

	mockPub := &mockPublisher{}
	publisher := NewOraclePublisher(mockPub, privKey)

	err := publisher.PublishOutcome(context.Background(), pool, 0.0)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != ErrOraclePoolAlreadyResolved {
		t.Errorf("expected ErrOraclePoolAlreadyResolved, got %v", err)
	}
}

// TestOracleEventSignatureRoundTrip tests signature verification.
func TestOracleEventSignatureRoundTrip(t *testing.T) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewOraclePublisher(mockPub, privKey)

	var creatorKey [32]byte
	copy(creatorKey[:], privKey.Public().(ed25519.PublicKey))

	pool, _ := NewOraclePool("Test?", OraclePredictionBoolean, "manual", creatorKey,
		time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))

	err := publisher.PublishPoolCreated(context.Background(), pool)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Unmarshal and verify signature.
	var gossipMsg pb.GossipMessage
	err = proto.Unmarshal(mockPub.published[0].data, &gossipMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	event := gossipMsg.GetOracleEvent()
	if event == nil {
		t.Fatal("expected oracle event")
	}
	if len(event.Signature) == 0 {
		t.Fatal("expected signature")
	}

	// Verify the signature through receiver.
	store := NewOraclePoolStore()
	receiver := NewOracleReceiver(store)
	err = receiver.verifyEventSignature(event)
	if err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
}

// BenchmarkOraclePublisher_PublishPoolCreated benchmarks pool creation publishing.
func BenchmarkOraclePublisher_PublishPoolCreated(b *testing.B) {
	mockPub := &mockPublisher{}
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewOraclePublisher(mockPub, privKey)

	var creatorKey [32]byte
	copy(creatorKey[:], privKey.Public().(ed25519.PublicKey))

	pool, _ := NewOraclePool("Benchmark question?", OraclePredictionBoolean, "manual", creatorKey,
		time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mockPub.published = nil
		pub.PublishPoolCreated(ctx, pool)
	}
}

// BenchmarkOracleReceiver_HandleMessage benchmarks message handling.
func BenchmarkOracleReceiver_HandleMessage(b *testing.B) {
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mockPub := &mockPublisher{}
	publisher := NewOraclePublisher(mockPub, privKey)

	var creatorKey [32]byte
	copy(creatorKey[:], privKey.Public().(ed25519.PublicKey))

	pool, _ := NewOraclePool("Benchmark?", OraclePredictionBoolean, "manual", creatorKey,
		time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))

	publisher.PublishPoolCreated(context.Background(), pool)
	data := mockPub.published[0].data

	store := NewOraclePoolStore()
	receiver := NewOracleReceiver(store)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		receiver.HandleMessage(data)
	}
}
