package styles

import "github.com/charmbracelet/lipgloss"

// Theme defines centralized styling for TUI rendering.
type Theme struct {
Base     lipgloss.Style
Header   lipgloss.Style
Tab      lipgloss.Style
TabActive lipgloss.Style
Muted    lipgloss.Style
Good     lipgloss.Style
Warn     lipgloss.Style
Bad      lipgloss.Style
Panel    lipgloss.Style
Help     lipgloss.Style
}

// NewTheme returns the default TUI theme.
func NewTheme() Theme {
return Theme{
Base:      lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
Header:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("45")),
Tab:       lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("250")),
TabActive: lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("45")),
Muted:     lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
Good:      lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
Warn:      lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
Bad:       lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
Panel:     lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(1),
Help:      lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Background(lipgloss.Color("236")).Padding(0, 1),
}
}
