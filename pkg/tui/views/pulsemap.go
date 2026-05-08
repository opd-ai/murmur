package views

import (
"fmt"
"math"
"strings"

tea "github.com/charmbracelet/bubbletea"
)

// PulseNode is a simplified render node for terminal Pulse Map view.
type PulseNode struct {
ID       string
X        float64
Y        float64
Activity float64
}

// PulseMapModel renders terminal Pulse Map interactions.
type PulseMapModel struct {
CameraX  float64
CameraY  float64
Zoom     float64
Focus    int
Nodes    []PulseNode
dragging bool
status   string
}

// NewPulseMapModel returns a Pulse Map model with seed nodes.
func NewPulseMapModel() PulseMapModel {
return PulseMapModel{
Zoom: 1,
Nodes: []PulseNode{
{ID: "self", X: 0, Y: 0, Activity: 0.8},
{ID: "peer-a", X: 6, Y: -2, Activity: 0.5},
{ID: "peer-b", X: -4, Y: 3, Activity: 0.2},
{ID: "peer-c", X: 9, Y: 5, Activity: 0.7},
},
status: "Use hjkl/arrows to pan, +/- to zoom, enter/click to select",
}
}

// Update handles pulse map interactions.
func (m PulseMapModel) Update(msg tea.Msg) (PulseMapModel, tea.Cmd) {
switch t := msg.(type) {
case tea.KeyMsg:
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
m.status = "Selected node: " + m.Nodes[m.Focus].ID
}
case "n":
m.CameraX, m.CameraY, m.Zoom = 0, 0, 1
m.status = "Viewport fit/reset"
}
case tea.MouseMsg:
switch t.Action {
case tea.MouseActionPress:
if t.Button == tea.MouseButtonLeft {
m.dragging = true
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
m.Focus = (m.Focus + 1) % len(m.Nodes)
m.status = "Selected node: " + m.Nodes[m.Focus].ID
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

// View renders pulse map content.
func (m PulseMapModel) View(width int) string {
if len(m.Nodes) == 0 {
return "No nodes"
}
var b strings.Builder
b.WriteString(fmt.Sprintf("Camera x=%.1f y=%.1f zoom=%.2f\n", m.CameraX, m.CameraY, m.Zoom))
b.WriteString("Nodes:\n")
for i, n := range m.Nodes {
marker := " "
if i == m.Focus {
marker = "▶"
}
dx := (n.X - m.CameraX) * m.Zoom
dy := (n.Y - m.CameraY) * m.Zoom
glyph := activityGlyph(n.Activity)
b.WriteString(fmt.Sprintf("%s %s %-8s (%5.1f,%5.1f)\n", marker, glyph, n.ID, dx, dy))
}
b.WriteString("\nStatus: " + m.status)
return b.String()
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

func clampZoom(z float64) float64 {
return math.Max(0.2, math.Min(5, z))
}
