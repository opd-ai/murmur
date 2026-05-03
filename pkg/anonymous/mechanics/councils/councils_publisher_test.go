// Package mechanics - Phantom Council network propagation tests.
// Per ROADMAP.md line 547, tests for council event publishing and receiving.
package councils

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

// TestCouncilPublisher_Creation tests CouncilPublisher instantiation.
func TestCouncilPublisher_Creation(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	pub := NewCouncilPublisher(mockPub, privateKey)
	if pub == nil {
		t.Fatal("NewCouncilPublisher returned nil")
	}
	if pub.topic != mechanics.TopicAnonymousMechanics {
		t.Errorf("wrong topic: got %s, want %s", pub.topic, mechanics.TopicAnonymousMechanics)
	}
}

// TestCouncilPublisher_NilPublisher tests handling when publisher is nil.
func TestCouncilPublisher_NilPublisher(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewCouncilPublisher(nil, privateKey)

	council := &PhantomCouncil{
		Name:      "Test",
		CreatedAt: time.Now(),
	}

	err := pub.PublishCouncilCreated(context.Background(), council)
	if err != mechanics.ErrPublisherNotSet {
		t.Errorf("expected mechanics.ErrPublisherNotSet, got %v", err)
	}
}

// TestCouncilPublisher_NilCouncil tests handling when council is nil.
func TestCouncilPublisher_NilCouncil(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	pub := NewCouncilPublisher(mockPub, privateKey)

	err := pub.PublishCouncilCreated(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil council")
	}
}

// TestCouncilPublisher_NilPrivateKey tests handling when private key is nil.
func TestCouncilPublisher_NilPrivateKey(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	pub := NewCouncilPublisher(mockPub, nil)

	council := &PhantomCouncil{
		Name:      "Test",
		CreatedAt: time.Now(),
	}

	err := pub.PublishCouncilCreated(context.Background(), council)
	if err != mechanics.ErrMissingPrivateKey {
		t.Errorf("expected mechanics.ErrMissingPrivateKey, got %v", err)
	}
}

// TestCouncilPublisher_PublishCouncilCreated tests successful council creation publication.
func TestCouncilPublisher_PublishCouncilCreated(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	pub := NewCouncilPublisher(mockPub, privateKey)

	var creatorKey [32]byte
	copy(creatorKey[:], pubKey)

	council := &PhantomCouncil{
		Name:             "Test Council",
		CreatorKey:       creatorKey,
		CreatedAt:        time.Now(),
		MinResonance:     200,
		State:            CouncilActive,
		memberByKey:      make(map[string]*CouncilMember),
		applicationByKey: make(map[string]*CouncilApplication),
	}
	rand.Read(council.ID[:])

	err := pub.PublishCouncilCreated(context.Background(), council)
	if err != nil {
		t.Fatalf("PublishCouncilCreated failed: %v", err)
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

	councilEvent := gossipMsg.GetCouncilEvent()
	if councilEvent == nil {
		t.Fatal("no council event in message")
	}
	if councilEvent.EventType != pb.CouncilEventType_COUNCIL_EVENT_CREATED {
		t.Errorf("wrong event type: got %v", councilEvent.EventType)
	}
	if len(councilEvent.Signature) == 0 {
		t.Error("event has no signature")
	}
}

// TestCouncilPublisher_PublishMemberJoined tests member join publication.
func TestCouncilPublisher_PublishMemberJoined(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	pub := NewCouncilPublisher(mockPub, privateKey)

	var councilID [32]byte
	rand.Read(councilID[:])

	var memberKey [32]byte
	copy(memberKey[:], pubKey)

	member := &CouncilMember{
		SpecterKey: memberKey,
		Status:     MemberActive,
		JoinedAt:   time.Now(),
	}

	err := pub.PublishMemberJoined(context.Background(), councilID, member)
	if err != nil {
		t.Fatalf("PublishMemberJoined failed: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mockPub.Published))
	}

	var gossipMsg pb.GossipMessage
	proto.Unmarshal(mockPub.Published[0].Data, &gossipMsg)
	event := gossipMsg.GetCouncilEvent()

	if event.EventType != pb.CouncilEventType_COUNCIL_EVENT_MEMBER_JOIN {
		t.Errorf("wrong event type: got %v", event.EventType)
	}
}

// TestCouncilPublisher_PublishProposal tests proposal publication.
func TestCouncilPublisher_PublishProposal(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	pub := NewCouncilPublisher(mockPub, privateKey)

	var councilID [32]byte
	rand.Read(councilID[:])

	var proposerKey [32]byte
	copy(proposerKey[:], pubKey)

	proposal := &CouncilProposal{
		ProposerKey: proposerKey,
		Text:        "Test Proposal",
		CreatedAt:   time.Now(),
		Votes:       make(map[string]VoteValue),
	}
	rand.Read(proposal.ID[:])

	err := pub.PublishProposal(context.Background(), councilID, proposal)
	if err != nil {
		t.Fatalf("PublishProposal failed: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mockPub.Published))
	}

	var gossipMsg pb.GossipMessage
	proto.Unmarshal(mockPub.Published[0].Data, &gossipMsg)
	event := gossipMsg.GetCouncilEvent()

	if event.EventType != pb.CouncilEventType_COUNCIL_EVENT_PROPOSAL {
		t.Errorf("wrong event type: got %v", event.EventType)
	}
}

// TestCouncilPublisher_PublishVote tests vote publication.
func TestCouncilPublisher_PublishVote(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	pub := NewCouncilPublisher(mockPub, privateKey)

	var councilID, proposalID, voterKey [32]byte
	rand.Read(councilID[:])
	rand.Read(proposalID[:])
	copy(voterKey[:], pubKey)

	err := pub.PublishVote(context.Background(), councilID, proposalID, voterKey, VoteFor)
	if err != nil {
		t.Fatalf("PublishVote failed: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mockPub.Published))
	}

	var gossipMsg pb.GossipMessage
	proto.Unmarshal(mockPub.Published[0].Data, &gossipMsg)
	event := gossipMsg.GetCouncilEvent()

	if event.EventType != pb.CouncilEventType_COUNCIL_EVENT_VOTE {
		t.Errorf("wrong event type: got %v", event.EventType)
	}
	if event.Vote.Choice != pb.VoteChoice_VOTE_CHOICE_YES {
		t.Errorf("wrong vote choice: got %v", event.Vote.Choice)
	}
}

// TestCouncilPublisher_PublishProposalResolved tests proposal resolution publication.
func TestCouncilPublisher_PublishProposalResolved(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	pub := NewCouncilPublisher(mockPub, privateKey)

	var councilID [32]byte
	rand.Read(councilID[:])

	var proposerKey [32]byte
	copy(proposerKey[:], pubKey)

	proposal := &CouncilProposal{
		ProposerKey: proposerKey,
		Text:        "Resolved Proposal",
		CreatedAt:   time.Now(),
		Resolved:    true,
		Passed:      true,
		Votes:       make(map[string]VoteValue),
	}
	rand.Read(proposal.ID[:])

	err := pub.PublishProposalResolved(context.Background(), councilID, proposal)
	if err != nil {
		t.Fatalf("PublishProposalResolved failed: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mockPub.Published))
	}

	var gossipMsg pb.GossipMessage
	proto.Unmarshal(mockPub.Published[0].Data, &gossipMsg)
	event := gossipMsg.GetCouncilEvent()

	if event.EventType != pb.CouncilEventType_COUNCIL_EVENT_RESOLVED {
		t.Errorf("wrong event type: got %v", event.EventType)
	}
}

// TestCouncilPublisher_PublishCouncilDissolved tests dissolution publication.
func TestCouncilPublisher_PublishCouncilDissolved(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	pub := NewCouncilPublisher(mockPub, privateKey)

	var councilID [32]byte
	rand.Read(councilID[:])

	err := pub.PublishCouncilDissolved(context.Background(), councilID)
	if err != nil {
		t.Fatalf("PublishCouncilDissolved failed: %v", err)
	}

	if len(mockPub.Published) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mockPub.Published))
	}

	var gossipMsg pb.GossipMessage
	proto.Unmarshal(mockPub.Published[0].Data, &gossipMsg)
	event := gossipMsg.GetCouncilEvent()

	if event.EventType != pb.CouncilEventType_COUNCIL_EVENT_DISSOLVED {
		t.Errorf("wrong event type: got %v", event.EventType)
	}
}

// TestCouncilReceiver_Creation tests CouncilReceiver instantiation.
func TestCouncilReceiver_Creation(t *testing.T) {
	store := NewCouncilStore()
	receiver := NewCouncilReceiver(store)
	if receiver == nil {
		t.Fatal("NewCouncilReceiver returned nil")
	}
	if receiver.GetCouncilStore() != store {
		t.Error("GetCouncilStore returned wrong store")
	}
}

// TestCouncilReceiver_HandleMessage_NonCouncilEvent tests ignoring non-council events.
func TestCouncilReceiver_HandleMessage_NonCouncilEvent(t *testing.T) {
	store := NewCouncilStore()
	receiver := NewCouncilReceiver(store)

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
		t.Errorf("expected no error for non-council event, got %v", err)
	}
}

// TestCouncilReceiver_HandleMessage_InvalidData tests handling of invalid data.
func TestCouncilReceiver_HandleMessage_InvalidData(t *testing.T) {
	store := NewCouncilStore()
	receiver := NewCouncilReceiver(store)

	err := receiver.HandleMessage([]byte("invalid data"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

// TestCouncilReceiver_HandleMessage_MissingSignature tests rejection of unsigned events.
func TestCouncilReceiver_HandleMessage_MissingSignature(t *testing.T) {
	store := NewCouncilStore()
	receiver := NewCouncilReceiver(store)

	var founderKey [32]byte
	rand.Read(founderKey[:])

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_CouncilEvent{
			CouncilEvent: &pb.CouncilEvent{
				EventType: pb.CouncilEventType_COUNCIL_EVENT_CREATED,
				Council: &pb.PhantomCouncil{
					Id:            make([]byte, 32),
					FounderPubkey: founderKey[:],
					CreatedAt:     time.Now().Unix(),
				},
				CouncilId: make([]byte, 32),
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

// TestCouncilReceiver_HandleMessage_ValidCouncilCreated tests successful council reception.
func TestCouncilReceiver_HandleMessage_ValidCouncilCreated(t *testing.T) {
	store := NewCouncilStore()
	receiver := NewCouncilReceiver(store)

	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var founderKey [32]byte
	copy(founderKey[:], pubKey)

	var councilID [32]byte
	rand.Read(councilID[:])

	now := time.Now()
	pbCouncil := &pb.PhantomCouncil{
		Id:            councilID[:],
		Name:          "Test Council",
		FounderPubkey: founderKey[:],
		CreatedAt:     now.Unix(),
		MinResonance:  200,
		State:         pb.CouncilState_COUNCIL_STATE_ACTIVE,
	}

	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_CREATED,
		Council:   pbCouncil,
		CouncilId: councilID[:],
		Timestamp: now.Unix(),
	}

	// Sign the event.
	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_CouncilEvent{
			CouncilEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Verify council was added.
	retrieved := store.GetCouncil(councilID)
	if retrieved == nil {
		t.Fatal("council not found in store")
	}
	if retrieved.Name != "Test Council" {
		t.Errorf("wrong name: got %q", retrieved.Name)
	}
}

// TestCouncilReceiver_HandleMessage_MemberJoined tests member join processing.
func TestCouncilReceiver_HandleMessage_MemberJoined(t *testing.T) {
	store := NewCouncilStore()
	receiver := NewCouncilReceiver(store)

	// First create a council.
	pubKey, _, _ := ed25519.GenerateKey(rand.Reader)

	var creatorKey [32]byte
	copy(creatorKey[:], pubKey)

	council := &PhantomCouncil{
		Name:             "Test Council",
		CreatorKey:       creatorKey,
		CreatedAt:        time.Now(),
		State:            CouncilActive,
		memberByKey:      make(map[string]*CouncilMember),
		applicationByKey: make(map[string]*CouncilApplication),
	}
	rand.Read(council.ID[:])
	store.AddCouncil(council)

	// Now create a member join event.
	memberPubKey, memberPrivKey, _ := ed25519.GenerateKey(rand.Reader)
	var memberKey [32]byte
	copy(memberKey[:], memberPubKey)

	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_MEMBER_JOIN,
		CouncilId: council.ID[:],
		Member: &pb.CouncilMember{
			SpecterPubkey: memberKey[:],
			JoinedAt:      time.Now().Unix(),
			VoteWeight:    1,
		},
		Timestamp: time.Now().Unix(),
	}

	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(memberPrivKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_CouncilEvent{
			CouncilEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Verify member was added.
	council = store.GetCouncil(council.ID)
	if len(council.Members) != 1 {
		t.Errorf("expected 1 member, got %d", len(council.Members))
	}
}

// TestCouncilReceiver_HandleMessage_Proposal tests proposal processing.
func TestCouncilReceiver_HandleMessage_Proposal(t *testing.T) {
	store := NewCouncilStore()
	receiver := NewCouncilReceiver(store)

	// First create a council.
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var creatorKey [32]byte
	copy(creatorKey[:], pubKey)

	council := &PhantomCouncil{
		Name:             "Test Council",
		CreatorKey:       creatorKey,
		CreatedAt:        time.Now(),
		State:            CouncilActive,
		memberByKey:      make(map[string]*CouncilMember),
		applicationByKey: make(map[string]*CouncilApplication),
	}
	rand.Read(council.ID[:])
	store.AddCouncil(council)

	// Create a proposal event.
	var proposalID [32]byte
	rand.Read(proposalID[:])

	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_PROPOSAL,
		CouncilId: council.ID[:],
		Proposal: &pb.CouncilProposal{
			Id:             proposalID[:],
			ProposerPubkey: creatorKey[:],
			Title:          "Test Proposal",
			CreatedAt:      time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	}

	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_CouncilEvent{
			CouncilEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Verify proposal was added.
	council = store.GetCouncil(council.ID)
	if len(council.Proposals) != 1 {
		t.Errorf("expected 1 proposal, got %d", len(council.Proposals))
	}
}

// TestCouncilReceiver_HandleMessage_Vote tests vote processing.
func TestCouncilReceiver_HandleMessage_Vote(t *testing.T) {
	store := NewCouncilStore()
	receiver := NewCouncilReceiver(store)

	// First create a council with a proposal.
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var creatorKey [32]byte
	copy(creatorKey[:], pubKey)

	var proposalID [32]byte
	rand.Read(proposalID[:])

	council := &PhantomCouncil{
		Name:             "Test Council",
		CreatorKey:       creatorKey,
		CreatedAt:        time.Now(),
		State:            CouncilActive,
		memberByKey:      make(map[string]*CouncilMember),
		applicationByKey: make(map[string]*CouncilApplication),
		Proposals: []*CouncilProposal{
			{
				ID:          proposalID,
				ProposerKey: creatorKey,
				Text:        "Test",
				CreatedAt:   time.Now(),
				Votes:       make(map[string]VoteValue),
			},
		},
	}
	rand.Read(council.ID[:])
	store.AddCouncil(council)

	// Create a vote event.
	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_VOTE,
		CouncilId: council.ID[:],
		Vote: &pb.CouncilVote{
			VoterPubkey: creatorKey[:],
			Choice:      pb.VoteChoice_VOTE_CHOICE_YES,
			Timestamp:   time.Now().Unix(),
		},
		Proposal: &pb.CouncilProposal{
			Id: proposalID[:],
		},
		Timestamp: time.Now().Unix(),
	}

	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_CouncilEvent{
			CouncilEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Verify vote was recorded.
	council = store.GetCouncil(council.ID)
	if len(council.Proposals[0].Votes) != 1 {
		t.Errorf("expected 1 vote, got %d", len(council.Proposals[0].Votes))
	}
}

// TestCouncilReceiver_HandleMessage_Dissolved tests dissolution processing.
func TestCouncilReceiver_HandleMessage_Dissolved(t *testing.T) {
	store := NewCouncilStore()
	receiver := NewCouncilReceiver(store)

	// First create a council.
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var creatorKey [32]byte
	copy(creatorKey[:], pubKey)

	council := &PhantomCouncil{
		Name:             "Test Council",
		CreatorKey:       creatorKey,
		CreatedAt:        time.Now(),
		State:            CouncilActive,
		memberByKey:      make(map[string]*CouncilMember),
		applicationByKey: make(map[string]*CouncilApplication),
	}
	rand.Read(council.ID[:])
	store.AddCouncil(council)

	// Create a dissolution event.
	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_DISSOLVED,
		CouncilId: council.ID[:],
		Timestamp: time.Now().Unix(),
	}

	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_CouncilEvent{
			CouncilEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)
	err := receiver.HandleMessage(data)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}

	// Verify council was disbanded.
	council = store.GetCouncil(council.ID)
	if council.State != CouncilDisbanded {
		t.Errorf("expected disbanded state, got %v", council.State)
	}
}

// TestCouncilPublisher_RoundTrip tests publishing and receiving council events.
func TestCouncilPublisher_RoundTrip(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	publisher := NewCouncilPublisher(mockPub, privateKey)
	store := NewCouncilStore()
	receiver := NewCouncilReceiver(store)

	var creatorKey [32]byte
	copy(creatorKey[:], pubKey)

	council := &PhantomCouncil{
		Name:             "Round-Trip Council",
		CreatorKey:       creatorKey,
		CreatedAt:        time.Now(),
		MinResonance:     200,
		State:            CouncilActive,
		memberByKey:      make(map[string]*CouncilMember),
		applicationByKey: make(map[string]*CouncilApplication),
	}
	rand.Read(council.ID[:])

	// Publish.
	err := publisher.PublishCouncilCreated(context.Background(), council)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	// Receive.
	err = receiver.HandleMessage(mockPub.Published[0].Data)
	if err != nil {
		t.Fatalf("receive failed: %v", err)
	}

	// Verify.
	retrieved := store.GetCouncil(council.ID)
	if retrieved == nil {
		t.Fatal("council not found after round-trip")
	}
	if retrieved.Name != council.Name {
		t.Errorf("name mismatch: got %q, want %q", retrieved.Name, council.Name)
	}
}

// TestCouncilPublisher_SignatureDataConsistency tests signature data consistency.
func TestCouncilPublisher_SignatureDataConsistency(t *testing.T) {
	mockPub := &mechanics.MockPublisher{}
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	publisher := NewCouncilPublisher(mockPub, privateKey)
	receiver := NewCouncilReceiver(NewCouncilStore())

	var councilID [32]byte
	rand.Read(councilID[:])

	now := time.Now()
	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_DISSOLVED,
		CouncilId: councilID[:],
		Timestamp: now.Unix(),
	}

	pubSigData := publisher.eventSignatureData(event)
	recSigData := receiver.eventSignatureData(event)

	if string(pubSigData) != string(recSigData) {
		t.Error("signature data mismatch between publisher and receiver")
	}
}

// BenchmarkCouncilPublisher_Publish benchmarks council publishing.
func BenchmarkCouncilPublisher_Publish(b *testing.B) {
	mockPub := &mechanics.MockPublisher{}
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	publisher := NewCouncilPublisher(mockPub, privateKey)

	var creatorKey [32]byte
	copy(creatorKey[:], pubKey)

	council := &PhantomCouncil{
		Name:             "Benchmark Council",
		CreatorKey:       creatorKey,
		CreatedAt:        time.Now(),
		State:            CouncilActive,
		memberByKey:      make(map[string]*CouncilMember),
		applicationByKey: make(map[string]*CouncilApplication),
	}
	rand.Read(council.ID[:])

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = publisher.PublishCouncilCreated(ctx, council)
	}
}

// BenchmarkCouncilReceiver_HandleMessage benchmarks council receiving.
func BenchmarkCouncilReceiver_HandleMessage(b *testing.B) {
	pubKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	var founderKey [32]byte
	copy(founderKey[:], pubKey)

	now := time.Now()
	pbCouncil := &pb.PhantomCouncil{
		Id:            make([]byte, 32),
		Name:          "Benchmark Council",
		FounderPubkey: founderKey[:],
		CreatedAt:     now.Unix(),
		State:         pb.CouncilState_COUNCIL_STATE_ACTIVE,
	}

	event := &pb.CouncilEvent{
		EventType: pb.CouncilEventType_COUNCIL_EVENT_CREATED,
		Council:   pbCouncil,
		CouncilId: pbCouncil.Id,
		Timestamp: now.Unix(),
	}

	store := NewCouncilStore()
	receiver := NewCouncilReceiver(store)

	sigData := receiver.eventSignatureData(event)
	event.Signature = ed25519.Sign(privateKey, sigData)

	gossipMsg := &pb.GossipMessage{
		Content: &pb.GossipMessage_CouncilEvent{
			CouncilEvent: event,
		},
	}

	data, _ := proto.Marshal(gossipMsg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Generate unique ID for each iteration.
		rand.Read(pbCouncil.Id)
		event.CouncilId = pbCouncil.Id
		data, _ = proto.Marshal(gossipMsg)
		_ = receiver.HandleMessage(data)
	}
}
