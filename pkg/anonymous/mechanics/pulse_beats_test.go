package mechanics

import (
	"crypto/rand"
	"testing"
	"time"
)

func TestNewBeatJournal(t *testing.T) {
	j := NewBeatJournal()
	if j == nil {
		t.Fatal("NewBeatJournal returned nil")
	}
	if j.Count() != 0 {
		t.Errorf("new journal should be empty, got %d", j.Count())
	}
}

func TestBeatJournal_AddBeat(t *testing.T) {
	j := NewBeatJournal()

	beat := &PulseBeat{
		Type:      BeatGift,
		Priority:  BeatPriorityNormal,
		Title:     "Test Gift",
		CreatedAt: time.Now(),
	}
	rand.Read(beat.ID[:])

	j.AddBeat(beat)

	if j.Count() != 1 {
		t.Errorf("expected 1 entry, got %d", j.Count())
	}
}

func TestBeatJournal_AddBeat_Duplicate(t *testing.T) {
	j := NewBeatJournal()

	beat := &PulseBeat{
		Type:      BeatGift,
		Priority:  BeatPriorityNormal,
		CreatedAt: time.Now(),
	}
	rand.Read(beat.ID[:])

	j.AddBeat(beat)
	j.AddBeat(beat) // Duplicate.

	if j.Count() != 1 {
		t.Errorf("duplicate should be ignored, got %d entries", j.Count())
	}
}

func TestBeatJournal_GetRecent(t *testing.T) {
	j := NewBeatJournal()

	// Add 5 beats.
	for i := 0; i < 5; i++ {
		beat := &PulseBeat{
			Type:      BeatType(i%5 + 1),
			Priority:  BeatPriorityNormal,
			Title:     "Beat",
			CreatedAt: time.Now(),
		}
		rand.Read(beat.ID[:])
		j.AddBeat(beat)
	}

	recent := j.GetRecent(3)
	if len(recent) != 3 {
		t.Errorf("expected 3 recent, got %d", len(recent))
	}

	// First in result should be newest.
	if recent[0].ReceivedAt.Before(recent[2].ReceivedAt) {
		t.Error("recent should be ordered newest first")
	}
}

func TestBeatJournal_GetByType(t *testing.T) {
	j := NewBeatJournal()

	// Add mixed types.
	for i := 0; i < 10; i++ {
		beat := &PulseBeat{
			Type:      BeatType(i%2 + 1), // Alternating Gift/Hunt.
			CreatedAt: time.Now(),
		}
		rand.Read(beat.ID[:])
		j.AddBeat(beat)
	}

	gifts := j.GetByType(BeatGift)
	if len(gifts) != 5 {
		t.Errorf("expected 5 gift beats, got %d", len(gifts))
	}

	hunts := j.GetByType(BeatHunt)
	if len(hunts) != 5 {
		t.Errorf("expected 5 hunt beats, got %d", len(hunts))
	}
}

func TestBeatJournal_GetUnread(t *testing.T) {
	j := NewBeatJournal()

	beat1 := &PulseBeat{Type: BeatGift, CreatedAt: time.Now()}
	beat2 := &PulseBeat{Type: BeatHunt, CreatedAt: time.Now()}
	rand.Read(beat1.ID[:])
	rand.Read(beat2.ID[:])

	j.AddBeat(beat1)
	j.AddBeat(beat2)

	unread := j.GetUnread()
	if len(unread) != 2 {
		t.Errorf("expected 2 unread, got %d", len(unread))
	}

	// Mark one as read.
	j.MarkRead(beat1.ID)

	unread = j.GetUnread()
	if len(unread) != 1 {
		t.Errorf("expected 1 unread after marking, got %d", len(unread))
	}
}

func TestBeatJournal_MarkRead(t *testing.T) {
	j := NewBeatJournal()

	beat := &PulseBeat{Type: BeatGift, CreatedAt: time.Now()}
	rand.Read(beat.ID[:])
	j.AddBeat(beat)

	// First mark should succeed.
	if !j.MarkRead(beat.ID) {
		t.Error("first MarkRead should succeed")
	}

	// Second mark should fail (already read).
	if j.MarkRead(beat.ID) {
		t.Error("second MarkRead should return false")
	}

	// Non-existent ID should fail.
	var badID [32]byte
	rand.Read(badID[:])
	if j.MarkRead(badID) {
		t.Error("MarkRead on non-existent ID should return false")
	}
}

func TestBeatJournal_MarkAllRead(t *testing.T) {
	j := NewBeatJournal()

	for i := 0; i < 5; i++ {
		beat := &PulseBeat{Type: BeatGift, CreatedAt: time.Now()}
		rand.Read(beat.ID[:])
		j.AddBeat(beat)
	}

	count := j.MarkAllRead()
	if count != 5 {
		t.Errorf("expected 5 marked, got %d", count)
	}

	if j.UnreadCount() != 0 {
		t.Errorf("expected 0 unread, got %d", j.UnreadCount())
	}

	// Marking again should return 0.
	count = j.MarkAllRead()
	if count != 0 {
		t.Errorf("expected 0 on second mark, got %d", count)
	}
}

func TestBeatJournal_CountByType(t *testing.T) {
	j := NewBeatJournal()

	for i := 0; i < 3; i++ {
		beat := &PulseBeat{Type: BeatGift, CreatedAt: time.Now()}
		rand.Read(beat.ID[:])
		j.AddBeat(beat)
	}
	for i := 0; i < 2; i++ {
		beat := &PulseBeat{Type: BeatHunt, CreatedAt: time.Now()}
		rand.Read(beat.ID[:])
		j.AddBeat(beat)
	}

	counts := j.CountByType()
	if counts[BeatGift] != 3 {
		t.Errorf("expected 3 gifts, got %d", counts[BeatGift])
	}
	if counts[BeatHunt] != 2 {
		t.Errorf("expected 2 hunts, got %d", counts[BeatHunt])
	}
}

func TestBeatJournal_GetSince(t *testing.T) {
	j := NewBeatJournal()

	// Add beat before.
	beat1 := &PulseBeat{Type: BeatGift, CreatedAt: time.Now()}
	rand.Read(beat1.ID[:])
	j.AddBeat(beat1)

	checkpoint := time.Now()
	time.Sleep(10 * time.Millisecond)

	// Add beat after.
	beat2 := &PulseBeat{Type: BeatHunt, CreatedAt: time.Now()}
	rand.Read(beat2.ID[:])
	j.AddBeat(beat2)

	since := j.GetSince(checkpoint)
	if len(since) != 1 {
		t.Errorf("expected 1 entry since checkpoint, got %d", len(since))
	}
}

func TestBeatJournal_MaxSize(t *testing.T) {
	j := NewBeatJournal()

	// Add more than max entries.
	for i := 0; i < BeatJournalMaxSize+100; i++ {
		beat := &PulseBeat{Type: BeatGift, CreatedAt: time.Now()}
		rand.Read(beat.ID[:])
		j.AddBeat(beat)
	}

	if j.Count() > BeatJournalMaxSize {
		t.Errorf("journal should cap at %d, got %d", BeatJournalMaxSize, j.Count())
	}
}

func TestNewBeatQueue(t *testing.T) {
	q := NewBeatQueue()
	if q == nil {
		t.Fatal("NewBeatQueue returned nil")
	}
	if q.PendingCount() != 0 {
		t.Errorf("new queue should be empty")
	}
}

func TestBeatQueue_Enqueue(t *testing.T) {
	q := NewBeatQueue()

	beat := &PulseBeat{Type: BeatGift, Priority: BeatPriorityNormal, CreatedAt: time.Now()}
	rand.Read(beat.ID[:])

	q.Enqueue(beat)

	if q.PendingCount() != 1 {
		t.Errorf("expected 1 pending, got %d", q.PendingCount())
	}
}

func TestBeatQueue_Enqueue_Priority(t *testing.T) {
	q := NewBeatQueue()

	low := &PulseBeat{Type: BeatGift, Priority: BeatPriorityLow, CreatedAt: time.Now()}
	high := &PulseBeat{Type: BeatHunt, Priority: BeatPriorityHigh, CreatedAt: time.Now()}
	rand.Read(low.ID[:])
	rand.Read(high.ID[:])

	q.Enqueue(low)
	q.Enqueue(high) // High should jump ahead.

	active := q.Tick()
	if len(active) == 0 {
		t.Fatal("expected at least 1 active beat")
	}

	// First active should be high priority.
	if active[0].Priority != BeatPriorityHigh {
		t.Errorf("expected high priority first, got %v", active[0].Priority)
	}
}

func TestBeatQueue_Tick(t *testing.T) {
	q := NewBeatQueue()

	// Add beats.
	for i := 0; i < 5; i++ {
		beat := &PulseBeat{Type: BeatGift, Priority: BeatPriorityNormal, CreatedAt: time.Now()}
		rand.Read(beat.ID[:])
		q.Enqueue(beat)
	}

	active := q.Tick()

	// Should only show BeatMaxVisible.
	if len(active) > BeatMaxVisible {
		t.Errorf("expected max %d active, got %d", BeatMaxVisible, len(active))
	}

	// Remaining should be pending.
	if q.PendingCount() != 5-BeatMaxVisible {
		t.Errorf("expected %d pending, got %d", 5-BeatMaxVisible, q.PendingCount())
	}
}

func TestBeatQueue_Clear(t *testing.T) {
	q := NewBeatQueue()

	for i := 0; i < 5; i++ {
		beat := &PulseBeat{Type: BeatGift, Priority: BeatPriorityNormal, CreatedAt: time.Now()}
		rand.Read(beat.ID[:])
		q.Enqueue(beat)
	}

	q.Clear()

	if q.PendingCount() != 0 || q.ActiveCount() != 0 {
		t.Error("clear should remove all beats")
	}
}

func TestNewBeatStore(t *testing.T) {
	s := NewBeatStore()
	if s == nil {
		t.Fatal("NewBeatStore returned nil")
	}
	if s.Journal() == nil {
		t.Error("journal should not be nil")
	}
	if s.Queue() == nil {
		t.Error("queue should not be nil")
	}
}

func TestBeatStore_Receive(t *testing.T) {
	s := NewBeatStore()

	beat := &PulseBeat{Type: BeatGift, Priority: BeatPriorityNormal, CreatedAt: time.Now()}
	rand.Read(beat.ID[:])

	s.Receive(beat)

	if s.Journal().Count() != 1 {
		t.Errorf("expected 1 journal entry, got %d", s.Journal().Count())
	}
	if s.Queue().PendingCount() != 1 {
		t.Errorf("expected 1 pending, got %d", s.Queue().PendingCount())
	}
}

func TestBeatStore_Tick(t *testing.T) {
	s := NewBeatStore()

	beat := &PulseBeat{Type: BeatGift, Priority: BeatPriorityNormal, CreatedAt: time.Now()}
	rand.Read(beat.ID[:])
	s.Receive(beat)

	active := s.Tick()
	if len(active) != 1 {
		t.Errorf("expected 1 active, got %d", len(active))
	}
}

func TestPulseBeat_IsRead(t *testing.T) {
	beat := &PulseBeat{Type: BeatGift}

	if beat.IsRead() {
		t.Error("new beat should not be read")
	}

	now := time.Now()
	beat.ReadAt = &now

	if !beat.IsRead() {
		t.Error("beat should be read after setting ReadAt")
	}
}

func TestPulseBeat_IsExpired(t *testing.T) {
	beat := &PulseBeat{Type: BeatGift}

	// No expiry set.
	if beat.IsExpired() {
		t.Error("beat without expiry should not be expired")
	}

	// Future expiry.
	beat.ExpiresAt = time.Now().Add(time.Hour)
	if beat.IsExpired() {
		t.Error("beat with future expiry should not be expired")
	}

	// Past expiry.
	beat.ExpiresAt = time.Now().Add(-time.Hour)
	if !beat.IsExpired() {
		t.Error("beat with past expiry should be expired")
	}
}

func TestPulseBeat_TimeRemaining(t *testing.T) {
	beat := &PulseBeat{Type: BeatGift}

	// No expiry.
	if beat.TimeRemaining() != 0 {
		t.Error("beat without expiry should have 0 remaining")
	}

	// Future expiry.
	beat.ExpiresAt = time.Now().Add(time.Hour)
	remaining := beat.TimeRemaining()
	if remaining < 59*time.Minute || remaining > 61*time.Minute {
		t.Errorf("unexpected time remaining: %v", remaining)
	}

	// Past expiry.
	beat.ExpiresAt = time.Now().Add(-time.Hour)
	if beat.TimeRemaining() != 0 {
		t.Error("expired beat should have 0 remaining")
	}
}

func TestBeatTypeString(t *testing.T) {
	tests := []struct {
		t    BeatType
		want string
	}{
		{BeatGift, "Gift"},
		{BeatHunt, "Hunt"},
		{BeatForge, "Forge"},
		{BeatChain, "Chain"},
		{BeatTerritory, "Territory"},
		{BeatSpark, "Spark"},
		{BeatPuzzle, "Puzzle"},
		{BeatCouncil, "Council"},
		{BeatMark, "Mark"},
		{BeatWave, "Wave"},
		{BeatType(99), "Unknown"},
	}

	for _, tt := range tests {
		got := BeatTypeString(tt.t)
		if got != tt.want {
			t.Errorf("BeatTypeString(%d) = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestBeatPriorityString(t *testing.T) {
	tests := []struct {
		p    BeatPriority
		want string
	}{
		{BeatPriorityLow, "Low"},
		{BeatPriorityNormal, "Normal"},
		{BeatPriorityHigh, "High"},
		{BeatPriorityUrgent, "Urgent"},
		{BeatPriority(99), "Unknown"},
	}

	for _, tt := range tests {
		got := BeatPriorityString(tt.p)
		if got != tt.want {
			t.Errorf("BeatPriorityString(%d) = %q, want %q", tt.p, got, tt.want)
		}
	}
}

func BenchmarkBeatJournal_AddBeat(b *testing.B) {
	j := NewBeatJournal()

	beats := make([]*PulseBeat, b.N)
	for i := 0; i < b.N; i++ {
		beats[i] = &PulseBeat{Type: BeatGift, CreatedAt: time.Now()}
		rand.Read(beats[i].ID[:])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j.AddBeat(beats[i])
	}
}

func BenchmarkBeatQueue_Tick(b *testing.B) {
	q := NewBeatQueue()

	// Pre-fill queue.
	for i := 0; i < 100; i++ {
		beat := &PulseBeat{Type: BeatGift, Priority: BeatPriorityNormal, CreatedAt: time.Now()}
		rand.Read(beat.ID[:])
		q.Enqueue(beat)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Tick()
	}
}
