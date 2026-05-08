package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opd-ai/murmur/pkg/anonymous/resonance"
	"github.com/opd-ai/murmur/pkg/anonymous/specters"
)

// AnonymousModel handles Specters, Shroud, and Resonance displays.
type AnonymousModel struct {
	Session    *SessionState
	Active     int
	Score      *resonance.Score
	Status     string
	Shroud     string
	HasPrimary bool
	HasBackup  bool
	CircuitAge string
	Milestones []string
}

// NewAnonymousModel creates anonymous-layer model.
func NewAnonymousModel(session *SessionState) AnonymousModel {
	return AnonymousModel{
		Session: session,
		Score:   resonance.NewScore(),
		Status:  "n: new specter, s: switch specter, g: phantom gift, m: mark",
		Shroud:  "circuit: standby | relays: 0 | route: unavailable",
		Milestones: []string{
			"Shade (25)", "Wraith (50)", "Shade-Wraith (75)", "Phantom (100)",
			"Council-Eligible (200)", "Abyss (500)",
			"Milestone 7", "Milestone 8", "Milestone 9", "Milestone 10", "Milestone 11", "Milestone 12", "Milestone 13",
		},
	}
}

// Update handles anonymous interactions.
func (m AnonymousModel) Update(msg tea.Msg) (AnonymousModel, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "n":
		s, err := specters.NewSpecter()
		if err != nil {
			m.Status = "specter creation failed: " + err.Error()
			return m, nil
		}
		m.Session.Specters = append(m.Session.Specters, s)
		m.Active = len(m.Session.Specters) - 1
		m.Status = "specter created: " + s.Name
		m.Score.AddPublication()
	case "s":
		if len(m.Session.Specters) > 0 {
			m.Active = (m.Active + 1) % len(m.Session.Specters)
			m.Status = "active specter: " + m.Session.Specters[m.Active].Name
		}
	case "g":
		m.Status = "phantom gift queued"
		m.Score.AddGiftGiven()
	case "m":
		m.Status = "specter mark placed"
		m.Score.AddEndorsement(false)
	case "p":
		m.Status = "mini-games menu opened (puzzles/hunts/oracle/forge/shadowplay/councils)"
		m.Score.AddGameResult(true)
	case "c":
		m.HasPrimary = true
		m.HasBackup = true
		m.CircuitAge = "0m"
		m.Shroud = "circuit: active | relays: 3 | route: primary+backup"
		m.Status = "shroud circuit established"
	}
	return m, nil
}

// View renders anonymous layer panel.
func (m AnonymousModel) View(width int) string {
	active := "<none>"
	if len(m.Session.Specters) > 0 {
		active = m.Session.Specters[m.Active].Name
	}
	score := m.Score.Compute()
	rank := resonance.RankFromScore(score).String()
	echo := m.Score.EchoIndex()
	return fmt.Sprintf("Active Specter: %s\nSpecter count: %d\nShroud: %s\nShroud health: primary=%t backup=%t age=%s\nResonance: %d (%s)\nEcho Index: %.2f\n\nMilestones:\n%s\n\nStatus: %s", active, len(m.Session.Specters), m.Shroud, m.HasPrimary, m.HasBackup, m.CircuitAge, score, rank, echo, strings.Join(m.Milestones, "\n"), m.Status)
}
