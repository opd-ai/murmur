package bridge

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

// EventMsg wraps external events as tea messages.
type EventMsg struct {
	Type    string
	Payload any
}

// EventStream bridges generic events into Bubble Tea messages.
type EventStream struct {
	ch          chan EventMsg
	unsubscribe func()
}

// NewEventStream creates an event stream around an existing channel.
func NewEventStream(ch chan EventMsg, unsubscribe func()) *EventStream {
	if ch == nil {
		return nil
	}
	return &EventStream{ch: ch, unsubscribe: unsubscribe}
}

// Close unsubscribes and closes internal resources.
func (s *EventStream) Close() {
	if s == nil {
		return
	}
	if s.unsubscribe != nil {
		s.unsubscribe()
	}
}

// WaitCmd returns a command that waits for one event.
func (s *EventStream) WaitCmd(ctx context.Context) tea.Cmd {
	if s == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-s.ch:
			return ev
		}
	}
}
