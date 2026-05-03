// Package mechanics - Oracle Pools persistence.
// Per ANONYMOUS_GAME_MECHANICS.md, all game state must survive application restarts.
package oracle

import (
	"fmt"
	"time"


	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// PersistentOracleStore wraps OraclePoolStore with Bbolt persistence.
type PersistentOracleStore struct {
	*OraclePoolStore
	db *store.DB
}

// NewPersistentOracleStore creates an oracle store with Bbolt persistence.
func NewPersistentOracleStore(db *store.DB) (*PersistentOracleStore, error) {
	ps := &PersistentOracleStore{
		OraclePoolStore: NewOraclePoolStore(),
		db:              db,
	}

	if db != nil {
		if err := ps.loadFromDB(); err != nil {
			return nil, fmt.Errorf("loading oracle pools from database: %w", err)
		}
	}

	return ps, nil
}

// loadFromDB loads all oracle pools from Bbolt into memory.
func (ps *PersistentOracleStore) loadFromDB() error {
	return ps.db.ForEach(store.BucketOracles, func(key, value []byte) error {
		var pbPool pb.OraclePool
		if err := proto.Unmarshal(value, &pbPool); err != nil {
			return nil // Skip corrupt entries.
		}

		pool := protoToOraclePool(&pbPool)
		if pool == nil {
			return nil
		}

		ps.OraclePoolStore.mu.Lock()
		ps.OraclePoolStore.pools[pool.ID] = pool
		if pool.State != OraclePoolResolved && pool.State != OraclePoolCancelled {
			ps.OraclePoolStore.active = append(ps.OraclePoolStore.active, pool)
		} else {
			ps.OraclePoolStore.history = append(ps.OraclePoolStore.history, pool)
		}
		ps.OraclePoolStore.mu.Unlock()

		return nil
	})
}

// AddPool adds a new pool and persists it.
func (ps *PersistentOracleStore) AddPool(p *OraclePool) error {
	ps.OraclePoolStore.AddPool(p)

	if ps.db != nil {
		if err := ps.persistPool(p); err != nil {
			ps.OraclePoolStore.mu.Lock()
			delete(ps.OraclePoolStore.pools, p.ID)
			ps.OraclePoolStore.mu.Unlock()
			return fmt.Errorf("persisting oracle pool: %w", err)
		}
	}

	return nil
}

// persistPool saves an oracle pool to Bbolt.
func (ps *PersistentOracleStore) persistPool(p *OraclePool) error {
	pbPool := oraclePoolToProto(p)
	data, err := proto.Marshal(pbPool)
	if err != nil {
		return fmt.Errorf("marshaling oracle pool: %w", err)
	}
	return ps.db.Put(store.BucketOracles, p.ID[:], data)
}

// UpdateAndPersist updates pool state and persists changes.
func (ps *PersistentOracleStore) UpdateAndPersist(p *OraclePool) error {
	if ps.db != nil {
		return ps.persistPool(p)
	}
	return nil
}

// oraclePoolToProto converts an OraclePool to its protobuf representation.
func oraclePoolToProto(p *OraclePool) *pb.OraclePool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	state := pb.OracleState_ORACLE_STATE_UNSPECIFIED
	switch p.State {
	case OraclePoolOpen:
		state = pb.OracleState_ORACLE_STATE_OPEN
	case OraclePoolPending:
		state = pb.OracleState_ORACLE_STATE_CLOSED
	case OraclePoolResolved:
		state = pb.OracleState_ORACLE_STATE_RESOLVED
	case OraclePoolCancelled:
		state = pb.OracleState_ORACLE_STATE_CANCELLED
	}

	pbPool := &pb.OraclePool{
		Id:            p.ID[:],
		CreatorPubkey: p.CreatorKey[:],
		Question:      p.Question,
		CreatedAt:     p.CreatedAt.Unix(),
		ClosesAt:      p.Deadline.Unix(),
		ResolvesAt:    p.ResolutionTime.Unix(),
		State:         state,
	}

	// Set outcome if resolved.
	if p.Outcome != nil {
		pbPool.WinningOption = uint32(*p.Outcome)
	}

	return pbPool
}

// protoToOraclePool converts a protobuf OraclePool to an OraclePool.
func protoToOraclePool(pbPool *pb.OraclePool) *OraclePool {
	if len(pbPool.Id) != 32 || len(pbPool.CreatorPubkey) != 32 {
		return nil
	}

	state := OraclePoolOpen
	switch pbPool.State {
	case pb.OracleState_ORACLE_STATE_OPEN:
		state = OraclePoolOpen
	case pb.OracleState_ORACLE_STATE_CLOSED:
		state = OraclePoolPending
	case pb.OracleState_ORACLE_STATE_RESOLVED:
		state = OraclePoolResolved
	case pb.OracleState_ORACLE_STATE_CANCELLED:
		state = OraclePoolCancelled
	}

	pool := &OraclePool{
		Question:         pbPool.Question,
		PredictionType:   OraclePredictionBoolean, // Default.
		ResolutionMethod: "manual",                // Default.
		CreatedAt:        time.Unix(pbPool.CreatedAt, 0),
		Deadline:         time.Unix(pbPool.ClosesAt, 0),
		ResolutionTime:   time.Unix(pbPool.ResolvesAt, 0),
		State:            state,
		commitments:      make(map[string]*Commitment),
		predictions:      make(map[string]*Prediction),
	}
	copy(pool.ID[:], pbPool.Id)
	copy(pool.CreatorKey[:], pbPool.CreatorPubkey)

	// Set outcome if resolved.
	if state == OraclePoolResolved {
		outcome := float64(pbPool.WinningOption)
		pool.Outcome = &outcome
	}

	return pool
}
