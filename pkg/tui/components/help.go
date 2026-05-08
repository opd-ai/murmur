package components

import (
"strings"

"github.com/opd-ai/murmur/pkg/tui/styles"
)

// HelpOverlay renders key help content.
func HelpOverlay(theme styles.Theme, lines []string) string {
return theme.Panel.Render(strings.Join(lines, "\n"))
}
