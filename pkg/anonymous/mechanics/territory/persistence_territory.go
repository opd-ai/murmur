// Package mechanics - Territory Drift persistence.
// Per ANONYMOUS_GAME_MECHANICS.md, all game state must survive application restarts.
package territory

import (
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"

	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// PersistentTerritoryManager wraps TerritoryManager with Bbolt persistence.
type PersistentTerritoryManager struct {
	*TerritoryManager
	db *store.DB
}

// NewPersistentTerritoryManager creates a territory manager with Bbolt persistence.
func NewPersistentTerritoryManager(db *store.DB) (*PersistentTerritoryManager, error) {
	pm := &PersistentTerritoryManager{
		TerritoryManager: NewTerritoryManager(),
		db:               db,
	}

	if db != nil {
		if err := pm.loadFromDB(); err != nil {
			return nil, fmt.Errorf("loading territories from database: %w", err)
		}
	}

	return pm, nil
}

// loadFromDB loads all territories from Bbolt into memory.
func (pm *PersistentTerritoryManager) loadFromDB() error {
	return pm.db.ForEach(store.BucketTerritories, func(key, value []byte) error {
		var pbTerritory pb.Territory
		if err := proto.Unmarshal(value, &pbTerritory); err != nil {
			return nil // Skip corrupt entries.
		}

		territory := protoToTerritory(&pbTerritory)
		if territory == nil {
			return nil
		}

		pm.TerritoryManager.mu.Lock()
		pm.TerritoryManager.territories[territory.ID] = territory
		pm.TerritoryManager.mu.Unlock()

		return nil
	})
}

// AddTerritory adds a new territory and persists it.
func (pm *PersistentTerritoryManager) AddTerritory(t *Territory) error {
	pm.TerritoryManager.AddTerritory(t)

	if pm.db != nil {
		if err := pm.persistTerritory(t); err != nil {
			pm.TerritoryManager.mu.Lock()
			delete(pm.TerritoryManager.territories, t.ID)
			pm.TerritoryManager.mu.Unlock()
			return fmt.Errorf("persisting territory: %w", err)
		}
	}

	return nil
}

// persistTerritory saves a territory to Bbolt.
func (pm *PersistentTerritoryManager) persistTerritory(t *Territory) error {
	pbTerritory := territoryToProto(t)
	data, err := proto.Marshal(pbTerritory)
	if err != nil {
		return fmt.Errorf("marshaling territory: %w", err)
	}
	return pm.db.Put(store.BucketTerritories, []byte(t.ID), data)
}

// UpdateAndPersist updates territory state and persists changes.
func (pm *PersistentTerritoryManager) UpdateAndPersist(t *Territory) error {
	if pm.db != nil {
		return pm.persistTerritory(t)
	}
	return nil
}

// territoryToProto converts a Territory to its protobuf representation.
func territoryToProto(t *Territory) *pb.Territory {
	t.mu.RLock()
	defer t.mu.RUnlock()

	pbTerritory := &pb.Territory{
		Id:          []byte(t.ID),
		Name:        t.ID,
		Influence:   uint32(t.totalInfluence()),
		LastUpdated: t.LastReset.Unix(),
	}

	if t.Controller != nil {
		pbTerritory.ControllerPubkey = t.Controller[:]
	}

	// Convert influence map to contenders.
	for hexKey, influence := range t.Influence {
		keyBytes := hexToKeyBytes(hexKey)
		if keyBytes != nil {
			pbContender := &pb.TerritoryContender{
				SpecterPubkey: keyBytes,
				Influence:     uint32(influence),
				LastAction:    t.LastReset.Unix(),
			}
			pbTerritory.Contenders = append(pbTerritory.Contenders, pbContender)
		}
	}

	return pbTerritory
}

// totalInfluence sums all influence in the territory.
func (t *Territory) totalInfluence() float64 {
	var total float64
	for _, inf := range t.Influence {
		total += inf
	}
	return total
}

// protoToTerritory converts a protobuf Territory to a Territory.
func protoToTerritory(pbTerritory *pb.Territory) *Territory {
	if len(pbTerritory.Id) == 0 {
		return nil
	}

	territory := &Territory{
		ID:        string(pbTerritory.Id),
		LastReset: time.Unix(pbTerritory.LastUpdated, 0),
		Influence: make(map[string]float64),
	}

	if len(pbTerritory.ControllerPubkey) == 32 {
		var controller [32]byte
		copy(controller[:], pbTerritory.ControllerPubkey)
		territory.Controller = &controller
		territory.State = TerritoryControlled
	} else {
		territory.State = TerritoryNeutral
	}

	// Convert contenders to influence map.
	for _, pbContender := range pbTerritory.Contenders {
		if len(pbContender.SpecterPubkey) == 32 {
			hexKey := mechanics.KeyToHex(pbContender.SpecterPubkey)
			territory.Influence[hexKey] = float64(pbContender.Influence)
		}
	}

	if len(territory.Influence) > 1 {
		territory.State = TerritoryContested
	}

	return territory
}

// hexToKeyBytes converts a hex string to bytes for persistence layer.
func hexToKeyBytes(h string) []byte {
	if len(h) != 64 {
		return nil
	}
	result := make([]byte, 32)
	for i := 0; i < 32; i++ {
		var b byte
		for j := 0; j < 2; j++ {
			c := h[i*2+j]
			switch {
			case c >= '0' && c <= '9':
				b = b*16 + (c - '0')
			case c >= 'a' && c <= 'f':
				b = b*16 + (c - 'a' + 10)
			case c >= 'A' && c <= 'F':
				b = b*16 + (c - 'A' + 10)
			default:
				return nil
			}
		}
		result[i] = b
	}
	return result
}
