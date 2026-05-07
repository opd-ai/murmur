package pulsemap

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/ui"
)

func TestNodeActionHandlers_ShowUnavailableToast(t *testing.T) {
	g := &Game{}

	g.handleNodeDetailSendGift("node-1")
	if g.toast == nil || g.toast.message == "" {
		t.Fatal("expected toast after Send Gift action")
	}

	g.handleNodeDetailPlaceMark("node-1")
	if g.toast == nil || g.toast.message == "" {
		t.Fatal("expected toast after Place Mark action")
	}

	g.handleNodeDetailSendWhisper("node-1")
	if g.toast == nil || g.toast.message == "" {
		t.Fatal("expected toast after Send Whisper action")
	}
}

func TestJoinGameAction_ShowsUnavailableToast(t *testing.T) {
	g := &Game{}
	g.handleRadialMenuAction(ui.ActionJoinGame, "node-1")
	if g.toast == nil {
		t.Fatal("expected toast after Join Game action")
	}
	if g.toast.message == "" {
		t.Fatal("expected non-empty Join Game toast message")
	}
}
