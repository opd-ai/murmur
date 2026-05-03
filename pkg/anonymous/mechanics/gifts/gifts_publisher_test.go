// Package mechanics - Phantom Gift network propagation tests.
// Per ROADMAP.md line 517, tests for gift event publishing and receiving.
package gifts

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// TestGiftPublisher_Creation tests GiftPublisher instantiation.
func TestGiftPublisher_Creation(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	pub := NewGiftPublisher(mockPub, privateKey)
	if pub == nil {
		t.Fatal("NewGiftPublisher returned nil")
	}
	if pub.topic != mechanics.TopicAnonymousMechanics {
		t.Errorf("wrong topic: got %s, want %s", pub.topic, mechanics.TopicAnonymousMechanics)
	}
}

// TestGiftPublisher_NilPublisher tests handling when publisher is nil.
func TestGiftPublisher_NilPublisher(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewGiftPublisher(nil, privateKey)

	gift := &Gift{
		Effect:    EffectSoftGlowPulse,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(GiftDuration),
	}

	err := pub.PublishGiftCreated(context.Background(), gift)
	if err != mechanics.ErrPublisherNotSet {
		t.Errorf("expected mechanics.ErrPublisherNotSet, got %v", err)
	}
}

// TestGiftPublisher_NilGift tests handling when gift is nil.
func TestGiftPublisher_NilGift(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewGiftPublisher(mockPub, privateKey)

	err := pub.PublishGiftCreated(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil gift")
	}
}

// TestGiftPublisher_NilPrivateKey tests handling when private key is nil.
func TestGiftPublisher_NilPrivateKey(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	pub := NewGiftPublisher(mockPub, nil)

	gift := &Gift{
		Effect:    EffectSoftGlowPulse,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(GiftDuration),
	}

	err := pub.PublishGiftCreated(context.Background(), gift)
	if err != mechanics.ErrMissingPrivateKey {
		t.Errorf("expected mechanics.ErrMissingPrivateKey, got %v", err)
	}
}

// TestGiftPublisher_PublishGiftCreated tests successful gift publication.
func TestGiftPublisher_PublishGiftCreated(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewGiftPublisher(mockPub, privateKey)

	var senderKey, recipientKey [32]byte
	rand.Read(senderKey[:])
	rand.Read(recipientKey[:])

	gift := &Gift{
		SenderPubKey: senderKey,
		RecipientKey: recipientKey[:],
		Effect:       EffectSoftGlowPulse,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(GiftDuration),
	}
	copy(gift.ID[:], []byte("test-gift-id-1234567890123456"))

	err := pub.PublishGiftCreated(context.Background(), gift)
	if err != nil {
		t.Fatalf("PublishGiftCreated failed: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mockPub.Published))
	}

	// Verify topic.
	if mockPub.Published[0].Topic != mechanics.TopicAnonymousMechanics {
		t.Errorf("wrong topic: got %s", mockPub.Published[0].Topic)
	}

	// Unmarshal and verify.
	var gossipMsg pb.GossipMessage
	if err := proto.Unmarshal(mockPub.Published[0].Data, &gossipMsg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	giftEvent := gossipMsg.GetGiftEvent()
	if giftEvent == nil {
		t.Fatal("no gift event in message")
	}
	if giftEvent.Gift == nil {
		t.Fatal("gift event has no gift")
	}
	if len(giftEvent.Signature) == 0 {
		t.Error("event has no signature")
	}
}

// TestGiftPublisher_AllEffectTypes tests publishing gifts with various effects.
func TestGiftPublisher_AllEffectTypes(t *testing.T) {
	effects := []EffectType{
		EffectSoftGlowPulse,
		EffectOrbitingGeometric,
		EffectMultiParticleSystem,
	}

	for _, effect := range effects {
		t.Run(EffectName(effect), func(t *testing.T) {
			mockPub := &mechanics.MockPublisher{}
			_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
			pub := NewGiftPublisher(mockPub, privateKey)

			var senderKey [32]byte
			rand.Read(senderKey[:])
			recipientKey := make([]byte, 32)
			rand.Read(recipientKey)

			gift := &Gift{
				SenderPubKey: senderKey,
				RecipientKey: recipientKey,
				Effect:       effect,
				CreatedAt:    time.Now(),
				ExpiresAt:    time.Now().Add(GiftDuration),
			}
			rand.Read(gift.ID[:])

			err := pub.PublishGiftCreated(context.Background(), gift)
			if err != nil {
				t.Fatalf("failed to publish gift with effect %d: %v", effect, err)
			}

			if len(mockPub.Published) != 1 {
				t.Fatalf("expected 1 message, got %d", len(mockPub.Published))
			}
		})
	}
}

// TestGiftReceiver_Creation tests GiftReceiver instantiation.
func TestGiftReceiver_Creation(t *testing.T) {
	store := NewGiftStore()
	receiver := NewGiftReceiver(store)
	if receiver == nil {
		t.Fatal("NewGiftReceiver returned nil")
	}
	if receiver.GetGiftStore() != store {
		t.Error("GetGiftStore returned wrong store")
	}
}

// TestGiftReceiver_HandleMessage_NonGiftEvent tests ignoring non-gift events.
func TestGiftReceiver_HandleMessage_NonGiftEvent(t *testing.T) {
	store := NewGiftStore()
	receiver := NewGiftReceiver(store)

	// Create a non-gift gossip message.
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
		t.Errorf("expected no error for non-gift event, got %v", err)
	}
}

// TestGiftReceiver_HandleMessage_InvalidData tests handling of invalid data.
func TestGiftReceiver_HandleMessage_InvalidData(t *testing.T) {
	store := NewGiftStore()
	receiver := NewGiftReceiver(store)

	err := receiver.HandleMessage([]byte("invalid data"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

// TestGiftReceiver_HandleMessage_MissingSignature tests rejection of unsigned events.
func TestGiftReceiver_HandleMessage_MissingSignature(t *testing.T) {
	store := NewGiftStore()
	receiver := NewGiftReceiver(store)

	var senderKey [32]byte
	rand.Read(senderKey[:])
	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_GiftEvent{
			GiftEvent: &pb.GiftEvent{
				Gift: &pb.PhantomGift{
					Id:              make([]byte, 32),
					SenderPubkey:    senderKey[:],
					RecipientPubkey: recipientKey,
					EffectType:      uint32(EffectSoftGlowPulse),
					CreatedAt:       time.Now().Unix(),
					ExpiresAt:       time.Now().Add(GiftDuration).Unix(),
				},
				Timestamp: time.Now().Unix(),
				// No Signature field set.
			},
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != mechanics.ErrMissingSignature {
		t.Errorf("expected mechanics.ErrMissingSignature, got %v", err)
	}
}

// TestGiftReceiver_HandleMessage_ValidGift tests successful gift reception.
func TestGiftReceiver_HandleMessage_ValidGift(t *testing.T) {
	store := NewGiftStore()
	receiver := NewGiftReceiver(store)

	// Create a signed gift event.
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var senderKey [32]byte
	copy(senderKey[:], pubKey)
	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	var giftID [32]byte
	rand.Read(giftID[:])

	now := time.Now()
	pbGift := &pb.PhantomGift{
		Id:              giftID[:],
		SenderPubkey:    senderKey[:],
		RecipientPubkey: recipientKey,
		EffectType:      uint32(EffectSoftGlowPulse),
		CreatedAt:       now.Unix(),
		ExpiresAt:       now.Add(GiftDuration).Unix(),
		Signature:       make([]byte, 64), // Gift signature (separate from event signature).
	}

	event := &pb.GiftEvent{
		Gift:      pbGift,
		Timestamp: now.Unix(),
	}

	// Sign the event.
	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_GiftEvent{
			GiftEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Verify gift was added.
	retrieved, err := store.GetGift(giftID)
	if err != nil {
		t.Fatalf("GetGift failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("gift not found in store")
	}
	if retrieved.Effect != EffectSoftGlowPulse {
		t.Errorf("wrong effect: got %d, want %d", retrieved.Effect, EffectSoftGlowPulse)
	}
}

// TestGiftReceiver_HandleMessage_ExpiredGift tests rejection of expired gifts.
func TestGiftReceiver_HandleMessage_ExpiredGift(t *testing.T) {
	store := NewGiftStore()
	receiver := NewGiftReceiver(store)

	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var senderKey [32]byte
	copy(senderKey[:], pubKey)
	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	var giftID [32]byte
	rand.Read(giftID[:])

	// Create an expired gift.
	expiredTime := time.Now().Add(-48 * time.Hour)
	pbGift := &pb.PhantomGift{
		Id:              giftID[:],
		SenderPubkey:    senderKey[:],
		RecipientPubkey: recipientKey,
		EffectType:      uint32(EffectSoftGlowPulse),
		CreatedAt:       expiredTime.Unix(),
		ExpiresAt:       expiredTime.Add(24 * time.Hour).Unix(), // Expired.
		Signature:       make([]byte, 64),
	}

	event := &pb.GiftEvent{
		Gift:      pbGift,
		Timestamp: time.Now().Unix(),
	}

	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_GiftEvent{
			GiftEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != ErrGiftExpired {
		t.Errorf("expected ErrGiftExpired, got %v", err)
	}
}

// TestGiftReceiver_HandleMessage_DuplicateGift tests rejection of duplicate gifts.
func TestGiftReceiver_HandleMessage_DuplicateGift(t *testing.T) {
	store := NewGiftStore()
	receiver := NewGiftReceiver(store)

	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var senderKey [32]byte
	copy(senderKey[:], pubKey)
	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	var giftID [32]byte
	rand.Read(giftID[:])

	now := time.Now()
	pbGift := &pb.PhantomGift{
		Id:              giftID[:],
		SenderPubkey:    senderKey[:],
		RecipientPubkey: recipientKey,
		EffectType:      uint32(EffectSoftGlowPulse),
		CreatedAt:       now.Unix(),
		ExpiresAt:       now.Add(GiftDuration).Unix(),
		Signature:       make([]byte, 64),
	}

	event := &pb.GiftEvent{
		Gift:      pbGift,
		Timestamp: now.Unix(),
	}

	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_GiftEvent{
			GiftEvent: event,
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
	if err != ErrDuplicateGift {
		t.Errorf("expected ErrDuplicateGift, got %v", err)
	}
}

// TestGiftReceiver_HandleMessage_InvalidSignature tests rejection of bad signatures.
func TestGiftReceiver_HandleMessage_InvalidSignature(t *testing.T) {
	store := NewGiftStore()
	receiver := NewGiftReceiver(store)

	var senderKey [32]byte
	rand.Read(senderKey[:])
	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	var giftID [32]byte
	rand.Read(giftID[:])

	now := time.Now()
	pbGift := &pb.PhantomGift{
		Id:              giftID[:],
		SenderPubkey:    senderKey[:],
		RecipientPubkey: recipientKey,
		EffectType:      uint32(EffectSoftGlowPulse),
		CreatedAt:       now.Unix(),
		ExpiresAt:       now.Add(GiftDuration).Unix(),
		Signature:       make([]byte, 64), // Gift signature.
	}

	event := &pb.GiftEvent{
		Gift:      pbGift,
		Timestamp: now.Unix(),
		Signature: []byte("invalid-signature-that-is-wrong"),
	}

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_GiftEvent{
			GiftEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != mechanics.ErrSignatureFailed {
		t.Errorf("expected mechanics.ErrSignatureFailed, got %v", err)
	}
}

// TestGiftReceiver_HandleMessage_MissingGift tests handling of event with nil gift.
func TestGiftReceiver_HandleMessage_MissingGift(t *testing.T) {
	store := NewGiftStore()
	receiver := NewGiftReceiver(store)

	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	event := &pb.GiftEvent{
		Gift:      nil, // Missing gift.
		Timestamp: time.Now().Unix(),
	}

	// Can't properly sign without a gift, but we test the nil gift path.
	event.Signature = ed25519.Sign(privateKey, []byte("dummy"))

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_GiftEvent{
			GiftEvent: event,
		},
	}

	_ = pubKey
	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != mechanics.ErrSignatureFailed {
		t.Errorf("expected mechanics.ErrSignatureFailed for nil gift, got %v", err)
	}
}

// TestGiftPublisher_RoundTrip tests publishing and receiving a gift.
func TestGiftPublisher_RoundTrip(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	publisher := NewGiftPublisher(mockPub, privateKey)
	store := NewGiftStore()
	receiver := NewGiftReceiver(store)

	var senderKey [32]byte
	copy(senderKey[:], pubKey)
	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	gift := &Gift{
		SenderPubKey: senderKey,
		RecipientKey: recipientKey,
		Effect:       EffectOrbitingGeometric,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(GiftDuration),
		Signature:    make([]byte, 64), // Placeholder gift signature.
	}
	rand.Read(gift.ID[:])

	// Publish.
	err := publisher.PublishGiftCreated(context.Background(), gift)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	// Receive.
	err = receiver.HandleMessage(mockPub.Published[0].Data)
	if err != nil {
		t.Fatalf("receive failed: %v", err)
	}

	// Verify.
	retrieved, err := store.GetGift(gift.ID)
	if err != nil {
		t.Fatalf("GetGift failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("gift not found after round-trip")
	}
	if retrieved.Effect != gift.Effect {
		t.Errorf("effect mismatch: got %d, want %d", retrieved.Effect, gift.Effect)
	}
	if string(retrieved.RecipientKey) != string(gift.RecipientKey) {
		t.Error("recipient key mismatch")
	}
}

// TestGiftPublisher_MultipleGifts tests publishing multiple gifts.
func TestGiftPublisher_MultipleGifts(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	publisher := NewGiftPublisher(mockPub, privateKey)
	store := NewGiftStore()
	receiver := NewGiftReceiver(store)

	var senderKey [32]byte
	copy(senderKey[:], pubKey)

	for i := 0; i < 5; i++ {
		recipientKey := make([]byte, 32)
		rand.Read(recipientKey)

		gift := &Gift{
			SenderPubKey: senderKey,
			RecipientKey: recipientKey,
			Effect:       EffectType(uint8(i) + 1),
			CreatedAt:    time.Now(),
			ExpiresAt:    time.Now().Add(GiftDuration),
			Signature:    make([]byte, 64),
		}
		rand.Read(gift.ID[:])

		err := publisher.PublishGiftCreated(context.Background(), gift)
		if err != nil {
			t.Fatalf("publish %d failed: %v", i, err)
		}
	}

	if len(mockPub.Published) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(mockPub.Published))
	}

	// Receive all.
	for i, msg := range mockPub.Published {
		err := receiver.HandleMessage(msg.Data)
		if err != nil {
			t.Fatalf("receive %d failed: %v", i, err)
		}
	}

	// Verify count.
	if store.Count() != 5 {
		t.Errorf("expected 5 gifts in store, got %d", store.Count())
	}
}

// TestGiftReceiver_GetGiftStore tests the GetGiftStore accessor.
func TestGiftReceiver_GetGiftStore(t *testing.T) {
	store := NewGiftStore()
	receiver := NewGiftReceiver(store)

	retrieved := receiver.GetGiftStore()
	if retrieved != store {
		t.Error("GetGiftStore returned different store instance")
	}
}

// TestGiftPublisher_SignatureDataConsistency tests that signature data is consistent.
func TestGiftPublisher_SignatureDataConsistency(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	publisher := NewGiftPublisher(mockPub, privateKey)
	receiver := NewGiftReceiver(NewGiftStore())

	var senderKey [32]byte
	rand.Read(senderKey[:])
	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	now := time.Now()
	pbGift := &pb.PhantomGift{
		Id:              make([]byte, 32),
		SenderPubkey:    senderKey[:],
		RecipientPubkey: recipientKey,
		EffectType:      uint32(EffectSoftGlowPulse),
		CreatedAt:       now.Unix(),
		ExpiresAt:       now.Add(GiftDuration).Unix(),
	}
	rand.Read(pbGift.Id)

	event := &pb.GiftEvent{
		Gift:      pbGift,
		Timestamp: now.Unix(),
	}

	// Get signature data from both publisher and receiver.
	pubSigData := publisher.eventSignatureData(event)
	recSigData := receiver.eventSignatureData(event)

	if string(pubSigData) != string(recSigData) {
		t.Error("signature data mismatch between publisher and receiver")
	}
}

// BenchmarkGiftPublisher_Publish benchmarks gift publishing.
func BenchmarkGiftPublisher_Publish(b *testing.B) {
	mockPub := &mechanics.MockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	publisher := NewGiftPublisher(mockPub, privateKey)

	var senderKey [32]byte
	rand.Read(senderKey[:])
	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	gift := &Gift{
		SenderPubKey: senderKey,
		RecipientKey: recipientKey,
		Effect:       EffectSoftGlowPulse,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(GiftDuration),
		Signature:    make([]byte, 64),
	}
	rand.Read(gift.ID[:])

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = publisher.PublishGiftCreated(ctx, gift)
	}
}

// BenchmarkGiftReceiver_HandleMessage benchmarks gift receiving.
func BenchmarkGiftReceiver_HandleMessage(b *testing.B) {
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var senderKey [32]byte
	copy(senderKey[:], pubKey)
	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	now := time.Now()
	pbGift := &pb.PhantomGift{
		Id:              make([]byte, 32),
		SenderPubkey:    senderKey[:],
		RecipientPubkey: recipientKey,
		EffectType:      uint32(EffectSoftGlowPulse),
		CreatedAt:       now.Unix(),
		ExpiresAt:       now.Add(GiftDuration).Unix(),
		Signature:       make([]byte, 64),
	}

	event := &pb.GiftEvent{
		Gift:      pbGift,
		Timestamp: now.Unix(),
	}

	store := NewGiftStore()
	receiver := NewGiftReceiver(store)

	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_GiftEvent{
			GiftEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Generate unique ID for each iteration to avoid duplicate errors.
		rand.Read(pbGift.Id)
		data, _ = proto.Marshal(gossipMsg)
		_ = receiver.HandleMessage(data)
	}
}
