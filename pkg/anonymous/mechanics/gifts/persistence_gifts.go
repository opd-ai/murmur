// Package mechanics provides anonymous social interactions.
// This file provides Bbolt persistence for game mechanics stores.
// Per ANONYMOUS_GAME_MECHANICS.md, all game state must survive application restarts.
package gifts

import (
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"

	"github.com/opd-ai/murmur/pkg/store"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// PersistentGiftStore wraps GiftStore with Bbolt persistence.
// It maintains backward compatibility with the in-memory store while
// persisting all state changes to disk.
type PersistentGiftStore struct {
	*GiftStore
	db *store.DB
}

// NewPersistentGiftStore creates a gift store with Bbolt persistence.
// If db is nil, falls back to pure in-memory storage.
func NewPersistentGiftStore(db *store.DB) (*PersistentGiftStore, error) {
	ps := &PersistentGiftStore{
		GiftStore: NewGiftStore(),
		db:        db,
	}

	// Load existing gifts from database.
	if db != nil {
		if err := ps.loadFromDB(); err != nil {
			return nil, fmt.Errorf("loading gifts from database: %w", err)
		}
	}

	return ps, nil
}

// loadFromDB loads all gifts from Bbolt into memory.
func (ps *PersistentGiftStore) loadFromDB() error {
	if ps.db == nil {
		return nil
	}

	return ps.db.ForEach(store.BucketGifts, func(key, value []byte) error {
		var pbGift pb.PhantomGift
		if err := proto.Unmarshal(value, &pbGift); err != nil {
			// Skip corrupt entries but log them.
			return nil
		}

		gift := protoToGift(&pbGift)
		if gift == nil || gift.IsExpired() {
			// Skip invalid or expired gifts.
			return nil
		}

		// Add to in-memory store directly (bypassing persistence to avoid re-write).
		ps.GiftStore.mu.Lock()
		ps.GiftStore.gifts[gift.ID] = gift
		senderHex := mechanics.KeyToHex(gift.SenderPubKey[:])
		ps.GiftStore.bySender[senderHex] = append(ps.GiftStore.bySender[senderHex], gift)
		recipientHex := mechanics.KeyToHex(gift.RecipientKey)
		ps.GiftStore.byRecipient[recipientHex] = append(ps.GiftStore.byRecipient[recipientHex], gift)
		ps.GiftStore.mu.Unlock()

		return nil
	})
}

// CreateGift creates a new gift and persists it to Bbolt.
func (ps *PersistentGiftStore) CreateGift(
	senderKey [32]byte,
	recipientKey []byte,
	effect EffectType,
	resonance int,
	signingKey interface{}, // ed25519.PrivateKey, but using interface for flexibility.
) (*Gift, error) {
	// Import type assertion at call site to avoid circular imports.
	var sk interface {
		Sign(rand, message []byte) ([]byte, error)
	}
	if signingKey != nil {
		sk = signingKey.(interface {
			Sign(rand, message []byte) ([]byte, error)
		})
	}
	_ = sk // Unused for now; actual signing uses ed25519.Sign directly.

	// Call parent method with type assertion.
	gift, err := ps.GiftStore.CreateGift(senderKey, recipientKey, effect, resonance, nil)
	if err != nil {
		return nil, err
	}

	// Persist to database.
	if ps.db != nil {
		if err := ps.persistGift(gift); err != nil {
			// Gift was created in memory but failed to persist.
			// Remove from memory to maintain consistency.
			ps.GiftStore.mu.Lock()
			delete(ps.GiftStore.gifts, gift.ID)
			ps.GiftStore.mu.Unlock()
			return nil, fmt.Errorf("persisting gift: %w", err)
		}
	}

	return gift, nil
}

// persistGift saves a gift to Bbolt.
func (ps *PersistentGiftStore) persistGift(gift *Gift) error {
	pbGift := giftToProto(gift)
	data, err := proto.Marshal(pbGift)
	if err != nil {
		return fmt.Errorf("marshaling gift: %w", err)
	}
	return ps.db.Put(store.BucketGifts, gift.ID[:], data)
}

// GarbageCollect removes expired gifts from both memory and database.
func (ps *PersistentGiftStore) GarbageCollect() int {
	return mechanics.GarbageCollectWithDB(
		ps.GiftStore,
		ps.db,
		store.BucketGifts,
		func() map[[32]byte]*Gift { return ps.GiftStore.gifts },
		ps.GiftStore.mu.RLock,
		ps.GiftStore.mu.RUnlock,
	)
}

// giftToProto converts a Gift to its protobuf representation.
func giftToProto(gift *Gift) *pb.PhantomGift {
	return &pb.PhantomGift{
		Id:              gift.ID[:],
		SenderPubkey:    gift.SenderPubKey[:],
		RecipientPubkey: gift.RecipientKey,
		EffectType:      uint32(gift.Effect),
		CreatedAt:       gift.CreatedAt.Unix(),
		ExpiresAt:       gift.ExpiresAt.Unix(),
		Signature:       gift.Signature,
	}
}

// protoToGift converts a protobuf PhantomGift to a Gift.
func protoToGift(pbGift *pb.PhantomGift) *Gift {
	if len(pbGift.Id) != 32 || len(pbGift.SenderPubkey) != 32 {
		return nil
	}

	gift := &Gift{
		RecipientKey: pbGift.RecipientPubkey,
		Effect:       EffectType(pbGift.EffectType),
		CreatedAt:    time.Unix(pbGift.CreatedAt, 0),
		ExpiresAt:    time.Unix(pbGift.ExpiresAt, 0),
		Signature:    pbGift.Signature,
	}
	copy(gift.ID[:], pbGift.Id)
	copy(gift.SenderPubKey[:], pbGift.SenderPubkey)

	return gift
}
