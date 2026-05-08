package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opd-ai/murmur/pkg/anonymous/specters"
	"github.com/opd-ai/murmur/pkg/identity/modes"
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
		pulse:      views.NewPulseMapModel(session),
		identity:   views.NewIdentityModel(session),
		waves:      views.NewWavesModel(session),
		anonymous:  views.NewAnonymousModel(session),
		onboarding: views.NewOnboardingModel(session),
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
	cmds := make([]tea.Cmd, 0, 2)
	cmds = append(cmds, m.pulse.InitCmd())
	if m.stream != nil {
		cmds = append(cmds, m.stream.WaitCmd(m.ctx))
	}
	return tea.Batch(cmds...)
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
		if m.session.Settings.ShowSettings {
			switch t.String() {
			case "r":
				m.session.Config.EnableRelay = !m.session.Config.EnableRelay
				return m, m.emitActionCmd("config_toggle", map[string]any{"field": "enable_relay", "value": m.session.Config.EnableRelay})
			case "o":
				m.session.Config.EnableTor = !m.session.Config.EnableTor
				return m, m.emitActionCmd("config_toggle", map[string]any{"field": "enable_tor", "value": m.session.Config.EnableTor})
			case "i":
				m.session.Config.EnableI2P = !m.session.Config.EnableI2P
				return m, m.emitActionCmd("config_toggle", map[string]any{"field": "enable_i2p", "value": m.session.Config.EnableI2P})
			case "h":
				m.session.Config.EnableHealthEndpoint = !m.session.Config.EnableHealthEndpoint
				return m, m.emitActionCmd("config_toggle", map[string]any{"field": "enable_health_endpoint", "value": m.session.Config.EnableHealthEndpoint})
			case "t":
				switch m.session.Settings.Theme {
				case "default":
					m.session.Settings.Theme = "midnight"
				case "midnight":
					m.session.Settings.Theme = "high-contrast"
				default:
					m.session.Settings.Theme = "default"
				}
				return m, m.emitActionCmd("theme_changed", map[string]any{"theme": m.session.Settings.Theme})
			case "v":
				m.session.Settings.ContrastMode = !m.session.Settings.ContrastMode
				return m, m.emitActionCmd("contrast_mode", map[string]any{"enabled": m.session.Settings.ContrastMode})
			case "w":
				if s, err := specters.NewSpecter(); err == nil {
					m.session.Settings.WordlistSample = s.Name
				}
				return m, m.emitActionCmd("wordlist_regenerate", map[string]any{"sample": m.session.Settings.WordlistSample})
			}
		}
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
		case "ctrl+,":
			m.session.Settings.ShowSettings = !m.session.Settings.ShowSettings
			return m, m.emitActionCmd("toggle_settings", map[string]any{"visible": m.session.Settings.ShowSettings})
		case "tab":
			m.active = (m.active + 1) % len(tabNames)
			return m, nil
		case "shift+tab":
			m.active = (m.active - 1 + len(tabNames)) % len(tabNames)
			return m, nil
		case "1", "2", "3", "4", "5", "6", "7", "8", "9", "0":
			if m.session.Settings.ShowSettings {
				return m, m.applySettingKey(t.String())
			}
			m.active = int(t.String()[0] - '1')
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
	if m.session.Settings.ShowSettings {
		parts = append(parts, components.HelpOverlay(m.theme, []string{
			fmt.Sprintf("Settings modal (Ctrl+, to close)"),
			fmt.Sprintf("1-4 privacy mode  | current=%s", m.session.ModeManager.Current().String()),
			fmt.Sprintf("5 blend down      | blend=%.0f%%", m.session.Settings.LayerBlend*100),
			fmt.Sprintf("6 blend up        | anonymous-only=%t", m.session.Settings.AnonymousOnly),
			fmt.Sprintf("7 heatmap [%t] 8 marks [%t] 9 gifts [%t] 0 echo [%t]",
				m.session.Settings.Overlays["heatmap"],
				m.session.Settings.Overlays["marks"],
				m.session.Settings.Overlays["gifts"],
				m.session.Settings.Overlays["echo"],
			),
			fmt.Sprintf("config relay=%t tor=%t i2p=%t health=%t",
				m.session.Config.EnableRelay,
				m.session.Config.EnableTor,
				m.session.Config.EnableI2P,
				m.session.Config.EnableHealthEndpoint,
			),
			fmt.Sprintf("config keys: r=relay o=tor i=i2p h=health | dataDir=%s", m.session.Config.DataDir),
			fmt.Sprintf("theme: %s (t cycle) | contrast mode (v): %t", m.session.Settings.Theme, m.session.Settings.ContrastMode),
			fmt.Sprintf("wordlist: %s | sample: %s (w regenerate)", m.session.Settings.WordlistSource, m.session.Settings.WordlistSample),
		}))
	}
	if m.showHelp {
		parts = append(parts, components.HelpOverlay(m.theme, []string{
			"Global: q/Ctrl+C quit, Tab/Shift+Tab switch views, 1-6 jump view",
			"Pulse Map: hjkl/arrows pan, +/- zoom, n/home fit-reset, / search, enter/click detail, z double-tap center, m actions",
			"Identity: g generate keypair+mnemonic, d declaration mode, n/b select name/bio field, u publish declaration",
			"Waves: c compose, y reply-to-last toggle, a amplify, 1-8 wave type, t/T TTL, enter submit",
			"Anonymous: n/s specters, c shroud, r relays, w whisper, p mini-games, 1-6 game boards",
			"Onboarding: enter advance, space skip+resume, i invitation warm-start, r recovery branch, b returning mode, x/a hint dismiss/ack",
			"Networking: d refresh dht, n relay diagnostics, g simulate gossip, r reset rate-limit indicator, protocol refs visible",
			"Settings: Ctrl+, toggle modal, 1-4 mode, 5/6 layer blend, 7-0 overlays, r/o/i/h config toggles, t theme, v contrast, w wordlist regen",
		}))
	}

	return strings.Join(parts, "\n\n")
}

func (m Model) applySettingKey(key string) tea.Cmd {
	switch key {
	case "1":
		_ = m.session.ModeManager.Transition(modes.Open)
		m.session.Settings.AnonymousOnly = false
	case "2":
		m.session.ModeManager.SetSpecterAvailable(true)
		_ = m.session.ModeManager.Transition(modes.Hybrid)
		m.session.Settings.AnonymousOnly = false
	case "3":
		m.session.ModeManager.SetSpecterAvailable(true)
		_ = m.session.ModeManager.Transition(modes.Guarded)
		m.session.Settings.AnonymousOnly = false
	case "4":
		m.session.ModeManager.SetSpecterAvailable(true)
		m.session.ModeManager.SetShroudAvailable(true)
		_ = m.session.ModeManager.Transition(modes.Fortress)
		m.session.Settings.AnonymousOnly = true
	case "5":
		m.session.Settings.LayerBlend -= 0.1
		if m.session.Settings.LayerBlend < 0 {
			m.session.Settings.LayerBlend = 0
		}
	case "6":
		m.session.Settings.LayerBlend += 0.1
		if m.session.Settings.LayerBlend > 1 {
			m.session.Settings.LayerBlend = 1
		}
	case "7":
		m.session.Settings.Overlays["heatmap"] = !m.session.Settings.Overlays["heatmap"]
	case "8":
		m.session.Settings.Overlays["marks"] = !m.session.Settings.Overlays["marks"]
	case "9":
		m.session.Settings.Overlays["gifts"] = !m.session.Settings.Overlays["gifts"]
	case "0":
		m.session.Settings.Overlays["echo"] = !m.session.Settings.Overlays["echo"]
	}
	return m.emitActionCmd("settings_changed", map[string]any{
		"mode":             m.session.ModeManager.Current().String(),
		"layer_blend":      m.session.Settings.LayerBlend,
		"anonymous_only":   m.session.Settings.AnonymousOnly,
		"settings_visible": m.session.Settings.ShowSettings,
		"overlays":         m.session.Settings.Overlays,
		"config": map[string]any{
			"enable_relay":   m.session.Config.EnableRelay,
			"enable_tor":     m.session.Config.EnableTor,
			"enable_i2p":     m.session.Config.EnableI2P,
			"health_enabled": m.session.Config.EnableHealthEndpoint,
		},
		"theme":           m.session.Settings.Theme,
		"contrast_mode":   m.session.Settings.ContrastMode,
		"wordlist_sample": m.session.Settings.WordlistSample,
	})
}

func (m Model) emitActionCmd(action string, payload map[string]any) tea.Cmd {
	if m.stream == nil {
		return nil
	}
	return m.stream.EmitCmd("UserAction", map[string]any{"action": action, "payload": payload})
}
