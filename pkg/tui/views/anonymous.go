package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opd-ai/murmur/pkg/anonymous/resonance"
	"github.com/opd-ai/murmur/pkg/anonymous/specters"
)

type relayStatusRow struct {
	PeerID   string
	LastSeen string
	Quality  string
}

// AnonymousModel handles Specters, Shroud, and Resonance displays.
type AnonymousModel struct {
	Session       *SessionState
	Active        int
	Score         *resonance.Score
	Status        string
	Shroud        string
	HasPrimary    bool
	HasBackup     bool
	CircuitAge    string
	WhisperState  string
	Relays        []relayStatusRow
	Milestones    []string
	MiniGames     []string
	OverlayLegend []string
}

// NewAnonymousModel creates anonymous-layer model.
func NewAnonymousModel(session *SessionState) AnonymousModel {
	return AnonymousModel{
		Session:      session,
		Score:        resonance.NewScore(),
		Status:       "n:new specter s:switch c:circuit r:relays w:whisper p:mini-games",
		Shroud:       "circuit: standby | relays: 0 | route: unavailable",
		WhisperState: "idle",
		Relays: []relayStatusRow{
			{PeerID: "relay-a", LastSeen: "just now", Quality: "high"},
			{PeerID: "relay-b", LastSeen: "8s ago", Quality: "medium"},
		},
		MiniGames:     []string{"Cipher Puzzles", "Specter Hunts", "Territory Drift", "Oracle Pools", "Sigil Forge", "Shadow Play"},
		OverlayLegend: []string{"marks: ✦", "gifts: ♦", "echo: 🔥/🌊"},
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
		m.Status = "mini-games menu opened"
		m.Score.AddGameResult(true)
	case "1":
		m.Status = "cipher puzzles: active challenge listed"
	case "2":
		m.Status = "specter hunts: tracking board opened"
	case "3":
		m.Status = "territory drift: influence board opened"
	case "4":
		m.Status = "oracle pools: contribution panel opened"
	case "5":
		m.Status = "sigil forge: craft progress shown"
	case "6":
		m.Status = "shadow play: matchmaking status shown"
	case "c":
		m.HasPrimary = true
		m.HasBackup = true
		m.CircuitAge = "0m"
		m.Shroud = "circuit: active | relays: 3 | route: primary+backup"
		m.Status = "shroud circuit established"
	case "r":
		m.Relays = append(m.Relays, relayStatusRow{PeerID: fmt.Sprintf("relay-%d", len(m.Relays)+1), LastSeen: "now", Quality: "medium"})
		m.Status = "relay discovery updated"
	case "w":
		m.WhisperState = fmt.Sprintf("route=3-hop delivered_at=%s", time.Now().Format("15:04:05"))
		m.Status = "whisper routed over shroud"
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

	relayLines := make([]string, 0, len(m.Relays))
	for _, r := range m.Relays {
		relayLines = append(relayLines, fmt.Sprintf("- %s seen=%s quality=%s", r.PeerID, r.LastSeen, r.Quality))
	}

	return fmt.Sprintf("Active Specter: %s\nSpecter count: %d\nShroud: %s\nShroud health: primary=%t backup=%t age=%s\nWhisper: %s\nResonance: %d (%s)\nEcho Index: %.2f\n\nRelay discovery:\n%s\n\nMini-games (1-6): %s\nOverlay legends: %s\n\nMilestones:\n%s\n\nStatus: %s",
		active,
		len(m.Session.Specters),
		m.Shroud,
		m.HasPrimary,
		m.HasBackup,
		m.CircuitAge,
		m.WhisperState,
		score,
		rank,
		echo,
		strings.Join(relayLines, "\n"),
		strings.Join(m.MiniGames, ", "),
		strings.Join(m.OverlayLegend, " | "),
		strings.Join(m.Milestones, "\n"),
		m.Status,
	)
}
