// Package pulsemap provides the force-directed graph visualization (Pulse Map).
// This file implements the ebiten.Game interface for the main rendering loop.
//

//go:build !test
// +build !test

package pulsemap

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
	"github.com/opd-ai/murmur/pkg/pulsemap/interaction"
	"github.com/opd-ai/murmur/pkg/pulsemap/layout"
	"github.com/opd-ai/murmur/pkg/pulsemap/rendering"
	"github.com/opd-ai/murmur/pkg/store"
	"github.com/opd-ai/murmur/pkg/ui"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// Game implements ebiten.Game for the Pulse Map visualization.
// Per TECHNICAL_IMPLEMENTATION.md §2, this is the Ebitengine game loop
// that runs at 60fps and draws the force-directed social graph.
type Game struct {
	// engine is the force-directed layout engine.
	engine *layout.Engine

	// renderer draws nodes/edges to the screen.
	renderer *rendering.Renderer

	// camera handles viewport transformations.
	camera *interaction.Camera

	// input tracks user interaction state.
	input *interaction.InputState

	// composePanel is the Wave composition UI panel.
	composePanel *ui.ComposePanel

	// nodeDetailPanel is the node information slide-in panel.
	nodeDetailPanel *ui.NodeDetailPanel

	// searchBar is the node search interface.
	searchBar *ui.SearchBar

	// bookmarkManager handles node bookmarks.
	bookmarkManager *BookmarkManager

	// viewportControls provides zoom preset buttons.
	viewportControls *ui.ViewportControls

	// radialMenu is the right-click context menu for node actions.
	// Per AUDIT.md HIGH finding: this was previously instantiated but not wired.
	radialMenu *ui.RadialMenu

	// keypair is the Surface Layer identity for signing Waves.
	keypair *keys.KeyPair

	// pubsub is the GossipSub instance for publishing Waves.
	pubsub *gossip.PubSub

	// store provides access to persisted data for cross-layer artifact queries.
	store *store.DB

	// ctx is the application context for async operations.
	ctx context.Context

	// screenWidth and screenHeight track window dimensions.
	screenWidth  int
	screenHeight int

	// dragStart tracks where dragging began.
	dragStartX, dragStartY int
	isDragging             bool

	// lastSelectedNode tracks the previously selected node to avoid redundant updates.
	lastSelectedNode string

	// frame counter for diagnostics.
	frameCount uint64

	// shutdown signals that the game loop should terminate.
	shutdown chan struct{}
}

// NewGame creates a new Pulse Map game instance.
// Per AUDIT.md remediation, this wires the Ebitengine game loop.
func NewGame(ctx context.Context, keypair *keys.KeyPair, pubsub *gossip.PubSub, db *store.DB, dataDir string) (*Game, error) {
	// Create layout engine with initial self node.
	engine := layout.NewEngine()

	// Add self node at center (ID "self" is a placeholder until we wire identity).
	selfNode := &layout.Node{
		ID:          "self",
		Connections: 0,
		Activity:    0.0,
	}
	engine.AddNode(selfNode)

	// Create renderer with store access for cross-layer artifact queries.
	renderer, err := rendering.NewRenderer(engine, db)
	if err != nil {
		return nil, fmt.Errorf("creating renderer: %w", err)
	}

	// Add self node to renderer for visual display.
	renderer.AddNode(&rendering.NodeData{
		ID:          "self",
		DisplayName: "Self",
		PublicKey:   []byte{128, 128, 128}, // Placeholder gray
		IsSpecter:   false,
		Connections: 0,
		Activity:    0.0,
		Resonance:   0.0,
		HasRing:     false,
		RingColor:   rendering.ColorFromHash([]byte{128, 128, 128}, false),
	})

	// Get the camera from the renderer (it creates one internally).
	camera := renderer.Camera()

	// Get the input state from the renderer.
	input := renderer.InputState()

	// Create compose panel with Wave submission callback.
	theme := ui.DefaultTheme()
	game := &Game{
		engine:       engine,
		renderer:     renderer,
		camera:       camera,
		input:        input,
		keypair:      keypair,
		pubsub:       pubsub,
		store:        db,
		ctx:          ctx,
		screenWidth:  800,
		screenHeight: 600,
		shutdown:     make(chan struct{}),
	}

	// Create compose panel with submission callback.
	game.composePanel = ui.NewComposePanel(theme, game.handleWaveSubmit)

	// Create node detail panel with interaction callbacks.
	game.nodeDetailPanel = ui.NewNodeDetailPanel(theme, ui.NodeDetailCallbacks{
		OnComposeWave: game.handleNodeDetailComposeWave,
		OnSendGift:    game.handleNodeDetailSendGift,
		OnPlaceMark:   game.handleNodeDetailPlaceMark,
		OnSendWhisper: game.handleNodeDetailSendWhisper,
		OnClose:       game.handleNodeDetailClose,
	})

	// Create search bar with search and select callbacks.
	game.searchBar = ui.NewSearchBar(theme, ui.SearchCallbacks{
		OnSearch: game.handleSearch,
		OnSelect: game.handleSearchSelect,
		OnClose:  game.handleSearchClose,
	})

	// Initialize bookmark manager.
	bookmarkMgr, err := NewBookmarkManager(dataDir)
	if err != nil {
		log.Printf("Warning: failed to initialize bookmark manager: %v", err)
		// Non-fatal: bookmarks will be disabled but app continues
	}
	game.bookmarkManager = bookmarkMgr

	// Create viewport controls with zoom preset callbacks.
	// Per ROADMAP.md line 682, this provides Macro/Meso/Micro preset zoom buttons.
	game.viewportControls = ui.NewViewportControls(theme, ui.ViewportCallbacks{
		OnMacro: func() { camera.SetZoomPresetMacro() },
		OnMeso:  func() { camera.SetZoomPresetMeso() },
		OnMicro: func() { camera.SetZoomPresetMicro() },
	})

	// Create radial menu with action callbacks for all 6 actions.
	// Per AUDIT.md HIGH finding: radial menu was implemented but not wired.
	game.radialMenu = ui.NewRadialMenu(theme, ui.RadialMenuCallbacks{
		OnAction: game.handleRadialMenuAction,
	})

	return game, nil
}

// Update is called every tick (1/60 second).
// Per ebiten.Game interface, this handles input and updates game state.
func (g *Game) Update() error {
	if g.shouldShutdown() {
		return ebiten.Termination
	}

	g.handleWindowResize()
	g.handleComposePanelToggle()
	g.handleSearchBarToggle()

	panelActive := g.searchBar.Visible() || g.nodeDetailPanel.Visible() || g.composePanel.Visible()
	if !panelActive {
		g.handleNavigationHotkeys()
	}
	g.handleNodeSelection()

	if g.updateActivePanels() {
		return nil
	}

	if err := g.renderer.Update(); err != nil {
		return err
	}

	g.handleZoom()
	g.handleDragging()
	g.engine.Tick()
	g.frameCount++

	return nil
}

// handleWindowResize checks for window size changes and updates screen dimensions.
func (g *Game) handleWindowResize() {
	w, h := ebiten.WindowSize()
	if w != g.screenWidth || h != g.screenHeight {
		g.screenWidth, g.screenHeight = w, h
	}
}

// handleNavigationHotkeys processes H/Home/N keys when no text panel is active.
func (g *Game) handleNavigationHotkeys() {
	g.handleFindSelf()
	g.handleNetworkView()
	g.handleBookmarkKeys()
}

// updateActivePanels updates visible panels and returns true if input was consumed.
func (g *Game) updateActivePanels() bool {
	if g.searchBar.Visible() && g.searchBar.Update() {
		return true
	}
	if g.nodeDetailPanel.Visible() && g.nodeDetailPanel.Update() {
		return true
	}
	if g.composePanel.Visible() && g.composePanel.Update() {
		return true
	}
	// Radial menu is updated after the panel chain; it closes if Escape or right-click is pressed.
	if g.radialMenu.Visible() && g.radialMenu.Update() {
		return true
	}
	if g.viewportControls.Update() {
		return true
	}
	return false
}

func (g *Game) shouldShutdown() bool {
	select {
	case <-g.shutdown:
		return true
	default:
		return false
	}
}

func (g *Game) handleComposePanelToggle() {
	ctrlPressed := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	if inpututil.IsKeyJustPressed(ebiten.KeyN) && ctrlPressed {
		g.composePanel.Toggle()
	}
}

// handleSearchBarToggle opens the search bar when Ctrl+F is pressed.
// Per ROADMAP.md line 670, this provides search by display name, fingerprint, or pseudonym.
func (g *Game) handleSearchBarToggle() {
	ctrlPressed := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	if inpututil.IsKeyJustPressed(ebiten.KeyF) && ctrlPressed {
		g.searchBar.Toggle()
	}
}

// handleFindSelf centers the camera on the user's own node when the Home key or 'H' key is pressed.
// Per ROADMAP.md line 672, this provides a "Find Self" button to center view on own node.
func (g *Game) handleFindSelf() {
	// Home key or 'H' key to center on self node.
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) || inpututil.IsKeyJustPressed(ebiten.KeyH) {
		g.centerOnSelfNode()
	}
}

// handleBookmarkKeys handles keyboard shortcuts for bookmark management.
// Ctrl+B: Add/update bookmark for currently selected node
// Ctrl+Shift+B: Remove bookmark for currently selected node
// Ctrl+1-9: Navigate to bookmark by index
func (g *Game) handleBookmarkKeys() {
	if g.bookmarkManager == nil {
		return
	}

	ctrlPressed := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	shiftPressed := ebiten.IsKeyPressed(ebiten.KeyShift)

	if g.handleAddBookmark(ctrlPressed, shiftPressed) {
		return
	}
	if g.handleRemoveBookmark(ctrlPressed, shiftPressed) {
		return
	}
	g.handleNavigateToBookmark(ctrlPressed)
}

// handleAddBookmark handles Ctrl+B to add bookmark for selected node.
func (g *Game) handleAddBookmark(ctrlPressed, shiftPressed bool) bool {
	if ctrlPressed && !shiftPressed && inpututil.IsKeyJustPressed(ebiten.KeyB) {
		if g.input.SelectedNodeID != "" {
			g.addBookmarkForSelectedNode()
		}
		return true
	}
	return false
}

// handleRemoveBookmark handles Ctrl+Shift+B to remove bookmark for selected node.
func (g *Game) handleRemoveBookmark(ctrlPressed, shiftPressed bool) bool {
	if ctrlPressed && shiftPressed && inpututil.IsKeyJustPressed(ebiten.KeyB) {
		if g.input.SelectedNodeID != "" {
			g.removeBookmarkForSelectedNode()
		}
		return true
	}
	return false
}

// handleNavigateToBookmark handles Ctrl+1-9 to navigate to bookmark by index.
func (g *Game) handleNavigateToBookmark(ctrlPressed bool) {
	if !ctrlPressed {
		return
	}
	for i := ebiten.Key1; i <= ebiten.Key9; i++ {
		if inpututil.IsKeyJustPressed(i) {
			index := int(i - ebiten.Key1)
			g.navigateToBookmark(index)
			return
		}
	}
}

// addBookmarkForSelectedNode adds a bookmark for the currently selected node.
func (g *Game) addBookmarkForSelectedNode() {
	nodeID := g.input.SelectedNodeID
	if nodeID == "" {
		return
	}

	// Get node position from layout engine
	positions := g.engine.Positions().Get()
	pos, ok := positions[nodeID]
	if !ok {
		log.Printf("Warning: cannot bookmark node %s: position not found", nodeID)
		return
	}

	// Get node display name (fallback to ID if not found)
	label := nodeID
	// TODO: Get display name from node data when available
	// For now, use node ID truncated to 16 chars
	if len(label) > 16 {
		label = label[:16] + "..."
	}

	if err := g.bookmarkManager.Add(nodeID, label, pos.X, pos.Y); err != nil {
		log.Printf("Error adding bookmark: %v", err)
	} else {
		log.Printf("Bookmarked node: %s", label)
	}
}

// removeBookmarkForSelectedNode removes the bookmark for the currently selected node.
func (g *Game) removeBookmarkForSelectedNode() {
	nodeID := g.input.SelectedNodeID
	if nodeID == "" {
		return
	}

	if err := g.bookmarkManager.Remove(nodeID); err != nil {
		log.Printf("Error removing bookmark: %v", err)
	} else {
		log.Printf("Removed bookmark for node: %s", nodeID)
	}
}

// navigateToBookmark animates the camera to the bookmark at the given index.
func (g *Game) navigateToBookmark(index int) {
	bookmarks := g.bookmarkManager.List()
	if index >= len(bookmarks) {
		return // Index out of range
	}

	bookmark := bookmarks[index]
	// Animate to bookmark position with comfortable zoom level
	g.camera.AnimateToWithZoom(bookmark.X, bookmark.Y, 1.5)
	log.Printf("Navigating to bookmark: %s", bookmark.Label)
}

// centerOnSelfNode animates the camera to the self node's position.
// The self node is always at the center of the layout (0, 0) per game initialization.
func (g *Game) centerOnSelfNode() {
	// Get the position of the self node from the layout engine.
	positions := g.engine.Positions().Get()
	if selfPos, ok := positions["self"]; ok {
		// Animate camera to self node position with default zoom.
		g.camera.AnimateToWithZoom(selfPos.X, selfPos.Y, 1.0)
	} else {
		// Fallback: center at origin (where self node should be).
		g.camera.AnimateToWithZoom(0, 0, 1.0)
	}
}

// centerOnNetwork animates the camera to show the entire network from a global perspective.
// Per ROADMAP.md line 681: network-centric view as alternative to ego-centric view.
func (g *Game) centerOnNetwork() {
	positions := g.engine.Positions().Get()
	if len(positions) == 0 {
		return
	}

	centroidX, centroidY := g.computeNetworkCentroid(positions)
	targetScale := g.computeFitZoom(positions)
	g.camera.AnimateToWithZoom(centroidX, centroidY, targetScale)
}

// computeNetworkCentroid calculates the average position of all nodes.
func (g *Game) computeNetworkCentroid(positions map[string]layout.Position) (float64, float64) {
	var sumX, sumY float64
	for _, pos := range positions {
		sumX += pos.X
		sumY += pos.Y
	}
	return sumX / float64(len(positions)), sumY / float64(len(positions))
}

// computeFitZoom calculates zoom level to fit the entire network in view with margin.
func (g *Game) computeFitZoom(positions map[string]layout.Position) float64 {
	minX, maxX, minY, maxY := g.computeNetworkBounds(positions)

	networkWidth := maxX - minX
	networkHeight := maxY - minY
	if networkWidth < 1 {
		networkWidth = 1
	}
	if networkHeight < 1 {
		networkHeight = 1
	}

	scaleX := float64(g.screenWidth) * 0.8 / networkWidth
	scaleY := float64(g.screenHeight) * 0.8 / networkHeight
	targetScale := scaleX
	if scaleY < targetScale {
		targetScale = scaleY
	}

	return constrainZoom(targetScale)
}

// computeNetworkBounds finds the bounding box of all node positions.
func (g *Game) computeNetworkBounds(positions map[string]layout.Position) (minX, maxX, minY, maxY float64) {
	first := true
	for _, pos := range positions {
		if first {
			minX, maxX, minY, maxY = g.initializeBounds(pos)
			first = false
		} else {
			minX, maxX, minY, maxY = g.updateBounds(pos, minX, maxX, minY, maxY)
		}
	}
	return minX, maxX, minY, maxY
}

// initializeBounds sets initial boundary values from first position.
func (g *Game) initializeBounds(pos layout.Position) (minX, maxX, minY, maxY float64) {
	return pos.X, pos.X, pos.Y, pos.Y
}

// updateBounds extends bounds to include new position.
func (g *Game) updateBounds(pos layout.Position, minX, maxX, minY, maxY float64) (float64, float64, float64, float64) {
	if pos.X < minX {
		minX = pos.X
	}
	if pos.X > maxX {
		maxX = pos.X
	}
	if pos.Y < minY {
		minY = pos.Y
	}
	if pos.Y > maxY {
		maxY = pos.Y
	}
	return minX, maxX, minY, maxY
}

// constrainZoom clamps zoom level to valid range.
func constrainZoom(scale float64) float64 {
	const minZoom = 0.1
	const maxZoom = 2.0
	if scale < minZoom {
		return minZoom
	}
	if scale > maxZoom {
		return maxZoom
	}
	return scale
}

// handleNetworkView centers the camera on the network centroid when 'N' key is pressed.
// Per ROADMAP.md line 681: network-centric view for global perspective.
func (g *Game) handleNetworkView() {
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		g.centerOnNetwork()
	}
}

func (g *Game) handleZoom() {
	_, dy := ebiten.Wheel()
	if dy == 0 {
		return
	}

	zoomFactor := 1.0 + dy*0.1
	mx, my := ebiten.CursorPosition()
	g.camera.Zoom(zoomFactor, float64(mx), float64(my),
		float64(g.screenWidth), float64(g.screenHeight))
}

func (g *Game) handleDragging() {
	// Guard: clear orphaned drag state if the left button is no longer held.
	// This handles the case where the user dragged outside the window and
	// released the button there — Ebitengine never fires a just-released event.
	if g.isDragging && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.isDragging = false
		g.renderer.HandleMouseUp()
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		// Perform node hit-testing via the renderer; this sets SelectedNodeID if
		// a node was clicked, or starts the renderer's drag state if not.
		g.renderer.HandleMouseDown(float64(mx), float64(my))
		// Only begin the camera-pan drag when no node was selected.
		if g.input.SelectedNodeID == "" {
			g.dragStartX, g.dragStartY = mx, my
			g.isDragging = true
		}
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		g.isDragging = false
		g.renderer.HandleMouseUp()
	}

	if g.isDragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.updatePanPosition()
	}

	// Right-click opens the radial menu on the hovered node.
	// Guard: do not open while a modal panel is visible.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		nodeID := g.input.SelectedNodeID
		if nodeID != "" && !g.anyModalVisible() {
			mx, my := ebiten.CursorPosition()
			g.radialMenu.Show(float64(mx), float64(my), nodeID)
		}
	}
}

func (g *Game) updatePanPosition() {
	mx, my := ebiten.CursorPosition()
	dx := float64(mx - g.dragStartX)
	dy := float64(my - g.dragStartY)
	g.camera.Pan(dx, dy)
	g.dragStartX, g.dragStartY = mx, my
}

// Draw renders the Pulse Map to the screen.
// Per ebiten.Game interface, this is called after Update().
func (g *Game) Draw(screen *ebiten.Image) {
	// Delegate to renderer which handles all drawing logic.
	g.renderer.Draw(screen)

	// Draw viewport controls (always visible, bottom layer of UI).
	g.viewportControls.Draw(screen)

	// Draw radial menu above the graph layer, below modal panels.
	if g.radialMenu.Visible() {
		g.radialMenu.Draw(screen)
	}

	// Draw node detail panel overlay if visible.
	if g.nodeDetailPanel.Visible() {
		g.nodeDetailPanel.Draw(screen)
	}

	// Draw search bar overlay if visible.
	if g.searchBar.Visible() {
		g.searchBar.Draw(screen)
	}

	// Draw compose panel overlay if visible (topmost).
	if g.composePanel.Visible() {
		g.composePanel.Draw(screen)
	}
}

// Layout returns the game's screen dimensions.
// Per ebiten.Game interface, this is called when window is resized.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.screenWidth = outsideWidth
	g.screenHeight = outsideHeight
	return outsideWidth, outsideHeight
}

// Shutdown signals the game loop to terminate cleanly.
// This causes Update() to return ebiten.Termination, which exits ebiten.RunGame().
func (g *Game) Shutdown() {
	select {
	case <-g.shutdown:
		// Already closed.
	default:
		close(g.shutdown)
	}
}

// handleWaveSubmit is the callback for Wave composition panel submission.
// It creates a Wave, computes PoW, signs it, wraps it in an envelope, and publishes to GossipSub.
// Per AUDIT.md remediation, this enables user Wave creation.
func (g *Game) handleWaveSubmit(content string, waveType uint8, targetNodeID string) {
	if g.keypair == nil || g.pubsub == nil {
		log.Printf("Cannot submit Wave: keypair or pubsub not initialized")
		return
	}

	// Create Wave asynchronously to avoid blocking UI (PoW takes 2-5 seconds).
	go func() {
		log.Printf("Creating Wave with %d bytes content...", len(content))

		// Create Wave with PoW.
		opts := waves.DefaultCreateOptions()
		wave, err := waves.Create(waves.WaveType(waveType), []byte(content), g.keypair, opts)
		if err != nil {
			log.Printf("Failed to create Wave: %v", err)
			return
		}

		log.Printf("Wave created with ID %x, computing envelope...", wave.WaveId)

		// Wrap in MurmurEnvelope per TECHNICAL_IMPLEMENTATION.md wire format.
		envelope := &pb.MurmurEnvelope{
			Version:       1,
			Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
			Payload:       mustMarshal(wave),
			SenderPubkey:  g.keypair.PublicKey,
			Signature:     wave.Signature,
			TimestampUnix: wave.CreatedAt,
			MessageId:     wave.WaveId,
		}

		envelopeBytes := mustMarshal(envelope)

		// Publish to /murmur/waves/1 topic.
		ctx, cancel := context.WithTimeout(g.ctx, 5*time.Second)
		defer cancel()

		if err := g.pubsub.Publish(ctx, "/murmur/waves/1", envelopeBytes); err != nil {
			log.Printf("Failed to publish Wave: %v", err)
			return
		}

		log.Printf("Published Wave %x to network", wave.WaveId)
	}()
}

// mustMarshal marshals a proto message or panics.
// Used for internal messages that should always serialize successfully.
func mustMarshal(m proto.Message) []byte {
	b, err := proto.Marshal(m)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal proto message: %v", err))
	}
	return b
}

// handleNodeSelection checks if a node was selected and shows the detail panel.
// Per ROADMAP.md line 664-669, clicking a node opens the Node Detail Panel.
func (g *Game) handleNodeSelection() {
	// Check if a node is selected in the input state.
	selectedID := g.input.SelectedNodeID
	if selectedID == "" || selectedID == g.lastSelectedNode {
		return
	}

	// Node selection changed - fetch node info and show panel.
	g.lastSelectedNode = selectedID
	nodeInfo := g.buildNodeInfo(selectedID)
	if nodeInfo != nil {
		g.nodeDetailPanel.Show(nodeInfo)
	}
}

// buildNodeInfo constructs NodeInfo from store and renderer data.
// This queries the database and renderer state to populate the detail panel.
func (g *Game) buildNodeInfo(nodeID string) *ui.NodeInfo {
	// Query node data from renderer.
	nodeData := g.renderer.GetNodeData(nodeID)
	if nodeData == nil {
		return nil
	}

	// Query recent Waves from this node.
	recentWaves := g.getRecentWaves(nodeID, 10)

	// Query connections.
	connections := g.getConnections(nodeID)

	// Build NodeInfo struct.
	return &ui.NodeInfo{
		PublicKey:       fmt.Sprintf("%x", nodeData.PublicKey),
		DisplayName:     nodeData.DisplayName,
		Fingerprint:     fmt.Sprintf("%x", nodeData.PublicKey)[:8],
		IsSpecter:       nodeData.IsSpecter,
		IsSurface:       !nodeData.IsSpecter,
		IsSelf:          nodeID == "self",
		Resonance:       int(nodeData.Resonance),
		ResonanceRank:   g.getResonanceRank(int(nodeData.Resonance)),
		ConnectionCount: nodeData.Connections,
		Connections:     connections,
		RecentWaves:     recentWaves,
	}
}

// getRecentWaves queries recent Waves from the given node.
func (g *Game) getRecentWaves(nodeID string, limit int) []ui.WaveInfo {
	// TODO: Query from store when Wave indexing by author is implemented.
	// For now, return empty list.
	return []ui.WaveInfo{}
}

// getConnections queries connections for the given node.
func (g *Game) getConnections(nodeID string) []string {
	// TODO: Query from renderer or store when connection list is implemented.
	// For now, return empty list.
	return []string{}
}

// getResonanceRank converts a Resonance score to a milestone name.
func (g *Game) getResonanceRank(resonance int) string {
	switch {
	case resonance >= 500:
		return "Abyss"
	case resonance >= 200:
		return "Council-Eligible"
	case resonance >= 100:
		return "Phantom"
	case resonance >= 75:
		return "Shade-Wraith"
	case resonance >= 50:
		return "Wraith"
	case resonance >= 25:
		return "Shade"
	default:
		return "Novice"
	}
}

// handleNodeDetailComposeWave is called when user clicks "Compose Wave" in the detail panel.
func (g *Game) handleNodeDetailComposeWave(nodeID string) {
	log.Printf("Compose Wave to node %s", nodeID)
	// Open compose panel with target node pre-filled.
	g.composePanel.Show()
}

// handleNodeDetailSendGift is called when user clicks "Send Gift" in the detail panel.
func (g *Game) handleNodeDetailSendGift(nodeID string) {
	log.Printf("Send Gift to node %s", nodeID)
	// TODO: Open gift selection UI when Phantom Gift UI is implemented.
}

// handleNodeDetailPlaceMark is called when user clicks "Place Mark" in the detail panel.
func (g *Game) handleNodeDetailPlaceMark(nodeID string) {
	log.Printf("Place Mark on node %s", nodeID)
	// TODO: Open mark type selection UI when Specter Mark UI is implemented.
}

// handleNodeDetailSendWhisper is called when user clicks "Send Whisper" in the detail panel.
func (g *Game) handleNodeDetailSendWhisper(nodeID string) {
	log.Printf("Send Whisper to node %s", nodeID)
	// TODO: Open whisper compose UI when Whisper Chain UI is implemented.
}

// handleNodeDetailClose is called when user closes the detail panel.
func (g *Game) handleNodeDetailClose() {
	log.Printf("Node detail panel closed")
	g.input.ClearSelection()
}

// anyModalVisible returns true if any modal panel that should block the radial menu is open.
func (g *Game) anyModalVisible() bool {
	return g.composePanel.Visible() || g.searchBar.Visible() || g.nodeDetailPanel.Visible()
}

// handleRadialMenuAction dispatches radial menu action callbacks.
// Per AUDIT.md HIGH finding: all 6 RadialMenuAction cases are handled here.
func (g *Game) handleRadialMenuAction(action ui.RadialMenuAction, nodeID string) {
	switch action {
	case ui.ActionComposeWave:
		g.handleNodeDetailComposeWave(nodeID)
	case ui.ActionSendGift:
		g.handleNodeDetailSendGift(nodeID)
	case ui.ActionPlaceMark:
		g.handleNodeDetailPlaceMark(nodeID)
	case ui.ActionSendWhisper:
		g.handleNodeDetailSendWhisper(nodeID)
	case ui.ActionJoinGame:
		// TODO: Open active mini-game for this node (see ANONYMOUS_GAME_MECHANICS.md).
		log.Printf("Join game with node %s (not yet implemented)", nodeID)
	case ui.ActionViewDetail:
		if nodeInfo := g.buildNodeInfo(nodeID); nodeInfo != nil {
			g.nodeDetailPanel.Show(nodeInfo)
		}
	}
}

// handleSearch is called when user types in the search bar.
// It searches all nodes by display name, pseudonym, or node ID.
func (g *Game) handleSearch(query string) []ui.SearchResult {
	if query == "" {
		return nil
	}

	// Build list of all nodes from renderer.
	nodes := g.renderer.GetAllNodes()
	allResults := make([]ui.SearchResult, 0, len(nodes))
	for _, node := range nodes {
		result := ui.SearchResult{
			NodeID:      node.ID,
			DisplayName: node.DisplayName,
			Pseudonym:   "", // TODO: Add pseudonym field to NodeData if needed
			IsSpecter:   node.IsSpecter,
			Relevance:   1.0, // Default relevance
			Resonance:   node.Resonance,
		}
		allResults = append(allResults, result)
	}

	// Filter results by query.
	return ui.FilterResults(query, allResults)
}

// handleSearchSelect is called when user selects a search result.
// It centers the camera on the selected node.
func (g *Game) handleSearchSelect(nodeID string) {
	log.Printf("Search selected node %s", nodeID)
	// Center camera on selected node.
	g.renderer.FocusNode(nodeID)
	// Select the node so detail panel can show.
	g.input.SelectNode(nodeID)
}

// handleSearchClose is called when user closes the search bar.
func (g *Game) handleSearchClose() {
	log.Printf("Search bar closed")
}
