package input

import "github.com/charmbracelet/bubbles/key"

// KeyMap contains TUI keybindings.
type KeyMap struct {
	Quit     key.Binding
	Help     key.Binding
	NextTab  key.Binding
	PrevTab  key.Binding
	PanLeft  key.Binding
	PanDown  key.Binding
	PanUp    key.Binding
	PanRight key.Binding
	ZoomIn   key.Binding
	ZoomOut  key.Binding
}

// NewKeyMap creates default key bindings.
func NewKeyMap() KeyMap {
	return KeyMap{
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
		NextTab:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
		PrevTab:  key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev tab")),
		PanLeft:  key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h/←", "pan left")),
		PanDown:  key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "pan down")),
		PanUp:    key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "pan up")),
		PanRight: key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l/→", "pan right")),
		ZoomIn:   key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "zoom in")),
		ZoomOut:  key.NewBinding(key.WithKeys("-"), key.WithHelp("-", "zoom out")),
	}
}
