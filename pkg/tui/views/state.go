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
}

// NewSessionState creates default session state.
func NewSessionState() *SessionState {
	return &SessionState{
		ModeManager: modes.NewManager(),
		CreatedAt:   time.Now(),
	}
}
