// Package overlays - Echo Chain Pulse Map visualization.
// Per ANONYMOUS_GAME_MECHANICS.md: "Echo Chains create visible golden (Surface) or
// silver (Anonymous) arcs connecting amplification participants on the Pulse Map.
// Long chains (5+ nodes) develop a shimmer effect. The chain fades after 1 hour."
// Per ROADMAP.md line 565: "Pulse Map visualization — animated amplification trail between nodes"
//

//go:build !test
// +build !test

package overlays

import (
	"image/color"
	"math"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ChainLayer indicates which layer the chain belongs to.
type ChainLayer uint8

const (
	ChainLayerSurface   ChainLayer = iota + 1 // Golden arcs.
	ChainLayerAnonymous                       // Silver arcs.
)

// ChainNodeInfo contains node information for visualization.
type ChainNodeInfo struct {
	NodeID      [32]byte  // Public key of the amplifier.
	X, Y        float64   // Position on the Pulse Map.
	AmplifiedAt time.Time // When this amplification occurred.
	Position    int       // Position in chain (0 = first amplifier).
}

// EchoChainInfo contains chain information for visualization.
type EchoChainInfo struct {
	ChainID    [32]byte         // Unique chain identifier.
	OriginalID [32]byte         // ID of the original Wave.
	Layer      ChainLayer       // Surface or Anonymous.
	Nodes      []*ChainNodeInfo // Ordered list of chain participants.
	FormedAt   time.Time        // When chain reached minimum length.
	ExpiresAt  time.Time        // When chain visual expires.
	HasShimmer bool             // True if chain length >= 5.
}

// EchoChainOverlay renders Echo Chains on the Pulse Map.
type EchoChainOverlay struct {
	mu sync.RWMutex

	visible bool
	chains  map[[32]byte]*EchoChainInfo
	time    float64 // Animation time.

	// Visual settings.
	surfaceColor   color.RGBA // Golden arc color for Surface chains.
	anonymousColor color.RGBA // Silver arc color for Anonymous chains.
	shimmerColor   color.RGBA // Shimmer highlight color.
	nodeHighlight  color.RGBA // Node marker color.
	fadeColor      color.RGBA // Fading chain color.
}

// NewEchoChainOverlay creates a new Echo Chain overlay.
func NewEchoChainOverlay() *EchoChainOverlay {
	return &EchoChainOverlay{
		visible: true,
		chains:  make(map[[32]byte]*EchoChainInfo),
		surfaceColor: color.RGBA{
			R: 255,
			G: 200,
			B: 80,
			A: 200,
		},
		anonymousColor: color.RGBA{
			R: 180,
			G: 190,
			B: 210,
			A: 200,
		},
		shimmerColor: color.RGBA{
			R: 255,
			G: 255,
			B: 220,
			A: 180,
		},
		nodeHighlight: color.RGBA{
			R: 255,
			G: 255,
			B: 255,
			A: 220,
		},
		fadeColor: color.RGBA{
			R: 100,
			G: 100,
			B: 100,
			A: 80,
		},
	}
}

// SetVisible controls visibility.
func (o *EchoChainOverlay) SetVisible(visible bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.visible = visible
}

// IsVisible returns visibility status.
func (o *EchoChainOverlay) IsVisible() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.visible
}

// SetChain adds or updates an echo chain.
func (o *EchoChainOverlay) SetChain(chain *EchoChainInfo) {
	if chain == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.chains[chain.ChainID] = chain
}

// RemoveChain removes a chain by ID.
func (o *EchoChainOverlay) RemoveChain(id [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.chains, id)
}

// GetChain returns a chain by ID.
func (o *EchoChainOverlay) GetChain(id [32]byte) *EchoChainInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.chains[id]
}

// GetAllChains returns all chains.
func (o *EchoChainOverlay) GetAllChains() []*EchoChainInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	chains := make([]*EchoChainInfo, 0, len(o.chains))
	for _, c := range o.chains {
		chains = append(chains, c)
	}
	return chains
}

// GetActiveChains returns non-expired chains.
func (o *EchoChainOverlay) GetActiveChains() []*EchoChainInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	now := time.Now()
	var active []*EchoChainInfo
	for _, c := range o.chains {
		if now.Before(c.ExpiresAt) {
			active = append(active, c)
		}
	}
	return active
}

// UpdateNodePosition updates the position of a node in all chains.
func (o *EchoChainOverlay) UpdateNodePosition(nodeID [32]byte, x, y float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, chain := range o.chains {
		for _, node := range chain.Nodes {
			if node.NodeID == nodeID {
				node.X = x
				node.Y = y
			}
		}
	}
}

// Update advances animation state.
func (o *EchoChainOverlay) Update(dt float64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.time += dt
}

// Draw renders the echo chains.
func (o *EchoChainOverlay) Draw(screen *ebiten.Image, cameraX, cameraY, zoom float64) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.visible {
		return
	}

	screenW := float64(screen.Bounds().Dx())
	screenH := float64(screen.Bounds().Dy())
	centerX := screenW / 2
	centerY := screenH / 2

	now := time.Now()

	for _, chain := range o.chains {
		if len(chain.Nodes) < 2 {
			continue
		}

		// Skip expired chains.
		if now.After(chain.ExpiresAt) {
			continue
		}

		// Calculate fade factor based on time remaining.
		totalDuration := chain.ExpiresAt.Sub(chain.FormedAt).Seconds()
		remaining := chain.ExpiresAt.Sub(now).Seconds()
		fadeFactor := 1.0
		if totalDuration > 0 {
			fadeFactor = remaining / totalDuration
			if fadeFactor > 1 {
				fadeFactor = 1
			}
			if fadeFactor < 0 {
				fadeFactor = 0
			}
		}

		o.drawChain(screen, chain, centerX, centerY, cameraX, cameraY, zoom, float32(fadeFactor))
	}
}

// drawChain draws a single echo chain.
func (o *EchoChainOverlay) drawChain(screen *ebiten.Image, chain *EchoChainInfo, centerX, centerY, cameraX, cameraY, zoom float64, fadeFactor float32) {
	screenW := float64(screen.Bounds().Dx())
	screenH := float64(screen.Bounds().Dy())

	// Get base color based on layer.
	baseColor := o.surfaceColor
	if chain.Layer == ChainLayerAnonymous {
		baseColor = o.anonymousColor
	}

	// Apply fade factor to alpha.
	arcAlpha := uint8(float32(baseColor.A) * fadeFactor)
	arcColor := color.RGBA{
		R: baseColor.R,
		G: baseColor.G,
		B: baseColor.B,
		A: arcAlpha,
	}

	// Draw arcs between consecutive nodes.
	for i := 0; i < len(chain.Nodes)-1; i++ {
		node1 := chain.Nodes[i]
		node2 := chain.Nodes[i+1]

		// Transform to screen coordinates.
		sx1 := centerX + (node1.X-cameraX)*zoom
		sy1 := centerY + (node1.Y-cameraY)*zoom
		sx2 := centerX + (node2.X-cameraX)*zoom
		sy2 := centerY + (node2.Y-cameraY)*zoom

		// Check if segment is visible (at least one endpoint on screen).
		if !o.isSegmentVisible(sx1, sy1, sx2, sy2, screenW, screenH) {
			continue
		}

		// Draw the arc between nodes.
		o.drawArc(screen, float32(sx1), float32(sy1), float32(sx2), float32(sy2), float32(zoom), arcColor, chain.HasShimmer, i)

		// Draw node marker at start of segment.
		o.drawNodeMarker(screen, float32(sx1), float32(sy1), float32(zoom), fadeFactor, i == 0)
	}

	// Draw final node marker.
	lastNode := chain.Nodes[len(chain.Nodes)-1]
	sx := centerX + (lastNode.X-cameraX)*zoom
	sy := centerY + (lastNode.Y-cameraY)*zoom
	if sx >= -50 && sx <= screenW+50 && sy >= -50 && sy <= screenH+50 {
		o.drawNodeMarker(screen, float32(sx), float32(sy), float32(zoom), fadeFactor, false)
	}

	// Draw animated pulse traveling along the chain.
	o.drawPulse(screen, chain, centerX, centerY, cameraX, cameraY, zoom, fadeFactor)
}

// isSegmentVisible checks if a line segment is potentially visible.
func (o *EchoChainOverlay) isSegmentVisible(x1, y1, x2, y2, screenW, screenH float64) bool {
	margin := 100.0

	// Check if either point is on screen.
	if x1 >= -margin && x1 <= screenW+margin && y1 >= -margin && y1 <= screenH+margin {
		return true
	}
	if x2 >= -margin && x2 <= screenW+margin && y2 >= -margin && y2 <= screenH+margin {
		return true
	}

	// Check if segment crosses screen bounds.
	return o.segmentIntersectsRect(x1, y1, x2, y2, -margin, -margin, screenW+margin, screenH+margin)
}

// segmentIntersectsRect checks if a line segment intersects a rectangle.
func (o *EchoChainOverlay) segmentIntersectsRect(x1, y1, x2, y2, left, top, right, bottom float64) bool {
	// Simple bounding box check.
	minX := math.Min(x1, x2)
	maxX := math.Max(x1, x2)
	minY := math.Min(y1, y2)
	maxY := math.Max(y1, y2)

	return maxX >= left && minX <= right && maxY >= top && minY <= bottom
}

// drawArc draws a curved arc between two points.
func (o *EchoChainOverlay) drawArc(screen *ebiten.Image, x1, y1, x2, y2, zoom float32, arcColor color.RGBA, hasShimmer bool, segmentIndex int) {
	// Calculate arc parameters.
	dx := x2 - x1
	dy := y2 - y1
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))

	if dist < 5 {
		return // Too close to draw meaningful arc.
	}

	// Arc width based on zoom.
	arcWidth := 3.0 * zoom
	if arcWidth < 1.5 {
		arcWidth = 1.5
	}
	if arcWidth > 6 {
		arcWidth = 6
	}

	// Calculate control point for quadratic bezier (arc bulge).
	midX := (x1 + x2) / 2
	midY := (y1 + y2) / 2

	// Perpendicular offset for arc curve.
	perpX := -dy / dist
	perpY := dx / dist

	// Arc height based on distance.
	arcHeight := dist * 0.2
	if arcHeight < 10 {
		arcHeight = 10
	}
	if arcHeight > 50 {
		arcHeight = 50
	}

	// Alternate arc direction based on segment index.
	direction := float32(1.0)
	if segmentIndex%2 == 1 {
		direction = -1.0
	}

	controlX := midX + perpX*arcHeight*direction
	controlY := midY + perpY*arcHeight*direction

	// Draw arc as series of line segments (quadratic bezier approximation).
	segments := 12
	prevX, prevY := x1, y1

	for i := 1; i <= segments; i++ {
		t := float32(i) / float32(segments)

		// Quadratic bezier: (1-t)^2 * P0 + 2*(1-t)*t * P1 + t^2 * P2
		oneMinusT := 1 - t
		px := oneMinusT*oneMinusT*x1 + 2*oneMinusT*t*controlX + t*t*x2
		py := oneMinusT*oneMinusT*y1 + 2*oneMinusT*t*controlY + t*t*y2

		vector.StrokeLine(screen, prevX, prevY, px, py, float32(arcWidth), arcColor, true)

		prevX, prevY = px, py
	}

	// Draw shimmer effect for long chains.
	if hasShimmer {
		o.drawShimmer(screen, x1, y1, x2, y2, controlX, controlY, zoom, segmentIndex)
	}
}

// drawShimmer draws a shimmer effect along the arc.
func (o *EchoChainOverlay) drawShimmer(screen *ebiten.Image, x1, y1, x2, y2, ctrlX, ctrlY, zoom float32, segmentIndex int) {
	// Animated shimmer position along the arc.
	shimmerT := float32(math.Mod(o.time*0.5+float64(segmentIndex)*0.3, 1.0))

	// Calculate shimmer position on bezier curve.
	oneMinusT := 1 - shimmerT
	shimmerX := oneMinusT*oneMinusT*x1 + 2*oneMinusT*shimmerT*ctrlX + shimmerT*shimmerT*x2
	shimmerY := oneMinusT*oneMinusT*y1 + 2*oneMinusT*shimmerT*ctrlY + shimmerT*shimmerT*y2

	// Shimmer size with pulse effect.
	pulse := float32(math.Sin(o.time*4+float64(segmentIndex))*0.3 + 0.7)
	shimmerSize := 4 * zoom * pulse
	if shimmerSize < 2 {
		shimmerSize = 2
	}

	shimmerAlpha := uint8(180 * pulse)
	shimmerColor := color.RGBA{
		R: o.shimmerColor.R,
		G: o.shimmerColor.G,
		B: o.shimmerColor.B,
		A: shimmerAlpha,
	}

	// Draw shimmer glow.
	for i := 2; i >= 0; i-- {
		glowSize := shimmerSize * float32(1+i) / 2
		glowAlpha := uint8(float32(shimmerAlpha) / float32(i+1))
		glowColor := color.RGBA{shimmerColor.R, shimmerColor.G, shimmerColor.B, glowAlpha}
		vector.DrawFilledCircle(screen, shimmerX, shimmerY, glowSize, glowColor, true)
	}

	// Draw shimmer core.
	vector.DrawFilledCircle(screen, shimmerX, shimmerY, shimmerSize*0.5, color.RGBA{255, 255, 255, 200}, true)
}

// drawNodeMarker draws a highlight marker at a chain node position.
func (o *EchoChainOverlay) drawNodeMarker(screen *ebiten.Image, x, y, zoom, fadeFactor float32, isOrigin bool) {
	scale := zoom * 0.5
	if scale < 0.3 {
		scale = 0.3
	}
	if scale > 1.5 {
		scale = 1.5
	}

	// Base marker size.
	baseSize := 5 * scale
	if isOrigin {
		baseSize = 7 * scale // Origin node is larger.
	}

	// Pulse effect.
	pulse := float32(math.Sin(o.time*2)*0.2 + 0.8)
	size := baseSize * pulse

	// Apply fade.
	markerAlpha := uint8(float32(o.nodeHighlight.A) * fadeFactor * pulse)
	markerColor := color.RGBA{
		R: o.nodeHighlight.R,
		G: o.nodeHighlight.G,
		B: o.nodeHighlight.B,
		A: markerAlpha,
	}

	// Draw glow.
	glowAlpha := uint8(float32(markerAlpha) / 3)
	glowColor := color.RGBA{markerColor.R, markerColor.G, markerColor.B, glowAlpha}
	vector.DrawFilledCircle(screen, x, y, size*1.5, glowColor, true)

	// Draw marker.
	vector.DrawFilledCircle(screen, x, y, size, markerColor, true)

	// Origin node gets a special ring.
	if isOrigin {
		ringColor := color.RGBA{255, 220, 100, markerAlpha}
		vector.StrokeCircle(screen, x, y, size+2*scale, 2, ringColor, true)
	}
}

// drawPulse draws an animated pulse traveling along the chain.
func (o *EchoChainOverlay) drawPulse(screen *ebiten.Image, chain *EchoChainInfo, centerX, centerY, cameraX, cameraY, zoom float64, fadeFactor float32) {
	if len(chain.Nodes) < 2 {
		return
	}

	// Pulse travels the full chain length every 3 seconds.
	pulseT := float32(math.Mod(o.time/3.0, 1.0))

	// Calculate total chain length.
	totalLen := 0.0
	for i := 0; i < len(chain.Nodes)-1; i++ {
		node1 := chain.Nodes[i]
		node2 := chain.Nodes[i+1]
		dx := node2.X - node1.X
		dy := node2.Y - node1.Y
		totalLen += math.Sqrt(dx*dx + dy*dy)
	}

	if totalLen < 1 {
		return
	}

	// Find pulse position along chain.
	targetDist := pulseT * float32(totalLen)
	currentDist := float32(0)

	for i := 0; i < len(chain.Nodes)-1; i++ {
		node1 := chain.Nodes[i]
		node2 := chain.Nodes[i+1]

		dx := node2.X - node1.X
		dy := node2.Y - node1.Y
		segLen := float32(math.Sqrt(dx*dx + dy*dy))

		if currentDist+segLen >= targetDist {
			// Pulse is on this segment.
			t := (targetDist - currentDist) / segLen
			pulseX := node1.X + float64(t)*dx
			pulseY := node1.Y + float64(t)*dy

			// Transform to screen coordinates.
			sx := centerX + (pulseX-cameraX)*zoom
			sy := centerY + (pulseY-cameraY)*zoom

			// Draw pulse.
			pulseSize := float32(zoom) * 4 * fadeFactor
			if pulseSize < 2 {
				pulseSize = 2
			}

			pulseAlpha := uint8(200 * fadeFactor)
			pulseColor := color.RGBA{255, 255, 255, pulseAlpha}

			// Glow.
			glowColor := color.RGBA{255, 255, 200, uint8(float32(pulseAlpha) / 2)}
			vector.DrawFilledCircle(screen, float32(sx), float32(sy), pulseSize*2, glowColor, true)

			// Core.
			vector.DrawFilledCircle(screen, float32(sx), float32(sy), pulseSize, pulseColor, true)

			return
		}

		currentDist += segLen
	}
}

// ChainCount returns the total number of chains.
func (o *EchoChainOverlay) ChainCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.chains)
}

// ActiveChainCount returns the number of non-expired chains.
func (o *EchoChainOverlay) ActiveChainCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()

	now := time.Now()
	count := 0
	for _, c := range o.chains {
		if now.Before(c.ExpiresAt) {
			count++
		}
	}
	return count
}

// ShimmeringChainCount returns the number of chains with shimmer effect.
func (o *EchoChainOverlay) ShimmeringChainCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()

	now := time.Now()
	count := 0
	for _, c := range o.chains {
		if c.HasShimmer && now.Before(c.ExpiresAt) {
			count++
		}
	}
	return count
}

// ClearExpired removes expired chains.
func (o *EchoChainOverlay) ClearExpired() int {
	o.mu.Lock()
	defer o.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, chain := range o.chains {
		if now.After(chain.ExpiresAt) {
			delete(o.chains, id)
			removed++
		}
	}

	return removed
}

// GetChainsByLayer returns chains of a specific layer.
func (o *EchoChainOverlay) GetChainsByLayer(layer ChainLayer) []*EchoChainInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	now := time.Now()
	var chains []*EchoChainInfo
	for _, c := range o.chains {
		if c.Layer == layer && now.Before(c.ExpiresAt) {
			chains = append(chains, c)
		}
	}
	return chains
}

// ChainLayerString returns a human-readable name for a chain layer.
func ChainLayerString(layer ChainLayer) string {
	switch layer {
	case ChainLayerSurface:
		return "Surface"
	case ChainLayerAnonymous:
		return "Anonymous"
	default:
		return "Unknown"
	}
}

// AddNodeToChain adds a new node to an existing chain.
func (o *EchoChainOverlay) AddNodeToChain(chainID [32]byte, node *ChainNodeInfo) {
	if node == nil {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if chain, ok := o.chains[chainID]; ok {
		node.Position = len(chain.Nodes)
		chain.Nodes = append(chain.Nodes, node)

		// Update shimmer status.
		if len(chain.Nodes) >= 5 {
			chain.HasShimmer = true
		}
	}
}

// GetLongestChain returns the chain with the most nodes.
func (o *EchoChainOverlay) GetLongestChain() *EchoChainInfo {
	o.mu.RLock()
	defer o.mu.RUnlock()

	now := time.Now()
	var longest *EchoChainInfo
	maxLen := 0

	for _, c := range o.chains {
		if len(c.Nodes) > maxLen && now.Before(c.ExpiresAt) {
			longest = c
			maxLen = len(c.Nodes)
		}
	}

	return longest
}
