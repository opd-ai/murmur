package pulsemap

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/pulsemap/interaction"
	"github.com/opd-ai/murmur/pkg/pulsemap/layout"
	"github.com/opd-ai/murmur/pkg/pulsemap/rendering"
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

func TestTouchAndMouseTapParity_SelectionAndDragState(t *testing.T) {
	engine := layout.NewEngine()
	renderer, err := rendering.NewRenderer(engine, nil)
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	const nodeID = "node-parity"
	renderer.AddNode(&rendering.NodeData{ID: nodeID})
	engine.Positions().Swap(map[string]layout.Position{nodeID: {X: 0, Y: 0}})

	g := &Game{renderer: renderer}

	// Empty-area mouse click should not keep dragging active.
	renderer.HandleMouseDown(9999, 9999)
	renderer.HandleMouseUp()
	if renderer.InputState().Dragging {
		t.Fatal("mouse click on empty area should end with Dragging=false")
	}

	// Empty-area touch tap should mirror mouse behavior.
	g.handleTouchTap(9999, 9999)
	if renderer.InputState().Dragging {
		t.Fatal("touch tap on empty area should end with Dragging=false")
	}

	screenX, screenY := renderer.Camera().WorldToScreen(0, 0, 800, 600)

	// Mouse node tap should select the node.
	renderer.HandleMouseDown(screenX, screenY)
	renderer.HandleMouseUp()
	mouseSelected := renderer.SelectedNode()

	// Touch node tap should select the same node.
	renderer.InputState().ClearSelection()
	g.handleTouchTap(screenX, screenY)
	touchSelected := renderer.SelectedNode()

	if mouseSelected != nodeID {
		t.Fatalf("expected mouse-selected node %q, got %q", nodeID, mouseSelected)
	}
	if touchSelected != nodeID {
		t.Fatalf("expected touch-selected node %q, got %q", nodeID, touchSelected)
	}
}

func TestNodeActionsAndRadialActionsParity(t *testing.T) {
	theme := ui.DefaultTheme()
	g := &Game{
		composePanel:    ui.NewComposePanel(theme, func(string, uint8, string) {}),
		nodeDetailPanel: ui.NewNodeDetailPanel(theme, ui.NodeDetailCallbacks{}),
	}

	nodeInfo := &ui.NodeInfo{PublicKey: "pk", DisplayName: "Node", Fingerprint: "deadbeef"}

	// Compose path parity: node detail action and radial menu action should
	// produce the same panel visibility transition.
	g.composePanel.Hide()
	g.nodeDetailPanel.Show(nodeInfo)
	g.handleNodeDetailComposeWave("node-1")
	directComposeVisible := g.composePanel.Visible()
	directNodeDetailVisible := g.nodeDetailPanel.Visible()

	g.composePanel.Hide()
	g.nodeDetailPanel.Show(nodeInfo)
	g.handleRadialMenuAction(ui.ActionComposeWave, "node-1")
	radialComposeVisible := g.composePanel.Visible()
	radialNodeDetailVisible := g.nodeDetailPanel.Visible()

	if directComposeVisible != radialComposeVisible || directNodeDetailVisible != radialNodeDetailVisible {
		t.Fatalf("compose parity mismatch: direct=(compose:%v detail:%v) radial=(compose:%v detail:%v)",
			directComposeVisible, directNodeDetailVisible, radialComposeVisible, radialNodeDetailVisible)
	}

	// Unimplemented action parity: radial dispatch should reach the same handler
	// and produce the same user-visible toast message.
	tests := []struct {
		name   string
		action ui.RadialMenuAction
		direct func(nodeID string)
	}{
		{name: "gift", action: ui.ActionSendGift, direct: g.handleNodeDetailSendGift},
		{name: "mark", action: ui.ActionPlaceMark, direct: g.handleNodeDetailPlaceMark},
		{name: "whisper", action: ui.ActionSendWhisper, direct: g.handleNodeDetailSendWhisper},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g.toast = nil
			tc.direct("node-1")
			if g.toast == nil || g.toast.message == "" {
				t.Fatalf("expected direct %s action to set a toast", tc.name)
			}
			directMessage := g.toast.message

			g.toast = nil
			g.handleRadialMenuAction(tc.action, "node-1")
			if g.toast == nil || g.toast.message == "" {
				t.Fatalf("expected radial %s action to set a toast", tc.name)
			}

			if g.toast.message != directMessage {
				t.Fatalf("expected matching toast for %s action, direct=%q radial=%q", tc.name, directMessage, g.toast.message)
			}
		})
	}
}

func TestBookmarkHandlers_ShowUserFeedback(t *testing.T) {
	engine := layout.NewEngine()
	bookmarkMgr, err := NewBookmarkManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewBookmarkManager failed: %v", err)
	}

	g := &Game{
		engine:           engine,
		input:            interaction.NewInputState(),
		bookmarkManager:  bookmarkMgr,
		composePanel:     ui.NewComposePanel(ui.DefaultTheme(), func(string, uint8, string) {}),
		nodeDetailPanel:  ui.NewNodeDetailPanel(ui.DefaultTheme(), ui.NodeDetailCallbacks{}),
		searchBar:        ui.NewSearchBar(ui.DefaultTheme(), ui.SearchCallbacks{}),
		settingsPanel:    ui.NewSettingsPanel(ui.DefaultTheme(), nil),
		viewportControls: ui.NewViewportControls(ui.DefaultTheme(), ui.ViewportCallbacks{}),
	}

	// Failure path: selected node has no known position.
	g.input.SelectNode("missing-node")
	g.addBookmarkForSelectedNode()
	if g.toast == nil || g.toast.message == "" || !g.toast.isError {
		t.Fatal("expected error toast when bookmarking node without position")
	}

	// Success path: add, remove, and invalid navigate all emit user-visible feedback.
	g.toast = nil
	g.input.SelectNode("node-1")
	engine.Positions().Swap(map[string]layout.Position{"node-1": {X: 12, Y: 34}})
	g.addBookmarkForSelectedNode()
	if g.toast == nil || g.toast.message == "" || g.toast.isError {
		t.Fatal("expected success toast after bookmark add")
	}

	g.toast = nil
	g.removeBookmarkForSelectedNode()
	if g.toast == nil || g.toast.message == "" || g.toast.isError {
		t.Fatal("expected success toast after bookmark remove")
	}

	g.toast = nil
	g.navigateToBookmark(3)
	if g.toast == nil || g.toast.message == "" || !g.toast.isError {
		t.Fatal("expected error toast for empty bookmark slot")
	}
}

func TestHandleSettingChange_PrivacyModeFeedback(t *testing.T) {
	g := &Game{}

	g.handleSettingChange("privacy_mode", "unknown-mode")
	if g.toast == nil || g.toast.message == "" || !g.toast.isError {
		t.Fatal("expected error toast for unknown privacy mode")
	}
}
