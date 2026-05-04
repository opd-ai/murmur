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

	// frame counter for diagnostics.
	frameCount uint64

	// shutdown signals that the game loop should terminate.
	shutdown chan struct{}
}

// NewGame creates a new Pulse Map game instance.
// Per AUDIT.md remediation, this wires the Ebitengine game loop.
func NewGame(ctx context.Context, keypair *keys.KeyPair, pubsub *gossip.PubSub, db *store.DB) (*Game, error) {
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

	return game, nil
}

// Update is called every tick (1/60 second).
// Per ebiten.Game interface, this handles input and updates game state.
func (g *Game) Update() error {
	if g.shouldShutdown() {
		return ebiten.Termination
	}

	// Handle window resize per AUDIT.md LOW finding.
	// Query Ebitengine for current window size and update if changed.
	w, h := ebiten.WindowSize()
	if w != g.screenWidth || h != g.screenHeight {
		g.screenWidth, g.screenHeight = w, h
	}

	g.handleComposePanelToggle()
	g.handleFindSelf()

	if g.composePanel.Visible() && g.composePanel.Update() {
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

// handleFindSelf centers the camera on the user's own node when the Home key or 'H' key is pressed.
// Per ROADMAP.md line 672, this provides a "Find Self" button to center view on own node.
func (g *Game) handleFindSelf() {
	// Home key or 'H' key to center on self node.
	if inpututil.IsKeyJustPressed(ebiten.KeyHome) || inpututil.IsKeyJustPressed(ebiten.KeyH) {
		g.centerOnSelfNode()
	}
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
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.dragStartX, g.dragStartY = ebiten.CursorPosition()
		g.isDragging = true
	}

	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		g.isDragging = false
	}

	if g.isDragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.updatePanPosition()
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

	// Draw compose panel overlay if visible.
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
