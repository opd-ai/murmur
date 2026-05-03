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

	state := pb.CouncilState_COUNCIL_STATE_UNSPECIFIED
	switch c.State {
	case CouncilActive:
		state = pb.CouncilState_COUNCIL_STATE_ACTIVE
	case CouncilDisbanded:
		state = pb.CouncilState_COUNCIL_STATE_DISSOLVED
	}

	pbCouncil := &pb.PhantomCouncil{
		Id:            c.ID[:],
		Name:          c.Name,
		FounderPubkey: c.CreatorKey[:],
		CreatedAt:     c.CreatedAt.Unix(),
		MinResonance:  uint32(c.MinResonance),
		Quorum:        uint32(len(c.Members) / 2), // Majority quorum.
		State:         state,
	}

	// Convert members.
	for _, m := range c.Members {
		if m.Status == MemberActive {
			role := pb.CouncilRole_COUNCIL_ROLE_MEMBER
			if m.SpecterKey == c.CreatorKey {
				role = pb.CouncilRole_COUNCIL_ROLE_FOUNDER
			}
			pbMember := &pb.CouncilMember{
				SpecterPubkey: m.SpecterKey[:],
				JoinedAt:      m.JoinedAt.Unix(),
				VoteWeight:    uint32(1),
				Role:          role,
			}
			pbCouncil.Members = append(pbCouncil.Members, pbMember)
		}
	}

	// Convert proposals.
	for _, p := range c.Proposals {
		proposalState := pb.ProposalState_PROPOSAL_STATE_PENDING
		if p.Resolved {
			if p.Passed {
				proposalState = pb.ProposalState_PROPOSAL_STATE_PASSED
			} else {
				proposalState = pb.ProposalState_PROPOSAL_STATE_REJECTED
			}
		}

		pbProposal := &pb.CouncilProposal{
			Id:             p.ID[:],
			ProposerPubkey: p.ProposerKey[:],
			Title:          p.Text, // Text field used for proposal content.
			Description:    "",
			CreatedAt:      p.CreatedAt.Unix(),
			VotingEndsAt:   p.CreatedAt.Add(72 * time.Hour).Unix(), // 72h voting period.
			State:          proposalState,
		}
		pbCouncil.Proposals = append(pbCouncil.Proposals, pbProposal)
	}

	return pbCouncil
}

// protoToCouncil converts a protobuf PhantomCouncil to a PhantomCouncil.
func protoToCouncil(pbCouncil *pb.PhantomCouncil) *PhantomCouncil {
	if len(pbCouncil.Id) != 32 || len(pbCouncil.FounderPubkey) != 32 {
		return nil
	}

	state := CouncilActive
	switch pbCouncil.State {
	case pb.CouncilState_COUNCIL_STATE_ACTIVE:
		state = CouncilActive
	case pb.CouncilState_COUNCIL_STATE_DISSOLVED:
		state = CouncilDisbanded
	}

	council := &PhantomCouncil{
		Name:             pbCouncil.Name,
		CreatedAt:        time.Unix(pbCouncil.CreatedAt, 0),
		MinResonance:     float64(pbCouncil.MinResonance),
		State:            state,
		memberByKey:      make(map[string]*CouncilMember),
		applicationByKey: make(map[string]*CouncilApplication),
	}
	copy(council.ID[:], pbCouncil.Id)
	copy(council.CreatorKey[:], pbCouncil.FounderPubkey)

	// Convert members.
	for _, pbMember := range pbCouncil.Members {
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

	// Convert proposals.
	for _, pbProposal := range pbCouncil.Proposals {
		if len(pbProposal.Id) != 32 || len(pbProposal.ProposerPubkey) != 32 {
			continue
		}
		resolved := false
		passed := false
		switch pbProposal.State {
		case pb.ProposalState_PROPOSAL_STATE_PASSED:
			resolved = true
			passed = true
		case pb.ProposalState_PROPOSAL_STATE_REJECTED:
			resolved = true
			passed = false
		}

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

	return council
}
