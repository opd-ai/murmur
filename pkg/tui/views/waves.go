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
	Session        *SessionState
	Compose        bool
	Content        string
	TypeIndex      int
	TTL            time.Duration
	Difficulty     uint8
	LastWave       *pb.Wave
	Status         string
	ThreadPreview  []string
	WaveLog        []*pb.Wave
	ReplyToLast    bool
	Amplifications int
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
		Status:        "c: compose, 1-8 type, t/T TTL, y toggle reply, enter submit",
		ThreadPreview: []string{"<empty thread>"},
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
		m.WaveLog = append(m.WaveLog, t.wave)
		m.ThreadPreview = buildThreadPreview(m.WaveLog)
		m.Content = ""
		m.Compose = false
		m.Status = fmt.Sprintf("wave created: type=%d ttl=%ds nonce=%d", t.wave.WaveType, t.wave.TtlSeconds, t.wave.PowNonce)
		return m, nil
	case tea.KeyMsg:
		s := t.String()
		switch s {
		case "c":
			m.Compose = !m.Compose
			if m.Compose {
				m.Status = "compose enabled"
			}
		case "y":
			m.ReplyToLast = !m.ReplyToLast
			if m.ReplyToLast {
				m.Status = "reply mode enabled"
			} else {
				m.Status = "reply mode disabled"
			}
		case "t":
			m.TTL -= time.Hour
			if m.TTL < time.Hour {
				m.TTL = time.Hour
			}
		case "T":
			m.TTL += time.Hour
			if m.TTL > waves.MaxTTL {
				m.TTL = waves.MaxTTL
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
				if m.ReplyToLast && m.LastWave != nil {
					opts.ParentHash = m.LastWave.WaveId
					wt = waves.TypeReply
				}
				return m, func() tea.Msg {
					w, err := waves.Create(wt, []byte(content), m.Session.KeyPair, opts)
					return waveCreatedMsg{wave: w, err: err}
				}
			}
		case "a":
			if m.LastWave == nil || m.Session.KeyPair == nil {
				m.Status = "need a received wave and identity to amplify"
				return m, nil
			}
			if _, err := waves.CreateAmplificationWithComment(m.LastWave, m.Session.KeyPair, []byte("amplified via tui")); err != nil {
				m.Status = "amplify failed: " + err.Error()
				return m, nil
			}
			m.Amplifications++
			m.Status = fmt.Sprintf("wave amplified (%d total)", m.Amplifications)
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
	reply := "off"
	if m.ReplyToLast {
		reply = "on"
	}
	firstWaveHint := ""
	if len(m.WaveLog) == 0 {
		firstWaveHint = "First-wave tip: press c to compose, choose type 1, then Enter to publish.\n"
	}
	return fmt.Sprintf("Compose: %s\nReply-to-last: %s\nAmplifications: %d\nWave type: %d (%v)\nTTL: %s\nDifficulty: %d\nDraft: %s\n\n%sThread view:\n%s\n\nLast wave: %s\nStatus: %s", composeState, reply, m.Amplifications, m.TypeIndex+1, currentType, m.TTL, m.Difficulty, m.Content, firstWaveHint, strings.Join(m.ThreadPreview, "\n"), last, m.Status)
}

func buildThreadPreview(log []*pb.Wave) []string {
	if len(log) == 0 {
		return []string{"<empty thread>"}
	}
	lines := make([]string, 0, len(log))
	for _, w := range log {
		prefix := "•"
		if len(w.ParentHash) > 0 {
			prefix = "└─"
		}
		lines = append(lines, fmt.Sprintf("%s type=%d id=%x", prefix, w.WaveType, w.WaveId[:3]))
	}
	return lines
}
