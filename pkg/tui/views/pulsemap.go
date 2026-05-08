package views

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opd-ai/murmur/pkg/pulsemap/layout"
)

// PulseNode is a simplified render node for terminal Pulse Map view.
type PulseNode struct {
	ID          string
	X           float64
	Y           float64
	Activity    float64
	Connections int
}

type pulseTickMsg time.Time

// PulseMapModel renders terminal Pulse Map interactions.
type PulseMapModel struct {
	Session *SessionState

	CameraX float64
	CameraY float64
	Zoom    float64
	Focus   int
	Nodes   []PulseNode
	Edges   []layout.Edge

	dragging      bool
	status        string
	showDetail    bool
	showActions   bool
	searchMode    bool
	searchQuery   string
	searchMatches []int
	bookmarks     map[int]struct{}
	engine        *layout.Engine
	lastSelectAt  time.Time
}

// NewPulseMapModel returns a Pulse Map model with layout-backed seed graph.
func NewPulseMapModel(session *SessionState) PulseMapModel {
	nodes := []PulseNode{
		{ID: "self", Activity: 0.8, Connections: 3},
		{ID: "peer-a", Activity: 0.5, Connections: 2},
		{ID: "peer-b", Activity: 0.2, Connections: 1},
		{ID: "peer-c", Activity: 0.7, Connections: 2},
		{ID: "peer-d", Activity: 0.4, Connections: 2},
	}
	edges := []layout.Edge{
		{SourceID: "self", TargetID: "peer-a", Age: 1},
		{SourceID: "self", TargetID: "peer-b", Age: 2},
		{SourceID: "self", TargetID: "peer-c", Age: 1},
		{SourceID: "peer-a", TargetID: "peer-d", Age: 3},
	}

	engine := layout.NewEngine()
	for _, n := range nodes {
		engine.AddNode(&layout.Node{ID: n.ID, Connections: n.Connections, Activity: n.Activity})
	}
	for _, e := range edges {
		engine.AddEdge(e)
	}
	engine.Tick()

	m := PulseMapModel{
		Session:   session,
		Zoom:      1,
		Nodes:     nodes,
		Edges:     edges,
		bookmarks: make(map[int]struct{}),
		status:    "hjkl/arrows pan, +/- zoom, / search, enter detail, m actions",
		engine:    engine,
	}
	m.syncFromLayout()
	return m
}

// InitCmd starts the periodic layout tick loop.
func (m PulseMapModel) InitCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return pulseTickMsg(t) })
}

// Update handles pulse map interactions.
func (m PulseMapModel) Update(msg tea.Msg) (PulseMapModel, tea.Cmd) {
	switch t := msg.(type) {
	case pulseTickMsg:
		m.engine.Tick()
		m.syncFromLayout()
		return m, m.InitCmd()
	case tea.KeyMsg:
		if m.searchMode {
			return m.handleSearchKey(t)
		}
		switch t.String() {
		case "h", "left":
			m.CameraX -= 2 / m.Zoom
		case "j", "down":
			m.CameraY += 2 / m.Zoom
		case "k", "up":
			m.CameraY -= 2 / m.Zoom
		case "l", "right":
			m.CameraX += 2 / m.Zoom
		case "+", "=":
			m.Zoom = clampZoom(m.Zoom * 1.1)
		case "-":
			m.Zoom = clampZoom(m.Zoom / 1.1)
		case "enter":
			if len(m.Nodes) > 0 {
				m.Focus = (m.Focus + 1) % len(m.Nodes)
				m.showDetail = true
				m.status = "Node detail opened: " + m.Nodes[m.Focus].ID
				m.lastSelectAt = time.Now()
			}
		case "z":
			m.centerOnFocus()
			m.Zoom = clampZoom(m.Zoom * 1.2)
			m.status = "Centered and zoomed on focus"
		case "/":
			m.searchMode = true
			m.searchQuery = ""
			m.searchMatches = nil
			m.status = "Search mode"
		case "esc":
			m.showDetail = false
			m.showActions = false
			m.searchMode = false
		case "n", "home", "H":
			m.CameraX, m.CameraY, m.Zoom = 0, 0, 1
			m.status = "Viewport fit/reset"
		case "1":
			m.Zoom = 0.3
		case "2":
			m.Zoom = 1.0
		case "3":
			m.Zoom = 3.0
		case "m":
			m.showActions = !m.showActions
		case "ctrl+b":
			m.bookmarks[m.Focus] = struct{}{}
			m.status = "Bookmark added"
		case "ctrl+shift+b":
			delete(m.bookmarks, m.Focus)
			m.status = "Bookmark removed"
		case "ctrl+1", "ctrl+2", "ctrl+3", "ctrl+4", "ctrl+5", "ctrl+6", "ctrl+7", "ctrl+8", "ctrl+9":
			idx := int(t.String()[5] - '1')
			if idx >= 0 && idx < len(m.Nodes) {
				m.Focus = idx
				m.centerOnFocus()
				m.status = "Jumped to bookmark slot"
			}
		}
	case tea.MouseMsg:
		switch t.Action {
		case tea.MouseActionPress:
			if t.Button == tea.MouseButtonLeft {
				m.dragging = true
			}
			if t.Button == tea.MouseButtonRight {
				m.showActions = !m.showActions
			}
			if t.Button == tea.MouseButtonWheelUp {
				m.Zoom = clampZoom(m.Zoom * 1.1)
			}
			if t.Button == tea.MouseButtonWheelDown {
				m.Zoom = clampZoom(m.Zoom / 1.1)
			}
		case tea.MouseActionRelease:
			if t.Button == tea.MouseButtonLeft {
				m.dragging = false
				if len(m.Nodes) > 0 {
					if time.Since(m.lastSelectAt) < 500*time.Millisecond {
						m.centerOnFocus()
						m.Zoom = clampZoom(m.Zoom * 1.2)
						m.status = "Double-click center/zoom"
						m.lastSelectAt = time.Time{}
						return m, nil
					}
					m.Focus = (m.Focus + 1) % len(m.Nodes)
					m.showDetail = true
					m.status = "Selected node: " + m.Nodes[m.Focus].ID
					m.lastSelectAt = time.Now()
				}
			}
		case tea.MouseActionMotion:
			if m.dragging {
				m.CameraX -= 0.4 / m.Zoom
				m.CameraY -= 0.2 / m.Zoom
				m.status = "Panning"
			}
		}
	}
	return m, nil
}

func (m PulseMapModel) handleSearchKey(k tea.KeyMsg) (PulseMapModel, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.searchMode = false
		m.status = "Search canceled"
	case "enter":
		m.applySearch()
		m.searchMode = false
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
	default:
		if len(k.Runes) > 0 {
			m.searchQuery += string(k.Runes)
		}
	}
	return m, nil
}

func (m *PulseMapModel) applySearch() {
	q := strings.TrimSpace(strings.ToLower(m.searchQuery))
	m.searchMatches = m.searchMatches[:0]
	if q == "" {
		return
	}
	for i, n := range m.Nodes {
		if strings.Contains(strings.ToLower(n.ID), q) {
			m.searchMatches = append(m.searchMatches, i)
		}
	}
	if len(m.searchMatches) == 0 {
		m.status = "No search results"
		return
	}
	m.Focus = m.searchMatches[0]
	m.centerOnFocus()
	m.showDetail = true
	m.status = fmt.Sprintf("Search matched %d node(s)", len(m.searchMatches))
}

func (m *PulseMapModel) centerOnFocus() {
	if m.Focus < 0 || m.Focus >= len(m.Nodes) {
		return
	}
	m.CameraX = m.Nodes[m.Focus].X
	m.CameraY = m.Nodes[m.Focus].Y
}

func (m *PulseMapModel) syncFromLayout() {
	if m.engine == nil {
		return
	}
	positions := m.engine.Positions().Get()
	for i := range m.Nodes {
		if p, ok := positions[m.Nodes[i].ID]; ok {
			m.Nodes[i].X = p.X
			m.Nodes[i].Y = p.Y
		}
	}
}

// View renders pulse map content.
func (m PulseMapModel) View(width int) string {
	if len(m.Nodes) == 0 {
		return "No nodes"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Camera x=%.1f y=%.1f zoom=%.2f blend=%.0f%%\n", m.CameraX, m.CameraY, m.Zoom, m.Session.Settings.LayerBlend*100))
	if m.searchMode {
		b.WriteString("Search: " + m.searchQuery + "\n")
	}
	b.WriteString("Nodes:\n")
	for i, n := range m.Nodes {
		marker := " "
		if i == m.Focus {
			marker = "▶"
		}
		dx := (n.X - m.CameraX) * m.Zoom
		dy := (n.Y - m.CameraY) * m.Zoom
		glyph := activityGlyph(n.Activity)
		state := nodeStateLabel(n.Activity)
		bookmark := " "
		if _, ok := m.bookmarks[i]; ok {
			bookmark = "★"
		}
		markGlyph, giftGlyph, echoGlyph := " ", " ", " "
		if m.Session.Settings.Overlays["marks"] && i%2 == 0 {
			markGlyph = "✦"
		}
		if m.Session.Settings.Overlays["gifts"] && i%3 == 0 {
			giftGlyph = "♦"
		}
		if m.Session.Settings.Overlays["echo"] {
			switch {
			case n.Activity > 0.7:
				echoGlyph = "🔥"
			case n.Activity < 0.35:
				echoGlyph = "🌊"
			default:
				echoGlyph = "•"
			}
		}
		b.WriteString(fmt.Sprintf("%s%s %s %-8s (%6.1f,%6.1f) %s %s%s%s\n", marker, bookmark, glyph, n.ID, dx, dy, state, markGlyph, giftGlyph, echoGlyph))
	}
	b.WriteString("\nEdges:\n")
	for _, e := range m.Edges {
		b.WriteString(fmt.Sprintf("  %s ── %s (age %.0fd)\n", e.SourceID, e.TargetID, e.Age))
	}
	b.WriteString("\nMinimap:\n")
	b.WriteString(m.minimap())
	if m.showDetail {
		b.WriteString("\n\nNode Detail:\n")
		b.WriteString(m.nodeDetail())
	}
	if m.showActions {
		b.WriteString("\n\nActions: [compose wave] [send gift] [place mark] [send whisper]\n")
	}
	if m.Session.Settings.Overlays["heatmap"] {
		b.WriteString("\nHeatmap: " + heatmapBar(m.Nodes) + "\n")
	}
	b.WriteString(fmt.Sprintf("Overlay legends: marks[%t] gifts[%t] echo[%t] heatmap[%t]\n",
		m.Session.Settings.Overlays["marks"],
		m.Session.Settings.Overlays["gifts"],
		m.Session.Settings.Overlays["echo"],
		m.Session.Settings.Overlays["heatmap"],
	))
	b.WriteString("\nStatus: " + m.status)
	return b.String()
}

func (m PulseMapModel) nodeDetail() string {
	if m.Focus < 0 || m.Focus >= len(m.Nodes) {
		return "<none>"
	}
	n := m.Nodes[m.Focus]
	fingerprint := fmt.Sprintf("%x", hashNodeID(n.ID))
	if len(fingerprint) > 12 {
		fingerprint = fingerprint[:12]
	}
	return fmt.Sprintf("ID: %s\nFingerprint: %s\nActivity: %.2f\nConnections: %d", n.ID, fingerprint, n.Activity, n.Connections)
}

func (m PulseMapModel) minimap() string {
	if len(m.Nodes) == 0 {
		return "<empty>"
	}
	xs := make([]float64, 0, len(m.Nodes))
	ys := make([]float64, 0, len(m.Nodes))
	for _, n := range m.Nodes {
		xs = append(xs, n.X)
		ys = append(ys, n.Y)
	}
	sort.Float64s(xs)
	sort.Float64s(ys)
	return fmt.Sprintf("  x:[%.1f..%.1f] y:[%.1f..%.1f] focus:%s", xs[0], xs[len(xs)-1], ys[0], ys[len(ys)-1], m.Nodes[m.Focus].ID)
}

func hashNodeID(id string) []byte {
	h := 0
	for _, r := range id {
		h = (h*31 + int(r)) & 0x7fffffff
	}
	return []byte(fmt.Sprintf("%08x", h))
}

func activityGlyph(v float64) string {
	switch {
	case v >= 0.75:
		return "●"
	case v >= 0.5:
		return "◉"
	case v >= 0.25:
		return "○"
	default:
		return "·"
	}
}

func nodeStateLabel(activity float64) string {
	switch {
	case activity >= 0.75:
		return "\x1b[31mHOT\x1b[0m"
	case activity >= 0.5:
		return "\x1b[33mWARM\x1b[0m"
	case activity >= 0.25:
		return "\x1b[36mCOOL\x1b[0m"
	default:
		return "\x1b[34mIDLE\x1b[0m"
	}
}

func heatmapBar(nodes []PulseNode) string {
	if len(nodes) == 0 {
		return "<empty>"
	}
	var b strings.Builder
	for _, n := range nodes {
		switch {
		case n.Activity >= 0.75:
			b.WriteString("▓")
		case n.Activity >= 0.5:
			b.WriteString("▒")
		case n.Activity >= 0.25:
			b.WriteString("░")
		default:
			b.WriteString("·")
		}
	}
	return b.String()
}

func clampZoom(z float64) float64 {
	return math.Max(0.2, math.Min(5, z))
}
