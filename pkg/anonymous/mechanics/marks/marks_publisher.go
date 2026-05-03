// Package mechanics - Specter Mark network propagation.
// Per ROADMAP.md line 531, broadcasts mark placement and removal events.
package marks

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

// MarkPublisher handles publishing mark events to the anonymous mechanics topic.
// All mark events are broadcast on mechanics.TopicAnonymousMechanics (/murmur/anonymous/mechanics/1.0).
type MarkPublisher struct {
	publisher  mechanics.Publisher
	topic      string
	privateKey ed25519.PrivateKey
}

// NewMarkPublisher creates a new mark publisher.
// privateKey is used to sign events; it can be nil if only receiving events.
func NewMarkPublisher(pub mechanics.Publisher, privateKey ed25519.PrivateKey) *MarkPublisher {
	return &MarkPublisher{
		publisher:  pub,
		topic:      mechanics.TopicAnonymousMechanics,
		privateKey: privateKey,
	}
}

// PublishMarkPlaced broadcasts a new mark placement event.
func (m *MarkPublisher) PublishMarkPlaced(ctx context.Context, mark *Mark) error {
	if m.publisher == nil {
		return mechanics.ErrPublisherNotSet
	}
	if mark == nil {
		return fmt.Errorf("mark cannot be nil")
	}

	pbMark := markToProto(mark)
	event := &pb.MarkEvent{
		Mark:      pbMark,
		Timestamp: time.Now().Unix(),
	}

	return m.signAndPublish(ctx, event)
}

// signAndPublish signs the event and publishes it to the topic.
func (m *MarkPublisher) signAndPublish(ctx context.Context, event *pb.MarkEvent) error {
	if m.privateKey == nil {
		return mechanics.ErrMissingPrivateKey
	}

	// Create signature over event data.
	sigData := m.eventSignatureData(event)
	signature := ed25519.Sign(m.privateKey, sigData)
	event.Signature = signature

	// Wrap in GossipMessage.
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_MarkEvent{
			MarkEvent: event,
		},
	}

	data, err := proto.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal mark event: %w", err)
	}

	return m.publisher.Publish(ctx, m.topic, data)
}

// eventSignatureData creates the data to be signed for an event.
func (m *MarkPublisher) eventSignatureData(event *pb.MarkEvent) []byte {
	return computeMarkEventSignatureData(event)
}

// computeMarkEventSignatureData is the canonical computation of mark event signature data.
// This function is shared by both MarkPublisher and MarkReceiver to ensure signature
// verification uses the same algorithm as signature generation.
func computeMarkEventSignatureData(event *pb.MarkEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("mark-event-v1"))
	if event.Mark != nil {
		hash.Write(event.Mark.Id)
		hash.Write(event.Mark.SpecterPubkey)
		hash.Write(event.Mark.TargetPubkey)
		hash.Write([]byte(event.Mark.Content))
	}
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	return hash.Sum(nil)
}

// MarkReceiver handles incoming mark events from the network.
type MarkReceiver struct {
	markStore *MarkStore
}

// NewMarkReceiver creates a new mark receiver.
func NewMarkReceiver(store *MarkStore) *MarkReceiver {
	return &MarkReceiver{
		markStore: store,
	}
}

// HandleMessage processes an incoming mark event.
func (r *MarkReceiver) HandleMessage(data []byte) error {
	var gossipMsg pb.GossipMessage
	if err := proto.Unmarshal(data, &gossipMsg); err != nil {
		return fmt.Errorf("failed to unmarshal gossip message: %w", err)
	}

	markEvent := gossipMsg.GetMarkEvent()
	if markEvent == nil {
		return nil // Not a mark event.
	}

	// Verify signature.
	if err := r.verifyEventSignature(markEvent); err != nil {
		return err
	}

	return r.processEvent(markEvent)
}

// verifyEventSignature checks the event signature.
func (r *MarkReceiver) verifyEventSignature(event *pb.MarkEvent) error {
	if len(event.Signature) == 0 {
		return mechanics.ErrMissingSignature
	}

	// The sender public key is the mark's specter pubkey.
	if event.Mark == nil || len(event.Mark.SpecterPubkey) != 32 {
		return mechanics.ErrSignatureFailed
	}

	// Verify the mark's own signature if present.
	if len(event.Mark.Signature) == 0 {
		return mechanics.ErrMissingSignature
	}

	sigData := r.eventSignatureData(event)

	// Try verification with specter public key.
	if len(event.Mark.SpecterPubkey) == ed25519.PublicKeySize {
		if ed25519.Verify(event.Mark.SpecterPubkey, sigData, event.Signature) {
			return nil
		}
	}

	return mechanics.ErrSignatureFailed
}

// eventSignatureData creates the data that was signed.
func (r *MarkReceiver) eventSignatureData(event *pb.MarkEvent) []byte {
	return computeMarkEventSignatureData(event)
}

// processEvent handles the mark event.
func (r *MarkReceiver) processEvent(event *pb.MarkEvent) error {
	if event.Mark == nil {
		return fmt.Errorf("mark event missing mark data")
	}

	mark := protoToMark(event.Mark)
	if mark == nil {
		return fmt.Errorf("failed to convert mark from protobuf")
	}

	// Check for duplicate.
	existing, err := r.markStore.GetMark(mark.ID)
	if err == nil && existing != nil {
		return ErrMarkAlreadyPlaced
	}

	// Check expiration.
	if mark.IsExpired() {
		return ErrMarkNotFound // Return not found for expired marks.
	}

	// Add to store.
	return r.addMarkToStore(mark)
}

// addMarkToStore adds a received mark to the store.
func (r *MarkReceiver) addMarkToStore(mark *Mark) error {
	r.markStore.mu.Lock()
	defer r.markStore.mu.Unlock()

	// Check for duplicate again under lock.
	if _, exists := r.markStore.marks[mark.ID]; exists {
		return ErrMarkAlreadyPlaced
	}

	// Check marker-target constraint.
	markerHex := mechanics.KeyToHex(mark.MarkerKey[:])
	targetHex := mechanics.KeyToHex(mark.TargetKey)

	if targets, ok := r.markStore.markerTargets[markerHex]; ok {
		if targets[targetHex] {
			return ErrMarkAlreadyPlaced
		}
	}

	// Add to main index.
	r.markStore.marks[mark.ID] = mark

	// Update marker index.
	r.markStore.byMarker[markerHex] = append(r.markStore.byMarker[markerHex], mark)

	// Update target index.
	r.markStore.byTarget[targetHex] = append(r.markStore.byTarget[targetHex], mark)

	// Update marker-target tracking.
	if r.markStore.markerTargets[markerHex] == nil {
		r.markStore.markerTargets[markerHex] = make(map[string]bool)
	}
	r.markStore.markerTargets[markerHex][targetHex] = true

	return nil
}

// GetMarkStore returns the underlying mark store.
func (r *MarkReceiver) GetMarkStore() *MarkStore {
	return r.markStore
}
