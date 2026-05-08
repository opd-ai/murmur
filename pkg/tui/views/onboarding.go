package views

import (
"fmt"
"strings"

tea "github.com/charmbracelet/bubbletea"
"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

// OnboardingModel renders six-phase onboarding and first-week nudges.
type OnboardingModel struct {
Controller *flow.Controller
Nudges     []string
Status     string
}

// NewOnboardingModel creates an onboarding model.
func NewOnboardingModel() OnboardingModel {
controller := flow.NewController(flow.Callbacks{})
controller.Start()
return OnboardingModel{
Controller: controller,
Nudges: []string{
"Day 1: Explore the Pulse Map and select 3 nodes.",
"Day 2: Publish a Wave and reply to one thread.",
"Day 3: Create your first Specter in Hybrid mode.",
"Day 4: Try one anonymous mini-game.",
"Day 5: Place a Specter Mark and check Resonance.",
"Day 6: Tune privacy mode based on usage.",
"Day 7: Invite one peer and review your graph.",
},
Status: "enter: complete phase, space: skip onboarding",
}
}

// Update advances onboarding flow.
func (m OnboardingModel) Update(msg tea.Msg) (OnboardingModel, tea.Cmd) {
k, ok := msg.(tea.KeyMsg)
if !ok {
return m, nil
}
switch k.String() {
case "enter":
if !m.Controller.IsComplete() {
m.Controller.CompleteCurrentPhase()
m.Status = "advanced to " + m.Controller.CurrentPhase().String()
}
case " ":
m.Controller.Skip()
m.Status = "onboarding skipped"
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
if m.Controller.Progress(info.Phase).Completed {
marker = "[x]"
}
if info.Phase == phase && !m.Controller.IsComplete() {
marker = "[>]"
}
rows = append(rows, fmt.Sprintf("%s %s", marker, info.Title))
}
return fmt.Sprintf("Current phase: %s\nProgress: %.0f%%\n\n%s\n\nFirst-week nudges:\n%s\n\nStatus: %s", phase.String(), m.Controller.OverallProgress(), strings.Join(rows, "\n"), strings.Join(m.Nudges, "\n"), m.Status)
}
