// Package ui - Sigil Forge shared types.
// Types shared between forge.go and forge_stub.go to eliminate duplication.
// No build tags - available in all builds.

package ui

import (
	"sync"
	"time"
)

// ForgeType represents the type of Sigil Forge event.
type ForgeType uint8

const (
	ForgeTypeSigilArt   ForgeType = iota // Sigil art creation.
	ForgeTypeMicroFic                    // Micro-fiction writing.
	ForgeTypeRemixChain                  // Collaborative remix.
)

// ForgeEntryInfo contains information about a forge entry.
type ForgeEntryInfo struct {
	EntryID        [32]byte
	SpecterKey     [32]byte
	SpecterName    string
	Preview        string
	Amplifications int
	SubmittedAt    time.Time
	IsOwn          bool
	IsWinner       bool
}

// ForgeInfo contains information about a forge event.
type ForgeInfo struct {
	ForgeID   [32]byte
	Type      ForgeType
	Prompt    string
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
	IsActive  bool
	IsCreator bool
	Entries   []ForgeEntryInfo
}

// ForgePanelMode represents the current panel mode.
type ForgePanelMode uint8

const (
	ForgeModeView    ForgePanelMode = iota // Viewing forge details.
	ForgeModeCreate                        // Creating a new forge.
	ForgeModeSubmit                        // Submitting an entry.
	ForgeModeEntries                       // Browsing entries.
)

// ForgePanel provides UI for Sigil Forge interaction.
type ForgePanel struct {
	mu sync.RWMutex

	visible        bool
	forge          *ForgeInfo
	mode           ForgePanelMode
	selectedEntry  int
	scrollOffset   int
	entryText      string
	promptText     string
	selectedType   ForgeType
	durationChoice int
	errorMessage   string
	theme          Theme

	onCreate  func(forgeType ForgeType, prompt string, duration time.Duration)
	onSubmit  func(forgeID [32]byte, content string)
	onAmplify func(forgeID, entryID [32]byte)
}
