// Package mechanics - Sigil Forge persistence.
// Per ANONYMOUS_GAME_MECHANICS.md, all game state must survive application restarts.
package mechanics

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/zeebo/blake3"

	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// PersistentForgeStore wraps ForgeStore with Bbolt persistence.
type PersistentForgeStore struct {
	*ForgeStore
	db *store.DB
}

// NewPersistentForgeStore creates a forge store with Bbolt persistence.
func NewPersistentForgeStore(db *store.DB) (*PersistentForgeStore, error) {
	ps := &PersistentForgeStore{
		ForgeStore: NewForgeStore(),
		db:         db,
	}

	if db != nil {
		if err := ps.loadFromDB(); err != nil {
			return nil, fmt.Errorf("loading forges from database: %w", err)
		}
	}

	return ps, nil
}

// loadFromDB loads all forges from Bbolt into memory.
func (ps *PersistentForgeStore) loadFromDB() error {
	return ps.db.ForEach(store.BucketForge, func(key, value []byte) error {
		var pbForge pb.ForgeProject
		if err := proto.Unmarshal(value, &pbForge); err != nil {
			return nil // Skip corrupt entries.
		}

		forge := protoToForge(&pbForge)
		if forge == nil {
			return nil
		}

		ps.ForgeStore.mu.Lock()
		ps.ForgeStore.forges[hex.EncodeToString(forge.ID[:])] = forge
		ps.ForgeStore.mu.Unlock()

		return nil
	})
}

// AddForge adds a new forge and persists it.
func (ps *PersistentForgeStore) AddForge(forge *SigilForge) error {
	ps.ForgeStore.AddForge(forge)

	if ps.db != nil {
		if err := ps.persistForge(forge); err != nil {
			ps.ForgeStore.mu.Lock()
			delete(ps.ForgeStore.forges, hex.EncodeToString(forge.ID[:]))
			ps.ForgeStore.mu.Unlock()
			return fmt.Errorf("persisting forge: %w", err)
		}
	}

	return nil
}

// persistForge saves a forge to Bbolt.
func (ps *PersistentForgeStore) persistForge(forge *SigilForge) error {
	pbForge := forgeToProto(forge)
	data, err := proto.Marshal(pbForge)
	if err != nil {
		return fmt.Errorf("marshaling forge: %w", err)
	}
	return ps.db.Put(store.BucketForge, forge.ID[:], data)
}

// UpdateAndPersist updates forge state and persists changes.
func (ps *PersistentForgeStore) UpdateAndPersist(forge *SigilForge) error {
	if ps.db != nil {
		return ps.persistForge(forge)
	}
	return nil
}

// forgeToProto converts a SigilForge to its protobuf representation.
func forgeToProto(forge *SigilForge) *pb.ForgeProject {
	forge.mu.RLock()
	defer forge.mu.RUnlock()

	state := pb.ForgeState_FORGE_STATE_UNSPECIFIED
	switch forge.State {
	case ForgeActive:
		state = pb.ForgeState_FORGE_STATE_COLLECTING
	case ForgeEvaluating:
		state = pb.ForgeState_FORGE_STATE_FINALIZING
	case ForgeCompleted:
		state = pb.ForgeState_FORGE_STATE_COMPLETE
	}

	pbForge := &pb.ForgeProject{
		Id:            forge.ID[:],
		CreatorPubkey: forge.InitiatorKey[:],
		Name:          forge.Prompt,
		Seed:          forge.ID[:], // Using ID as seed.
		CreatedAt:     forge.CreatedAt.Unix(),
		Deadline:      forge.Deadline.Unix(),
		State:         state,
	}

	// Convert entries to contributions.
	for _, entry := range forge.Entries {
		pbContrib := &pb.ForgeContribution{
			SpecterPubkey: entry.SpecterKey[:],
			Contribution:  entry.Content,
			Timestamp:     entry.SubmittedAt.Unix(),
		}
		pbForge.Contributions = append(pbForge.Contributions, pbContrib)
	}

	return pbForge
}

// protoToForge converts a protobuf ForgeProject to a SigilForge.
func protoToForge(pbForge *pb.ForgeProject) *SigilForge {
	if len(pbForge.Id) != 32 || len(pbForge.CreatorPubkey) != 32 {
		return nil
	}

	state := ForgeActive
	switch pbForge.State {
	case pb.ForgeState_FORGE_STATE_COLLECTING:
		state = ForgeActive
	case pb.ForgeState_FORGE_STATE_FINALIZING:
		state = ForgeEvaluating
	case pb.ForgeState_FORGE_STATE_COMPLETE:
		state = ForgeCompleted
	}

	createdAt := time.Unix(pbForge.CreatedAt, 0)
	deadline := time.Unix(pbForge.Deadline, 0)

	forge := &SigilForge{
		Type:           ForgeSigilArt, // Default type.
		Prompt:         pbForge.Name,
		CreatedAt:      createdAt,
		Duration:       deadline.Sub(createdAt),
		Deadline:       deadline,
		State:          state,
		entryBySpecter: make(map[string]*ForgeEntry),
	}
	copy(forge.ID[:], pbForge.Id)
	copy(forge.InitiatorKey[:], pbForge.CreatorPubkey)

	// Convert contributions to entries.
	for _, pbContrib := range pbForge.Contributions {
		if len(pbContrib.SpecterPubkey) != 32 {
			continue
		}
		entry := &ForgeEntry{
			Content:      pbContrib.Contribution,
			SubmittedAt:  time.Unix(pbContrib.Timestamp, 0),
			amplifierSet: make(map[string]bool),
		}
		copy(entry.SpecterKey[:], pbContrib.SpecterPubkey)
		// Generate entry ID.
		entry.ID = computeEntryID(entry.SpecterKey, entry.Content)
		forge.Entries = append(forge.Entries, entry)
		forge.entryBySpecter[hex.EncodeToString(entry.SpecterKey[:])] = entry
	}

	return forge
}

// computeEntryID generates a deterministic entry ID.
func computeEntryID(specterKey [32]byte, content []byte) [32]byte {
	h := blake3.New()
	h.Write(specterKey[:])
	h.Write(content)
	var id [32]byte
	copy(id[:], h.Sum(nil))
	return id
}
