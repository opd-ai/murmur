// Package mechanics - Pulse Beats notification system.
// Per ANONYMOUS_GAME_MECHANICS.md §10, Pulse Beats are gamified notification events
// that transform routine network notifications into brief, engaging micro-interactions.
package mechanics

import (
	"sync"
	"time"
)

// BeatType identifies the type of Pulse Beat notification.
type BeatType uint8

// Beat types per ANONYMOUS_GAME_MECHANICS.md §10.
const (
	BeatGift      BeatType = iota + 1 // Phantom Gift arrived.
	BeatHunt                          // Specter Hunt began nearby.
	BeatForge                         // Sigil Forge is active.
	BeatChain                         // Echo Chain passed through user's node.
	BeatTerritory                     // User's territory is contested.
	BeatSpark                         // Surface Spark challenge started.
	BeatPuzzle                        // Cipher Puzzle available.
	BeatCouncil                       // Phantom Council activity.
	BeatMark                          // Specter Mark placed on user.
	BeatWave                          // Notable Wave interaction.
)

// BeatPriority indicates the urgency of a Pulse Beat.
type BeatPriority uint8

// Beat priorities for ordering.
const (
	BeatPriorityLow    BeatPriority = iota + 1 // Informational.
	BeatPriorityNormal                         // Standard notification.
	BeatPriorityHigh                           // Requires attention.
	BeatPriorityUrgent                         // Immediate attention.
)

// Beat display duration.
const (
	BeatDisplayDuration = 5 * time.Second        // Time beat is shown at viewport edge.
	BeatFadeDuration    = 500 * time.Millisecond // Fade out animation duration.
	BeatMaxVisible      = 3                      // Max beats visible simultaneously.
	BeatJournalMaxSize  = 1000                   // Max entries in beat journal.
)

// PulseBeat represents a single gamified notification event.
type PulseBeat struct {
	ID          [32]byte          // Unique beat ID.
	Type        BeatType          // Beat type (Gift, Hunt, etc.).
	Priority    BeatPriority      // Display priority.
	Title       string            // Brief title.
	Description string            // Optional longer description.
	TargetID    []byte            // Node ID to pan to when tapped.
	RelatedID   []byte            // Related entity ID (gift ID, hunt ID, etc.).
	Color       uint32            // RGBA color for beat glyph.
	CreatedAt   time.Time         // When the beat was created.
	ExpiresAt   time.Time         // When the beat expires (for timed events).
	ReadAt      *time.Time        // When the user dismissed/acknowledged.
	Metadata    map[string]string // Additional type-specific data.
}

// IsRead returns true if the beat has been acknowledged.
func (b *PulseBeat) IsRead() bool {
	return b.ReadAt != nil
}

// IsExpired returns true if the beat's event has passed.
func (b *PulseBeat) IsExpired() bool {
	if b.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(b.ExpiresAt)
}

// TimeRemaining returns time until expiration.
func (b *PulseBeat) TimeRemaining() time.Duration {
	if b.ExpiresAt.IsZero() {
		return 0
	}
	remaining := time.Until(b.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// BeatTypeString returns the human-readable name of a beat type.
func BeatTypeString(t BeatType) string {
	switch t {
	case BeatGift:
		return "Gift"
	case BeatHunt:
		return "Hunt"
	case BeatForge:
		return "Forge"
	case BeatChain:
		return "Chain"
	case BeatTerritory:
		return "Territory"
	case BeatSpark:
		return "Spark"
	case BeatPuzzle:
		return "Puzzle"
	case BeatCouncil:
		return "Council"
	case BeatMark:
		return "Mark"
	case BeatWave:
		return "Wave"
	default:
		return "Unknown"
	}
}

// BeatPriorityString returns the human-readable name of a priority.
func BeatPriorityString(p BeatPriority) string {
	switch p {
	case BeatPriorityLow:
		return "Low"
	case BeatPriorityNormal:
		return "Normal"
	case BeatPriorityHigh:
		return "High"
	case BeatPriorityUrgent:
		return "Urgent"
	default:
		return "Unknown"
	}
}

// BeatJournalEntry is a logged beat in the Beat Journal.
type BeatJournalEntry struct {
	Beat       *PulseBeat
	ReceivedAt time.Time
}

// BeatJournal stores the personal history of Pulse Beats.
// Per ANONYMOUS_GAME_MECHANICS.md, the journal records all Beats received,
// creating a personal history of notable network events.
type BeatJournal struct {
	mu      sync.RWMutex
	entries []*BeatJournalEntry
	byType  map[BeatType][]*BeatJournalEntry
	byID    map[[32]byte]*BeatJournalEntry
}

// NewBeatJournal creates a new Beat Journal.
func NewBeatJournal() *BeatJournal {
	return &BeatJournal{
		entries: make([]*BeatJournalEntry, 0),
		byType:  make(map[BeatType][]*BeatJournalEntry),
		byID:    make(map[[32]byte]*BeatJournalEntry),
	}
}

// AddBeat logs a new beat to the journal.
func (j *BeatJournal) AddBeat(beat *PulseBeat) {
	if beat == nil {
		return
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	// Check for duplicate.
	if _, exists := j.byID[beat.ID]; exists {
		return
	}

	entry := &BeatJournalEntry{
		Beat:       beat,
		ReceivedAt: time.Now(),
	}

	j.entries = append(j.entries, entry)
	j.byType[beat.Type] = append(j.byType[beat.Type], entry)
	j.byID[beat.ID] = entry

	// Enforce max size by removing oldest entries.
	for len(j.entries) > BeatJournalMaxSize {
		oldest := j.entries[0]
		j.entries = j.entries[1:]
		delete(j.byID, oldest.Beat.ID)
		// Note: byType is not cleaned up for performance; entries may linger.
	}
}

// GetRecent returns the N most recent journal entries.
func (j *BeatJournal) GetRecent(n int) []*BeatJournalEntry {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if n <= 0 {
		return nil
	}
	if n > len(j.entries) {
		n = len(j.entries)
	}

	// Return most recent (end of slice).
	result := make([]*BeatJournalEntry, n)
	start := len(j.entries) - n
	copy(result, j.entries[start:])

	// Reverse to get newest first.
	for i, k := 0, len(result)-1; i < k; i, k = i+1, k-1 {
		result[i], result[k] = result[k], result[i]
	}

	return result
}

// GetByType returns all entries of a specific beat type.
func (j *BeatJournal) GetByType(t BeatType) []*BeatJournalEntry {
	j.mu.RLock()
	defer j.mu.RUnlock()

	entries := j.byType[t]
	result := make([]*BeatJournalEntry, len(entries))
	copy(result, entries)
	return result
}

// GetUnread returns all unread entries.
func (j *BeatJournal) GetUnread() []*BeatJournalEntry {
	j.mu.RLock()
	defer j.mu.RUnlock()

	var result []*BeatJournalEntry
	for _, entry := range j.entries {
		if !entry.Beat.IsRead() {
			result = append(result, entry)
		}
	}
	return result
}

// MarkRead marks a beat as read by ID.
func (j *BeatJournal) MarkRead(beatID [32]byte) bool {
	j.mu.Lock()
	defer j.mu.Unlock()

	entry, ok := j.byID[beatID]
	if !ok {
		return false
	}

	if entry.Beat.ReadAt != nil {
		return false // Already read.
	}

	now := time.Now()
	entry.Beat.ReadAt = &now
	return true
}

// MarkAllRead marks all beats as read.
func (j *BeatJournal) MarkAllRead() int {
	j.mu.Lock()
	defer j.mu.Unlock()

	count := 0
	now := time.Now()
	for _, entry := range j.entries {
		if entry.Beat.ReadAt == nil {
			entry.Beat.ReadAt = &now
			count++
		}
	}
	return count
}

// Count returns the total number of entries.
func (j *BeatJournal) Count() int {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return len(j.entries)
}

// UnreadCount returns the number of unread entries.
func (j *BeatJournal) UnreadCount() int {
	j.mu.RLock()
	defer j.mu.RUnlock()

	count := 0
	for _, entry := range j.entries {
		if !entry.Beat.IsRead() {
			count++
		}
	}
	return count
}

// CountByType returns counts per beat type.
func (j *BeatJournal) CountByType() map[BeatType]int {
	j.mu.RLock()
	defer j.mu.RUnlock()

	counts := make(map[BeatType]int)
	for t, entries := range j.byType {
		counts[t] = len(entries)
	}
	return counts
}

// GetSince returns entries since a given time.
func (j *BeatJournal) GetSince(since time.Time) []*BeatJournalEntry {
	j.mu.RLock()
	defer j.mu.RUnlock()

	var result []*BeatJournalEntry
	for _, entry := range j.entries {
		if entry.ReceivedAt.After(since) {
			result = append(result, entry)
		}
	}
	return result
}

// Clear removes all entries from the journal.
func (j *BeatJournal) Clear() {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.entries = make([]*BeatJournalEntry, 0)
	j.byType = make(map[BeatType][]*BeatJournalEntry)
	j.byID = make(map[[32]byte]*BeatJournalEntry)
}

// BeatQueue manages the display queue of active Pulse Beats.
// At most BeatMaxVisible beats are shown at the viewport edge.
type BeatQueue struct {
	mu      sync.RWMutex
	pending []*PulseBeat // Beats waiting to be displayed.
	active  []*PulseBeat // Currently displayed beats.
}

// NewBeatQueue creates a new beat display queue.
func NewBeatQueue() *BeatQueue {
	return &BeatQueue{
		pending: make([]*PulseBeat, 0),
		active:  make([]*PulseBeat, 0),
	}
}

// Enqueue adds a beat to the display queue.
func (q *BeatQueue) Enqueue(beat *PulseBeat) {
	if beat == nil {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Insert by priority (higher priority first).
	inserted := false
	for i, b := range q.pending {
		if beat.Priority > b.Priority {
			// Insert before b.
			q.pending = append(q.pending[:i], append([]*PulseBeat{beat}, q.pending[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		q.pending = append(q.pending, beat)
	}
}

// Tick advances the queue state, returns beats that should be displayed.
// Call this each frame to update the display.
func (q *BeatQueue) Tick() []*PulseBeat {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()

	// Remove expired active beats.
	var stillActive []*PulseBeat
	for _, beat := range q.active {
		displayEnd := beat.CreatedAt.Add(BeatDisplayDuration)
		if now.Before(displayEnd) {
			stillActive = append(stillActive, beat)
		}
	}
	q.active = stillActive

	// Fill active from pending.
	for len(q.active) < BeatMaxVisible && len(q.pending) > 0 {
		beat := q.pending[0]
		q.pending = q.pending[1:]
		q.active = append(q.active, beat)
	}

	// Return copy of active beats.
	result := make([]*PulseBeat, len(q.active))
	copy(result, q.active)
	return result
}

// PendingCount returns the number of pending beats.
func (q *BeatQueue) PendingCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.pending)
}

// ActiveCount returns the number of currently displayed beats.
func (q *BeatQueue) ActiveCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.active)
}

// Clear removes all pending and active beats.
func (q *BeatQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.pending = make([]*PulseBeat, 0)
	q.active = make([]*PulseBeat, 0)
}

// BeatStore combines journal and queue for full beat management.
type BeatStore struct {
	journal *BeatJournal
	queue   *BeatQueue
	mu      sync.Mutex
}

// NewBeatStore creates a new beat store.
func NewBeatStore() *BeatStore {
	return &BeatStore{
		journal: NewBeatJournal(),
		queue:   NewBeatQueue(),
	}
}

// Receive processes an incoming beat, logging it and queueing for display.
func (s *BeatStore) Receive(beat *PulseBeat) {
	if beat == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.journal.AddBeat(beat)
	s.queue.Enqueue(beat)
}

// Tick advances the display queue and returns visible beats.
func (s *BeatStore) Tick() []*PulseBeat {
	return s.queue.Tick()
}

// Journal returns the beat journal.
func (s *BeatStore) Journal() *BeatJournal {
	return s.journal
}

// Queue returns the beat queue.
func (s *BeatStore) Queue() *BeatQueue {
	return s.queue
}

// UnreadCount returns number of unread journal entries.
func (s *BeatStore) UnreadCount() int {
	return s.journal.UnreadCount()
}

// MarkRead marks a beat as read.
func (s *BeatStore) MarkRead(beatID [32]byte) bool {
	return s.journal.MarkRead(beatID)
}
