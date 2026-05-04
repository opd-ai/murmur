// Package mechanics - Phantom Gift network propagation.
// Per ROADMAP.md line 517, broadcasts gift creation and receipt events.
package gifts

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"

	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

// GiftPublisher handles publishing gift events to the anonymous mechanics topic.
// All gift events are broadcast on mechanics.TopicAnonymousMechanics (/murmur/anonymous/mechanics/1.0).
type GiftPublisher struct {
	publisher  mechanics.Publisher
	topic      string
	privateKey ed25519.PrivateKey
}

// NewGiftPublisher creates a new gift publisher.
// privateKey is used to sign events; it can be nil if only receiving events.
func NewGiftPublisher(pub mechanics.Publisher, privateKey ed25519.PrivateKey) *GiftPublisher {
	return &GiftPublisher{
		publisher:  pub,
		topic:      mechanics.TopicAnonymousMechanics,
		privateKey: privateKey,
	}
}

// PublishGiftCreated broadcasts a new gift announcement.
func (g *GiftPublisher) PublishGiftCreated(ctx context.Context, gift *Gift) error {
	if g.publisher == nil {
		return mechanics.ErrPublisherNotSet
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
		return mechanics.ErrMissingPrivateKey
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
	return computeGiftEventSignatureData(event)
}

// computeGiftEventSignatureData is the canonical computation of gift event signature data.
// This function is shared by both GiftPublisher and GiftReceiver to ensure signature
// verification uses the same algorithm as signature generation.
func computeGiftEventSignatureData(event *pb.GiftEvent) []byte {
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
	return mechanics.ProcessGossipEvent(
		data,
		func(msg *pb.GossipMessage) *pb.GiftEvent { return msg.GetGiftEvent() },
		r.verifyEventSignature,
		r.processEvent,
	)
}

// verifyEventSignature checks the event signature.
func (r *GiftReceiver) verifyEventSignature(event *pb.GiftEvent) error {
	if event.Gift == nil || len(event.Gift.SenderPubkey) != 32 {
		return mechanics.ErrSignatureFailed
	}

	sigData := r.eventSignatureData(event)
	return mechanics.VerifyEd25519Signature(
		event.Signature,
		event.Gift.Signature,
		event.Gift.SenderPubkey,
		sigData,
	)
}

// eventSignatureData creates the data that was signed.
func (r *GiftReceiver) eventSignatureData(event *pb.GiftEvent) []byte {
	return computeGiftEventSignatureData(event)
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
	senderHex := mechanics.KeyToHex(gift.SenderPubKey[:])
	r.giftStore.bySender[senderHex] = append(r.giftStore.bySender[senderHex], gift)

	// Update recipient index.
	recipientHex := mechanics.KeyToHex(gift.RecipientKey)
	r.giftStore.byRecipient[recipientHex] = append(r.giftStore.byRecipient[recipientHex], gift)

	return nil
}

// GetGiftStore returns the underlying gift store.
func (r *GiftReceiver) GetGiftStore() *GiftStore {
	return r.giftStore
}
