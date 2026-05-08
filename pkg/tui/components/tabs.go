package components

import (
	"strings"

	"github.com/opd-ai/murmur/pkg/tui/styles"
)

// Tabs renders top-level tab navigation.
func Tabs(theme styles.Theme, labels []string, active int) string {
	parts := make([]string, 0, len(labels))
	for i, label := range labels {
		if i == active {
			parts = append(parts, theme.TabActive.Render(label))
			continue
		}
		parts = append(parts, theme.Tab.Render(label))
	}
	return strings.Join(parts, " ")
}
