// Package mechanics - Phantom Council network propagation.
// Per ROADMAP.md line 547, broadcasts council creation, admission, proposals, votes.
package councils

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

// CouncilPublisher handles publishing council events to the anonymous mechanics topic.
// All council events are broadcast on TopicAnonymousMechanics (/murmur/anonymous/mechanics/1.0).
type CouncilPublisher struct {
	publisher  Publisher
	topic      string
	privateKey ed25519.PrivateKey
}

// NewCouncilPublisher creates a new council publisher.
// privateKey is used to sign events; it can be nil if only receiving events.
func NewCouncilPublisher(pub Publisher, privateKey ed25519.PrivateKey) *CouncilPublisher {
	return &CouncilPublisher{
		publisher:  pub,
		topic:      TopicAnonymousMechanics,
		privateKey: privateKey,
	}
}

// PublishCouncilCreated broadcasts a new council creation event.
func (c *CouncilPublisher) PublishCouncilCreated(ctx context.Context, council *PhantomCouncil) error {
	if c.publisher == nil {
		return ErrPublisherNotSet
	}
	if council == nil {
		return fmt.Errorf("council cannot be nil")
	}

	pbCouncil := councilToProto(council)
	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_CREATED,
		Council:   pbCouncil,
		CouncilId: council.ID[:],
		Timestamp: time.Now().Unix(),
	}

	return c.signAndPublish(ctx, event)
}

// PublishMemberJoined broadcasts a member join event.
func (c *CouncilPublisher) PublishMemberJoined(ctx context.Context, councilID [32]byte, member *CouncilMember) error {
	if c.publisher == nil {
		return ErrPublisherNotSet
	}
	if member == nil {
		return fmt.Errorf("member cannot be nil")
	}

	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_MEMBER_JOIN,
		CouncilId: councilID[:],
		Member: &pb.CouncilMember{
			SpecterPubkey: member.SpecterKey[:],
			JoinedAt:      member.JoinedAt.Unix(),
			VoteWeight:    1,
			Role:          pb.CouncilRole_COUNCIL_ROLE_MEMBER,
		},
		Timestamp: time.Now().Unix(),
	}

	return c.signAndPublish(ctx, event)
}

// PublishProposal broadcasts a new proposal event.
func (c *CouncilPublisher) PublishProposal(ctx context.Context, councilID [32]byte, proposal *CouncilProposal) error {
	if c.publisher == nil {
		return ErrPublisherNotSet
	}
	if proposal == nil {
		return fmt.Errorf("proposal cannot be nil")
	}

	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_PROPOSAL,
		CouncilId: councilID[:],
		Proposal: &pb.CouncilProposal{
			Id:             proposal.ID[:],
			ProposerPubkey: proposal.ProposerKey[:],
			Title:          proposal.Text,
			CreatedAt:      proposal.CreatedAt.Unix(),
			VotingEndsAt:   proposal.CreatedAt.Add(72 * time.Hour).Unix(),
			State:          pb.ProposalState_PROPOSAL_STATE_PENDING,
		},
		Timestamp: time.Now().Unix(),
	}

	return c.signAndPublish(ctx, event)
}

// PublishVote broadcasts a vote event.
func (c *CouncilPublisher) PublishVote(ctx context.Context, councilID, proposalID, voterKey [32]byte, vote VoteValue) error {
	if c.publisher == nil {
		return ErrPublisherNotSet
	}

	voteChoice := pb.VoteChoice_VOTE_CHOICE_UNSPECIFIED
	switch vote {
	case VoteFor:
		voteChoice = pb.VoteChoice_VOTE_CHOICE_YES
	case VoteAgainst:
		voteChoice = pb.VoteChoice_VOTE_CHOICE_NO
	case VoteAbstain:
		voteChoice = pb.VoteChoice_VOTE_CHOICE_ABSTAIN
	}

	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_VOTE,
		CouncilId: councilID[:],
		Vote: &pb.CouncilVote{
			VoterPubkey: voterKey[:],
			Choice:      voteChoice,
			Timestamp:   time.Now().Unix(),
		},
		// Store proposal ID in the Proposal field for reference.
		Proposal: &pb.CouncilProposal{
			Id: proposalID[:],
		},
		Timestamp: time.Now().Unix(),
	}

	return c.signAndPublish(ctx, event)
}

// PublishProposalResolved broadcasts a proposal resolution event.
func (c *CouncilPublisher) PublishProposalResolved(ctx context.Context, councilID [32]byte, proposal *CouncilProposal) error {
	if c.publisher == nil {
		return ErrPublisherNotSet
	}
	if proposal == nil {
		return fmt.Errorf("proposal cannot be nil")
	}

	state := pb.ProposalState_PROPOSAL_STATE_REJECTED
	if proposal.Passed {
		state = pb.ProposalState_PROPOSAL_STATE_PASSED
	}

	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_RESOLVED,
		CouncilId: councilID[:],
		Proposal: &pb.CouncilProposal{
			Id:             proposal.ID[:],
			ProposerPubkey: proposal.ProposerKey[:],
			Title:          proposal.Text,
			CreatedAt:      proposal.CreatedAt.Unix(),
			State:          state,
		},
		Timestamp: time.Now().Unix(),
	}

	return c.signAndPublish(ctx, event)
}

// PublishCouncilDissolved broadcasts a council dissolution event.
func (c *CouncilPublisher) PublishCouncilDissolved(ctx context.Context, councilID [32]byte) error {
	if c.publisher == nil {
		return ErrPublisherNotSet
	}

	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_DISSOLVED,
		CouncilId: councilID[:],
		Timestamp: time.Now().Unix(),
	}

	return c.signAndPublish(ctx, event)
}

// signAndPublish signs the event and publishes it to the topic.
func (c *CouncilPublisher) signAndPublish(ctx context.Context, event *pb.CouncilEvent) error {
	if c.privateKey == nil {
		return ErrMissingPrivateKey
	}

	// Create signature over event data.
	sigData := c.eventSignatureData(event)
	signature := ed25519.Sign(c.privateKey, sigData)
	event.Signature = signature

	// Wrap in GossipMessage.
	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_CouncilEvent{
			CouncilEvent: event,
		},
	}

	data, err := proto.Marshal(gossipMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal council event: %w", err)
	}

	return c.publisher.Publish(ctx, c.topic, data)
}

// eventSignatureData creates the data to be signed for an event.
func (c *CouncilPublisher) eventSignatureData(event *pb.CouncilEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("council-event-v1"))
	binary.Write(hash, binary.BigEndian, int32(event.EventType))
	hash.Write(event.CouncilId)
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	if event.Member != nil {
		hash.Write(event.Member.SpecterPubkey)
	}
	if event.Proposal != nil {
		hash.Write(event.Proposal.Id)
	}
	if event.Vote != nil {
		hash.Write(event.Vote.VoterPubkey)
	}
	return hash.Sum(nil)
}

// CouncilReceiver handles incoming council events from the network.
type CouncilReceiver struct {
	councilStore *CouncilStore
}

// NewCouncilReceiver creates a new council receiver.
func NewCouncilReceiver(store *CouncilStore) *CouncilReceiver {
	return &CouncilReceiver{
		councilStore: store,
	}
}

// HandleMessage processes an incoming council event.
func (r *CouncilReceiver) HandleMessage(data []byte) error {
	var gossipMsg pb.GossipMessage
	if err := proto.Unmarshal(data, &gossipMsg); err != nil {
		return fmt.Errorf("failed to unmarshal gossip message: %w", err)
	}

	councilEvent := gossipMsg.GetCouncilEvent()
	if councilEvent == nil {
		return nil // Not a council event.
	}

	// Verify signature.
	if err := r.verifyEventSignature(councilEvent); err != nil {
		return err
	}

	return r.processEvent(councilEvent)
}

// verifyEventSignature checks the event signature.
func (r *CouncilReceiver) verifyEventSignature(event *pb.CouncilEvent) error {
	if len(event.Signature) == 0 {
		return ErrMissingSignature
	}

	// For council events, we need to extract the sender's public key.
	// This can come from different places depending on event type.
	var senderPubkey []byte

	switch event.EventType {
	case pb.CouncilEventType_COUNCIL_EVENT_CREATED:
		if event.Council != nil {
			senderPubkey = event.Council.FounderPubkey
		}
	case pb.CouncilEventType_COUNCIL_EVENT_MEMBER_JOIN:
		if event.Member != nil {
			senderPubkey = event.Member.SpecterPubkey
		}
	case pb.CouncilEventType_COUNCIL_EVENT_PROPOSAL:
		if event.Proposal != nil {
			senderPubkey = event.Proposal.ProposerPubkey
		}
	case pb.CouncilEventType_COUNCIL_EVENT_VOTE:
		if event.Vote != nil {
			senderPubkey = event.Vote.VoterPubkey
		}
	case pb.CouncilEventType_COUNCIL_EVENT_RESOLVED, pb.CouncilEventType_COUNCIL_EVENT_DISSOLVED:
		// For these events, we get the founder key from the stored council.
		// Only the council founder can dissolve or resolve proposals.
		if len(event.CouncilId) == 32 {
			var councilID [32]byte
			copy(councilID[:], event.CouncilId)
			council := r.councilStore.GetCouncil(councilID)
			if council != nil {
				senderPubkey = council.CreatorKey[:]
			}
		}
	}

	if len(senderPubkey) != ed25519.PublicKeySize {
		return ErrSignatureFailed
	}

	sigData := r.eventSignatureData(event)
	if !ed25519.Verify(senderPubkey, sigData, event.Signature) {
		return ErrSignatureFailed
	}

	return nil
}

// eventSignatureData creates the data that was signed.
func (r *CouncilReceiver) eventSignatureData(event *pb.CouncilEvent) []byte {
	hash := blake3.New()
	hash.Write([]byte("council-event-v1"))
	binary.Write(hash, binary.BigEndian, int32(event.EventType))
	hash.Write(event.CouncilId)
	binary.Write(hash, binary.BigEndian, event.Timestamp)
	if event.Member != nil {
		hash.Write(event.Member.SpecterPubkey)
	}
	if event.Proposal != nil {
		hash.Write(event.Proposal.Id)
	}
	if event.Vote != nil {
		hash.Write(event.Vote.VoterPubkey)
	}
	return hash.Sum(nil)
}

// processEvent handles the specific event type.
func (r *CouncilReceiver) processEvent(event *pb.CouncilEvent) error {
	switch event.EventType {
	case pb.CouncilEventType_COUNCIL_EVENT_CREATED:
		return r.handleCouncilCreated(event)
	case pb.CouncilEventType_COUNCIL_EVENT_MEMBER_JOIN:
		return r.handleMemberJoined(event)
	case pb.CouncilEventType_COUNCIL_EVENT_PROPOSAL:
		return r.handleProposal(event)
	case pb.CouncilEventType_COUNCIL_EVENT_VOTE:
		return r.handleVote(event)
	case pb.CouncilEventType_COUNCIL_EVENT_RESOLVED:
		return r.handleProposalResolved(event)
	case pb.CouncilEventType_COUNCIL_EVENT_DISSOLVED:
		return r.handleCouncilDissolved(event)
	default:
		return nil // Ignore unknown event types.
	}
}

// handleCouncilCreated processes a council creation event.
func (r *CouncilReceiver) handleCouncilCreated(event *pb.CouncilEvent) error {
	if event.Council == nil {
		return fmt.Errorf("council creation event missing council data")
	}

	council := protoToCouncil(event.Council)
	if council == nil {
		return fmt.Errorf("failed to convert council from protobuf")
	}

	// Check if council already exists.
	var councilID [32]byte
	copy(councilID[:], event.CouncilId)
	if existing := r.councilStore.GetCouncil(councilID); existing != nil {
		return nil // Already exists, skip.
	}

	r.councilStore.AddCouncil(council)
	return nil
}

// handleMemberJoined processes a member join event.
func (r *CouncilReceiver) handleMemberJoined(event *pb.CouncilEvent) error {
	if event.Member == nil {
		return fmt.Errorf("member join event missing member data")
	}

	var councilID [32]byte
	copy(councilID[:], event.CouncilId)

	council := r.councilStore.GetCouncil(councilID)
	if council == nil {
		return fmt.Errorf("council not found")
	}

	var memberKey [32]byte
	copy(memberKey[:], event.Member.SpecterPubkey)

	// Check if already a member.
	council.mu.Lock()
	defer council.mu.Unlock()

	keyHex := fmt.Sprintf("%x", memberKey[:])
	if _, exists := council.memberByKey[keyHex]; exists {
		return nil // Already a member.
	}

	member := &CouncilMember{
		SpecterKey: memberKey,
		Status:     MemberActive,
		JoinedAt:   time.Unix(event.Member.JoinedAt, 0),
	}

	council.Members = append(council.Members, member)
	council.memberByKey[keyHex] = member

	return nil
}

// handleProposal processes a new proposal event.
func (r *CouncilReceiver) handleProposal(event *pb.CouncilEvent) error {
	if event.Proposal == nil {
		return fmt.Errorf("proposal event missing proposal data")
	}

	var councilID [32]byte
	copy(councilID[:], event.CouncilId)

	council := r.councilStore.GetCouncil(councilID)
	if council == nil {
		return fmt.Errorf("council not found")
	}

	var proposalID [32]byte
	copy(proposalID[:], event.Proposal.Id)

	var proposerKey [32]byte
	copy(proposerKey[:], event.Proposal.ProposerPubkey)

	council.mu.Lock()
	defer council.mu.Unlock()

	// Check if proposal already exists.
	for _, p := range council.Proposals {
		if p.ID == proposalID {
			return nil // Already exists.
		}
	}

	proposal := &CouncilProposal{
		ID:          proposalID,
		ProposerKey: proposerKey,
		Text:        event.Proposal.Title,
		CreatedAt:   time.Unix(event.Proposal.CreatedAt, 0),
		Votes:       make(map[string]VoteValue),
		Resolved:    false,
		Passed:      false,
	}

	council.Proposals = append(council.Proposals, proposal)
	return nil
}

// handleVote processes a vote event.
func (r *CouncilReceiver) handleVote(event *pb.CouncilEvent) error {
	if event.Vote == nil {
		return fmt.Errorf("vote event missing vote data")
	}

	var councilID [32]byte
	copy(councilID[:], event.CouncilId)

	council := r.councilStore.GetCouncil(councilID)
	if council == nil {
		return fmt.Errorf("council not found")
	}

	// Proposal ID comes from the Proposal field for vote events.
	if event.Proposal == nil {
		return fmt.Errorf("vote event missing proposal reference")
	}

	var proposalID [32]byte
	copy(proposalID[:], event.Proposal.Id)

	var voterKey [32]byte
	copy(voterKey[:], event.Vote.VoterPubkey)

	council.mu.Lock()
	defer council.mu.Unlock()

	// Find the proposal.
	var proposal *CouncilProposal
	for _, p := range council.Proposals {
		if p.ID == proposalID {
			proposal = p
			break
		}
	}

	if proposal == nil {
		return ErrCouncilProposalNotFound
	}

	// Convert vote choice.
	var vote VoteValue
	switch event.Vote.Choice {
	case pb.VoteChoice_VOTE_CHOICE_YES:
		vote = VoteFor
	case pb.VoteChoice_VOTE_CHOICE_NO:
		vote = VoteAgainst
	default:
		vote = VoteAbstain
	}

	voterHex := fmt.Sprintf("%x", voterKey[:])
	proposal.Votes[voterHex] = vote

	return nil
}

// handleProposalResolved processes a proposal resolution event.
func (r *CouncilReceiver) handleProposalResolved(event *pb.CouncilEvent) error {
	if event.Proposal == nil {
		return fmt.Errorf("proposal resolved event missing proposal data")
	}

	var councilID [32]byte
	copy(councilID[:], event.CouncilId)

	council := r.councilStore.GetCouncil(councilID)
	if council == nil {
		return fmt.Errorf("council not found")
	}

	var proposalID [32]byte
	copy(proposalID[:], event.Proposal.Id)

	council.mu.Lock()
	defer council.mu.Unlock()

	for _, p := range council.Proposals {
		if p.ID == proposalID {
			p.Resolved = true
			p.Passed = event.Proposal.State == pb.ProposalState_PROPOSAL_STATE_PASSED
			return nil
		}
	}

	return ErrCouncilProposalNotFound
}

// handleCouncilDissolved processes a council dissolution event.
func (r *CouncilReceiver) handleCouncilDissolved(event *pb.CouncilEvent) error {
	var councilID [32]byte
	copy(councilID[:], event.CouncilId)

	council := r.councilStore.GetCouncil(councilID)
	if council == nil {
		return nil // Already doesn't exist.
	}

	council.mu.Lock()
	council.State = CouncilDisbanded
	council.mu.Unlock()

	return nil
}

// GetCouncilStore returns the underlying council store.
func (r *CouncilReceiver) GetCouncilStore() *CouncilStore {
	return r.councilStore
}
