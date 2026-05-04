// Package mechanics - Phantom Councils persistence.
// Per ANONYMOUS_GAME_MECHANICS.md, all game state must survive application restarts.
package councils

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// PersistentCouncilStore wraps CouncilStore with Bbolt persistence.
type PersistentCouncilStore struct {
	*CouncilStore
	db *store.DB
}

// NewPersistentCouncilStore creates a council store with Bbolt persistence.
func NewPersistentCouncilStore(db *store.DB) (*PersistentCouncilStore, error) {
	ps := &PersistentCouncilStore{
		CouncilStore: NewCouncilStore(),
		db:           db,
	}

	if db != nil {
		if err := ps.loadFromDB(); err != nil {
			return nil, fmt.Errorf("loading councils from database: %w", err)
		}
	}

	return ps, nil
}

// loadFromDB loads all councils from Bbolt into memory.
func (ps *PersistentCouncilStore) loadFromDB() error {
	return ps.db.ForEach(store.BucketCouncils, func(key, value []byte) error {
		var pbCouncil pb.PhantomCouncil
		if err := proto.Unmarshal(value, &pbCouncil); err != nil {
			return nil // Skip corrupt entries.
		}

		council := protoToCouncil(&pbCouncil)
		if council == nil {
			return nil
		}

		ps.CouncilStore.mu.Lock()
		ps.CouncilStore.councils[hex.EncodeToString(council.ID[:])] = council
		ps.CouncilStore.mu.Unlock()

		return nil
	})
}

// AddCouncil adds a new council and persists it.
func (ps *PersistentCouncilStore) AddCouncil(c *PhantomCouncil) error {
	ps.CouncilStore.AddCouncil(c)

	if ps.db != nil {
		if err := ps.persistCouncil(c); err != nil {
			ps.CouncilStore.mu.Lock()
			delete(ps.CouncilStore.councils, hex.EncodeToString(c.ID[:]))
			ps.CouncilStore.mu.Unlock()
			return fmt.Errorf("persisting council: %w", err)
		}
	}

	return nil
}

// persistCouncil saves a council to Bbolt.
func (ps *PersistentCouncilStore) persistCouncil(c *PhantomCouncil) error {
	pbCouncil := councilToProto(c)
	data, err := proto.Marshal(pbCouncil)
	if err != nil {
		return fmt.Errorf("marshaling council: %w", err)
	}
	return ps.db.Put(store.BucketCouncils, c.ID[:], data)
}

// UpdateAndPersist updates council state and persists changes.
func (ps *PersistentCouncilStore) UpdateAndPersist(c *PhantomCouncil) error {
	if ps.db != nil {
		return ps.persistCouncil(c)
	}
	return nil
}

// PruneDisbanded removes disbanded councils from memory and database.
func (ps *PersistentCouncilStore) PruneDisbanded() int {
	ps.CouncilStore.mu.RLock()
	var disbandedIDs [][32]byte
	for _, c := range ps.CouncilStore.councils {
		if c.State == CouncilDisbanded {
			disbandedIDs = append(disbandedIDs, c.ID)
		}
	}
	ps.CouncilStore.mu.RUnlock()

	pruned := ps.CouncilStore.PruneDisbanded()

	if ps.db != nil {
		for _, id := range disbandedIDs {
			_ = ps.db.Delete(store.BucketCouncils, id[:])
		}
	}

	return pruned
}

// councilToProto converts a PhantomCouncil to its protobuf representation.
func councilToProto(c *PhantomCouncil) *pb.PhantomCouncil {
	c.mu.RLock()
	defer c.mu.RUnlock()

	pbCouncil := &pb.PhantomCouncil{
		Id:            c.ID[:],
		Name:          c.Name,
		FounderPubkey: c.CreatorKey[:],
		CreatedAt:     c.CreatedAt.Unix(),
		MinResonance:  uint32(c.MinResonance),
		Quorum:        uint32(len(c.Members) / 2), // Majority quorum.
		State:         convertCouncilState(c.State),
		Members:       convertMembersToProto(c.Members, c.CreatorKey),
		Proposals:     convertProposalsToProto(c.Proposals),
	}

	return pbCouncil
}

// convertCouncilState maps internal CouncilState to protobuf CouncilState.
func convertCouncilState(state CouncilState) pb.CouncilState {
	switch state {
	case CouncilActive:
		return pb.CouncilState_COUNCIL_STATE_ACTIVE
	case CouncilDisbanded:
		return pb.CouncilState_COUNCIL_STATE_DISSOLVED
	default:
		return pb.CouncilState_COUNCIL_STATE_UNSPECIFIED
	}
}

// convertMembersToProto converts council members to protobuf representation.
func convertMembersToProto(members []*CouncilMember, creatorKey [32]byte) []*pb.CouncilMember {
	var pbMembers []*pb.CouncilMember
	for _, m := range members {
		if m.Status != MemberActive {
			continue
		}
		role := pb.CouncilRole_COUNCIL_ROLE_MEMBER
		if m.SpecterKey == creatorKey {
			role = pb.CouncilRole_COUNCIL_ROLE_FOUNDER
		}
		pbMembers = append(pbMembers, &pb.CouncilMember{
			SpecterPubkey: m.SpecterKey[:],
			JoinedAt:      m.JoinedAt.Unix(),
			VoteWeight:    uint32(1),
			Role:          role,
		})
	}
	return pbMembers
}

// convertProposalsToProto converts council proposals to protobuf representation.
func convertProposalsToProto(proposals []*CouncilProposal) []*pb.CouncilProposal {
	var pbProposals []*pb.CouncilProposal
	for _, p := range proposals {
		pbProposals = append(pbProposals, &pb.CouncilProposal{
			Id:             p.ID[:],
			ProposerPubkey: p.ProposerKey[:],
			Title:          p.Text,
			Description:    "",
			CreatedAt:      p.CreatedAt.Unix(),
			VotingEndsAt:   p.CreatedAt.Add(72 * time.Hour).Unix(),
			State:          convertProposalState(*p),
		})
	}
	return pbProposals
}

// convertProposalState maps proposal resolution state to protobuf ProposalState.
func convertProposalState(p CouncilProposal) pb.ProposalState {
	if !p.Resolved {
		return pb.ProposalState_PROPOSAL_STATE_PENDING
	}
	if p.Passed {
		return pb.ProposalState_PROPOSAL_STATE_PASSED
	}
	return pb.ProposalState_PROPOSAL_STATE_REJECTED
}

// protoToCouncil converts a protobuf PhantomCouncil to a PhantomCouncil.
func protoToCouncil(pbCouncil *pb.PhantomCouncil) *PhantomCouncil {
	if len(pbCouncil.Id) != 32 || len(pbCouncil.FounderPubkey) != 32 {
		return nil
	}

	council := &PhantomCouncil{
		Name:             pbCouncil.Name,
		CreatedAt:        time.Unix(pbCouncil.CreatedAt, 0),
		MinResonance:     float64(pbCouncil.MinResonance),
		State:            convertProtoCouncilState(pbCouncil.State),
		memberByKey:      make(map[string]*CouncilMember),
		applicationByKey: make(map[string]*CouncilApplication),
	}
	copy(council.ID[:], pbCouncil.Id)
	copy(council.CreatorKey[:], pbCouncil.FounderPubkey)

	populateMembersFromProto(council, pbCouncil.Members)
	populateProposalsFromProto(council, pbCouncil.Proposals)

	return council
}

// convertProtoCouncilState maps protobuf CouncilState to internal CouncilState.
func convertProtoCouncilState(state pb.CouncilState) CouncilState {
	switch state {
	case pb.CouncilState_COUNCIL_STATE_ACTIVE:
		return CouncilActive
	case pb.CouncilState_COUNCIL_STATE_DISSOLVED:
		return CouncilDisbanded
	default:
		return CouncilActive
	}
}

// populateMembersFromProto converts and adds protobuf members to council.
func populateMembersFromProto(council *PhantomCouncil, pbMembers []*pb.CouncilMember) {
	for _, pbMember := range pbMembers {
		if len(pbMember.SpecterPubkey) != 32 {
			continue
		}
		member := &CouncilMember{
			JoinedAt: time.Unix(pbMember.JoinedAt, 0),
			Status:   MemberActive,
		}
		copy(member.SpecterKey[:], pbMember.SpecterPubkey)
		council.Members = append(council.Members, member)
		council.memberByKey[hex.EncodeToString(member.SpecterKey[:])] = member
	}
}

// populateProposalsFromProto converts and adds protobuf proposals to council.
func populateProposalsFromProto(council *PhantomCouncil, pbProposals []*pb.CouncilProposal) {
	for _, pbProposal := range pbProposals {
		if len(pbProposal.Id) != 32 || len(pbProposal.ProposerPubkey) != 32 {
			continue
		}
		resolved, passed := convertProtoProposalState(pbProposal.State)
		proposal := &CouncilProposal{
			Text:      pbProposal.Title,
			CreatedAt: time.Unix(pbProposal.CreatedAt, 0),
			Resolved:  resolved,
			Passed:    passed,
			Votes:     make(map[string]VoteValue),
		}
		copy(proposal.ID[:], pbProposal.Id)
		copy(proposal.ProposerKey[:], pbProposal.ProposerPubkey)
		council.Proposals = append(council.Proposals, proposal)
	}
}

// convertProtoProposalState converts protobuf ProposalState to resolved/passed flags.
func convertProtoProposalState(state pb.ProposalState) (resolved, passed bool) {
	switch state {
	case pb.ProposalState_PROPOSAL_STATE_PASSED:
		return true, true
	case pb.ProposalState_PROPOSAL_STATE_REJECTED:
		return true, false
	default:
		return false, false
	}
}
