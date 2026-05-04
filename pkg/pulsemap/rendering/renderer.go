// Package rendering provides Ebitengine-based rendering for the Pulse Map.
// This file contains the main Renderer type that coordinates layout, camera,
// and drawing of nodes/edges.
//

//go:build !test
// +build !test

package rendering

import (
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/opd-ai/murmur/pkg/pulsemap/interaction"
	"github.com/opd-ai/murmur/pkg/pulsemap/layout"
	"github.com/opd-ai/murmur/pkg/pulsemap/rendering/effects"
	"github.com/opd-ai/murmur/pkg/store"
)

// Renderer coordinates rendering of the Pulse Map.
// It reads node positions from the layout engine's double-buffered positions
// and transforms them to screen coordinates via the camera.
type Renderer struct {
	mu sync.RWMutex

	// engine is the force-directed layout engine.
	engine *layout.Engine

	// camera handles viewport transformations.
	camera *interaction.Camera

	// input tracks user interaction state.
	input *interaction.InputState

	// shaders contains compiled Kage shaders for effects.
	shaders *effects.Shaders

	// nodeData maps node IDs to their visual properties.
	nodeData map[string]*NodeData

	// edges holds all edges to render.
	edges []EdgeData

	// amplificationTrails holds amplification relationships to visualize.
	amplificationTrails []AmplificationTrailData

	// store provides access to persisted data for cross-layer artifact queries.
	// Per AUDIT.md HIGH finding "Cross-layer visibility not implemented", this enables
	// Surface users to see anonymous artifacts (Marks, Gifts, mini-games) on their Pulse Map.
	store *store.DB

	// backgroundColor is the Pulse Map background color.
	backgroundColor color.RGBA

	// screenWidth and screenHeight are the current screen dimensions.
	screenWidth, screenHeight int

	// time tracks elapsed time for animations.
	time float32
}

// NodeData holds visual properties for a renderable node.
type NodeData struct {
	ID          string
	DisplayName string  // Display name or Specter pseudonym
	PublicKey   []byte  // For color derivation
	IsSpecter   bool    // True if this is a Specter node
	Connections int     // Connection count
	Activity    float64 // Activity metric
	Resonance   float64 // Resonance score (Specters only)
	HasRing     bool    // True if mode ring should be shown
	RingColor   color.RGBA
}

// EdgeData holds visual properties for a renderable edge.
type EdgeData struct {
	SourceID             string
	TargetID             string
	Age                  float64 // Connection age in days
	Active               bool    // True if recently propagated a Wave
	InteractionFrequency float64 // Message exchange rate (messages per day)
}

// AmplificationTrailData holds visual properties for an amplification relationship.
// Per ROADMAP.md line 621, amplification trails show visual connection between
// amplifier and original author.
type AmplificationTrailData struct {
	AmplifierID   string  // Node ID of the amplifier
	OriginalID    string  // Node ID of the original author
	AmplifiedAt   int64   // Unix timestamp when amplified
	WaveID        []byte  // ID of the amplified Wave
	HasComment    bool    // True if amplification includes a comment
	RecentSeconds float64 // How many seconds ago this amplification occurred (for fade animation)
}

// NewRenderer creates a new Pulse Map renderer.
func NewRenderer(engine *layout.Engine, db *store.DB) (*Renderer, error) {
	shaders, err := effects.LoadShaders()
	if err != nil {
		// Shaders may fail to load in some environments; continue without them.
		shaders = nil
	}

	return &Renderer{
		engine:              engine,
		camera:              interaction.NewCamera(),
		input:               interaction.NewInputState(),
		shaders:             shaders,
		nodeData:            make(map[string]*NodeData),
		edges:               make([]EdgeData, 0),
		amplificationTrails: make([]AmplificationTrailData, 0),
		store:               db,
		backgroundColor:     color.RGBA{10, 12, 18, 255}, // Dark background per PULSE_MAP.md
		screenWidth:         800,
		screenHeight:        600,
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

// AddAmplificationTrail adds an amplification relationship to visualize.
func (r *Renderer) AddAmplificationTrail(trail AmplificationTrailData) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.amplificationTrails = append(r.amplificationTrails, trail)
}

// ClearAmplificationTrails removes all amplification trails.
func (r *Renderer) ClearAmplificationTrails() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.amplificationTrails = r.amplificationTrails[:0]
}

// SetAmplificationTrails replaces all amplification trails.
func (r *Renderer) SetAmplificationTrails(trails []AmplificationTrailData) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.amplificationTrails = trails
}

// Update performs per-frame updates.
func (r *Renderer) Update() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Update animation time.
	r.time += 1.0 / float32(TargetFPS)

	// Update camera animations.
	if r.camera != nil {
		r.camera.Update()
	}

	return nil
}

// Draw renders the Pulse Map to the given screen.
// This is the main draw loop called by Ebitengine.
func (r *Renderer) Draw(screen *ebiten.Image) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get screen dimensions.
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	r.screenWidth = w
	r.screenHeight = h

	// Clear to background color.
	screen.Fill(r.backgroundColor)

	if r.engine == nil || r.camera == nil {
		return
	}

	// Get current node positions from the double-buffered layout.
	positions := r.engine.Positions().Get()
	if len(positions) == 0 {
		return
	}

	// Compute visible bounds for culling.
	minX, minY, maxX, maxY := r.camera.ViewBounds(float64(w), float64(h))

	// Calculate zoom level for detail decisions.
	zoom := ZoomLevelFromScale(r.camera.Scale)

	// Draw edges first (below nodes).
	r.drawEdges(screen, positions, zoom)

	// Draw amplification trails (above edges, below nodes).
	r.drawAmplificationTrails(screen, positions, zoom)

	// Draw nodes on top.
	r.drawNodes(screen, positions, minX, minY, maxX, maxY, zoom)
}

// drawEdges renders all edges between nodes with pulse animations.
func (r *Renderer) drawEdges(screen *ebiten.Image, positions map[string]layout.Position, zoom ZoomLevel) {
	screenW := float64(r.screenWidth)
	screenH := float64(r.screenHeight)

	for _, edge := range r.edges {
		srcPos, srcOK := positions[edge.SourceID]
		dstPos, dstOK := positions[edge.TargetID]
		if !srcOK || !dstOK {
			continue
		}

		// Transform and cull.
		srcScreenX, srcScreenY, dstScreenX, dstScreenY, visible := r.transformAndCullLine(srcPos.X, srcPos.Y, dstPos.X, dstPos.Y, screenW, screenH)
		if !visible {
			continue
		}

		// Build edge style from data.
		style := EdgeStyle{
			Color:                color.RGBA{100, 120, 140, 255}, // Default edge color
			Age:                  edge.Age,
			Active:               edge.Active,
			InteractionFrequency: edge.InteractionFrequency,
		}

		// Use time-based rendering for pulse animations on active edges.
		RenderEdgeWithTime(screen, float32(srcScreenX), float32(srcScreenY),
			float32(dstScreenX), float32(dstScreenY), style, zoom, float64(r.time))
	}
}

// drawAmplificationTrails renders amplification relationships between nodes.
// Per ROADMAP.md line 621, amplification trails are visual connections between
// amplifier and original author, distinct from regular edges.
func (r *Renderer) drawAmplificationTrails(screen *ebiten.Image, positions map[string]layout.Position, zoom ZoomLevel) {
	screenW := float64(r.screenWidth)
	screenH := float64(r.screenHeight)

	for _, trail := range r.amplificationTrails {
		ampPos, ampOK := positions[trail.AmplifierID]
		origPos, origOK := positions[trail.OriginalID]
		if !ampOK || !origOK {
			continue
		}

		// Transform and cull.
		ampScreenX, ampScreenY, origScreenX, origScreenY, visible := r.transformAndCullLine(ampPos.X, ampPos.Y, origPos.X, origPos.Y, screenW, screenH)
		if !visible {
			continue
		}

		// Render amplification trail with distinctive style.
		RenderAmplificationTrail(screen,
			float32(ampScreenX), float32(ampScreenY),
			float32(origScreenX), float32(origScreenY),
			trail, zoom, float64(r.time))
	}
}

// drawNodes renders all visible nodes.
func (r *Renderer) drawNodes(screen *ebiten.Image, positions map[string]layout.Position,
	minX, minY, maxX, maxY float64, zoom ZoomLevel,
) {
	screenW := float64(r.screenWidth)
	screenH := float64(r.screenHeight)
	margin := 50.0 // Render nodes slightly outside visible area for smooth scrolling.

	for id, pos := range positions {
		// Frustum culling in world space.
		if pos.X < minX-margin || pos.X > maxX+margin ||
			pos.Y < minY-margin || pos.Y > maxY+margin {
			continue
		}

		// Get node visual data.
		data, ok := r.nodeData[id]
		if !ok {
			// Node not in render data; use default style.
			data = &NodeData{
				ID:          id,
				Connections: 1,
			}
		}

		// Transform to screen coordinates.
		screenX, screenY := r.camera.WorldToScreen(pos.X, pos.Y, screenW, screenH)

		// Build node style.
		style := r.buildNodeStyle(data)

		// Render the node.
		RenderNode(screen, float32(screenX), float32(screenY), style, zoom)

		// Render glow effect for active/selected nodes.
		if r.shaders != nil && (style.HasHalo || style.Selected) {
			r.drawNodeGlow(screen, float32(screenX), float32(screenY), style)
		}

		// Render cross-layer artifacts (Specter Marks, Gifts, etc.) if store is available.
		// Per AUDIT.md HIGH finding "Cross-layer visibility not implemented", this enables
		// Surface users to see anonymous activity on their Pulse Map.
		if r.store != nil {
			r.drawCrossLayerArtifacts(screen, data, float32(screenX), float32(screenY))
		}

		// Render text label at Micro zoom level.
		RenderTextLabel(screen, float32(screenX), float32(screenY), data.DisplayName, data.IsSpecter, zoom)
	}
}

// buildNodeStyle creates a NodeStyle from NodeData.
func (r *Renderer) buildNodeStyle(data *NodeData) NodeStyle {
	// Derive color from public key or use default.
	var coreColor color.RGBA
	if len(data.PublicKey) >= 3 {
		coreColor = ColorFromHash(data.PublicKey, data.IsSpecter)
	} else {
		if data.IsSpecter {
			coreColor = color.RGBA{100, 150, 200, 255} // Cool blue for Specters
		} else {
			coreColor = color.RGBA{200, 150, 100, 255} // Warm orange for Surface
		}
	}

	style := NodeStyle{
		CoreColor:   coreColor,
		RingColor:   data.RingColor,
		HasRing:     data.HasRing,
		HasHalo:     data.Activity > 0,
		HaloAlpha:   float32(data.Activity) / 100.0, // Normalize to 0-1
		IsSpecter:   data.IsSpecter,
		Selected:    r.input.SelectedNodeID == data.ID,
		Connections: data.Connections,
		Activity:    data.Activity,
		Resonance:   data.Resonance,
	}

	// Clamp halo alpha.
	if style.HaloAlpha > 1.0 {
		style.HaloAlpha = 1.0
	}

	return style
}

// drawNodeGlow renders a glow effect around a node.
func (r *Renderer) drawNodeGlow(screen *ebiten.Image, x, y float32, style NodeStyle) {
	if r.shaders == nil || r.shaders.Glow == nil {
		return
	}

	uniforms := effects.GlowUniforms{
		Time:          r.time,
		GlowIntensity: style.HaloAlpha,
		GlowColor: [4]float32{
			float32(style.CoreColor.R) / 255.0,
			float32(style.CoreColor.G) / 255.0,
			float32(style.CoreColor.B) / 255.0,
			1.0,
		},
	}

	// Glow size is 3x node radius.
	radius := computeNodeRadius(style)
	r.shaders.DrawGlow(screen, x, y, radius*6, uniforms)
}

// transformAndCullLine transforms two world positions to screen coordinates and checks if visible.
// Returns screen coordinates and true if the line should be drawn, or false if culled.
func (r *Renderer) transformAndCullLine(
	srcWorldX, srcWorldY, dstWorldX, dstWorldY float64,
	screenW, screenH float64,
) (srcScreenX, srcScreenY, dstScreenX, dstScreenY float64, visible bool) {
	// Transform world coordinates to screen coordinates.
	srcScreenX, srcScreenY = r.camera.WorldToScreen(srcWorldX, srcWorldY, screenW, screenH)
	dstScreenX, dstScreenY = r.camera.WorldToScreen(dstWorldX, dstWorldY, screenW, screenH)

	// Cull lines completely outside screen (with margin).
	margin := 50.0
	visible = r.lineIntersectsRect(srcScreenX, srcScreenY, dstScreenX, dstScreenY,
		-margin, -margin, screenW+margin, screenH+margin)

	return srcScreenX, srcScreenY, dstScreenX, dstScreenY, visible
}

// lineIntersectsRect checks if a line segment intersects a rectangle.
// Uses Cohen-Sutherland-like approach for efficiency.
func (r *Renderer) lineIntersectsRect(x1, y1, x2, y2, minX, minY, maxX, maxY float64) bool {
	// Quick check: if both endpoints are on the same side of the rect, no intersection.
	if (x1 < minX && x2 < minX) || (x1 > maxX && x2 > maxX) ||
		(y1 < minY && y2 < minY) || (y1 > maxY && y2 > maxY) {
		return false
	}
	return true
}

// Layout returns the preferred layout size for Ebitengine.
// Per PULSE_MAP.md, the Pulse Map should be resizable.
func (r *Renderer) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// HandleMouseDown processes mouse down events for interaction.
func (r *Renderer) HandleMouseDown(x, y float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if clicking on a node.
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

// HandleMouseWheel processes mouse wheel events for zooming.
func (r *Renderer) HandleMouseWheel(x, y, deltaY float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.camera == nil {
		return
	}

	// Zoom factor based on wheel delta.
	factor := 1.0
	if deltaY > 0 {
		factor = 1.1
	} else if deltaY < 0 {
		factor = 0.9
	}

	r.camera.Zoom(factor, x, y, float64(r.screenWidth), float64(r.screenHeight))
}

// hitTestNodes finds the node at the given screen position.
func (r *Renderer) hitTestNodes(screenX, screenY float64) string {
	if r.engine == nil || r.camera == nil {
		return ""
	}

	// Convert screen to world coordinates.
	worldX, worldY := r.camera.ScreenToWorld(screenX, screenY,
		float64(r.screenWidth), float64(r.screenHeight))

	// Get current positions.
	positions := r.engine.Positions().Get()

	// Check each node for hit.
	for id, pos := range positions {
		data := r.nodeData[id]
		if data == nil {
			continue
		}

		// Calculate hit radius (slightly larger than visual for easier clicking).
		style := r.buildNodeStyle(data)
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

// drawCrossLayerArtifacts renders anonymous artifacts (Specter Marks, Phantom Gifts, etc.)
// overlaid on a node. This implements the Shadow Gradient visibility mechanism per PRODUCT_VISION.md:
// "Open-mode users see the anonymous layer's effects everywhere."
func (r *Renderer) drawCrossLayerArtifacts(screen *ebiten.Image, nodeData *NodeData, nodeX, nodeY float32) {
	// Query marks for this node from the store.
	// Use PublicKey as the target identifier.
	if len(nodeData.PublicKey) == 0 {
		return // No pubkey, can't query
	}

	marks, err := r.store.ListMarksForTarget(nodeData.PublicKey)
	if err != nil || len(marks) == 0 {
		return // No marks or query failed
	}

	// Render marks as orbiting icons.
	// Per ANONYMOUS_GAME_MECHANICS.md, marks appear as orbiting sigil icons on marked nodes.
	for i, mark := range marks {
		if mark == nil {
			continue
		}

		// Calculate age for visibility decay (marks decay over 30 days).
		createdAt := time.Unix(mark.CreatedAt, 0)
		expiresAt := time.Unix(mark.ExpiresAt, 0)
		age := time.Since(createdAt)
		lifetime := expiresAt.Sub(createdAt)

		// Skip expired marks.
		if time.Now().After(expiresAt) {
			continue
		}

		// Calculate visibility (1.0 → 0.0 linear decay over lifetime).
		visibility := float32(1.0 - (float64(age) / float64(lifetime)))
		if visibility < 0 {
			visibility = 0
		}

		// Stack orbits for multiple marks.
		orbitRadius := 24.0 + float32(i)*6.0

		// Orbit angle based on elapsed time and mark ID for variety.
		// Use mark ID's first byte to seed unique orbit speed.
		orbitSpeed := 0.5 + float32(mark.Id[0]%64)/128.0 // 0.5 to 1.0 rad/sec
		orbitAngle := float32(r.time) * orbitSpeed

		// Calculate orbit position.
		x := nodeX + float32(math.Cos(float64(orbitAngle)))*orbitRadius
		y := nodeY + float32(math.Sin(float64(orbitAngle)))*orbitRadius

		// Draw mark icon as a small circle with pulsing glow.
		// Color based on first byte of Specter pubkey for variety.
		alpha := uint8(visibility * 200) // 0-200 alpha
		markColor := color.RGBA{
			R: 100 + mark.SpecterPubkey[0]%100,
			G: 150,
			B: 200 + mark.SpecterPubkey[1]%55,
			A: alpha,
		}

		// Draw outer glow (pulsing).
		pulsePhase := float32(math.Sin(float64(r.time) * 2))
		glowRadius := 5.0 + pulsePhase*2.0
		glowAlpha := uint8(float32(alpha) * 0.3)
		glowColor := color.RGBA{markColor.R, markColor.G, markColor.B, glowAlpha}
		vector.DrawFilledCircle(screen, x, y, glowRadius, glowColor, false)

		// Draw core icon.
		vector.DrawFilledCircle(screen, x, y, 3.0, markColor, false)
	}
}
