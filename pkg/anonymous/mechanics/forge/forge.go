// Package mechanics implements anonymous layer game mechanics for MURMUR.
// Sigil Forge: Timed creative challenges where Specters compete to produce
// compelling content evaluated by amplification.
// Per ANONYMOUS_GAME_MECHANICS.md, Sigil Forge requires Resonance 50 (Wraith milestone).
package forge

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// Sigil Forge constants per ANONYMOUS_GAME_MECHANICS.md.
const (
	// ForgeMinResonance is the minimum Specter Resonance to initiate a forge.
	ForgeMinResonance = 50

	// ForgeMaxPromptLength is the maximum prompt length (256 bytes per spec).
	ForgeMaxPromptLength = 256

	// ForgeDuration30Min is 30 minute forge duration.
	ForgeDuration30Min = 30 * time.Minute

	// ForgeDuration60Min is 60 minute forge duration.
	ForgeDuration60Min = 60 * time.Minute

	// ForgeMaxEntries limits entries per forge.
	ForgeMaxEntries = 100

	// ForgeMaxEntrySize is max entry content size (2048 bytes for micro-fiction).
	ForgeMaxEntrySize = 2048

	// ForgeWinnerDisplayDuration is how long winner is displayed.
	ForgeWinnerDisplayDuration = 24 * time.Hour

	// ForgeBonusDecayDays is the decay period for forge bonuses.
	ForgeBonusDecayDays = 14
)

// ForgeType represents the type of Sigil Forge event.
type ForgeType uint8

const (
	// ForgeSigilArt is visual art creation from seed.
	ForgeSigilArt ForgeType = iota + 1

	// ForgeMicroFiction is short-form creative writing.
	ForgeMicroFiction

	// ForgeRemixChain is collaborative derivative creation.
	ForgeRemixChain
)

// ForgeState represents the state of a Sigil Forge event.
type ForgeState uint8

const (
	// ForgeActive means submissions are being accepted.
	ForgeActive ForgeState = iota

	// ForgeEvaluating means evaluation period (amplifications tallied).
	ForgeEvaluating

	// ForgeCompleted means winner has been determined.
	ForgeCompleted

	// ForgeExpired means forge expired without completion.
	ForgeExpired
)

// ForgeTypeString returns a human-readable string for ForgeType.
func ForgeTypeString(t ForgeType) string {
	switch t {
	case ForgeSigilArt:
		return "Sigil Art"
	case ForgeMicroFiction:
		return "Micro Fiction"
	case ForgeRemixChain:
		return "Remix Chain"
	default:
		return "Unknown"
	}
}

// ForgeStateString returns a human-readable string for ForgeState.
func ForgeStateString(s ForgeState) string {
	switch s {
	case ForgeActive:
		return "Active"
	case ForgeEvaluating:
		return "Evaluating"
	case ForgeCompleted:
		return "Completed"
	case ForgeExpired:
		return "Expired"
	default:
		return "Unknown"
	}
}

// Forge errors.
var (
	ErrForgeInsufficientResonance = errors.New("insufficient resonance to create forge")
	ErrForgePromptTooLong         = errors.New("forge prompt exceeds maximum length")
	ErrForgeInvalidDuration       = errors.New("invalid forge duration")
	ErrForgeInvalidType           = errors.New("invalid forge type")
	ErrForgeClosed                = errors.New("forge is not accepting entries")
	ErrForgeEntryTooLarge         = errors.New("forge entry exceeds maximum size")
	ErrForgeDuplicateEntry        = errors.New("specter has already submitted entry")
	ErrForgeEntryNotFound         = errors.New("forge entry not found")
	ErrForgeNoEntries             = errors.New("forge has no entries to evaluate")
	ErrForgeRemixNoParent         = errors.New("remix entry requires parent entry")
)

// SigilForge represents a timed creative challenge event.
type SigilForge struct {
	mu sync.RWMutex

	// ID uniquely identifies this forge (BLAKE3 hash).
	ID [32]byte

	// Type is the forge type (sigil_art, micro_fiction, remix_chain).
	Type ForgeType

	// Prompt is the creative prompt (max 256 bytes).
	Prompt string

	// InitiatorKey is the Specter public key that created this forge.
	InitiatorKey [32]byte

	// CreatedAt is when the forge was created.
	CreatedAt time.Time

	// Duration is the forge duration.
	Duration time.Duration

	// Deadline is when submissions close.
	Deadline time.Time

	// State is the current forge state.
	State ForgeState

	// Entries is the list of submitted entries.
	Entries []*ForgeEntry

	// entryBySpecter maps Specter key to their entry for deduplication.
	entryBySpecter map[string]*ForgeEntry

	// WinnerID is the ID of the winning entry (set after completion).
	WinnerID [32]byte

	// WinnerDisplayUntil is when winner display expires.
	WinnerDisplayUntil time.Time
}

// ForgeEntry represents a submission to a Sigil Forge.
type ForgeEntry struct {
	// ID uniquely identifies this entry.
	ID [32]byte

	// ForgeID links to the parent forge.
	ForgeID [32]byte

	// SpecterKey is the submitting Specter's public key.
	SpecterKey [32]byte

	// Content is the entry content (sigil seed, text, etc.).
	Content []byte

	// ParentEntryID for remix chains (zero if first in chain).
	ParentEntryID [32]byte

	// SubmittedAt is when the entry was submitted.
	SubmittedAt time.Time

	// Amplifications tracks weighted amplification count.
	Amplifications float64

	// AmplifierSet tracks who has amplified (for deduplication).
	amplifierSet map[string]bool

	// Rank is the final rank (1 = winner), set after evaluation.
	Rank int
}

// Amplification represents an amplification of a forge entry.
type Amplification struct {
	// SpecterKey is the amplifier's public key.
	SpecterKey [32]byte

	// EntryID is the entry being amplified.
	EntryID [32]byte

	// AmplifierResonance is the amplifier's current Resonance.
	AmplifierResonance float64

	// Weight is the weighted amplification value.
	Weight float64
}

// NewSigilForge creates a new Sigil Forge event.
func NewSigilForge(
	forgeType ForgeType,
	prompt string,
	initiator [32]byte,
	duration time.Duration,
	initiatorResonance int,
) (*SigilForge, error) {
	// Validate initiator resonance per RESONANCE_SYSTEM.md.
	if initiatorResonance < ForgeMinResonance {
		return nil, ErrForgeInsufficientResonance
	}

	// Validate forge type.
	if forgeType < ForgeSigilArt || forgeType > ForgeRemixChain {
		return nil, ErrForgeInvalidType
	}

	// Validate prompt length.
	if len(prompt) > ForgeMaxPromptLength {
		return nil, ErrForgePromptTooLong
	}

	// Validate duration.
	if duration != ForgeDuration30Min && duration != ForgeDuration60Min {
		return nil, ErrForgeInvalidDuration
	}

	now := time.Now()

	// Generate deterministic forge ID.
	var idSeed []byte
	idSeed = append(idSeed, initiator[:]...)
	idSeed = append(idSeed, []byte(prompt)...)
	idSeed = append(idSeed, byte(forgeType))

	var nonce [8]byte
	rand.Read(nonce[:])
	idSeed = append(idSeed, nonce[:]...)

	id := sha256.Sum256(idSeed)

	return &SigilForge{
		ID:             id,
		Type:           forgeType,
		Prompt:         prompt,
		InitiatorKey:   initiator,
		CreatedAt:      now,
		Duration:       duration,
		Deadline:       now.Add(duration),
		State:          ForgeActive,
		Entries:        make([]*ForgeEntry, 0),
		entryBySpecter: make(map[string]*ForgeEntry),
	}, nil
}

// IsActive returns true if the forge is accepting entries.
func (f *SigilForge) IsActive() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.State == ForgeActive
}

// IsCompleted returns true if the forge has been evaluated.
func (f *SigilForge) IsCompleted() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.State == ForgeCompleted
}

// UpdateState updates the forge state based on current time.
func (f *SigilForge) UpdateState() {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()

	switch f.State {
	case ForgeActive:
		if now.After(f.Deadline) {
			if len(f.Entries) > 0 {
				f.State = ForgeEvaluating
			} else {
				f.State = ForgeExpired
			}
		}
	case ForgeEvaluating:
		// Evaluation happens via Evaluate() method.
	case ForgeCompleted:
		// No state change needed.
	}
}

// SubmitEntry adds a new entry to the forge.
func (f *SigilForge) SubmitEntry(
	specter [32]byte,
	content []byte,
	parentID [32]byte,
) (*ForgeEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := f.validateSubmission(specter, content, parentID); err != nil {
		return nil, err
	}

	entry := f.createEntry(specter, content, parentID)
	f.registerEntry(entry, specter)

	return entry, nil
}

// validateSubmission checks if an entry can be submitted.
func (f *SigilForge) validateSubmission(specter [32]byte, content []byte, parentID [32]byte) error {
	if err := f.checkForgeActive(); err != nil {
		return err
	}
	if err := f.checkEntrySize(content); err != nil {
		return err
	}
	if err := f.checkDuplicateEntry(specter); err != nil {
		return err
	}
	return f.checkRemixParent(parentID)
}

// checkForgeActive verifies the forge is accepting entries.
func (f *SigilForge) checkForgeActive() error {
	if f.State != ForgeActive {
		return ErrForgeClosed
	}
	if time.Now().After(f.Deadline) {
		f.State = ForgeEvaluating
		return ErrForgeClosed
	}
	return nil
}

// checkEntrySize validates content size.
func (f *SigilForge) checkEntrySize(content []byte) error {
	if len(content) > ForgeMaxEntrySize {
		return ErrForgeEntryTooLarge
	}
	return nil
}

// checkDuplicateEntry verifies specter hasn't already submitted.
func (f *SigilForge) checkDuplicateEntry(specter [32]byte) error {
	if _, exists := f.entryBySpecter[mechanics.KeyToHex(specter[:])]; exists {
		return ErrForgeDuplicateEntry
	}
	return nil
}

// checkRemixParent validates parent exists for remix chains.
func (f *SigilForge) checkRemixParent(parentID [32]byte) error {
	if f.Type != ForgeRemixChain || parentID == ([32]byte{}) {
		return nil
	}
	for _, e := range f.Entries {
		if e.ID == parentID {
			return nil
		}
	}
	return ErrForgeRemixNoParent
}

// createEntry constructs a new forge entry.
func (f *SigilForge) createEntry(specter [32]byte, content []byte, parentID [32]byte) *ForgeEntry {
	var idSeed []byte
	idSeed = append(idSeed, f.ID[:]...)
	idSeed = append(idSeed, specter[:]...)
	idSeed = append(idSeed, content...)

	return &ForgeEntry{
		ID:             sha256.Sum256(idSeed),
		ForgeID:        f.ID,
		SpecterKey:     specter,
		Content:        content,
		ParentEntryID:  parentID,
		SubmittedAt:    time.Now(),
		Amplifications: 0,
		amplifierSet:   make(map[string]bool),
	}
}

// registerEntry adds an entry to the forge's tracking.
func (f *SigilForge) registerEntry(entry *ForgeEntry, specter [32]byte) {
	f.Entries = append(f.Entries, entry)
	f.entryBySpecter[mechanics.KeyToHex(specter[:])] = entry
}

// GetEntry retrieves an entry by ID.
func (f *SigilForge) GetEntry(entryID [32]byte) *ForgeEntry {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, e := range f.Entries {
		if e.ID == entryID {
			return e
		}
	}
	return nil
}

// GetEntryBySpecter retrieves entry by Specter key.
func (f *SigilForge) GetEntryBySpecter(specter [32]byte) *ForgeEntry {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.entryBySpecter[mechanics.KeyToHex(specter[:])]
}

// AmplifyEntry adds an amplification to an entry.
func (f *SigilForge) AmplifyEntry(
	entryID [32]byte,
	amplifier [32]byte,
	amplifierResonance float64,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Find entry.
	var entry *ForgeEntry
	for _, e := range f.Entries {
		if e.ID == entryID {
			entry = e
			break
		}
	}
	if entry == nil {
		return ErrForgeEntryNotFound
	}

	// Check amplifier hasn't already amplified.
	ampHex := mechanics.KeyToHex(amplifier[:])
	if entry.amplifierSet[ampHex] {
		return nil // Silently ignore duplicate.
	}

	// Compute weighted amplification.
	// Weight scales with amplifier's Resonance.
	weight := 1.0 + (amplifierResonance / 100.0)

	entry.Amplifications += weight
	entry.amplifierSet[ampHex] = true

	return nil
}

// Evaluate tallies amplifications and determines winner.
func (f *SigilForge) Evaluate() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.State != ForgeEvaluating && f.State != ForgeActive {
		return nil // Already evaluated or not ready.
	}

	if len(f.Entries) == 0 {
		f.State = ForgeExpired
		return ErrForgeNoEntries
	}

	// For remix chains, calculate chain scores (all contributors share).
	if f.Type == ForgeRemixChain {
		f.evaluateRemixChain()
	}

	// Sort entries by amplifications descending.
	sort.Slice(f.Entries, func(i, j int) bool {
		return f.Entries[i].Amplifications > f.Entries[j].Amplifications
	})

	// Assign ranks.
	for i, entry := range f.Entries {
		entry.Rank = i + 1
	}

	// Set winner.
	f.WinnerID = f.Entries[0].ID
	f.WinnerDisplayUntil = time.Now().Add(ForgeWinnerDisplayDuration)
	f.State = ForgeCompleted

	return nil
}

// evaluateRemixChain calculates scores for remix chains.
// In remix chains, all contributors share the chain's total amplifications.
func (f *SigilForge) evaluateRemixChain() {
	childMap := f.buildChildMap()

	for _, entry := range f.Entries {
		if entry.ParentEntryID == ([32]byte{}) {
			f.distributeChainScore(entry, childMap)
		}
	}
}

// buildChildMap creates a mapping from parent entries to their children.
func (f *SigilForge) buildChildMap() map[[32]byte][]*ForgeEntry {
	childMap := make(map[[32]byte][]*ForgeEntry)
	for _, entry := range f.Entries {
		if entry.ParentEntryID != ([32]byte{}) {
			childMap[entry.ParentEntryID] = append(childMap[entry.ParentEntryID], entry)
		}
	}
	return childMap
}

// distributeChainScore calculates and distributes scores equally among chain members.
func (f *SigilForge) distributeChainScore(root *ForgeEntry, childMap map[[32]byte][]*ForgeEntry) {
	chainTotal := f.calculateChainScore(root, childMap)
	chainMembers := f.collectChainMembers(root, childMap)

	if len(chainMembers) == 0 {
		return
	}

	sharedScore := chainTotal / float64(len(chainMembers))
	for _, member := range chainMembers {
		member.Amplifications = sharedScore
	}
}

// calculateChainScore recursively sums amplifications for an entry and descendants.
func (f *SigilForge) calculateChainScore(entry *ForgeEntry, childMap map[[32]byte][]*ForgeEntry) float64 {
	total := entry.Amplifications
	for _, child := range childMap[entry.ID] {
		total += f.calculateChainScore(child, childMap)
	}
	return total
}

// collectChainMembers gathers all entries in a chain starting from root.
func (f *SigilForge) collectChainMembers(root *ForgeEntry, childMap map[[32]byte][]*ForgeEntry) []*ForgeEntry {
	var members []*ForgeEntry
	var collect func(e *ForgeEntry)
	collect = func(e *ForgeEntry) {
		members = append(members, e)
		for _, child := range childMap[e.ID] {
			collect(child)
		}
	}
	collect(root)
	return members
}

// GetWinner returns the winning entry after evaluation.
func (f *SigilForge) GetWinner() *ForgeEntry {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.State != ForgeCompleted {
		return nil
	}

	return f.GetEntry(f.WinnerID)
}

// GetLeaderboard returns entries sorted by amplifications.
func (f *SigilForge) GetLeaderboard() []*ForgeEntry {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Return copy sorted by amplifications.
	result := make([]*ForgeEntry, len(f.Entries))
	copy(result, f.Entries)

	sort.Slice(result, func(i, j int) bool {
		return result[i].Amplifications > result[j].Amplifications
	})

	return result
}

// EntryCount returns the number of entries.
func (f *SigilForge) EntryCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.Entries)
}

// ComputeForgeWinnerBonus calculates Resonance bonus for forge winner.
// Per spec: forge_bonus = 4 * ln(1 + weighted_amplifications).
func ComputeForgeWinnerBonus(weightedAmplifications float64) float64 {
	return 4.0 * math.Log(1.0+weightedAmplifications)
}

// ComputeForgeParticipationBonus calculates Resonance bonus for participants.
// Per spec: participation_bonus = 2 * ln(1 + own_amplifications_received).
func ComputeForgeParticipationBonus(ownAmplifications float64) float64 {
	return 2.0 * math.Log(1.0+ownAmplifications)
}

// ForgeStore manages active and historical Sigil Forge events.
type ForgeStore struct {
	mu     sync.RWMutex
	forges map[string]*SigilForge
}

// NewForgeStore creates a new ForgeStore.
func NewForgeStore() *ForgeStore {
	return &ForgeStore{
		forges: make(map[string]*SigilForge),
	}
}

// AddForge adds a forge to the store.
func (s *ForgeStore) AddForge(forge *SigilForge) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.forges[hex.EncodeToString(forge.ID[:])] = forge
}

// GetForge retrieves a forge by ID.
func (s *ForgeStore) GetForge(id [32]byte) *SigilForge {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.forges[hex.EncodeToString(id[:])]
}

// RemoveForge removes a forge from the store.
func (s *ForgeStore) RemoveForge(id [32]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.forges, hex.EncodeToString(id[:]))
}

// GetActiveForges returns all forges in Active state.
func (s *ForgeStore) GetActiveForges() []*SigilForge {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []*SigilForge
	for _, forge := range s.forges {
		if forge.IsActive() {
			active = append(active, forge)
		}
	}
	return active
}

// GetForgesByType returns forges filtered by type.
func (s *ForgeStore) GetForgesByType(forgeType ForgeType) []*SigilForge {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*SigilForge
	for _, forge := range s.forges {
		if forge.Type == forgeType {
			result = append(result, forge)
		}
	}
	return result
}

// Count returns the number of forges in the store.
func (s *ForgeStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.forges)
}

// UpdateAllStates updates state for all stored forges.
func (s *ForgeStore) UpdateAllStates() {
	s.mu.RLock()
	forges := make([]*SigilForge, 0, len(s.forges))
	for _, f := range s.forges {
		forges = append(forges, f)
	}
	s.mu.RUnlock()

	for _, forge := range forges {
		forge.UpdateState()
	}
}

// PruneExpired removes expired forges older than the retention period.
func (s *ForgeStore) PruneExpired(retention time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-retention)
	pruned := 0

	for id, forge := range s.forges {
		if forge.State == ForgeExpired && forge.CreatedAt.Before(cutoff) {
			delete(s.forges, id)
			pruned++
		}
		if forge.State == ForgeCompleted &&
			forge.WinnerDisplayUntil.Before(cutoff) {
			delete(s.forges, id)
			pruned++
		}
	}

	return pruned
}
