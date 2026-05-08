package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opd-ai/murmur/pkg/tui/bridge"
"github.com/opd-ai/murmur/pkg/tui/components"
"github.com/opd-ai/murmur/pkg/tui/input"
"github.com/opd-ai/murmur/pkg/tui/styles"
"github.com/opd-ai/murmur/pkg/tui/views"
)

const (
tabPulseMap = iota
tabIdentity
tabWaves
tabAnonymous
tabOnboarding
tabNetworking
)

var tabNames = []string{"Pulse Map", "Identity", "Waves", "Anonymous", "Onboarding", "Networking"}

// Config configures the TUI runtime.
type Config struct {
	EventStream *bridge.EventStream
}

// Model is the root Bubble Tea model.
type Model struct {
ctx      context.Context
cancel   context.CancelFunc
width    int
height   int
active   int
showHelp bool

theme   styles.Theme
keys    input.KeyMap
session *views.SessionState

pulse      views.PulseMapModel
identity   views.IdentityModel
waves      views.WavesModel
anonymous  views.AnonymousModel
onboarding views.OnboardingModel
networking views.NetworkingModel

stream *bridge.EventStream
}

// NewModel creates the root TUI model.
func NewModel(cfg Config) Model {
ctx, cancel := context.WithCancel(context.Background())
session := views.NewSessionState()
	stream := cfg.EventStream
return Model{
ctx:        ctx,
cancel:     cancel,
theme:      styles.NewTheme(),
keys:       input.NewKeyMap(),
session:    session,
pulse:      views.NewPulseMapModel(),
identity:   views.NewIdentityModel(session),
waves:      views.NewWavesModel(session),
anonymous:  views.NewAnonymousModel(session),
onboarding: views.NewOnboardingModel(),
networking: views.NewNetworkingModel(),
stream:     stream,
}
}

// NewProgram creates a Bubble Tea program configured for TUI operation.
func NewProgram(cfg Config) *tea.Program {
m := NewModel(cfg)
return tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
}

// Init starts asynchronous subscriptions.
func (m Model) Init() tea.Cmd {
if m.stream == nil {
return nil
}
return m.stream.WaitCmd(m.ctx)
}

// Update handles root-level messages and routes to active views.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
switch t := msg.(type) {
case tea.WindowSizeMsg:
m.width = t.Width
m.height = t.Height
return m, nil
	case bridge.EventMsg:
		m.networking.ApplyEventType(t.Type)
if m.stream != nil {
return m, m.stream.WaitCmd(m.ctx)
}
return m, nil
case tea.KeyMsg:
switch t.String() {
case "q", "ctrl+c":
m.cancel()
if m.stream != nil {
m.stream.Close()
}
return m, tea.Quit
case "?":
m.showHelp = !m.showHelp
return m, nil
case "tab":
m.active = (m.active + 1) % len(tabNames)
return m, nil
case "shift+tab":
m.active = (m.active - 1 + len(tabNames)) % len(tabNames)
return m, nil
case "1", "2", "3", "4", "5", "6":
m.active = int(t.String()[0]-'1')
if m.active >= len(tabNames) {
m.active = 0
}
return m, nil
}
}

switch m.active {
case tabPulseMap:
next, cmd := m.pulse.Update(msg)
m.pulse = next
return m, cmd
case tabIdentity:
next, cmd := m.identity.Update(msg)
m.identity = next
return m, cmd
case tabWaves:
next, cmd := m.waves.Update(msg)
m.waves = next
return m, cmd
case tabAnonymous:
next, cmd := m.anonymous.Update(msg)
m.anonymous = next
return m, cmd
case tabOnboarding:
next, cmd := m.onboarding.Update(msg)
m.onboarding = next
return m, cmd
case tabNetworking:
next, cmd := m.networking.Update(msg)
m.networking = next
return m, cmd
}

return m, nil
}

// View renders the full TUI.
func (m Model) View() string {
header := m.theme.Header.Render("MURMUR TUI") + "\n" + components.Tabs(m.theme, tabNames, m.active)

body := ""
switch m.active {
case tabPulseMap:
body = m.pulse.View(m.width)
case tabIdentity:
body = m.identity.View(m.width)
case tabWaves:
body = m.waves.View(m.width)
case tabAnonymous:
body = m.anonymous.View(m.width)
case tabOnboarding:
body = m.onboarding.View(m.width)
case tabNetworking:
body = m.networking.View(m.width)
}

status := components.StatusBar(m.theme, "tab: cycle views | 1-6 jump views | ?: help", fmt.Sprintf("view=%s", tabNames[m.active]))

parts := []string{header, m.theme.Panel.Render(body), status}
if m.showHelp {
parts = append(parts, components.HelpOverlay(m.theme, []string{
"Global: q/Ctrl+C quit, Tab/Shift+Tab switch views, 1-6 jump view",
"Pulse Map: hjkl/arrows pan, +/- zoom, n fit/reset, enter/click select",
"Identity: g generate keypair+mnemonic, 1-4 privacy mode",
"Waves: c compose, 1-8 wave type, enter submit",
"Anonymous: n new specter, s switch specter, g gift, m mark, p mini-games",
"Onboarding: enter advance phase, space skip",
"Networking: d refresh dht, r reset rate-limit indicator",
}))
}

return strings.Join(parts, "\n\n")
}
