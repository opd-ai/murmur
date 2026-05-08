package views

import (
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/specters"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
)

// SessionState holds mutable session state shared across TUI views.
type SessionState struct {
	KeyPair     *keys.KeyPair
	ModeManager *modes.Manager
	Specters    []*specters.Specter
	CreatedAt   time.Time
	Settings    SettingsState
	Actions     []string
}

// SettingsState stores cross-view runtime settings.
type SettingsState struct {
	ShowSettings  bool
	LayerBlend    float64
	AnonymousOnly bool
}

// NewSessionState creates default session state.
func NewSessionState() *SessionState {
	return &SessionState{
		ModeManager: modes.NewManager(),
		CreatedAt:   time.Now(),
		Settings: SettingsState{
			LayerBlend: 0.5,
		},
	}
}
