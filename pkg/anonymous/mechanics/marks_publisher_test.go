// Package mechanics - Specter Mark network propagation tests.
// Per ROADMAP.md line 531, tests for mark event publishing and receiving.
package mechanics

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// TestMarkPublisher_Creation tests MarkPublisher instantiation.
func TestMarkPublisher_Creation(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	pub := NewMarkPublisher(mockPub, privateKey)
	if pub == nil {
		t.Fatal("NewMarkPublisher returned nil")
	}
	if pub.topic != TopicAnonymousMechanics {
		t.Errorf("wrong topic: got %s, want %s", pub.topic, TopicAnonymousMechanics)
	}
}

// TestMarkPublisher_NilPublisher tests handling when publisher is nil.
func TestMarkPublisher_NilPublisher(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewMarkPublisher(nil, privateKey)

	mark := &Mark{
		Category:  MarkWatcher,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(MarkDuration),
	}

	err := pub.PublishMarkPlaced(context.Background(), mark)
	if err != ErrPublisherNotSet {
		t.Errorf("expected ErrPublisherNotSet, got %v", err)
	}
}

// TestMarkPublisher_NilMark tests handling when mark is nil.
func TestMarkPublisher_NilMark(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewMarkPublisher(mockPub, privateKey)

	err := pub.PublishMarkPlaced(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil mark")
	}
}

// TestMarkPublisher_NilPrivateKey tests handling when private key is nil.
func TestMarkPublisher_NilPrivateKey(t *testing.T) {
	mockPub := &mockPublisher{}
	pub := NewMarkPublisher(mockPub, nil)

	mark := &Mark{
		Category:  MarkWatcher,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(MarkDuration),
	}

	err := pub.PublishMarkPlaced(context.Background(), mark)
	if err != ErrMissingPrivateKey {
		t.Errorf("expected ErrMissingPrivateKey, got %v", err)
	}
}

// TestMarkPublisher_PublishMarkPlaced tests successful mark publication.
func TestMarkPublisher_PublishMarkPlaced(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewMarkPublisher(mockPub, privateKey)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	mark := &Mark{
		MarkerKey: markerKey,
		TargetKey: targetKey,
		Category:  MarkWatcher,
		Note:      "Test note",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(MarkDuration),
	}
	copy(mark.ID[:], []byte("test-mark-id-1234567890123456"))

	err := pub.PublishMarkPlaced(context.Background(), mark)
	if err != nil {
		t.Fatalf("PublishMarkPlaced failed: %v", err)
	}

	if len(mockPub.published) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mockPub.published))
	}

	// Verify topic.
	if mockPub.published[0].topic != TopicAnonymousMechanics {
		t.Errorf("wrong topic: got %s", mockPub.published[0].topic)
	}

	// Unmarshal and verify.
	var gossipMsg pb.GossipMessage
	if err := proto.Unmarshal(mockPub.published[0].data, &gossipMsg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	markEvent := gossipMsg.GetMarkEvent()
	if markEvent == nil {
		t.Fatal("no mark event in message")
	}
	if markEvent.Mark == nil {
		t.Fatal("mark event has no mark")
	}
	if len(markEvent.Signature) == 0 {
		t.Error("event has no signature")
	}
}

// TestMarkPublisher_AllCategories tests publishing marks with all categories.
func TestMarkPublisher_AllCategories(t *testing.T) {
	categories := []MarkCategory{
		MarkWatcher,
		MarkAlly,
		MarkRival,
	}

	for _, cat := range categories {
		t.Run(CategoryString(cat), func(t *testing.T) {
			mockPub := &mockPublisher{}
			_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
			pub := NewMarkPublisher(mockPub, privateKey)

			var markerKey [32]byte
			rand.Read(markerKey[:])
			targetKey := make([]byte, 32)
			rand.Read(targetKey)

			mark := &Mark{
				MarkerKey: markerKey,
				TargetKey: targetKey,
				Category:  cat,
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(MarkDuration),
			}
			rand.Read(mark.ID[:])

			err := pub.PublishMarkPlaced(context.Background(), mark)
			if err != nil {
				t.Fatalf("failed to publish mark with category %d: %v", cat, err)
			}

			if len(mockPub.published) != 1 {
				t.Fatalf("expected 1 message, got %d", len(mockPub.published))
			}
		})
	}
}

// TestMarkReceiver_Creation tests MarkReceiver instantiation.
func TestMarkReceiver_Creation(t *testing.T) {
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)
	if receiver == nil {
		t.Fatal("NewMarkReceiver returned nil")
	}
	if receiver.GetMarkStore() != store {
		t.Error("GetMarkStore returned wrong store")
	}
}

// TestMarkReceiver_HandleMessage_NonMarkEvent tests ignoring non-mark events.
func TestMarkReceiver_HandleMessage_NonMarkEvent(t *testing.T) {
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	// Create a non-mark gossip message.
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_HuntEvent{
			HuntEvent: &pb.HuntEvent{
				EventType: pb.HuntEventType_HUNT_EVENT_CREATED,
			},
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != nil {
		t.Errorf("expected no error for non-mark event, got %v", err)
	}
}

// TestMarkReceiver_HandleMessage_InvalidData tests handling of invalid data.
func TestMarkReceiver_HandleMessage_InvalidData(t *testing.T) {
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	err := receiver.HandleMessage([]byte("invalid data"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

// TestMarkReceiver_HandleMessage_MissingSignature tests rejection of unsigned events.
func TestMarkReceiver_HandleMessage_MissingSignature(t *testing.T) {
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_MarkEvent{
			MarkEvent: &pb.MarkEvent{
				Mark: &pb.SpecterMark{
					Id:            make([]byte, 32),
					SpecterPubkey: markerKey[:],
					TargetPubkey:  targetKey,
					Content:       "Test",
					CreatedAt:     time.Now().Unix(),
					ExpiresAt:     time.Now().Add(MarkDuration).Unix(),
				},
				Timestamp: time.Now().Unix(),
				// No Signature field set.
			},
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != ErrMissingSignature {
		t.Errorf("expected ErrMissingSignature, got %v", err)
	}
}

// TestMarkReceiver_HandleMessage_ValidMark tests successful mark reception.
func TestMarkReceiver_HandleMessage_ValidMark(t *testing.T) {
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	// Create a signed mark event.
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var markerKey [32]byte
	copy(markerKey[:], pubKey)
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	var markID [32]byte
	rand.Read(markID[:])

	now := time.Now()
	pbMark := &pb.SpecterMark{
		Id:            markID[:],
		SpecterPubkey: markerKey[:],
		TargetPubkey:  targetKey,
		Content:       "Test mark",
		CreatedAt:     now.Unix(),
		ExpiresAt:     now.Add(MarkDuration).Unix(),
		Signature:     make([]byte, 64), // Mark signature (separate from event signature).
	}

	event := &pb.MarkEvent{
		Mark:      pbMark,
		Timestamp: now.Unix(),
	}

	// Sign the event.
	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_MarkEvent{
			MarkEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Verify mark was added.
	retrieved, err := store.GetMark(markID)
	if err != nil {
		t.Fatalf("GetMark failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("mark not found in store")
	}
	if retrieved.Note != "Test mark" {
		t.Errorf("wrong note: got %q, want %q", retrieved.Note, "Test mark")
	}
}

// TestMarkReceiver_HandleMessage_ExpiredMark tests rejection of expired marks.
func TestMarkReceiver_HandleMessage_ExpiredMark(t *testing.T) {
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var markerKey [32]byte
	copy(markerKey[:], pubKey)
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	var markID [32]byte
	rand.Read(markID[:])

	// Create an expired mark.
	expiredTime := time.Now().Add(-48 * time.Hour)
	pbMark := &pb.SpecterMark{
		Id:            markID[:],
		SpecterPubkey: markerKey[:],
		TargetPubkey:  targetKey,
		Content:       "Test",
		CreatedAt:     expiredTime.Unix(),
		ExpiresAt:     expiredTime.Add(24 * time.Hour).Unix(), // Expired.
		Signature:     make([]byte, 64),
	}

	event := &pb.MarkEvent{
		Mark:      pbMark,
		Timestamp: time.Now().Unix(),
	}

	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_MarkEvent{
			MarkEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != ErrMarkNotFound {
		t.Errorf("expected ErrMarkNotFound for expired mark, got %v", err)
	}
}

// TestMarkReceiver_HandleMessage_DuplicateMark tests rejection of duplicate marks.
func TestMarkReceiver_HandleMessage_DuplicateMark(t *testing.T) {
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var markerKey [32]byte
	copy(markerKey[:], pubKey)
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	var markID [32]byte
	rand.Read(markID[:])

	now := time.Now()
	pbMark := &pb.SpecterMark{
		Id:            markID[:],
		SpecterPubkey: markerKey[:],
		TargetPubkey:  targetKey,
		Content:       "Test",
		CreatedAt:     now.Unix(),
		ExpiresAt:     now.Add(MarkDuration).Unix(),
		Signature:     make([]byte, 64),
	}

	event := &pb.MarkEvent{
		Mark:      pbMark,
		Timestamp: now.Unix(),
	}

	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_MarkEvent{
			MarkEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)

	// First message should succeed.
	err := receiver.HandleMessage(data)
	if err != nil {
		t.Fatalf("first HandleMessage failed: %v", err)
	}

	// Second message with same ID should fail.
	err = receiver.HandleMessage(data)
	if err != ErrMarkAlreadyPlaced {
		t.Errorf("expected ErrMarkAlreadyPlaced, got %v", err)
	}
}

// TestMarkReceiver_HandleMessage_InvalidSignature tests rejection of bad signatures.
func TestMarkReceiver_HandleMessage_InvalidSignature(t *testing.T) {
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	var markID [32]byte
	rand.Read(markID[:])

	now := time.Now()
	pbMark := &pb.SpecterMark{
		Id:            markID[:],
		SpecterPubkey: markerKey[:],
		TargetPubkey:  targetKey,
		Content:       "Test",
		CreatedAt:     now.Unix(),
		ExpiresAt:     now.Add(MarkDuration).Unix(),
		Signature:     make([]byte, 64), // Mark signature.
	}

	event := &pb.MarkEvent{
		Mark:      pbMark,
		Timestamp: now.Unix(),
		Signature: []byte("invalid-signature-that-is-wrong"),
	}

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_MarkEvent{
			MarkEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != ErrSignatureFailed {
		t.Errorf("expected ErrSignatureFailed, got %v", err)
	}
}

// TestMarkReceiver_HandleMessage_MissingMark tests handling of event with nil mark.
func TestMarkReceiver_HandleMessage_MissingMark(t *testing.T) {
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	event := &pb.MarkEvent{
		Mark:      nil, // Missing mark.
		Timestamp: time.Now().Unix(),
	}

	// Can't properly sign without a mark.
	event.Signature = ed25519.Sign(privateKey, []byte("dummy"))

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_MarkEvent{
			MarkEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != ErrSignatureFailed {
		t.Errorf("expected ErrSignatureFailed for nil mark, got %v", err)
	}
}

// TestMarkPublisher_RoundTrip tests publishing and receiving a mark.
func TestMarkPublisher_RoundTrip(t *testing.T) {
	mockPub := &mockPublisher{}
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	publisher := NewMarkPublisher(mockPub, privateKey)
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	var markerKey [32]byte
	copy(markerKey[:], pubKey)
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	mark := &Mark{
		MarkerKey: markerKey,
		TargetKey: targetKey,
		Category:  MarkAlly,
		Note:      "Test round-trip",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(MarkDuration),
		Signature: make([]byte, 64), // Placeholder mark signature.
	}
	rand.Read(mark.ID[:])

	// Publish.
	err := publisher.PublishMarkPlaced(context.Background(), mark)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	// Receive.
	err = receiver.HandleMessage(mockPub.published[0].data)
	if err != nil {
		t.Fatalf("receive failed: %v", err)
	}

	// Verify.
	retrieved, err := store.GetMark(mark.ID)
	if err != nil {
		t.Fatalf("GetMark failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("mark not found after round-trip")
	}
	if retrieved.Note != mark.Note {
		t.Errorf("note mismatch: got %q, want %q", retrieved.Note, mark.Note)
	}
	if string(retrieved.TargetKey) != string(mark.TargetKey) {
		t.Error("target key mismatch")
	}
}

// TestMarkPublisher_MultipleMarks tests publishing multiple marks.
func TestMarkPublisher_MultipleMarks(t *testing.T) {
	mockPub := &mockPublisher{}
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	publisher := NewMarkPublisher(mockPub, privateKey)
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	var markerKey [32]byte
	copy(markerKey[:], pubKey)

	for i := 0; i < 5; i++ {
		targetKey := make([]byte, 32)
		rand.Read(targetKey)

		mark := &Mark{
			MarkerKey: markerKey,
			TargetKey: targetKey,
			Category:  MarkCategory(uint8(i%3) + 1),
			Note:      "Test",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(MarkDuration),
			Signature: make([]byte, 64),
		}
		rand.Read(mark.ID[:])

		err := publisher.PublishMarkPlaced(context.Background(), mark)
		if err != nil {
			t.Fatalf("publish %d failed: %v", i, err)
		}
	}

	if len(mockPub.published) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(mockPub.published))
	}

	// Receive all.
	for i, msg := range mockPub.published {
		err := receiver.HandleMessage(msg.data)
		if err != nil {
			t.Fatalf("receive %d failed: %v", i, err)
		}
	}

	// Verify count.
	if store.Count() != 5 {
		t.Errorf("expected 5 marks in store, got %d", store.Count())
	}
}

// TestMarkReceiver_GetMarkStore tests the GetMarkStore accessor.
func TestMarkReceiver_GetMarkStore(t *testing.T) {
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	retrieved := receiver.GetMarkStore()
	if retrieved != store {
		t.Error("GetMarkStore returned different store instance")
	}
}

// TestMarkPublisher_SignatureDataConsistency tests that signature data is consistent.
func TestMarkPublisher_SignatureDataConsistency(t *testing.T) {
	mockPub := &mockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	publisher := NewMarkPublisher(mockPub, privateKey)
	receiver := NewMarkReceiver(NewMarkStore())

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	now := time.Now()
	pbMark := &pb.SpecterMark{
		Id:            make([]byte, 32),
		SpecterPubkey: markerKey[:],
		TargetPubkey:  targetKey,
		Content:       "Test",
		CreatedAt:     now.Unix(),
		ExpiresAt:     now.Add(MarkDuration).Unix(),
	}
	rand.Read(pbMark.Id)

	event := &pb.MarkEvent{
		Mark:      pbMark,
		Timestamp: now.Unix(),
	}

	// Get signature data from both publisher and receiver.
	pubSigData := publisher.eventSignatureData(event)
	recSigData := receiver.eventSignatureData(event)

	if string(pubSigData) != string(recSigData) {
		t.Error("signature data mismatch between publisher and receiver")
	}
}

// TestMarkReceiver_MarkerTargetConstraint tests marker-target uniqueness.
func TestMarkReceiver_MarkerTargetConstraint(t *testing.T) {
	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var markerKey [32]byte
	copy(markerKey[:], pubKey)
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	now := time.Now()

	// First mark.
	var markID1 [32]byte
	rand.Read(markID1[:])
	pbMark1 := &pb.SpecterMark{
		Id:            markID1[:],
		SpecterPubkey: markerKey[:],
		TargetPubkey:  targetKey,
		Content:       "First",
		CreatedAt:     now.Unix(),
		ExpiresAt:     now.Add(MarkDuration).Unix(),
		Signature:     make([]byte, 64),
	}

	event1 := &pb.MarkEvent{
		Mark:      pbMark1,
		Timestamp: now.Unix(),
	}
	sigData1 := receiver.eventSignatureData(event1)
	event1.Signature = ed25519.Sign(privateKey, sigData1)

	gossipMsg1 := &pb.GossipMessage{
		Content: &pb.GossipMessage_MarkEvent{MarkEvent: event1},
	}
	data1, _ := proto.Marshal(gossipMsg1)

	// First should succeed.
	err := receiver.HandleMessage(data1)
	if err != nil {
		t.Fatalf("first mark failed: %v", err)
	}

	// Second mark with same marker and target but different ID.
	var markID2 [32]byte
	rand.Read(markID2[:])
	pbMark2 := &pb.SpecterMark{
		Id:            markID2[:],
		SpecterPubkey: markerKey[:],
		TargetPubkey:  targetKey, // Same target.
		Content:       "Second",
		CreatedAt:     now.Unix(),
		ExpiresAt:     now.Add(MarkDuration).Unix(),
		Signature:     make([]byte, 64),
	}

	event2 := &pb.MarkEvent{
		Mark:      pbMark2,
		Timestamp: now.Unix(),
	}
	sigData2 := receiver.eventSignatureData(event2)
	event2.Signature = ed25519.Sign(privateKey, sigData2)

	gossipMsg2 := &pb.GossipMessage{
		Content: &pb.GossipMessage_MarkEvent{MarkEvent: event2},
	}
	data2, _ := proto.Marshal(gossipMsg2)

	// Second should fail due to marker-target constraint.
	err = receiver.HandleMessage(data2)
	if err != ErrMarkAlreadyPlaced {
		t.Errorf("expected ErrMarkAlreadyPlaced for same marker-target, got %v", err)
	}
}

// BenchmarkMarkPublisher_Publish benchmarks mark publishing.
func BenchmarkMarkPublisher_Publish(b *testing.B) {
	mockPub := &mockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	publisher := NewMarkPublisher(mockPub, privateKey)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	mark := &Mark{
		MarkerKey: markerKey,
		TargetKey: targetKey,
		Category:  MarkWatcher,
		Note:      "Benchmark",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(MarkDuration),
		Signature: make([]byte, 64),
	}
	rand.Read(mark.ID[:])

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = publisher.PublishMarkPlaced(ctx, mark)
	}
}

// BenchmarkMarkReceiver_HandleMessage benchmarks mark receiving.
func BenchmarkMarkReceiver_HandleMessage(b *testing.B) {
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var markerKey [32]byte
	copy(markerKey[:], pubKey)

	now := time.Now()
	pbMark := &pb.SpecterMark{
		Id:            make([]byte, 32),
		SpecterPubkey: markerKey[:],
		TargetPubkey:  make([]byte, 32),
		Content:       "Test",
		CreatedAt:     now.Unix(),
		ExpiresAt:     now.Add(MarkDuration).Unix(),
		Signature:     make([]byte, 64),
	}

	event := &pb.MarkEvent{
		Mark:      pbMark,
		Timestamp: now.Unix(),
	}

	store := NewMarkStore()
	receiver := NewMarkReceiver(store)

	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_MarkEvent{
			MarkEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Generate unique IDs to avoid duplicate errors.
		rand.Read(pbMark.Id)
		rand.Read(pbMark.TargetPubkey)
		data, _ = proto.Marshal(gossipMsg)
		_ = receiver.HandleMessage(data)
	}
}
