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

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/opd-ai/murmur/pkg/pulsemap/interaction"
	"github.com/opd-ai/murmur/pkg/pulsemap/layout"
	"github.com/opd-ai/murmur/pkg/pulsemap/overlays"
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

	// backgroundColor is the Pulse Map background color (deprecated - use background renderer).
	backgroundColor color.RGBA

	// background renders the procedural gradient background with noise.
	// Per ROADMAP.md line 686, this creates a dark blue-gray gradient with procedural noise.
	background *BackgroundRenderer

	// particles renders ambient drifting particles for atmospheric depth.
	// Per ROADMAP.md line 687, this creates a sparse particle field.
	particles *AmbientParticleField

	// screenWidth and screenHeight are the current screen dimensions.
	screenWidth, screenHeight int

	// time tracks elapsed time for animations.
	time float32

	// Layer images for framebuffer compositing.
	// Per ROADMAP.md line 688, separate layers are composited for
	// background/nodes/overlays/UI to improve rendering performance.
	backgroundLayer *ebiten.Image
	graphLayer      *ebiten.Image
	overlayLayer    *ebiten.Image
	uiLayer         *ebiten.Image

	// batchRenderer accumulates draw commands for batched execution.
	// Per ROADMAP.md line 692, batched rendering groups operations by type
	// to reduce draw call overhead and improve GPU utilization.
	batchRenderer *BatchRenderer

	// layerBlend controls Surface/Anonymous layer opacity blend.
	// Per AUDIT.md MEDIUM fix: overlay layer was never populated; LayerBlend drives it.
	layerBlend *overlays.LayerBlend

	// specterEmitters maps Specter node IDs to their particle emitters for the overlay layer.
	// Per AUDIT.md MEDIUM fix: specterEmitters were defined in overlays but never used in Renderer.
	specterEmitters map[string]*overlays.ParticleEmitter
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
		backgroundColor:     color.RGBA{10, 12, 18, 255}, // Fallback solid color
		background:          NewBackgroundRenderer(),     // Procedural gradient with noise per ROADMAP.md line 686
		particles:           NewAmbientParticleField(),   // Sparse drifting particles per ROADMAP.md line 687
		screenWidth:         800,
		screenHeight:        600,
		batchRenderer:       NewBatchRenderer(), // Batched rendering per ROADMAP.md line 692
		layerBlend:          overlays.NewDefaultBlend(),
		specterEmitters:     make(map[string]*overlays.ParticleEmitter),
	}, nil
}

// SetCamera sets the camera for viewport transformations.
func (r *Renderer) SetCamera(camera *interaction.Camera) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.camera = camera
}

// SetLayerBlend sets the Surface/Anonymous layer blend ratio.
// Per AUDIT.md MEDIUM fix: exposes layerBlend to the UI for toggle control.
func (r *Renderer) SetLayerBlend(blend *overlays.LayerBlend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.layerBlend = blend
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

	// Resize framebuffer layers here, while the write lock is already held,
	// to avoid the lock-downgrade race window that existed in Draw().
	// Per audit MEDIUM finding: RLock→unlock→Lock→unlock→RLock creates a gap
	// where concurrent goroutines could observe inconsistent state.
	w, h := ebiten.WindowSize()
	if w > 0 && h > 0 && (w != r.screenWidth || h != r.screenHeight || r.backgroundLayer == nil) {
		r.screenWidth = w
		r.screenHeight = h
		r.ensureLayers(w, h)
	}

	// Update camera animations.
	if r.camera != nil {
		r.camera.Update()

		// Update ambient particles with camera position for parallax.
		if r.particles != nil {
			dt := 1.0 / float64(TargetFPS)
			r.particles.Update(dt, r.camera.X, r.camera.Y, r.screenWidth, r.screenHeight)
		}
	}

	return nil
}

// ensureLayers creates or resizes layer images to match screen dimensions.
// Per ROADMAP.md line 688, separate layers are composited for
// background/nodes/overlays/UI to improve rendering performance.
func (r *Renderer) ensureLayers(w, h int) {
	if r.backgroundLayer == nil || r.backgroundLayer.Bounds().Dx() != w || r.backgroundLayer.Bounds().Dy() != h {
		r.backgroundLayer = ebiten.NewImage(w, h)
	}
	if r.graphLayer == nil || r.graphLayer.Bounds().Dx() != w || r.graphLayer.Bounds().Dy() != h {
		r.graphLayer = ebiten.NewImage(w, h)
	}
	if r.overlayLayer == nil || r.overlayLayer.Bounds().Dx() != w || r.overlayLayer.Bounds().Dy() != h {
		r.overlayLayer = ebiten.NewImage(w, h)
	}
	if r.uiLayer == nil || r.uiLayer.Bounds().Dx() != w || r.uiLayer.Bounds().Dy() != h {
		r.uiLayer = ebiten.NewImage(w, h)
	}
}

// Draw renders the Pulse Map to the given screen.
// This is the main draw loop called by Ebitengine.
// Per ROADMAP.md line 688, rendering uses separate framebuffer layers
// (background/graph/overlays/UI) composited for improved performance.
func (r *Renderer) Draw(screen *ebiten.Image) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Layers are sized in Update() under the write lock; no lock downgrade needed here.
	// Guard against Draw() being called before the first Update() initialises layers.
	if r.backgroundLayer == nil {
		screen.Fill(r.backgroundColor)
		return
	}

	// Clear all layers for fresh frame.
	r.backgroundLayer.Clear()
	r.graphLayer.Clear()
	r.overlayLayer.Clear()
	r.uiLayer.Clear()

	// Layer 1: Draw background (gradient + noise).
	// Per ROADMAP.md line 686, this creates a dark blue-gray gradient with procedural noise.
	if r.background != nil {
		r.background.Draw(r.backgroundLayer)
	} else {
		// Fallback to solid color if background renderer failed.
		r.backgroundLayer.Fill(r.backgroundColor)
	}

	// Layer 2: Draw ambient particles for atmospheric depth.
	// Per ROADMAP.md line 687, this creates a sparse drifting particle field.
	// Note: Particles drawn on background layer for performance (avoid extra composite).
	if r.particles != nil && r.camera != nil {
		r.particles.Draw(r.backgroundLayer, r.camera.X, r.camera.Y)
	}

	if r.engine == nil || r.camera == nil {
		// Composite background layer to screen.
		screen.DrawImage(r.backgroundLayer, nil)
		return
	}

	// Get current node positions from the double-buffered layout.
	positions := r.engine.Positions().Get()
	if len(positions) == 0 {
		// Composite background layer to screen.
		screen.DrawImage(r.backgroundLayer, nil)
		return
	}

	// Inject a frame-current NodePositionFunc into the store so that
	// cross-layer spatial queries (GetActivePuzzlesNearNode, etc.) can
	// filter by actual Pulse Map proximity rather than returning all records.
	// Per ROADMAP.md: "Replace placeholder cross-layer spatial queries with
	// actual location-aware selectors".
	if r.store != nil {
		r.store.SetNodePositioner(r.buildNodePositionFunc(positions))
	}

	// Compute visible bounds for culling.
	minX, minY, maxX, maxY := r.camera.ViewBounds(float64(r.screenWidth), float64(r.screenHeight))

	// Calculate zoom level for detail decisions.
	zoom := ZoomLevelFromScale(r.camera.Scale)

	// Layer 3: Draw graph (edges + nodes) using batched rendering.
	// Per ROADMAP.md line 692, batch rendering groups operations by type
	// to reduce draw call overhead and improve GPU utilization.
	r.batchRenderer.Clear()

	// Accumulate edges into batch.
	r.accumulateEdges(positions, zoom)

	// Accumulate amplification trails into batch.
	r.accumulateAmplificationTrails(positions, zoom)

	// Accumulate nodes into batch.
	r.accumulateNodes(positions, minX, minY, maxX, maxY, zoom)

	// Execute all batched draw commands at once.
	r.batchRenderer.Flush(r.graphLayer)

	// Layer 4: Overlays (Specter particle emitters for Anonymous layer visibility).
	// Per AUDIT.md MEDIUM fix: populate overlayLayer with Specter emitter particles
	// when AnonymousOpacity > 0. Previously this layer was always empty.
	if r.layerBlend != nil && r.layerBlend.AnonymousOpacity > 0 {
		for id, pos := range positions {
			data := r.nodeData[id]
			if data == nil || !data.IsSpecter {
				continue
			}
			emitter, ok := r.specterEmitters[id]
			if !ok {
				emitter = overlays.NewParticleEmitter(20, 0.5)
				r.specterEmitters[id] = emitter
			}
			nodeRadius := float32(computeNodeRadius(r.buildNodeStyle(data)))
			emitter.Update(1.0/60.0, float32(pos.X), float32(pos.Y), nodeRadius, float32(data.Resonance))
			emitter.Render(r.overlayLayer,
				float32(r.camera.X), float32(r.camera.Y), float32(r.camera.Scale))
		}
	}

	// Layer 5: UI elements (controls, notifications, etc.).
	// Currently empty - future implementation for UI widgets.
	// This layer will hold viewport controls, zoom buttons, notifications, etc.

	// Composite all layers to screen in order.
	screen.DrawImage(r.backgroundLayer, nil)
	screen.DrawImage(r.graphLayer, nil)
	screen.DrawImage(r.overlayLayer, nil)
	screen.DrawImage(r.uiLayer, nil)
}

// drawEdges renders all edges between nodes with pulse animations.
func (r *Renderer) drawEdges(screen *ebiten.Image, positions map[string]layout.Position, zoom ZoomLevel) {
	r.iterateEdges(positions, zoom, func(srcX, srcY, dstX, dstY float32, style EdgeStyle) {
		RenderEdgeWithTime(screen, srcX, srcY, dstX, dstY, style, zoom, float64(r.time))
	})
}

// iterateEdges processes all edges with culling and style building, invoking callback for visible edges.
func (r *Renderer) iterateEdges(positions map[string]layout.Position, zoom ZoomLevel, callback func(srcX, srcY, dstX, dstY float32, style EdgeStyle)) {
	screenW := float64(r.screenWidth)
	screenH := float64(r.screenHeight)

	for _, edge := range r.edges {
		srcPos, srcOK := positions[edge.SourceID]
		dstPos, dstOK := positions[edge.TargetID]
		if !srcOK || !dstOK {
			continue
		}

		srcScreenX, srcScreenY, dstScreenX, dstScreenY, visible := r.transformAndCullLine(srcPos.X, srcPos.Y, dstPos.X, dstPos.Y, screenW, screenH)
		if !visible {
			continue
		}

		style := EdgeStyle{
			Color:                color.RGBA{100, 120, 140, 255},
			Age:                  edge.Age,
			Active:               edge.Active,
			InteractionFrequency: edge.InteractionFrequency,
		}

		callback(float32(srcScreenX), float32(srcScreenY), float32(dstScreenX), float32(dstScreenY), style)
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

		// Render cross-layer artifacts (Specter Marks, Gifts, etc.) only at Micro zoom.
		// This avoids expensive per-node store reads when details are too small to be
		// meaningful and prevents transition stutter during pan/zoom at wider views.
		if r.store != nil && zoom == ZoomMicro {
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
	// Clear any orphaned drag state before starting a new interaction.
	// Per AUDIT.md: mirrors the guard in game.go handleDragging() — if Dragging
	// is stuck true from a prior unclosed drag, reset it first.
	if r.input.Dragging {
		r.input.EndDrag()
	}
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

		// Calculate hit radius in world units: use the larger of the zoom-adjusted
		// visual radius and a constant minimum to avoid misses at high zoom or
		// bloated zones at low zoom. Per AUDIT.md HIGH finding.
		const baseHitRadius = 8.0 // world units, matches rBase in computeNodeRadius
		style := r.buildNodeStyle(data)
		radius := math.Max(float64(computeNodeRadius(style))/r.camera.Scale, baseHitRadius)

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

// NodeAtScreen returns the node ID under the given screen-space position.
// Returns an empty string when no node is hit.
func (r *Renderer) NodeAtScreen(x, y float64) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.hitTestNodes(x, y)
}

// GetNodeData returns the NodeData for the given node ID, or nil if not found.
// This is used by the Node Detail Panel to query node information.
func (r *Renderer) GetNodeData(nodeID string) *NodeData {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.nodeData[nodeID]
}

// GetAllNodes returns a copy of all node data for searching.
// This is used by the search bar to build search results.
func (r *Renderer) GetAllNodes() []*NodeData {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*NodeData, 0, len(r.nodeData))
	for _, node := range r.nodeData {
		// Return a copy to avoid concurrent access issues
		nodeCopy := *node
		result = append(result, &nodeCopy)
	}
	return result
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
	// Per PLAN.md Step 6, this function was refactored to reduce cyclomatic
	// complexity from 34 to <15 by extracting artifact drawing into helpers.
	r.drawSpecterMarks(screen, nodeData, nodeX, nodeY)
	r.drawPhantomGifts(screen, nodeData, nodeX, nodeY)
	r.drawCipherPuzzles(screen, nodeData, nodeX, nodeY)
	r.drawSpecterHunts(screen, nodeData, nodeX, nodeY)
	r.drawTerritoryInfluence(screen, nodeData, nodeX, nodeY)
	r.drawOraclePools(screen, nodeData, nodeX, nodeY)
	r.drawForgeProjects(screen, nodeData, nodeX, nodeY)
	r.drawShadowPlays(screen, nodeData, nodeX, nodeY)
	r.drawPhantomCouncils(screen, nodeData, nodeX, nodeY)
	// Note: Masked Events rendering deferred per PLAN.md (full implementation future work).
}

// buildNodePositionFunc constructs a store.NodePositionFunc snapshot from the
// current frame's layout positions and node metadata.
// It must be called while r.mu is held (read or write).
// Per PULSE_MAP.md §2, coordinates are in force-directed layout units.
func (r *Renderer) buildNodePositionFunc(positions map[string]layout.Position) store.NodePositionFunc {
	pubkeyToPos := make(map[string][2]float64, len(r.nodeData))
	for id, data := range r.nodeData {
		if len(data.PublicKey) == 0 {
			continue
		}
		pos, ok := positions[id]
		if !ok {
			continue
		}
		pubkeyToPos[string(data.PublicKey)] = [2]float64{pos.X, pos.Y}
	}
	return func(pubkey []byte) (x, y float64, ok bool) {
		p, found := pubkeyToPos[string(pubkey)]
		if !found {
			return 0, 0, false
		}
		return p[0], p[1], true
	}
}

// accumulateEdges adds all edges to the batch renderer.
// This replaces the old drawEdges method for batched rendering.
func (r *Renderer) accumulateEdges(positions map[string]layout.Position, zoom ZoomLevel) {
	r.iterateEdges(positions, zoom, func(srcX, srcY, dstX, dstY float32, style EdgeStyle) {
		r.batchRenderer.AddEdge(srcX, srcY, dstX, dstY, style, zoom)
	})
}

// accumulateAmplificationTrails adds all amplification trails to the batch renderer.
// This replaces the old drawAmplificationTrails method for batched rendering.
func (r *Renderer) accumulateAmplificationTrails(positions map[string]layout.Position, zoom ZoomLevel) {
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

		// Calculate fade alpha.
		baseAlpha := calculateTrailFade(trail.RecentSeconds)
		if baseAlpha < 10 {
			continue // Skip nearly invisible trails
		}

		// Add to batch renderer.
		r.batchRenderer.AddTrail(float32(ampScreenX), float32(ampScreenY),
			float32(origScreenX), float32(origScreenY),
			baseAlpha, trail.HasComment, float64(r.time))
	}
}

// accumulateNodes adds all visible nodes to the batch renderer.
// This replaces the old drawNodes method for batched rendering.
func (r *Renderer) accumulateNodes(positions map[string]layout.Position,
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

		// Calculate radius.
		radius := computeNodeRadius(style)

		// Add node to batch renderer.
		r.batchRenderer.AddNode(float32(screenX), float32(screenY), radius, style)

		// Render glow effect for active/selected nodes (not batched - uses shaders).
		if r.shaders != nil && (style.HasHalo || style.Selected) {
			r.drawNodeGlow(r.graphLayer, float32(screenX), float32(screenY), style)
		}

		// Render cross-layer artifacts (not batched - complex custom rendering).
		// Per AUDIT.md HIGH finding "Cross-layer visibility not implemented", this enables
		// Surface users to see anonymous activity on their Pulse Map.
		if r.store != nil && zoom == ZoomMicro {
			r.drawCrossLayerArtifacts(r.graphLayer, data, float32(screenX), float32(screenY))
		}

		// Render text label at Micro zoom level (not batched - uses text rendering).
		RenderTextLabel(r.graphLayer, float32(screenX), float32(screenY), data.DisplayName, data.IsSpecter, zoom)
	}
}
