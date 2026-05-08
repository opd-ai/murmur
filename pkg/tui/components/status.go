package components

import "github.com/opd-ai/murmur/pkg/tui/styles"

// StatusBar renders a single-line status bar.
func StatusBar(theme styles.Theme, left, right string) string {
if right == "" {
return theme.Help.Render(left)
}
return theme.Help.Render(left + " | " + right)
}
