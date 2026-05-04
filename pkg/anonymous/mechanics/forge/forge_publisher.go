// Package mechanics - Sigil Forge network propagation.
// Per ROADMAP.md line 474, broadcasts forge events, entries, votes.
package forge

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"

	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"

	pb "github.com/opd-ai/murmur/proto"
)

// ForgePublisher handles publishing forge events to the anonymous mechanics topic.
// All forge events are broadcast on mechanics.TopicAnonymousMechanics (/murmur/anonymous/mechanics/1.0).
type ForgePublisher struct {
	publisher  mechanics.Publisher
	topic      string
	privateKey ed25519.PrivateKey
}

// NewForgePublisher creates a new forge publisher.
// privateKey is used to sign events; it can be nil if only receiving events.
func NewForgePublisher(pub mechanics.Publisher, privateKey ed25519.PrivateKey) *ForgePublisher {
	return &ForgePublisher{
		publisher:  pub,
		topic:      mechanics.TopicAnonymousMechanics,
		privateKey: privateKey,
	}
}

// PublishForgeCreated broadcasts a new forge announcement.
// Per ANONYMOUS_GAME_MECHANICS.md, forge creation requires Resonance ≥50.
func (f *ForgePublisher) PublishForgeCreated(ctx context.Context, forge *SigilForge) error {
	if f.publisher == nil {
		return mechanics.ErrPublisherNotSet
	}
	if forge == nil {
		return fmt.Errorf("forge cannot be nil")
	}

	pbForge := forgeToProto(forge)
	event := &pb.ForgeEvent{
		EventType: pb.ForgeEventType_FORGE_EVENT_CREATED,
		Project:   pbForge,
		ProjectId: forge.ID[:],
		Timestamp: time.Now().Unix(),
	}

	return f.signAndPublish(ctx, event)
}

// PublishEntry broadcasts a forge entry submission.
func (f *ForgePublisher) PublishEntry(
	ctx context.Context,
	forgeID [32]byte,
	entry *ForgeEntry,
) error {
	if f.publisher == nil {
		return mechanics.ErrPublisherNotSet
	}
	if entry == nil {
		return fmt.Errorf("entry cannot be nil")
	}

	pbContrib := &pb.ForgeContribution{
		SpecterPubkey: entry.SpecterKey[:],
		Contribution:  entry.Content,
		Timestamp:     entry.SubmittedAt.Unix(),
	}

	event := &pb.ForgeEvent{
		EventType:    pb.ForgeEventType_FORGE_EVENT_CONTRIBUTION,
		ProjectId:    forgeID[:],
		Contribution: pbContrib,
		Timestamp:    time.Now().Unix(),
	}

	return f.signAndPublish(ctx, event)
}

// PublishAmplification broadcasts an amplification (vote) for an entry.
// Note: Amplification is tracked via the CONTRIBUTION event type.
func (f *ForgePublisher) PublishAmplification(
	ctx context.Context,
	forgeID [32]byte,
	entryID [32]byte,
	amplifierKey [32]byte,
) error {
	if f.publisher == nil {
		return mechanics.ErrPublisherNotSet
	}

	// Use contribution with entry ID reference.
	pbContrib := &pb.ForgeContribution{
		SpecterPubkey: amplifierKey[:],
		Contribution:  entryID[:], // Reference to the amplified entry.
		Timestamp:     time.Now().Unix(),
	}

	event := &pb.ForgeEvent{
		EventType:    pb.ForgeEventType_FORGE_EVENT_CONTRIBUTION,
		ProjectId:    forgeID[:],
		Contribution: pbContrib,
		Timestamp:    time.Now().Unix(),
	}

	return f.signAndPublish(ctx, event)
}

// PublishForgeFinalized broadcasts forge completion with winner.
func (f *ForgePublisher) PublishForgeFinalized(
	ctx context.Context,
	forge *SigilForge,
	winningSigil []byte,
) error {
	if f.publisher == nil {
		return mechanics.ErrPublisherNotSet
	}
	if forge == nil {
		return fmt.Errorf("forge cannot be nil")
	}

	pbForge := forgeToProto(forge)
	event := &pb.ForgeEvent{
		EventType:  pb.ForgeEventType_FORGE_EVENT_FINALIZED,
		Project:    pbForge,
		ProjectId:  forge.ID[:],
		FinalSigil: winningSigil,
		Timestamp:  time.Now().Unix(),
	}

	return f.signAndPublish(ctx, event)
}

// PublishForgeFailed broadcasts that a forge failed (not enough entries).
func (f *ForgePublisher) PublishForgeFailed(ctx context.Context, forgeID [32]byte) error {
	if f.publisher == nil {
		return mechanics.ErrPublisherNotSet
	}

	event := &pb.ForgeEvent{
		EventType: pb.ForgeEventType_FORGE_EVENT_FAILED,
		ProjectId: forgeID[:],
		Timestamp: time.Now().Unix(),
	}

	return f.signAndPublish(ctx, event)
}

// signAndPublish signs the event and publishes it to the topic.
func (f *ForgePublisher) signAndPublish(ctx context.Context, event *pb.ForgeEvent) error {
	if f.privateKey == nil {
		return mechanics.ErrMissingPrivateKey
	}

	// Create signature over event data.
	sigData := f.eventSignatureData(event)
	signature := ed25519.Sign(f.privateKey, sigData)
	event.Signature = signature

	// Wrap in GossipMessage.
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_ForgeEvent{
			ForgeEvent: event,
		},
	}

	data, err := proto.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal forge event: %w", err)
	}

	return f.publisher.Publish(ctx, f.topic, data)
}

// eventSignatureData creates the data to be signed for an event.
func (f *ForgePublisher) eventSignatureData(event *pb.ForgeEvent) []byte {
	return computeForgeEventSignatureData(event)
}

// computeForgeEventSignatureData is the canonical computation of forge event signature data.
// This function is shared by both ForgePublisher and ForgeReceiver to ensure signature
// verification uses the same algorithm as signature generation.
func computeForgeEventSignatureData(event *pb.ForgeEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("forge-event-v1"))
	binary.Write(hash, binary.BigEndian, int32(event.EventType))
	hash.Write(event.ProjectId)
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	if event.Contribution != nil {
		hash.Write(event.Contribution.SpecterPubkey)
		hash.Write(event.Contribution.Contribution)
	}
	return hash.Sum(nil)
}

// ForgeReceiver handles incoming forge events from the network.
type ForgeReceiver struct {
	forgeStore *ForgeStore
}

// NewForgeReceiver creates a new forge receiver.
func NewForgeReceiver(store *ForgeStore) *ForgeReceiver {
	return &ForgeReceiver{
		forgeStore: store,
	}
}

// HandleMessage processes an incoming forge event.
func (r *ForgeReceiver) HandleMessage(data []byte) error {
	return mechanics.ProcessGossipEvent(
		data,
		func(msg *pb.GossipMessage) *pb.ForgeEvent { return msg.GetForgeEvent() },
		r.verifyEventSignature,
		r.processEvent,
	)
}

// verifyEventSignature checks the event signature.
func (r *ForgeReceiver) verifyEventSignature(event *pb.ForgeEvent) error {
	if len(event.Signature) == 0 {
		return mechanics.ErrMissingSignature
	}

	switch event.EventType {
	case pb.ForgeEventType_FORGE_EVENT_CREATED:
		return r.verifyCreationSignature(event)
	case pb.ForgeEventType_FORGE_EVENT_CONTRIBUTION:
		return r.verifyContributionSignature(event)
	default:
		// For finalized/failed events, signature is from forge creator.
		return nil
	}
}

// verifyCreationSignature validates the signature for forge creation events.
func (r *ForgeReceiver) verifyCreationSignature(event *pb.ForgeEvent) error {
	if event.Project != nil && len(event.Project.CreatorPubkey) == ed25519.PublicKeySize {
		sigData := r.eventSignatureData(event)
		if !ed25519.Verify(event.Project.CreatorPubkey, sigData, event.Signature) {
			return mechanics.ErrSignatureFailed
		}
	}
	return nil
}

// verifyContributionSignature validates the signature for contribution events.
func (r *ForgeReceiver) verifyContributionSignature(event *pb.ForgeEvent) error {
	if event.Contribution != nil && len(event.Contribution.SpecterPubkey) == ed25519.PublicKeySize {
		sigData := r.eventSignatureData(event)
		if !ed25519.Verify(event.Contribution.SpecterPubkey, sigData, event.Signature) {
			return mechanics.ErrSignatureFailed
		}
	}
	return nil
}

// eventSignatureData creates the data that was signed.
func (r *ForgeReceiver) eventSignatureData(event *pb.ForgeEvent) []byte {
	return computeForgeEventSignatureData(event)
}

// processEvent handles the specific event type.
func (r *ForgeReceiver) processEvent(event *pb.ForgeEvent) error {
	switch event.EventType {
	case pb.ForgeEventType_FORGE_EVENT_CREATED:
		return r.handleForgeCreated(event)
	case pb.ForgeEventType_FORGE_EVENT_CONTRIBUTION:
		return r.handleContribution(event)
	case pb.ForgeEventType_FORGE_EVENT_FINALIZED:
		return r.handleForgeFinalized(event)
	case pb.ForgeEventType_FORGE_EVENT_FAILED:
		return r.handleForgeFailed(event)
	default:
		return nil // Ignore unknown event types.
	}
}

// handleForgeCreated processes a forge creation event.
func (r *ForgeReceiver) handleForgeCreated(event *pb.ForgeEvent) error {
	if event.Project == nil {
		return fmt.Errorf("forge creation event missing project data")
	}

	forge := protoToForge(event.Project)
	if forge == nil {
		return fmt.Errorf("failed to convert forge from protobuf")
	}

	// Add to store if not already present.
	if existing := r.forgeStore.GetForge(forge.ID); existing == nil {
		r.forgeStore.AddForge(forge)
	}

	return nil
}

// handleContribution processes a contribution (entry or amplification) event.
func (r *ForgeReceiver) handleContribution(event *pb.ForgeEvent) error {
	if event.Contribution == nil {
		return fmt.Errorf("contribution event missing contribution data")
	}

	var forgeID [32]byte
	copy(forgeID[:], event.ProjectId)

	forge := r.forgeStore.GetForge(forgeID)
	if forge == nil {
		return fmt.Errorf("forge not found: %x", forgeID)
	}

	// Check if this is a new entry or an amplification.
	contrib := event.Contribution
	if len(contrib.Contribution) == 32 {
		// Could be an amplification (entry ID reference).
		// Try to find existing entry with this ID.
		var entryID [32]byte
		copy(entryID[:], contrib.Contribution)
		for _, entry := range forge.Entries {
			if entry.ID == entryID {
				// This is an amplification.
				var amplifierKey [32]byte
				copy(amplifierKey[:], contrib.SpecterPubkey)
				forge.AmplifyEntry(entryID, amplifierKey, 1.0) // Default weight.
				return nil
			}
		}
	}

	// It's a new entry.
	var specterKey [32]byte
	copy(specterKey[:], contrib.SpecterPubkey)

	entry := &ForgeEntry{
		ForgeID:      forgeID,
		SpecterKey:   specterKey,
		Content:      contrib.Contribution,
		SubmittedAt:  time.Unix(contrib.Timestamp, 0),
		amplifierSet: make(map[string]bool),
	}
	entry.ID = computeEntryID(specterKey, entry.Content)

	forge.mu.Lock()
	defer forge.mu.Unlock()

	specterHex := hex.EncodeToString(specterKey[:])
	if forge.entryBySpecter[specterHex] != nil {
		return ErrForgeDuplicateEntry
	}

	forge.Entries = append(forge.Entries, entry)
	forge.entryBySpecter[specterHex] = entry

	return nil
}

// handleForgeFinalized processes a forge completion event.
func (r *ForgeReceiver) handleForgeFinalized(event *pb.ForgeEvent) error {
	var forgeID [32]byte
	copy(forgeID[:], event.ProjectId)

	forge := r.forgeStore.GetForge(forgeID)
	if forge == nil {
		// If forge not found, try to create from event data.
		if event.Project != nil {
			newForge := protoToForge(event.Project)
			if newForge != nil {
				r.forgeStore.AddForge(newForge)
				forge = newForge
			}
		}
		if forge == nil {
			return fmt.Errorf("forge not found: %x", forgeID)
		}
	}

	forge.mu.Lock()
	forge.State = ForgeCompleted
	forge.mu.Unlock()

	return nil
}

// handleForgeFailed processes a forge failure event.
func (r *ForgeReceiver) handleForgeFailed(event *pb.ForgeEvent) error {
	var forgeID [32]byte
	copy(forgeID[:], event.ProjectId)

	forge := r.forgeStore.GetForge(forgeID)
	if forge == nil {
		return fmt.Errorf("forge not found: %x", forgeID)
	}

	forge.mu.Lock()
	forge.State = ForgeExpired
	forge.mu.Unlock()

	return nil
}
