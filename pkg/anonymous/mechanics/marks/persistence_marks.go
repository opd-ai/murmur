// Package mechanics - Specter Marks persistence.
// Per ANONYMOUS_GAME_MECHANICS.md, all game state must survive application restarts.
package marks

import (
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// PersistentMarkStore wraps MarkStore with Bbolt persistence.
type PersistentMarkStore struct {
	*MarkStore
	db *store.DB
}

// NewPersistentMarkStore creates a mark store with Bbolt persistence.
func NewPersistentMarkStore(db *store.DB) (*PersistentMarkStore, error) {
	ps := &PersistentMarkStore{
		MarkStore: NewMarkStore(),
		db:        db,
	}

	if db != nil {
		if err := ps.loadFromDB(); err != nil {
			return nil, fmt.Errorf("loading marks from database: %w", err)
		}
	}

	return ps, nil
}

// loadFromDB loads all marks from Bbolt into memory.
func (ps *PersistentMarkStore) loadFromDB() error {
	return ps.db.ForEach(store.BucketMarks, func(key, value []byte) error {
		var pbMark pb.SpecterMark
		if err := proto.Unmarshal(value, &pbMark); err != nil {
			return nil // Skip corrupt entries.
		}

		mark := protoToMark(&pbMark)
		if mark == nil || mark.IsExpired() {
			return nil
		}

		ps.MarkStore.mu.Lock()
		ps.MarkStore.marks[mark.ID] = mark
		markerHex := keyToHex(mark.MarkerKey[:])
		targetHex := keyToHex(mark.TargetKey)
		ps.MarkStore.byMarker[markerHex] = append(ps.MarkStore.byMarker[markerHex], mark)
		ps.MarkStore.byTarget[targetHex] = append(ps.MarkStore.byTarget[targetHex], mark)
		if ps.MarkStore.markerTargets[markerHex] == nil {
			ps.MarkStore.markerTargets[markerHex] = make(map[string]bool)
		}
		ps.MarkStore.markerTargets[markerHex][targetHex] = true
		ps.MarkStore.mu.Unlock()

		return nil
	})
}

// PlaceMark creates a new mark and persists it.
func (ps *PersistentMarkStore) PlaceMark(
	markerKey [32]byte,
	targetKey []byte,
	category MarkCategory,
	note string,
	resonance int,
	signingKey interface{},
) (*Mark, error) {
	mark, err := ps.MarkStore.PlaceMark(markerKey, targetKey, category, note, resonance, nil)
	if err != nil {
		return nil, err
	}

	if ps.db != nil {
		if err := ps.persistMark(mark); err != nil {
			ps.MarkStore.mu.Lock()
			delete(ps.MarkStore.marks, mark.ID)
			ps.MarkStore.mu.Unlock()
			return nil, fmt.Errorf("persisting mark: %w", err)
		}
	}

	return mark, nil
}

// persistMark saves a mark to Bbolt.
func (ps *PersistentMarkStore) persistMark(mark *Mark) error {
	pbMark := markToProto(mark)
	data, err := proto.Marshal(pbMark)
	if err != nil {
		return fmt.Errorf("marshaling mark: %w", err)
	}
	return ps.db.Put(store.BucketMarks, mark.ID[:], data)
}

// GarbageCollect removes expired marks from memory and database.
func (ps *PersistentMarkStore) GarbageCollect() int {
	ps.MarkStore.mu.RLock()
	var expiredIDs [][32]byte
	for id, mark := range ps.MarkStore.marks {
		if mark.IsExpired() {
			expiredIDs = append(expiredIDs, id)
		}
	}
	ps.MarkStore.mu.RUnlock()

	removed := ps.MarkStore.GarbageCollect()

	if ps.db != nil {
		for _, id := range expiredIDs {
			_ = ps.db.Delete(store.BucketMarks, id[:])
		}
	}

	return removed
}

// markToProto converts a Mark to its protobuf representation.
func markToProto(mark *Mark) *pb.SpecterMark {
	return &pb.SpecterMark{
		Id:            mark.ID[:],
		SpecterPubkey: mark.MarkerKey[:],
		TargetPubkey:  mark.TargetKey,
		Content:       mark.Note,
		CreatedAt:     mark.CreatedAt.Unix(),
		ExpiresAt:     mark.ExpiresAt.Unix(),
		Signature:     mark.Signature,
	}
}

// protoToMark converts a protobuf SpecterMark to a Mark.
func protoToMark(pbMark *pb.SpecterMark) *Mark {
	if len(pbMark.Id) != 32 || len(pbMark.SpecterPubkey) != 32 {
		return nil
	}

	mark := &Mark{
		TargetKey:  pbMark.TargetPubkey,
		Note:       pbMark.Content,
		CreatedAt:  time.Unix(pbMark.CreatedAt, 0),
		ExpiresAt:  time.Unix(pbMark.ExpiresAt, 0),
		Visibility: 1.0,
		Signature:  pbMark.Signature,
	}
	copy(mark.ID[:], pbMark.Id)
	copy(mark.MarkerKey[:], pbMark.SpecterPubkey)

	return mark
}
