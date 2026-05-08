package views

import (
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/specters"
	"github.com/opd-ai/murmur/pkg/config"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
)

// SessionState holds mutable session state shared across TUI views.
type SessionState struct {
	KeyPair          *keys.KeyPair
	ModeManager      *modes.Manager
	Specters         []*specters.Specter
	CreatedAt        time.Time
	Settings         SettingsState
	Actions          []string
	Config           *config.Config
	OnboardingResume OnboardingResumeState
}

// SettingsState stores cross-view runtime settings.
type SettingsState struct {
	ShowSettings   bool
	LayerBlend     float64
	AnonymousOnly  bool
	Overlays       map[string]bool
	Theme          string
	ContrastMode   bool
	WordlistSource string
	WordlistSample string
}

// OnboardingResumeState tracks lightweight onboarding persistence for the TUI.
type OnboardingResumeState struct {
	CompletedPhases map[string]bool
	Skipped         bool
	ReturningUser   bool
}

// NewSessionState creates default session state.
func NewSessionState() *SessionState {
	return &SessionState{
		ModeManager: modes.NewManager(),
		CreatedAt:   time.Now(),
		Settings: SettingsState{
			LayerBlend: 0.5,
			Overlays: map[string]bool{
				"heatmap": true,
				"marks":   true,
				"gifts":   true,
				"echo":    true,
			},
			Theme:          "default",
			WordlistSource: "assets/wordlists/specter-names.txt",
		},
		Config: config.DefaultConfig(),
		OnboardingResume: OnboardingResumeState{
			CompletedPhases: make(map[string]bool),
		},
	}
}
