package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opd-ai/murmur/pkg/content/waves"
	pb "github.com/opd-ai/murmur/proto"
)

type waveCreatedMsg struct {
	wave *pb.Wave
	err  error
}

// WavesModel handles wave compose and thread interactions.
type WavesModel struct {
	Session       *SessionState
	Compose       bool
	Content       string
	TypeIndex     int
	TTL           time.Duration
	Difficulty    uint8
	LastWave      *pb.Wave
	Status        string
	ThreadPreview []string
}

var waveTypes = []waves.WaveType{
	waves.TypeSurface,
	waves.TypeReply,
	waves.TypeVeiled,
	waves.TypeSpecter,
	waves.TypeSigil,
	waves.TypeAbyssal,
	waves.TypeMasked,
	waves.TypeBeacon,
}

// NewWavesModel creates the waves view model.
func NewWavesModel(session *SessionState) WavesModel {
	return WavesModel{
		Session:       session,
		TTL:           waves.DefaultTTL,
		Difficulty:    waves.DefaultDifficulty,
		Status:        "c: compose, 1-8: wave type, enter: submit",
		ThreadPreview: []string{"root-wave", "└─ reply-wave", "   └─ nested-reply"},
	}
}

// Update handles wave interactions.
func (m WavesModel) Update(msg tea.Msg) (WavesModel, tea.Cmd) {
	switch t := msg.(type) {
	case waveCreatedMsg:
		if t.err != nil {
			m.Status = "wave create failed: " + t.err.Error()
			return m, nil
		}
		m.LastWave = t.wave
		m.Content = ""
		m.Compose = false
		m.Status = fmt.Sprintf("wave created: type=%d ttl=%ds", t.wave.WaveType, t.wave.TtlSeconds)
		return m, nil
	case tea.KeyMsg:
		s := t.String()
		switch s {
		case "c":
			m.Compose = !m.Compose
			if m.Compose {
				m.Status = "compose enabled"
			}
		case "enter":
			if m.Compose {
				content := strings.TrimSpace(m.Content)
				if content == "" {
					m.Status = "content required"
					return m, nil
				}
				if m.Session.KeyPair == nil {
					m.Status = "generate identity first"
					return m, nil
				}
				wt := waveTypes[m.TypeIndex]
				opts := waves.DefaultCreateOptions()
				opts.TTL = m.TTL
				opts.Difficulty = m.Difficulty
				return m, func() tea.Msg {
					w, err := waves.Create(wt, []byte(content), m.Session.KeyPair, opts)
					return waveCreatedMsg{wave: w, err: err}
				}
			}
		case "backspace":
			if m.Compose && len(m.Content) > 0 {
				m.Content = m.Content[:len(m.Content)-1]
			}
		default:
			if len(s) == 1 && s[0] >= '1' && s[0] <= '8' {
				m.TypeIndex = int(s[0] - '1')
				m.Status = fmt.Sprintf("wave type set to %d", m.TypeIndex+1)
				return m, nil
			}
			if m.Compose && len(t.Runes) > 0 {
				m.Content += string(t.Runes)
			}
		}
	}
	return m, nil
}

// View renders waves panel.
func (m WavesModel) View(width int) string {
	currentType := waveTypes[m.TypeIndex]
	last := "<none>"
	if m.LastWave != nil {
		last = fmt.Sprintf("id=%x.. hop=%d ttl=%ds", m.LastWave.WaveId[:4], m.LastWave.HopCount, m.LastWave.TtlSeconds)
	}
	composeState := "off"
	if m.Compose {
		composeState = "on"
	}
	return fmt.Sprintf("Compose: %s\nWave type: %d (%v)\nTTL: %s\nDifficulty: %d\nDraft: %s\n\nThread preview:\n%s\n\nLast wave: %s\nStatus: %s", composeState, m.TypeIndex+1, currentType, m.TTL, m.Difficulty, m.Content, strings.Join(m.ThreadPreview, "\n"), last, m.Status)
}
