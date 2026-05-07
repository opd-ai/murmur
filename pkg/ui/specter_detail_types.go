// Package ui - Specter detail panel shared types.
// Types shared between specter_detail.go and specter_detail_stub.go to eliminate duplication.

package ui

import (
	"sync"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

// SpecterDetailMode represents the panel tab/mode.
type SpecterDetailMode uint8

const (
	SpecterModeOverview SpecterDetailMode = iota // Overview with basic info.
	SpecterModeTrophies                          // Trophy display.
	SpecterModeActivity                          // Recent activity.
	SpecterModeInteract                          // Interaction options.
)

// SpecterInfo contains information about a Specter for display.
type SpecterInfo struct {
	ID             [32]byte
	Pseudonym      string
	Resonance      float64
	Rank           string
	CreatedAt      time.Time
	LastSeenAt     time.Time
	WaveCount      int
	GiftsSent      int
	GiftsReceived  int
	PuzzlesSolved  int
	HuntsCompleted int
	IsOwnSpecter   bool
}

// TrophyDisplayInfo contains trophy info for UI display.
type TrophyDisplayInfo struct {
	Trophy   mechanics.TrophyUnlock
	Def      *mechanics.TrophyDefinition
	Glyph    *mechanics.TrophyGlyph
	Selected bool
}

// SpecterDetailCallbacks provides callbacks for panel actions.
type SpecterDetailCallbacks struct {
	OnClose             func()
	OnSendGift          func(specterID [32]byte)
	OnViewWaves         func(specterID [32]byte)
	OnAddMark           func(specterID [32]byte)
	GetTrophies         func(specterID [32]byte) []TrophyDisplayInfo
	GetTotalTrophyCount func() int
}

// SpecterDetailPanel displays detailed information about a Specter.
type SpecterDetailPanel struct {
	mu sync.RWMutex

	visible        bool
	mode           SpecterDetailMode
	theme          Theme
	callbacks      SpecterDetailCallbacks
	specter        *SpecterInfo
	trophies       []TrophyDisplayInfo
	animTime       float64
	hoverButton    int
	trophyScroll   int
	selectedTrophy int
	trophyHover    int
	panelX, panelY int
	panelW, panelH int
	transition     float64
	closing        bool
}
