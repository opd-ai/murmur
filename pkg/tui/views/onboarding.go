package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	obbootstrap "github.com/opd-ai/murmur/pkg/onboarding/bootstrap"
	"github.com/opd-ai/murmur/pkg/onboarding/flow"
	"github.com/opd-ai/murmur/pkg/onboarding/tutorials"
)

type bootstrapProgressMsg struct {
	Progress obbootstrap.Progress
}

type bootstrapCompleteMsg struct {
	Peers int
}

type bootstrapErrorMsg struct {
	Err error
}

type mockConnector struct{ peers int }

func (m *mockConnector) Connect(ctx context.Context, addr string) (string, error) {
	_ = ctx
	m.peers++
	return addr, nil
}
func (m *mockConnector) PeerCount() int                           { return m.peers }
func (m *mockConnector) StartDiscovery(ctx context.Context) error { _ = ctx; m.peers += 2; return nil }

// OnboardingModel renders six-phase onboarding and first-week nudges.
type OnboardingModel struct {
	Session        *SessionState
	Controller     *flow.Controller
	Nudges         []string
	Status         string
	Bootstrap      *obbootstrap.Manager
	Hints          *tutorials.Manager
	Progress       obbootstrap.Progress
	InviteURI      string
	InviteStatus   string
	RecoveryBranch bool
}

// NewOnboardingModel creates an onboarding model.
func NewOnboardingModel(session *SessionState) OnboardingModel {
	controller := flow.NewController(flow.Callbacks{})
	controller.Start()
	connector := &mockConnector{}
	bm := obbootstrap.NewManager(obbootstrap.DefaultConfig(), connector, obbootstrap.Callbacks{})

	hints := tutorials.NewManager(tutorials.ManagerCallbacks{})
	_ = hints.TriggerHint(tutorials.HintPulseMapPan)

	return OnboardingModel{
		Session:    session,
		Controller: controller,
		Bootstrap:  bm,
		Hints:      hints,
		Nudges: []string{
			"Day 1: Explore the Pulse Map and select 3 nodes.",
			"Day 2: Publish a Wave and reply to one thread.",
			"Day 3: Create your first Specter in Hybrid mode.",
			"Day 4: Try one anonymous mini-game.",
			"Day 5: Place a Specter Mark and check Resonance.",
			"Day 6: Tune privacy mode based on usage.",
			"Day 7: Invite one peer and review your graph.",
		},
		Status: "enter: advance, space: skip, i: invitation, r: recovery, x/a: dismiss/ack hint",
	}
}

func (m OnboardingModel) startBootstrapCmd() tea.Cmd {
	if m.Bootstrap == nil {
		return nil
	}
	return func() tea.Msg {
		if err := m.Bootstrap.Start(context.Background()); err != nil {
			return bootstrapErrorMsg{Err: err}
		}
		return bootstrapCompleteMsg{Peers: m.Bootstrap.Progress().ConnectedPeers}
	}
}

func (m OnboardingModel) pollBootstrapCmd() tea.Cmd {
	if m.Bootstrap == nil {
		return nil
	}
	return tea.Tick(200*time.Millisecond, func(_ time.Time) tea.Msg {
		return bootstrapProgressMsg{Progress: m.Bootstrap.Progress()}
	})
}

// Update advances onboarding flow.
func (m OnboardingModel) Update(msg tea.Msg) (OnboardingModel, tea.Cmd) {
	switch t := msg.(type) {
	case bootstrapProgressMsg:
		m.Progress = t.Progress
		return m, m.pollBootstrapCmd()
	case bootstrapCompleteMsg:
		m.Status = fmt.Sprintf("bootstrap complete with %d peers", t.Peers)
		return m, nil
	case bootstrapErrorMsg:
		m.Status = "bootstrap failed: " + t.Err.Error()
		return m, nil
	}

	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "enter":
		if !m.Controller.IsComplete() {
			phase := m.Controller.CurrentPhase()
			m.Session.OnboardingResume.CompletedPhases[phase.String()] = true
			if phase == flow.PhaseIdentityCreation && m.Session.KeyPair == nil {
				kp, err := keys.GenerateKeyPair()
				if err == nil {
					m.Session.KeyPair = kp
				}
			}
			if phase == flow.PhaseNetworkBootstrap {
				m.Status = "bootstrap running..."
				m.Controller.CompleteCurrentPhase()
				return m, tea.Batch(m.startBootstrapCmd(), m.pollBootstrapCmd())
			}
			m.Controller.CompleteCurrentPhase()
			m.Status = "advanced to " + m.Controller.CurrentPhase().String()
		}
	case " ":
		m.Controller.Skip()
		m.Session.OnboardingResume.Skipped = true
		m.Status = "onboarding skipped (resume state saved)"
	case "i":
		m.InviteURI = "murmur://invite/demo"
		if _, err := obbootstrap.AcceptInvitation(m.InviteURI); err != nil {
			m.InviteStatus = "invitation warm-start: simulated"
		} else {
			m.InviteStatus = "invitation accepted"
		}
		m.Status = m.InviteStatus
	case "r":
		m.RecoveryBranch = !m.RecoveryBranch
		if m.RecoveryBranch {
			m.Status = "recovery onboarding branch enabled"
		} else {
			m.Status = "recovery onboarding branch disabled"
		}
	case "b":
		m.Session.OnboardingResume.ReturningUser = true
		m.Status = "returning-user continue screen enabled"
	case "x":
		m.Hints.DismissHint()
		m.Status = "hint dismissed"
	case "a":
		if h := m.Hints.ActiveHint(); h != nil {
			m.Hints.AcknowledgeHint(h.ID)
			m.Status = "hint acknowledged"
		}
	}
	return m, nil
}

// View renders onboarding phase state.
func (m OnboardingModel) View(width int) string {
	infos := flow.GetPhaseInfo()
	phase := m.Controller.CurrentPhase()
	rows := make([]string, 0, len(infos))
	for _, info := range infos {
		marker := "[ ]"
		if m.Controller.Progress(info.Phase).Completed || m.Session.OnboardingResume.CompletedPhases[info.Phase.String()] {
			marker = "[x]"
		}
		if info.Phase == phase && !m.Controller.IsComplete() {
			marker = "[>]"
		}
		rows = append(rows, fmt.Sprintf("%s %s", marker, info.Title))
	}

	hintLine := "<none>"
	if h := m.Hints.ActiveHint(); h != nil {
		hintLine = fmt.Sprintf("%s — %s", h.Title, h.Content)
	}

	completion := ""
	if m.Controller.IsComplete() {
		invite := "<none>"
		if m.Session.KeyPair != nil {
			full := fmt.Sprintf("%x", m.Session.KeyPair.PublicKey)
			if len(full) >= 12 {
				invite = "MURMUR-" + full[:6] + "-" + full[6:12]
			}
		}
		completion = fmt.Sprintf("\nCompletion summary: invite=%s returning=%t", invite, m.Session.OnboardingResume.ReturningUser)
	}

	bootstrapLine := fmt.Sprintf("Bootstrap: %s peers=%d/%d elapsed=%s invitation=%s", m.Progress.Status.String(), m.Progress.ConnectedPeers, m.Progress.TargetPeers, m.Progress.ElapsedTime.Round(time.Second), m.InviteStatus)
	return fmt.Sprintf("Current phase: %s\nProgress: %.0f%%\n%s\nRecovery branch: %t\nHint: %s\n\n%s\n\nFirst-week nudges:\n%s\n%s\n\nStatus: %s", phase.String(), m.Controller.OverallProgress(), bootstrapLine, m.RecoveryBranch, hintLine, strings.Join(rows, "\n"), strings.Join(m.Nudges, "\n"), completion, m.Status)
}
