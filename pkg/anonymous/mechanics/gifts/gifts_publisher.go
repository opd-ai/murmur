// Package mechanics - Phantom Gift network propagation.
// Per ROADMAP.md line 517, broadcasts gift creation and receipt events.
package gifts

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

// GiftPublisher handles publishing gift events to the anonymous mechanics topic.
// All gift events are broadcast on TopicAnonymousMechanics (/murmur/anonymous/mechanics/1.0).
type GiftPublisher struct {
	publisher  Publisher
	topic      string
	privateKey ed25519.PrivateKey
}

// NewGiftPublisher creates a new gift publisher.
// privateKey is used to sign events; it can be nil if only receiving events.
func NewGiftPublisher(pub Publisher, privateKey ed25519.PrivateKey) *GiftPublisher {
	return &GiftPublisher{
		publisher:  pub,
		topic:      TopicAnonymousMechanics,
		privateKey: privateKey,
	}
}

// PublishGiftCreated broadcasts a new gift announcement.
func (g *GiftPublisher) PublishGiftCreated(ctx context.Context, gift *Gift) error {
	if g.publisher == nil {
		return ErrPublisherNotSet
	}
	if gift == nil {
		return fmt.Errorf("gift cannot be nil")
	}

	pbGift := giftToProto(gift)
	event := &pb.GiftEvent{
		Gift:      pbGift,
		Timestamp: time.Now().Unix(),
	}

	return g.signAndPublish(ctx, event)
}

// signAndPublish signs the event and publishes it to the topic.
func (g *GiftPublisher) signAndPublish(ctx context.Context, event *pb.GiftEvent) error {
	if g.privateKey == nil {
		return ErrMissingPrivateKey
	}

	// Create signature over event data.
	sigData := g.eventSignatureData(event)
	signature := ed25519.Sign(g.privateKey, sigData)
	event.Signature = signature

	// Wrap in GossipMessage.
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_GiftEvent{
			GiftEvent: event,
		},
	}

	data, err := proto.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal gift event: %w", err)
	}

	return g.publisher.Publish(ctx, g.topic, data)
}

// eventSignatureData creates the data to be signed for an event.
func (g *GiftPublisher) eventSignatureData(event *pb.GiftEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("gift-event-v1"))
	if event.Gift != nil {
		hash.Write(event.Gift.Id)
		hash.Write(event.Gift.SenderPubkey)
		hash.Write(event.Gift.RecipientPubkey)
		binary.Write(hash, binary.BigEndian, event.Gift.EffectType)
	}
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	return hash.Sum(nil)
}

// GiftReceiver handles incoming gift events from the network.
type GiftReceiver struct {
	giftStore *GiftStore
}

// NewGiftReceiver creates a new gift receiver.
func NewGiftReceiver(store *GiftStore) *GiftReceiver {
	return &GiftReceiver{
		giftStore: store,
	}
}

// HandleMessage processes an incoming gift event.
func (r *GiftReceiver) HandleMessage(data []byte) error {
	var gossipMsg pb.GossipMessage
	if err := proto.Unmarshal(data, &gossipMsg); err != nil {
		return fmt.Errorf("failed to unmarshal gossip message: %w", err)
	}

	giftEvent := gossipMsg.GetGiftEvent()
	if giftEvent == nil {
		return nil // Not a gift event.
	}

	// Verify signature.
	if err := r.verifyEventSignature(giftEvent); err != nil {
		return err
	}

	return r.processEvent(giftEvent)
}

// verifyEventSignature checks the event signature.
func (r *GiftReceiver) verifyEventSignature(event *pb.GiftEvent) error {
	if len(event.Signature) == 0 {
		return ErrMissingSignature
	}

	// The sender public key is the gift's sender.
	if event.Gift == nil || len(event.Gift.SenderPubkey) != 32 {
		return ErrSignatureFailed
	}

	// For Phantom Gifts, the signature is verified using Ed25519.
	// The Specter's Curve25519 key is converted or they use a separate signing key.
	// We verify using the gift's own signature field if present.
	if len(event.Gift.Signature) == 0 {
		return ErrMissingSignature
	}

	sigData := r.eventSignatureData(event)

	// Try verification with sender public key (if it's an Ed25519 key).
	// Note: Specters use Curve25519 for DH, but may have an Ed25519 signing key.
	// For simplicity, we accept any valid 32-byte key as potential Ed25519 public key.
	if len(event.Gift.SenderPubkey) == ed25519.PublicKeySize {
		if ed25519.Verify(event.Gift.SenderPubkey, sigData, event.Signature) {
			return nil
		}
	}

	return ErrSignatureFailed
}

// eventSignatureData creates the data that was signed.
func (r *GiftReceiver) eventSignatureData(event *pb.GiftEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("gift-event-v1"))
	if event.Gift != nil {
		hash.Write(event.Gift.Id)
		hash.Write(event.Gift.SenderPubkey)
		hash.Write(event.Gift.RecipientPubkey)
		binary.Write(hash, binary.BigEndian, event.Gift.EffectType)
	}
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	return hash.Sum(nil)
}

// processEvent handles the gift event.
func (r *GiftReceiver) processEvent(event *pb.GiftEvent) error {
	if event.Gift == nil {
		return fmt.Errorf("gift event missing gift data")
	}

	gift := protoToGift(event.Gift)
	if gift == nil {
		return fmt.Errorf("failed to convert gift from protobuf")
	}

	// Check for duplicate.
	existing, err := r.giftStore.GetGift(gift.ID)
	if err == nil && existing != nil {
		return ErrDuplicateGift
	}

	// Check expiration.
	if gift.IsExpired() {
		return ErrGiftExpired
	}

	// Add to store.
	return r.addGiftToStore(gift)
}

// addGiftToStore adds a received gift to the store.
func (r *GiftReceiver) addGiftToStore(gift *Gift) error {
	r.giftStore.mu.Lock()
	defer r.giftStore.mu.Unlock()

	// Check for duplicate again under lock.
	if _, exists := r.giftStore.gifts[gift.ID]; exists {
		return ErrDuplicateGift
	}

	// Add to main index.
	r.giftStore.gifts[gift.ID] = gift

	// Update sender index.
	senderHex := keyToHex(gift.SenderPubKey[:])
	r.giftStore.bySender[senderHex] = append(r.giftStore.bySender[senderHex], gift)

	// Update recipient index.
	recipientHex := keyToHex(gift.RecipientKey)
	r.giftStore.byRecipient[recipientHex] = append(r.giftStore.byRecipient[recipientHex], gift)

	return nil
}

// GetGiftStore returns the underlying gift store.
func (r *GiftReceiver) GetGiftStore() *GiftStore {
	return r.giftStore
}
