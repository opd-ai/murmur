// Package mechanics - Specter Hunt network propagation.
// Per ROADMAP.md line 431, broadcasts Hunt events, fragment claims, clue reveals.
package hunts

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

// HuntPublisher handles publishing hunt events to the anonymous mechanics topic.
// All hunt events are broadcast on TopicAnonymousMechanics (/murmur/anonymous/mechanics/1.0).
type HuntPublisher struct {
	publisher  Publisher
	topic      string
	privateKey ed25519.PrivateKey
}

// NewHuntPublisher creates a new hunt publisher.
// privateKey is used to sign events; it can be nil if only receiving events.
func NewHuntPublisher(pub Publisher, privateKey ed25519.PrivateKey) *HuntPublisher {
	return &HuntPublisher{
		publisher:  pub,
		topic:      TopicAnonymousMechanics,
		privateKey: privateKey,
	}
}

// PublishHuntCreated broadcasts a new hunt announcement.
func (h *HuntPublisher) PublishHuntCreated(ctx context.Context, hunt *Hunt) error {
	if h.publisher == nil {
		return ErrPublisherNotSet
	}
	if hunt == nil {
		return fmt.Errorf("hunt cannot be nil")
	}

	pbHunt := huntToNetworkProto(hunt)
	event := &pb.HuntEvent{
		EventType:    pb.HuntEventType_HUNT_EVENT_CREATED,
		Hunt:         pbHunt,
		HuntId:       hunt.ID[:],
		Timestamp:    time.Now().Unix(),
		InitiatorKey: hunt.InitiatorKey[:],
	}

	return h.signAndPublish(ctx, event)
}

// PublishFragmentClaim broadcasts a fragment claim event.
func (h *HuntPublisher) PublishFragmentClaim(
	ctx context.Context,
	huntID [32]byte,
	fragmentIndex int,
	claimerKey [32]byte,
	proof *DHTProximityProof,
) error {
	if h.publisher == nil {
		return ErrPublisherNotSet
	}

	event := &pb.HuntEvent{
		EventType:     pb.HuntEventType_HUNT_EVENT_CLAIM,
		HuntId:        huntID[:],
		FragmentIndex: uint32(fragmentIndex),
		ClaimerKey:    claimerKey[:],
		Timestamp:     time.Now().Unix(),
	}

	// Include proximity proof attestations.
	if proof != nil {
		event.ProximityProof = proximityProofToProto(proof)
	}

	return h.signAndPublish(ctx, event)
}

// PublishClueReveal broadcasts a clue reveal event.
func (h *HuntPublisher) PublishClueReveal(
	ctx context.Context,
	huntID [32]byte,
	fragmentIndex int,
	clueIndex int,
	clueText string,
) error {
	if h.publisher == nil {
		return ErrPublisherNotSet
	}

	event := &pb.HuntEvent{
		EventType:     pb.HuntEventType_HUNT_EVENT_CLUE,
		HuntId:        huntID[:],
		FragmentIndex: uint32(fragmentIndex),
		ClueIndex:     uint32(clueIndex),
		ClueText:      clueText,
		Timestamp:     time.Now().Unix(),
	}

	return h.signAndPublish(ctx, event)
}

// PublishHuntCompleted broadcasts a hunt completion event.
func (h *HuntPublisher) PublishHuntCompleted(ctx context.Context, hunt *Hunt) error {
	if h.publisher == nil {
		return ErrPublisherNotSet
	}
	if hunt == nil {
		return fmt.Errorf("hunt cannot be nil")
	}

	event := &pb.HuntEvent{
		EventType: pb.HuntEventType_HUNT_EVENT_COMPLETED,
		HuntId:    hunt.ID[:],
		Timestamp: time.Now().Unix(),
	}

	// Include leaderboard in completion event.
	leaderboard := hunt.GetLeaderboard()
	event.LeaderboardEntries = make([]*pb.HuntLeaderboardEntry, len(leaderboard))
	for i, entry := range leaderboard {
		event.LeaderboardEntries[i] = &pb.HuntLeaderboardEntry{
			SpecterKey: entry.SpecterKey[:],
			Claims:     uint32(entry.Claims),
			Rank:       uint32(i + 1),
		}
	}

	return h.signAndPublish(ctx, event)
}

// PublishHuntExpired broadcasts a hunt expiration event.
func (h *HuntPublisher) PublishHuntExpired(ctx context.Context, huntID [32]byte) error {
	if h.publisher == nil {
		return ErrPublisherNotSet
	}

	event := &pb.HuntEvent{
		EventType: pb.HuntEventType_HUNT_EVENT_EXPIRED,
		HuntId:    huntID[:],
		Timestamp: time.Now().Unix(),
	}

	return h.signAndPublish(ctx, event)
}

// signAndPublish signs the event and publishes it to the topic.
func (h *HuntPublisher) signAndPublish(ctx context.Context, event *pb.HuntEvent) error {
	if h.privateKey == nil {
		return ErrMissingPrivateKey
	}

	// Create signature over event data.
	sigData := h.eventSignatureData(event)
	signature := ed25519.Sign(h.privateKey, sigData)
	event.Signature = signature

	// Include sender public key.
	pubKey := h.privateKey.Public().(ed25519.PublicKey)
	event.SenderPubkey = pubKey

	// Wrap in GossipMessage.
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_HuntEvent{
			HuntEvent: event,
		},
	}

	data, err := proto.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal hunt event: %w", err)
	}

	return h.publisher.Publish(ctx, h.topic, data)
}

// eventSignatureData creates the data to be signed for an event.
func (h *HuntPublisher) eventSignatureData(event *pb.HuntEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("hunt-event-v1"))
	binary.Write(hash, binary.BigEndian, int32(event.EventType))
	hash.Write(event.HuntId)
	hash.Write(event.ClaimerKey)
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	binary.Write(hash, binary.BigEndian, event.FragmentIndex)
	return hash.Sum(nil)
}

// HuntReceiver handles incoming hunt events from the network.
type HuntReceiver struct {
	huntStore *HuntStore
}

// NewHuntReceiver creates a new hunt receiver.
func NewHuntReceiver(store *HuntStore) *HuntReceiver {
	return &HuntReceiver{
		huntStore: store,
	}
}

// HandleMessage processes an incoming hunt event.
func (r *HuntReceiver) HandleMessage(data []byte) error {
	var gossipMsg pb.GossipMessage
	if err := proto.Unmarshal(data, &gossipMsg); err != nil {
		return fmt.Errorf("failed to unmarshal gossip message: %w", err)
	}

	huntEvent := gossipMsg.GetHuntEvent()
	if huntEvent == nil {
		return nil // Not a hunt event.
	}

	// Verify signature.
	if err := r.verifyEventSignature(huntEvent); err != nil {
		return err
	}

	return r.processEvent(huntEvent)
}

// verifyEventSignature checks the event signature.
func (r *HuntReceiver) verifyEventSignature(event *pb.HuntEvent) error {
	if len(event.Signature) == 0 {
		return ErrMissingSignature
	}
	if len(event.SenderPubkey) != ed25519.PublicKeySize {
		return ErrSignatureFailed
	}

	sigData := r.eventSignatureData(event)
	if !ed25519.Verify(event.SenderPubkey, sigData, event.Signature) {
		return ErrSignatureFailed
	}

	return nil
}

// eventSignatureData creates the data that was signed.
func (r *HuntReceiver) eventSignatureData(event *pb.HuntEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("hunt-event-v1"))
	binary.Write(hash, binary.BigEndian, int32(event.EventType))
	hash.Write(event.HuntId)
	hash.Write(event.ClaimerKey)
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	binary.Write(hash, binary.BigEndian, event.FragmentIndex)
	return hash.Sum(nil)
}

// processEvent handles the specific event type.
func (r *HuntReceiver) processEvent(event *pb.HuntEvent) error {
	switch event.EventType {
	case pb.HuntEventType_HUNT_EVENT_CREATED:
		return r.handleHuntCreated(event)
	case pb.HuntEventType_HUNT_EVENT_CLAIM:
		return r.handleFragmentClaim(event)
	case pb.HuntEventType_HUNT_EVENT_CLUE:
		return r.handleClueReveal(event)
	case pb.HuntEventType_HUNT_EVENT_COMPLETED:
		return r.handleHuntCompleted(event)
	case pb.HuntEventType_HUNT_EVENT_EXPIRED:
		return r.handleHuntExpired(event)
	default:
		return nil // Ignore unknown event types.
	}
}

// handleHuntCreated processes a hunt creation event.
func (r *HuntReceiver) handleHuntCreated(event *pb.HuntEvent) error {
	if event.Hunt == nil {
		return fmt.Errorf("hunt creation event missing hunt data")
	}

	hunt := networkProtoToHunt(event.Hunt)
	if hunt == nil {
		return fmt.Errorf("failed to convert hunt from protobuf")
	}

	// Add to store if not already present.
	if existing := r.huntStore.GetHunt(hunt.ID); existing == nil {
		r.huntStore.AddHunt(hunt)
	}

	return nil
}

// handleFragmentClaim processes a fragment claim event.
func (r *HuntReceiver) handleFragmentClaim(event *pb.HuntEvent) error {
	var huntID [32]byte
	copy(huntID[:], event.HuntId)

	hunt := r.huntStore.GetHunt(huntID)
	if hunt == nil {
		return ErrHuntNotFound
	}

	var claimerKey [32]byte
	copy(claimerKey[:], event.ClaimerKey)

	// Convert proximity proof from protobuf.
	proof := protoToProximityProof(event.ProximityProof, claimerKey)

	// Attempt to claim the fragment.
	return hunt.ClaimFragment(int(event.FragmentIndex), claimerKey, proof.ToLegacyProof())
}

// handleClueReveal processes a clue reveal event.
func (r *HuntReceiver) handleClueReveal(event *pb.HuntEvent) error {
	var huntID [32]byte
	copy(huntID[:], event.HuntId)

	hunt := r.huntStore.GetHunt(huntID)
	if hunt == nil {
		return ErrHuntNotFound
	}

	// Update clue visibility.
	hunt.RevealClues()
	return nil
}

// handleHuntCompleted processes a hunt completion event.
func (r *HuntReceiver) handleHuntCompleted(event *pb.HuntEvent) error {
	var huntID [32]byte
	copy(huntID[:], event.HuntId)

	hunt := r.huntStore.GetHunt(huntID)
	if hunt == nil {
		return ErrHuntNotFound
	}

	hunt.mu.Lock()
	hunt.State = HuntCompleted
	hunt.mu.Unlock()

	return nil
}

// handleHuntExpired processes a hunt expiration event.
func (r *HuntReceiver) handleHuntExpired(event *pb.HuntEvent) error {
	var huntID [32]byte
	copy(huntID[:], event.HuntId)

	hunt := r.huntStore.GetHunt(huntID)
	if hunt == nil {
		return ErrHuntNotFound
	}

	hunt.mu.Lock()
	hunt.State = HuntExpired
	hunt.mu.Unlock()

	return nil
}

// huntToNetworkProto converts a Hunt to the network protobuf format (pb.Hunt).
// This is different from huntToProto in persistence_hunts.go which uses SpecterHunt.
func huntToNetworkProto(h *Hunt) *pb.Hunt {
	if h == nil {
		return nil
	}

	pbHunt := &pb.Hunt{
		Id:            h.ID[:],
		Theme:         h.Theme,
		Seed:          h.Seed[:],
		InitiatorKey:  h.InitiatorKey[:],
		CreatedAt:     h.CreatedAt.Unix(),
		Duration:      int64(h.Duration.Seconds()),
		ExpiresAt:     h.ExpiresAt.Unix(),
		State:         pb.HuntState(h.State),
		FragmentCount: uint32(h.FragmentCount),
	}

	// Convert fragments.
	pbHunt.Fragments = make([]*pb.Fragment, len(h.Fragments))
	for i, f := range h.Fragments {
		pbHunt.Fragments[i] = fragmentToNetworkProto(f)
	}

	return pbHunt
}

// fragmentToNetworkProto converts a Fragment to protobuf format.
func fragmentToNetworkProto(f *Fragment) *pb.Fragment {
	if f == nil {
		return nil
	}

	pbFrag := &pb.Fragment{
		Index:         uint32(f.Index),
		LocationHash:  f.LocationHash[:],
		TargetPeerId:  f.TargetPeerID,
		Claimed:       f.Claimed,
		CluesRevealed: uint32(f.CluesRevealed),
		Clues:         f.Clues,
	}

	if f.ClaimerKey != nil {
		pbFrag.ClaimerKey = f.ClaimerKey[:]
	}
	if f.ClaimedAt != nil {
		pbFrag.ClaimedAt = f.ClaimedAt.Unix()
	}

	return pbFrag
}

// networkProtoToHunt converts a protobuf Hunt to native format.
func networkProtoToHunt(pbHunt *pb.Hunt) *Hunt {
	if pbHunt == nil {
		return nil
	}

	var id, seed, initiatorKey [32]byte
	copy(id[:], pbHunt.Id)
	copy(seed[:], pbHunt.Seed)
	copy(initiatorKey[:], pbHunt.InitiatorKey)

	hunt := &Hunt{
		ID:            id,
		Theme:         pbHunt.Theme,
		Seed:          seed,
		InitiatorKey:  initiatorKey,
		CreatedAt:     time.Unix(pbHunt.CreatedAt, 0),
		Duration:      time.Duration(pbHunt.Duration) * time.Second,
		ExpiresAt:     time.Unix(pbHunt.ExpiresAt, 0),
		State:         HuntState(pbHunt.State),
		FragmentCount: int(pbHunt.FragmentCount),
	}

	// Convert fragments.
	hunt.Fragments = make([]*Fragment, len(pbHunt.Fragments))
	for i, pbFrag := range pbHunt.Fragments {
		hunt.Fragments[i] = networkProtoToFragment(pbFrag)
	}

	return hunt
}

// networkProtoToFragment converts a protobuf Fragment to native format.
func networkProtoToFragment(pbFrag *pb.Fragment) *Fragment {
	if pbFrag == nil {
		return nil
	}

	var locationHash [32]byte
	copy(locationHash[:], pbFrag.LocationHash)

	frag := &Fragment{
		Index:         int(pbFrag.Index),
		LocationHash:  locationHash,
		TargetPeerID:  pbFrag.TargetPeerId,
		Claimed:       pbFrag.Claimed,
		CluesRevealed: int(pbFrag.CluesRevealed),
		Clues:         pbFrag.Clues,
	}

	if len(pbFrag.ClaimerKey) == 32 {
		var claimerKey [32]byte
		copy(claimerKey[:], pbFrag.ClaimerKey)
		frag.ClaimerKey = &claimerKey
	}

	if pbFrag.ClaimedAt > 0 {
		claimedAt := time.Unix(pbFrag.ClaimedAt, 0)
		frag.ClaimedAt = &claimedAt
	}

	return frag
}

// proximityProofToProto converts a DHTProximityProof to protobuf.
func proximityProofToProto(proof *DHTProximityProof) *pb.ProximityProof {
	if proof == nil {
		return nil
	}

	pbProof := &pb.ProximityProof{
		ClaimerPubkey:    proof.ClaimerPubKey[:],
		ClaimerPeerId:    proof.ClaimerPeerID,
		TargetHash:       proof.TargetHash[:],
		RoutingTableSize: uint32(proof.RoutingTableSize),
		Timestamp:        proof.Timestamp,
	}

	pbProof.Attestations = make([]*pb.ProximityAttestation, len(proof.Attestations))
	for i, att := range proof.Attestations {
		pbProof.Attestations[i] = &pb.ProximityAttestation{
			AttesterPubkey: att.AttesterPubKey[:],
			AttesterPeerId: att.AttesterPeerID,
			ClaimerPubkey:  att.ClaimerPubKey[:],
			TargetHash:     att.TargetHash[:],
			Timestamp:      att.Timestamp,
			XorDistance:    att.XORDistance,
			Signature:      att.Signature[:],
		}
	}

	return pbProof
}

// protoToProximityProof converts a protobuf ProximityProof to native format.
func protoToProximityProof(pbProof *pb.ProximityProof, claimerKey [32]byte) *DHTProximityProof {
	if pbProof == nil {
		return NewDHTProximityProof(claimerKey, "", [32]byte{}, 0)
	}

	var targetHash [32]byte
	copy(targetHash[:], pbProof.TargetHash)

	proof := &DHTProximityProof{
		ClaimerPubKey:    claimerKey,
		ClaimerPeerID:    pbProof.ClaimerPeerId,
		TargetHash:       targetHash,
		RoutingTableSize: int(pbProof.RoutingTableSize),
		Timestamp:        pbProof.Timestamp,
	}

	proof.Attestations = make([]ProximityAttestation, len(pbProof.Attestations))
	for i, pbAtt := range pbProof.Attestations {
		var attesterPubKey, attClaimerPubKey, attTargetHash [32]byte
		var signature [64]byte

		copy(attesterPubKey[:], pbAtt.AttesterPubkey)
		copy(attClaimerPubKey[:], pbAtt.ClaimerPubkey)
		copy(attTargetHash[:], pbAtt.TargetHash)
		copy(signature[:], pbAtt.Signature)

		proof.Attestations[i] = ProximityAttestation{
			AttesterPubKey: attesterPubKey,
			AttesterPeerID: pbAtt.AttesterPeerId,
			ClaimerPubKey:  attClaimerPubKey,
			TargetHash:     attTargetHash,
			Timestamp:      pbAtt.Timestamp,
			XORDistance:    pbAtt.XorDistance,
			Signature:      signature,
		}
	}

	return proof
}
