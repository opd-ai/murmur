// Package mechanics - Specter Hunts persistence.
// Per ANONYMOUS_GAME_MECHANICS.md, all game state must survive application restarts.
package hunts

import (
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// PersistentHuntStore wraps HuntStore with Bbolt persistence.
type PersistentHuntStore struct {
	*HuntStore
	db *store.DB
}

// NewPersistentHuntStore creates a hunt store with Bbolt persistence.
func NewPersistentHuntStore(db *store.DB) (*PersistentHuntStore, error) {
	ps := &PersistentHuntStore{
		HuntStore: NewHuntStore(),
		db:        db,
	}

	if db != nil {
		if err := ps.loadFromDB(); err != nil {
			return nil, fmt.Errorf("loading hunts from database: %w", err)
		}
	}

	return ps, nil
}

// loadFromDB loads all hunts from Bbolt into memory.
func (ps *PersistentHuntStore) loadFromDB() error {
	return ps.db.ForEach(store.BucketHunts, func(key, value []byte) error {
		var pbHunt pb.SpecterHunt
		if err := proto.Unmarshal(value, &pbHunt); err != nil {
			return nil // Skip corrupt entries.
		}

		hunt := protoToHunt(&pbHunt)
		if hunt == nil {
			return nil
		}

		ps.HuntStore.mu.Lock()
		ps.HuntStore.hunts[hunt.ID] = hunt
		if hunt.IsActive() {
			ps.HuntStore.active = append(ps.HuntStore.active, hunt)
		} else {
			ps.HuntStore.history = append(ps.HuntStore.history, hunt)
		}
		ps.HuntStore.mu.Unlock()

		return nil
	})
}

// AddHunt adds a new hunt and persists it.
func (ps *PersistentHuntStore) AddHunt(h *Hunt) error {
	ps.HuntStore.AddHunt(h)

	if ps.db != nil {
		if err := ps.persistHunt(h); err != nil {
			ps.HuntStore.mu.Lock()
			delete(ps.HuntStore.hunts, h.ID)
			ps.HuntStore.mu.Unlock()
			return fmt.Errorf("persisting hunt: %w", err)
		}
	}

	return nil
}

// persistHunt saves a hunt to Bbolt.
func (ps *PersistentHuntStore) persistHunt(h *Hunt) error {
	pbHunt := huntToProto(h)
	data, err := proto.Marshal(pbHunt)
	if err != nil {
		return fmt.Errorf("marshaling hunt: %w", err)
	}
	return ps.db.Put(store.BucketHunts, h.ID[:], data)
}

// UpdateAndPersist updates hunt state and persists changes.
func (ps *PersistentHuntStore) UpdateAndPersist(h *Hunt) error {
	if ps.db != nil {
		return ps.persistHunt(h)
	}
	return nil
}

// huntToProto converts a Hunt to its protobuf representation.
func huntToProto(h *Hunt) *pb.SpecterHunt {
	h.mu.RLock()
	defer h.mu.RUnlock()

	state := pb.HuntState_HUNT_STATE_UNSPECIFIED
	switch h.State {
	case HuntPending:
		state = pb.HuntState_HUNT_STATE_PENDING
	case HuntActive:
		state = pb.HuntState_HUNT_STATE_ACTIVE
	case HuntCompleted:
		state = pb.HuntState_HUNT_STATE_COMPLETED
	case HuntExpired:
		state = pb.HuntState_HUNT_STATE_CANCELLED // Using CANCELLED for expired.
	}

	pbHunt := &pb.SpecterHunt{
		Id:              h.ID[:],
		OrganizerPubkey: h.InitiatorKey[:],
		Title:           h.Theme,
		Description:     h.Theme,
		StartTime:       h.CreatedAt.Unix(),
		EndTime:         h.ExpiresAt.Unix(),
		State:           state,
		MaxParticipants: uint32(h.FragmentCount),
	}

	// Convert fragments to targets.
	for _, f := range h.Fragments {
		target := &pb.HuntTarget{
			Id:           f.LocationHash[:],
			LocationHash: f.LocationHash[:],
			Found:        f.Claimed,
			Points:       uint32(100), // Default points.
		}
		if f.ClaimerKey != nil {
			target.FinderPubkey = (*f.ClaimerKey)[:]
		}
		if f.ClaimedAt != nil {
			target.FoundAt = f.ClaimedAt.Unix()
		}
		pbHunt.Targets = append(pbHunt.Targets, target)
	}

	return pbHunt
}

// protoToHunt converts a protobuf SpecterHunt to a Hunt.
func protoToHunt(pbHunt *pb.SpecterHunt) *Hunt {
	if len(pbHunt.Id) != 32 || len(pbHunt.OrganizerPubkey) != 32 {
		return nil
	}

	state := HuntPending
	switch pbHunt.State {
	case pb.HuntState_HUNT_STATE_PENDING:
		state = HuntPending
	case pb.HuntState_HUNT_STATE_ACTIVE:
		state = HuntActive
	case pb.HuntState_HUNT_STATE_COMPLETED:
		state = HuntCompleted
	case pb.HuntState_HUNT_STATE_CANCELLED:
		state = HuntExpired
	}

	createdAt := time.Unix(pbHunt.StartTime, 0)
	expiresAt := time.Unix(pbHunt.EndTime, 0)

	hunt := &Hunt{
		Theme:         pbHunt.Title,
		CreatedAt:     createdAt,
		ExpiresAt:     expiresAt,
		Duration:      expiresAt.Sub(createdAt),
		State:         state,
		FragmentCount: len(pbHunt.Targets),
	}
	copy(hunt.ID[:], pbHunt.Id)
	copy(hunt.InitiatorKey[:], pbHunt.OrganizerPubkey)

	// Convert targets to fragments.
	for i, target := range pbHunt.Targets {
		fragment := &Fragment{
			Index:   i,
			Claimed: target.Found,
		}
		if len(target.LocationHash) == 32 {
			copy(fragment.LocationHash[:], target.LocationHash)
		}
		if len(target.FinderPubkey) == 32 {
			var claimer [32]byte
			copy(claimer[:], target.FinderPubkey)
			fragment.ClaimerKey = &claimer
		}
		if target.FoundAt > 0 {
			claimedAt := time.Unix(target.FoundAt, 0)
			fragment.ClaimedAt = &claimedAt
		}
		hunt.Fragments = append(hunt.Fragments, fragment)
	}

	return hunt
}
