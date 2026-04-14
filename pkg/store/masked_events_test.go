package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestMaskedEventStore_CreateAndGet(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)

	event := &StoredMaskedEvent{
		ID:                [32]byte{1, 2, 3, 4},
		Topic:             "Test Event Topic",
		CreatorSpecterKey: [32]byte{10, 20, 30},
		StartTime:         time.Now(),
		EndTime:           time.Now().Add(1 * time.Hour),
		Duration:          1 * time.Hour,
		MaxParticipants:   10,
		State:             0,
		CreatedAt:         time.Now(),
	}

	// Create event.
	err = store.CreateEvent(event)
	if err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	// Get event.
	retrieved, err := store.GetEvent(event.ID)
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}

	if retrieved.Topic != event.Topic {
		t.Errorf("Topic = %q, want %q", retrieved.Topic, event.Topic)
	}
	if retrieved.MaxParticipants != event.MaxParticipants {
		t.Errorf("MaxParticipants = %d, want %d", retrieved.MaxParticipants, event.MaxParticipants)
	}
}

func TestMaskedEventStore_CreateDuplicate(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)

	event := &StoredMaskedEvent{
		ID:    [32]byte{1},
		Topic: "Test",
	}

	store.CreateEvent(event)

	// Try to create again.
	err = store.CreateEvent(event)
	if err != ErrMaskedEventExists {
		t.Errorf("Expected ErrMaskedEventExists, got %v", err)
	}
}

func TestMaskedEventStore_GetNotFound(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)

	_, err = store.GetEvent([32]byte{99})
	if err != ErrMaskedEventNotFound {
		t.Errorf("Expected ErrMaskedEventNotFound, got %v", err)
	}
}

func TestMaskedEventStore_UpdateEvent(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)

	event := &StoredMaskedEvent{
		ID:    [32]byte{1},
		Topic: "Original",
		State: 0,
	}
	store.CreateEvent(event)

	// Update.
	event.Topic = "Updated"
	event.State = 1
	err = store.UpdateEvent(event)
	if err != nil {
		t.Fatalf("UpdateEvent failed: %v", err)
	}

	// Verify.
	retrieved, _ := store.GetEvent(event.ID)
	if retrieved.Topic != "Updated" {
		t.Errorf("Topic = %q, want %q", retrieved.Topic, "Updated")
	}
	if retrieved.State != 1 {
		t.Errorf("State = %d, want 1", retrieved.State)
	}
}

func TestMaskedEventStore_UpdateNotFound(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)

	event := &StoredMaskedEvent{ID: [32]byte{99}}
	err = store.UpdateEvent(event)
	if err != ErrMaskedEventNotFound {
		t.Errorf("Expected ErrMaskedEventNotFound, got %v", err)
	}
}

func TestMaskedEventStore_DeleteEvent(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)

	event := &StoredMaskedEvent{
		ID:        [32]byte{1},
		StartTime: time.Now(),
	}
	store.CreateEvent(event)

	// Add a participant.
	participant := &StoredMaskedParticipant{
		EventID:         event.ID,
		MaskedPublicKey: [32]byte{50},
		Pseudonym:       "Test Mask",
	}
	store.AddParticipant(participant)

	// Delete event.
	err = store.DeleteEvent(event.ID)
	if err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}

	// Verify event is gone.
	_, err = store.GetEvent(event.ID)
	if err != ErrMaskedEventNotFound {
		t.Error("Event should be deleted")
	}

	// Verify participant is also gone.
	_, err = store.GetParticipant(event.ID, participant.MaskedPublicKey)
	if err != ErrMaskedEventNotFound {
		t.Error("Participant should be deleted with event")
	}
}

func TestMaskedEventStore_Participants(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)

	event := &StoredMaskedEvent{ID: [32]byte{1}}
	store.CreateEvent(event)

	// Add participants.
	p1 := &StoredMaskedParticipant{
		EventID:         event.ID,
		MaskedPublicKey: [32]byte{10},
		Pseudonym:       "Flickering Mask",
		JoinedAt:        time.Now(),
	}
	p2 := &StoredMaskedParticipant{
		EventID:         event.ID,
		MaskedPublicKey: [32]byte{20},
		Pseudonym:       "Silent Echo",
		JoinedAt:        time.Now(),
	}

	err = store.AddParticipant(p1)
	if err != nil {
		t.Fatalf("AddParticipant 1 failed: %v", err)
	}
	err = store.AddParticipant(p2)
	if err != nil {
		t.Fatalf("AddParticipant 2 failed: %v", err)
	}

	// List participants.
	participants, err := store.ListParticipants(event.ID)
	if err != nil {
		t.Fatalf("ListParticipants failed: %v", err)
	}
	if len(participants) != 2 {
		t.Errorf("ListParticipants returned %d, want 2", len(participants))
	}

	// Count participants.
	count, err := store.CountParticipants(event.ID)
	if err != nil {
		t.Fatalf("CountParticipants failed: %v", err)
	}
	if count != 2 {
		t.Errorf("CountParticipants = %d, want 2", count)
	}
}

func TestMaskedEventStore_UpdateParticipant(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)

	event := &StoredMaskedEvent{ID: [32]byte{1}}
	store.CreateEvent(event)

	p := &StoredMaskedParticipant{
		EventID:         event.ID,
		MaskedPublicKey: [32]byte{10},
		WaveCount:       0,
	}
	store.AddParticipant(p)

	// Update.
	p.WaveCount = 5
	p.AmplificationsReceived = 3
	err = store.UpdateParticipant(p)
	if err != nil {
		t.Fatalf("UpdateParticipant failed: %v", err)
	}

	// Verify.
	retrieved, _ := store.GetParticipant(event.ID, p.MaskedPublicKey)
	if retrieved.WaveCount != 5 {
		t.Errorf("WaveCount = %d, want 5", retrieved.WaveCount)
	}
	if retrieved.AmplificationsReceived != 3 {
		t.Errorf("AmplificationsReceived = %d, want 3", retrieved.AmplificationsReceived)
	}
}

func TestMaskedEventStore_AddParticipant_NoEvent(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)

	p := &StoredMaskedParticipant{
		EventID:         [32]byte{99}, // Non-existent event
		MaskedPublicKey: [32]byte{10},
	}

	err = store.AddParticipant(p)
	if err != ErrMaskedEventNotFound {
		t.Errorf("Expected ErrMaskedEventNotFound, got %v", err)
	}
}

func TestMaskedEventStore_ListActiveEvents(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)

	// Create events with different states.
	pending := &StoredMaskedEvent{ID: [32]byte{1}, State: 0}
	active := &StoredMaskedEvent{ID: [32]byte{2}, State: 1}
	ended := &StoredMaskedEvent{ID: [32]byte{3}, State: 2}

	store.CreateEvent(pending)
	store.CreateEvent(active)
	store.CreateEvent(ended)

	// List active.
	events, err := store.ListActiveEvents()
	if err != nil {
		t.Fatalf("ListActiveEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("ListActiveEvents returned %d events, want 1", len(events))
	}
	if events[0].ID != active.ID {
		t.Error("Should return the active event")
	}
}

func TestMaskedEventStore_CleanupExpiredEvents(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)

	// Create an old event.
	old := &StoredMaskedEvent{
		ID:        [32]byte{1},
		StartTime: time.Now().Add(-2 * time.Hour),
		EndTime:   time.Now().Add(-1 * time.Hour),
	}
	store.CreateEvent(old)

	// Create a current event.
	current := &StoredMaskedEvent{
		ID:        [32]byte{2},
		StartTime: time.Now(),
		EndTime:   time.Now().Add(1 * time.Hour),
	}
	store.CreateEvent(current)

	// Cleanup events that ended before now.
	cleaned, err := store.CleanupExpiredEvents(time.Now())
	if err != nil {
		t.Fatalf("CleanupExpiredEvents failed: %v", err)
	}
	if cleaned != 1 {
		t.Errorf("Cleaned %d events, want 1", cleaned)
	}

	// Verify old is gone.
	_, err = store.GetEvent(old.ID)
	if err != ErrMaskedEventNotFound {
		t.Error("Old event should be cleaned")
	}

	// Verify current remains.
	_, err = store.GetEvent(current.ID)
	if err != nil {
		t.Error("Current event should remain")
	}
}

func TestMaskedEventStore_ListEventsByTimeRange(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)

	now := time.Now()

	// Create events at different times.
	e1 := &StoredMaskedEvent{ID: [32]byte{1}, StartTime: now.Add(-2 * time.Hour)}
	e2 := &StoredMaskedEvent{ID: [32]byte{2}, StartTime: now}
	e3 := &StoredMaskedEvent{ID: [32]byte{3}, StartTime: now.Add(2 * time.Hour)}

	store.CreateEvent(e1)
	store.CreateEvent(e2)
	store.CreateEvent(e3)

	// Query events starting within 1 hour of now.
	events, err := store.ListEventsByTimeRange(now.Add(-1*time.Hour), now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("ListEventsByTimeRange failed: %v", err)
	}

	// Should include e2 only.
	if len(events) != 1 {
		t.Errorf("ListEventsByTimeRange returned %d events, want 1", len(events))
	}
}

func TestEventIDStringParse(t *testing.T) {
	original := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	str := EventIDString(original)
	if len(str) != 64 {
		t.Errorf("EventIDString should return 64 char hex, got %d", len(str))
	}

	parsed, err := ParseEventID(str)
	if err != nil {
		t.Fatalf("ParseEventID failed: %v", err)
	}
	if parsed != original {
		t.Error("Parsed ID should match original")
	}
}

func TestParseEventID_Invalid(t *testing.T) {
	// Invalid hex.
	_, err := ParseEventID("not-hex")
	if err == nil {
		t.Error("Should fail for invalid hex")
	}

	// Wrong length.
	_, err = ParseEventID("0102030405")
	if err == nil {
		t.Error("Should fail for wrong length")
	}
}

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		s      []byte
		prefix []byte
		want   bool
	}{
		{[]byte("hello"), []byte("hel"), true},
		{[]byte("hello"), []byte("hello"), true},
		{[]byte("hello"), []byte("helloworld"), false},
		{[]byte("hello"), []byte("bye"), false},
		{[]byte(""), []byte(""), true},
		{[]byte("abc"), []byte(""), true},
	}

	for _, tc := range tests {
		got := hasPrefix(tc.s, tc.prefix)
		if got != tc.want {
			t.Errorf("hasPrefix(%q, %q) = %v, want %v", tc.s, tc.prefix, got, tc.want)
		}
	}
}

func TestNewMaskedEventStore(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := NewMaskedEventStore(db)
	if store == nil {
		t.Fatal("NewMaskedEventStore returned nil")
	}
}
