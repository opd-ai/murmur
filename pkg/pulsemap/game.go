// Package pulsemap provides the force-directed graph visualization (Pulse Map).
// This file implements the ebiten.Game interface for the main rendering loop.
//
//go:build !noebiten
// +build !noebiten

package pulsemap

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/opd-ai/murmur/pkg/pulsemap/interaction"
	"github.com/opd-ai/murmur/pkg/pulsemap/layout"
	"github.com/opd-ai/murmur/pkg/pulsemap/rendering"
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

	// screenWidth and screenHeight track window dimensions.
	screenWidth  int
	screenHeight int

	// dragStart tracks where dragging began.
	dragStartX, dragStartY int
	isDragging             bool

	// frame counter for diagnostics.
	frameCount uint64
}

// NewGame creates a new Pulse Map game instance.
// Per AUDIT.md remediation, this wires the Ebitengine game loop.
func NewGame() (*Game, error) {
	// Create layout engine with initial self node.
	engine := layout.NewEngine()

	// Create renderer.
	renderer, err := rendering.NewRenderer(engine)
	if err != nil {
		return nil, fmt.Errorf("creating renderer: %w", err)
	}

	// Get the camera from the renderer (it creates one internally).
	camera := renderer.Camera()

	// Get the input state from the renderer.
	input := renderer.InputState()

	return &Game{
		engine:       engine,
		renderer:     renderer,
		camera:       camera,
		input:        input,
		screenWidth:  800,
		screenHeight: 600,
	}, nil
}

// Update is called every tick (1/60 second).
// Per ebiten.Game interface, this handles input and updates game state.
func (g *Game) Update() error {
	// Update renderer (which updates camera animation and time).
	if err := g.renderer.Update(); err != nil {
		return err
	}

	// Handle mouse wheel zoom.
	_, dy := ebiten.Wheel()
	if dy != 0 {
		zoomFactor := 1.0 + dy*0.1
		mx, my := ebiten.CursorPosition()
		g.camera.Zoom(zoomFactor, float64(mx), float64(my),
			float64(g.screenWidth), float64(g.screenHeight))
	}

	// Handle mouse drag panning.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.dragStartX, g.dragStartY = ebiten.CursorPosition()
		g.isDragging = true
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		g.isDragging = false
	}
	if g.isDragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		dx := float64(mx - g.dragStartX)
		dy := float64(my - g.dragStartY)
		g.camera.Pan(dx, dy)
		g.dragStartX, g.dragStartY = mx, my
	}

	// Step the force-directed layout engine.
	g.engine.Tick()

	g.frameCount++
	return nil
}

// Draw renders the Pulse Map to the screen.
// Per ebiten.Game interface, this is called after Update().
func (g *Game) Draw(screen *ebiten.Image) {
	// Delegate to renderer which handles all drawing logic.
	g.renderer.Draw(screen)
}

// Layout returns the game's screen dimensions.
// Per ebiten.Game interface, this is called when window is resized.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.screenWidth = outsideWidth
	g.screenHeight = outsideHeight
	return outsideWidth, outsideHeight
}
