package gifts

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/store"
	"google.golang.org/protobuf/proto"
)

func TestNewPersistentGiftStore(t *testing.T) {
	// Test with nil DB (falls back to in-memory).
	ps, err := NewPersistentGiftStore(nil)
	if err != nil {
		t.Fatalf("NewPersistentGiftStore with nil DB failed: %v", err)
	}
	if ps == nil {
		t.Fatal("expected non-nil store, got nil")
	}
	if ps.GiftStore == nil {
		t.Fatal("expected embedded GiftStore, got nil")
	}

	// Test with actual DB.
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer db.Close()

	ps2, err := NewPersistentGiftStore(db)
	if err != nil {
		t.Fatalf("NewPersistentGiftStore with DB failed: %v", err)
	}
	if ps2 == nil {
		t.Fatal("expected non-nil store, got nil")
	}
	if ps2.db != db {
		t.Error("database reference mismatch")
	}
}

func TestPersistentGiftStore_CreateGift(t *testing.T) {
	// Create test database.
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer db.Close()

	ps, err := NewPersistentGiftStore(db)
	if err != nil {
		t.Fatalf("NewPersistentGiftStore failed: %v", err)
	}

	// Create test gift with sufficient resonance (25+ required for EffectSoftGlowPulse).
	var senderKey [32]byte
	copy(senderKey[:], []byte("test-sender-key-1234567890123456"))
	recipientKey := []byte("test-recipient-key-12345678901234567890123456789012")

	gift, err := ps.CreateGift(senderKey, recipientKey, EffectSoftGlowPulse, 25, nil)
	if err != nil {
		t.Fatalf("CreateGift failed: %v", err)
	}

	// Verify gift was created.
	if gift == nil {
		t.Fatal("expected non-nil gift, got nil")
	}
	if gift.Effect != EffectSoftGlowPulse {
		t.Errorf("effect mismatch: got %v, want %v", gift.Effect, EffectSoftGlowPulse)
	}

	// Verify gift was persisted to database.
	data, err := db.Get(store.BucketGifts, gift.ID[:])
	if err != nil {
		t.Fatalf("retrieving gift from database: %v", err)
	}
	if data == nil {
		t.Error("gift not found in database")
	}
}

func TestPersistentGiftStore_LoadFromDB(t *testing.T) {
	// Create test database and store.
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer db.Close()

	ps1, err := NewPersistentGiftStore(db)
	if err != nil {
		t.Fatalf("NewPersistentGiftStore failed: %v", err)
	}

	// Create several gifts with sufficient resonance.
	var senderKey1, senderKey2 [32]byte
	copy(senderKey1[:], []byte("sender1-1234567890123456789012345"))
	copy(senderKey2[:], []byte("sender2-1234567890123456789012345"))

	gift1, err := ps1.CreateGift(senderKey1, []byte("recipient1-123456789012345678901234567890"), EffectSoftGlowPulse, 25, nil)
	if err != nil {
		t.Fatalf("creating gift1: %v", err)
	}
	gift2, err := ps1.CreateGift(senderKey2, []byte("recipient2-123456789012345678901234567890"), EffectAuroraColorShift, 50, nil)
	if err != nil {
		t.Fatalf("creating gift2: %v", err)
	}

	// Close the store.
	db.Close()

	// Reopen database and create new store (should load existing gifts).
	db2, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("reopening database: %v", err)
	}
	defer db2.Close()

	ps2, err := NewPersistentGiftStore(db2)
	if err != nil {
		t.Fatalf("NewPersistentGiftStore on reload failed: %v", err)
	}

	// Verify gifts were loaded.
	loadedGift1, err := ps2.GetGift(gift1.ID)
	if err != nil {
		t.Fatalf("getting gift1: %v", err)
	}
	if loadedGift1 == nil {
		t.Error("gift1 not loaded from database")
	} else {
		if loadedGift1.Effect != gift1.Effect {
			t.Errorf("gift1 effect mismatch: got %v, want %v", loadedGift1.Effect, gift1.Effect)
		}
	}

	loadedGift2, err := ps2.GetGift(gift2.ID)
	if err != nil {
		t.Fatalf("getting gift2: %v", err)
	}
	if loadedGift2 == nil {
		t.Error("gift2 not loaded from database")
	} else {
		if loadedGift2.Effect != gift2.Effect {
			t.Errorf("gift2 effect mismatch: got %v, want %v", loadedGift2.Effect, gift2.Effect)
		}
	}
}

func TestPersistentGiftStore_GarbageCollect(t *testing.T) {
	// Create test database.
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer db.Close()

	ps, err := NewPersistentGiftStore(db)
	if err != nil {
		t.Fatalf("NewPersistentGiftStore failed: %v", err)
	}

	// Create an expired gift by directly adding to store.
	var senderKey [32]byte
	copy(senderKey[:], []byte("sender-12345678901234567890123456"))

	gift := &Gift{
		SenderPubKey: senderKey,
		RecipientKey: []byte("recipient-1234567890123456789012345678"),
		Effect:       EffectSoftGlowPulse,
		CreatedAt:    time.Now().Add(-48 * time.Hour), // Created 2 days ago
		ExpiresAt:    time.Now().Add(-24 * time.Hour), // Expired 1 day ago
	}
	copy(gift.ID[:], []byte("gift-id-1234567890123456789012345"))

	// Add to in-memory store and persist.
	ps.GiftStore.mu.Lock()
	ps.GiftStore.gifts[gift.ID] = gift
	ps.GiftStore.mu.Unlock()

	if err := ps.persistGift(gift); err != nil {
		t.Fatalf("persisting expired gift: %v", err)
	}

	// Verify gift exists before GC.
	existingGift, err := ps.GetGift(gift.ID)
	if err != nil {
		t.Fatalf("getting gift before GC: %v", err)
	}
	if existingGift == nil {
		t.Fatal("gift should exist before garbage collection")
	}

	// Run garbage collection.
	removed := ps.GarbageCollect()
	if removed == 0 {
		t.Error("expected at least 1 gift to be removed")
	}

	// Verify gift is removed from memory.
	removedGift, _ := ps.GetGift(gift.ID)
	if removedGift != nil {
		t.Error("expired gift should be removed from memory")
	}

	// Verify gift is removed from database.
	data, err := db.Get(store.BucketGifts, gift.ID[:])
	if err == nil && data != nil {
		t.Error("expired gift should be removed from database")
	}
}

func TestGiftToProto_RoundTrip(t *testing.T) {
	// Create test gift.
	var senderKey [32]byte
	var giftID [32]byte
	copy(senderKey[:], []byte("sender-key-1234567890123456789012"))
	copy(giftID[:], []byte("gift-id-123456789012345678901234"))

	gift := &Gift{
		ID:           giftID,
		SenderPubKey: senderKey,
		RecipientKey: []byte("recipient-key-12345678901234567890"),
		Effect:       EffectAuroraColorShift,
		CreatedAt:    time.Now().Truncate(time.Second),
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour).Truncate(time.Second),
		Signature:    []byte("signature-12345678901234567890"),
	}

	// Convert to proto and back.
	pbGift := giftToProto(gift)
	if pbGift == nil {
		t.Fatal("giftToProto returned nil")
	}

	roundTripped := protoToGift(pbGift)
	if roundTripped == nil {
		t.Fatal("protoToGift returned nil")
	}

	// Verify fields match.
	if roundTripped.ID != gift.ID {
		t.Error("ID mismatch after round trip")
	}
	if roundTripped.SenderPubKey != gift.SenderPubKey {
		t.Error("SenderPubKey mismatch after round trip")
	}
	if string(roundTripped.RecipientKey) != string(gift.RecipientKey) {
		t.Error("RecipientKey mismatch after round trip")
	}
	if roundTripped.Effect != gift.Effect {
		t.Errorf("Effect mismatch: got %v, want %v", roundTripped.Effect, gift.Effect)
	}
	if !roundTripped.CreatedAt.Equal(gift.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", roundTripped.CreatedAt, gift.CreatedAt)
	}
	if !roundTripped.ExpiresAt.Equal(gift.ExpiresAt) {
		t.Errorf("ExpiresAt mismatch: got %v, want %v", roundTripped.ExpiresAt, gift.ExpiresAt)
	}
	if string(roundTripped.Signature) != string(gift.Signature) {
		t.Error("Signature mismatch after round trip")
	}
}

func TestProtoToGift_InvalidInput(t *testing.T) {
	// Create a minimal valid gift for giftToProto
	var giftID, senderKey [32]byte
	copy(giftID[:], []byte("gift-id-123456789012345678901234"))
	copy(senderKey[:], []byte("sender-key-1234567890123456789012"))

	// Test with invalid ID length.
	pbGift := giftToProto(&Gift{
		ID:           giftID,
		SenderPubKey: senderKey,
	})
	pbGift.Id = []byte("short")
	if protoToGift(pbGift) != nil {
		t.Error("protoToGift with short ID should return nil")
	}

	// Test with invalid sender pubkey length.
	pbGift2 := giftToProto(&Gift{
		ID:           giftID,
		SenderPubKey: senderKey,
	})
	pbGift2.SenderPubkey = []byte("short")
	if protoToGift(pbGift2) != nil {
		t.Error("protoToGift with short sender pubkey should return nil")
	}
}

func TestPersistentGiftStore_CreateGift_PersistenceFailure(t *testing.T) {
	// Create test database.
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}

	ps, err := NewPersistentGiftStore(db)
	if err != nil {
		t.Fatalf("NewPersistentGiftStore failed: %v", err)
	}

	// Close database to simulate persistence failure.
	db.Close()

	// Attempt to create gift (should fail and roll back).
	var senderKey [32]byte
	copy(senderKey[:], []byte("sender-key-1234567890123456789012"))

	_, err = ps.CreateGift(senderKey, []byte("recipient-key-123456789012345678901234"), EffectSoftGlowPulse, 25, nil)
	if err == nil {
		t.Error("expected error when database is closed, got nil")
	}
}

func TestPersistentGiftStore_LoadFromDB_SkipsExpired(t *testing.T) {
	// Create test database.
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	defer db.Close()

	// Manually persist an expired gift.
	var giftID, senderKey [32]byte
	copy(giftID[:], []byte("expired-gift-id-12345678901234567"))
	copy(senderKey[:], []byte("sender-key-1234567890123456789012"))

	expiredGift := &Gift{
		ID:           giftID,
		SenderPubKey: senderKey,
		RecipientKey: []byte("recipient-key-12345678901234567890"),
		Effect:       EffectAuroraColorShift,
		CreatedAt:    time.Now().Add(-48 * time.Hour),
		ExpiresAt:    time.Now().Add(-24 * time.Hour), // Expired
	}

	pbGift := giftToProto(expiredGift)
	data, _ := proto.Marshal(pbGift)
	db.Put(store.BucketGifts, giftID[:], data)

	// Close and reopen database.
	db.Close()

	db2, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("reopening database: %v", err)
	}
	defer db2.Close()

	// Create new store (should skip expired gift during load).
	ps, err := NewPersistentGiftStore(db2)
	if err != nil {
		t.Fatalf("NewPersistentGiftStore failed: %v", err)
	}

	// Verify expired gift was not loaded.
	loadedGift, _ := ps.GetGift(giftID)
	if loadedGift != nil {
		t.Error("expired gift should not be loaded from database")
	}
}

func TestPersistentGiftStore_NilDB_Operations(t *testing.T) {
	// Create store with nil DB.
	ps, err := NewPersistentGiftStore(nil)
	if err != nil {
		t.Fatalf("NewPersistentGiftStore with nil DB failed: %v", err)
	}

	// All operations should work in-memory with sufficient resonance.
	var senderKey [32]byte
	copy(senderKey[:], []byte("sender-key-1234567890123456789012"))

	gift, err := ps.CreateGift(senderKey, []byte("recipient-key-123456789012345678901234"), EffectSoftGlowPulse, 25, nil)
	if err != nil {
		t.Fatalf("CreateGift with nil DB failed: %v", err)
	}

	retrievedGift, err := ps.GetGift(gift.ID)
	if err != nil {
		t.Fatalf("GetGift failed: %v", err)
	}
	if retrievedGift == nil {
		t.Error("gift should exist in memory even without DB")
	}

	// GC should work without DB.
	ps.GarbageCollect()
}
