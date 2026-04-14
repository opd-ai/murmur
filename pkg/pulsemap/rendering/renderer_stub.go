// Package rendering provides stub types for the Pulse Map renderer.
// This file is used when building with the noebiten tag.
//
//go:build noebiten
// +build noebiten

package rendering

import (
	"image/color"
	"sync"

	"github.com/opd-ai/murmur/pkg/pulsemap/interaction"
	"github.com/opd-ai/murmur/pkg/pulsemap/layout"
)

// Renderer coordinates rendering of the Pulse Map (stub implementation).
type Renderer struct {
	mu sync.RWMutex

	engine   *layout.Engine
	camera   *interaction.Camera
	input    *interaction.InputState
	nodeData map[string]*NodeData
	edges    []EdgeData

	backgroundColor color.RGBA
	screenWidth     int
	screenHeight    int
	time            float32
}

// NodeData holds visual properties for a renderable node.
type NodeData struct {
	ID          string
	PublicKey   []byte
	IsSpecter   bool
	Connections int
	Activity    float64
	Resonance   float64
	HasRing     bool
	RingColor   color.RGBA
}

// EdgeData holds visual properties for a renderable edge.
type EdgeData struct {
	SourceID string
	TargetID string
	Age      float64
	Active   bool
}

// NewRenderer creates a new Pulse Map renderer (stub).
func NewRenderer(engine *layout.Engine) (*Renderer, error) {
	return &Renderer{
		engine:          engine,
		camera:          interaction.NewCamera(),
		input:           interaction.NewInputState(),
		nodeData:        make(map[string]*NodeData),
		edges:           make([]EdgeData, 0),
		backgroundColor: color.RGBA{10, 12, 18, 255},
		screenWidth:     800,
		screenHeight:    600,
	}, nil
}

// SetCamera sets the camera for viewport transformations.
func (r *Renderer) SetCamera(camera *interaction.Camera) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.camera = camera
}

// Camera returns the current camera.
func (r *Renderer) Camera() *interaction.Camera {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.camera
}

// InputState returns the input state tracker.
func (r *Renderer) InputState() *interaction.InputState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.input
}

// AddNode adds a node to be rendered.
func (r *Renderer) AddNode(data *NodeData) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodeData[data.ID] = data
}

// RemoveNode removes a node from rendering.
func (r *Renderer) RemoveNode(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.nodeData, id)
}

// AddEdge adds an edge to be rendered.
func (r *Renderer) AddEdge(edge EdgeData) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.edges = append(r.edges, edge)
}

// ClearEdges removes all edges.
func (r *Renderer) ClearEdges() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.edges = r.edges[:0]
}

// SetEdges replaces all edges.
func (r *Renderer) SetEdges(edges []EdgeData) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.edges = edges
}

// Update performs per-frame updates (stub).
func (r *Renderer) Update() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.time += 1.0 / float32(TargetFPS)
	if r.camera != nil {
		r.camera.Update()
	}
	return nil
}

// Layout returns the preferred layout size.
func (r *Renderer) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// HandleMouseDown processes mouse down events.
func (r *Renderer) HandleMouseDown(x, y float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodeID := r.hitTestNodes(x, y)
	if nodeID != "" {
		r.input.SelectNode(nodeID)
	} else {
		r.input.ClearSelection()
		r.input.StartDrag(x, y)
	}
}

// HandleMouseUp processes mouse up events.
func (r *Renderer) HandleMouseUp() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.input.EndDrag()
}

// HandleMouseMove processes mouse move events.
func (r *Renderer) HandleMouseMove(x, y float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.input.Dragging && r.camera != nil {
		dx, dy := r.input.UpdateDrag(x, y)
		r.camera.Pan(dx, dy)
	}
}

// HandleMouseWheel processes mouse wheel events.
func (r *Renderer) HandleMouseWheel(x, y, deltaY float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.camera == nil {
		return
	}
	factor := 1.0
	if deltaY > 0 {
		factor = 1.1
	} else if deltaY < 0 {
		factor = 0.9
	}
	r.camera.Zoom(factor, x, y, float64(r.screenWidth), float64(r.screenHeight))
}

// hitTestNodes finds the node at the given screen position (stub).
func (r *Renderer) hitTestNodes(screenX, screenY float64) string {
	if r.engine == nil || r.camera == nil {
		return ""
	}
	worldX, worldY := r.camera.ScreenToWorld(screenX, screenY,
		float64(r.screenWidth), float64(r.screenHeight))

	positions := r.engine.Positions().Get()
	for id, pos := range positions {
		data := r.nodeData[id]
		if data == nil {
			continue
		}
		style := NodeStyle{
			IsSpecter:   data.IsSpecter,
			Connections: data.Connections,
			Activity:    data.Activity,
			Resonance:   data.Resonance,
		}
		radius := float64(computeNodeRadius(style)) * 1.5 / r.camera.Scale
		if interaction.HitTest(pos.X, pos.Y, worldX, worldY, radius) {
			return id
		}
	}
	return ""
}

// SelectedNode returns the currently selected node ID.
func (r *Renderer) SelectedNode() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.input.SelectedNodeID
}

// FocusNode animates the camera to center on a node.
func (r *Renderer) FocusNode(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.engine == nil || r.camera == nil {
		return
	}
	positions := r.engine.Positions().Get()
	pos, ok := positions[nodeID]
	if !ok {
		return
	}
	r.camera.AnimateToWithZoom(pos.X, pos.Y, 1.5)
}

// SetBackgroundColor sets the Pulse Map background color.
func (r *Renderer) SetBackgroundColor(c color.RGBA) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backgroundColor = c
}
