// Package layout - Viewport culling for force-directed layout.
// Per ROADMAP.md line 594: "Viewport culling — only compute forces for
// visible nodes".
// Per PULSE_MAP.md: Optimizes layout computation by only calculating forces
// for nodes currently visible in the viewport, plus a margin buffer.
package layout

import (
	"math"
	"sync"
)

// ViewportCulling handles visibility determination for layout optimization.
type ViewportCulling struct {
	mu sync.RWMutex

	// Viewport bounds in world coordinates.
	minX, minY float64
	maxX, maxY float64

	// Margin around viewport for force computation.
	// Nodes in this margin still participate in physics but may not be rendered.
	margin float64

	// Camera state.
	cameraX, cameraY float64
	zoom             float64
	screenW, screenH float64

	// Cached visibility state.
	visibleNodes  map[string]bool
	marginalNodes map[string]bool // In margin, not viewport.
	culledNodes   map[string]bool // Outside margin.

	// Statistics.
	lastVisibleCount  int
	lastMarginalCount int
	lastCulledCount   int
}

// NewViewportCulling creates a new viewport culling manager.
func NewViewportCulling() *ViewportCulling {
	return &ViewportCulling{
		margin:        200.0, // Default margin in world units.
		visibleNodes:  make(map[string]bool),
		marginalNodes: make(map[string]bool),
		culledNodes:   make(map[string]bool),
		zoom:          1.0,
		screenW:       800,
		screenH:       600,
	}
}

// SetMargin sets the margin around the viewport for force computation.
func (vc *ViewportCulling) SetMargin(margin float64) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.margin = margin
}

// SetCamera updates the camera position and zoom.
func (vc *ViewportCulling) SetCamera(x, y, zoom float64) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.cameraX = x
	vc.cameraY = y
	vc.zoom = zoom
	vc.updateBounds()
}

// SetScreenSize sets the screen dimensions for viewport calculation.
func (vc *ViewportCulling) SetScreenSize(width, height float64) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.screenW = width
	vc.screenH = height
	vc.updateBounds()
}

// updateBounds recalculates the viewport bounds in world coordinates.
func (vc *ViewportCulling) updateBounds() {
	// Convert screen dimensions to world dimensions.
	halfW := (vc.screenW / 2) / vc.zoom
	halfH := (vc.screenH / 2) / vc.zoom

	vc.minX = vc.cameraX - halfW
	vc.maxX = vc.cameraX + halfW
	vc.minY = vc.cameraY - halfH
	vc.maxY = vc.cameraY + halfH
}

// UpdateVisibility categorizes nodes as visible, marginal, or culled.
func (vc *ViewportCulling) UpdateVisibility(positions map[string]Position) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Clear caches.
	vc.visibleNodes = make(map[string]bool, len(positions))
	vc.marginalNodes = make(map[string]bool)
	vc.culledNodes = make(map[string]bool)

	marginMinX := vc.minX - vc.margin
	marginMaxX := vc.maxX + vc.margin
	marginMinY := vc.minY - vc.margin
	marginMaxY := vc.maxY + vc.margin

	for id, pos := range positions {
		if pos.X >= vc.minX && pos.X <= vc.maxX &&
			pos.Y >= vc.minY && pos.Y <= vc.maxY {
			vc.visibleNodes[id] = true
		} else if pos.X >= marginMinX && pos.X <= marginMaxX &&
			pos.Y >= marginMinY && pos.Y <= marginMaxY {
			vc.marginalNodes[id] = true
		} else {
			vc.culledNodes[id] = true
		}
	}

	vc.lastVisibleCount = len(vc.visibleNodes)
	vc.lastMarginalCount = len(vc.marginalNodes)
	vc.lastCulledCount = len(vc.culledNodes)
}

// IsVisible returns true if the node is in the viewport.
func (vc *ViewportCulling) IsVisible(nodeID string) bool {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.visibleNodes[nodeID]
}

// IsInMargin returns true if the node is in the margin but not viewport.
func (vc *ViewportCulling) IsInMargin(nodeID string) bool {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.marginalNodes[nodeID]
}

// IsCulled returns true if the node is outside the margin.
func (vc *ViewportCulling) IsCulled(nodeID string) bool {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.culledNodes[nodeID]
}

// ShouldComputeForces returns true if forces should be computed for this node.
// Nodes in the viewport and margin participate in physics.
func (vc *ViewportCulling) ShouldComputeForces(nodeID string) bool {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.visibleNodes[nodeID] || vc.marginalNodes[nodeID]
}

// GetVisibleNodes returns a slice of visible node IDs.
func (vc *ViewportCulling) GetVisibleNodes() []string {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	result := make([]string, 0, len(vc.visibleNodes))
	for id := range vc.visibleNodes {
		result = append(result, id)
	}
	return result
}

// GetActiveNodes returns nodes that should participate in physics.
// This includes both visible and marginal nodes.
func (vc *ViewportCulling) GetActiveNodes() []string {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	result := make([]string, 0, len(vc.visibleNodes)+len(vc.marginalNodes))
	for id := range vc.visibleNodes {
		result = append(result, id)
	}
	for id := range vc.marginalNodes {
		result = append(result, id)
	}
	return result
}

// GetBounds returns the current viewport bounds in world coordinates.
func (vc *ViewportCulling) GetBounds() (minX, minY, maxX, maxY float64) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.minX, vc.minY, vc.maxX, vc.maxY
}

// GetBoundsWithMargin returns the extended bounds including margin.
func (vc *ViewportCulling) GetBoundsWithMargin() (minX, minY, maxX, maxY float64) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.minX - vc.margin, vc.minY - vc.margin,
		vc.maxX + vc.margin, vc.maxY + vc.margin
}

// CullStats contains statistics about culling.
type CullStats struct {
	VisibleCount  int
	MarginalCount int
	CulledCount   int
	TotalCount    int
	CullRatio     float64 // Ratio of culled to total.
}

// GetStats returns culling statistics.
func (vc *ViewportCulling) GetStats() CullStats {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	total := vc.lastVisibleCount + vc.lastMarginalCount + vc.lastCulledCount
	ratio := 0.0
	if total > 0 {
		ratio = float64(vc.lastCulledCount) / float64(total)
	}

	return CullStats{
		VisibleCount:  vc.lastVisibleCount,
		MarginalCount: vc.lastMarginalCount,
		CulledCount:   vc.lastCulledCount,
		TotalCount:    total,
		CullRatio:     ratio,
	}
}

// FilterEdges returns only edges where at least one endpoint is active.
func (vc *ViewportCulling) FilterEdges(edges []Edge) []Edge {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	result := make([]Edge, 0, len(edges)/2)
	for _, e := range edges {
		srcActive := vc.visibleNodes[e.SourceID] || vc.marginalNodes[e.SourceID]
		tgtActive := vc.visibleNodes[e.TargetID] || vc.marginalNodes[e.TargetID]
		if srcActive || tgtActive {
			result = append(result, e)
		}
	}
	return result
}

// ContainsPoint checks if a world point is within the viewport.
func (vc *ViewportCulling) ContainsPoint(x, y float64) bool {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return x >= vc.minX && x <= vc.maxX && y >= vc.minY && y <= vc.maxY
}

// ContainsPointWithMargin checks if a point is within viewport + margin.
func (vc *ViewportCulling) ContainsPointWithMargin(x, y float64) bool {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	marginMinX := vc.minX - vc.margin
	marginMaxX := vc.maxX + vc.margin
	marginMinY := vc.minY - vc.margin
	marginMaxY := vc.maxY + vc.margin
	return x >= marginMinX && x <= marginMaxX && y >= marginMinY && y <= marginMaxY
}

// DistanceToViewport returns the distance from a point to the viewport edge.
// Returns 0 if the point is inside the viewport.
func (vc *ViewportCulling) DistanceToViewport(x, y float64) float64 {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	// Clamp to viewport bounds.
	clampedX := math.Max(vc.minX, math.Min(vc.maxX, x))
	clampedY := math.Max(vc.minY, math.Min(vc.maxY, y))

	// If point is inside, distance is 0.
	if x == clampedX && y == clampedY {
		return 0
	}

	// Calculate distance to nearest edge.
	dx := x - clampedX
	dy := y - clampedY
	return math.Sqrt(dx*dx + dy*dy)
}

// CulledEngine wraps Engine with viewport culling support.
// This is an optional enhancement that can be enabled for large graphs.
type CulledEngine struct {
	*Engine
	culling *ViewportCulling
	enabled bool
	mu      sync.RWMutex
}

// NewCulledEngine creates an engine with viewport culling support.
func NewCulledEngine() *CulledEngine {
	return &CulledEngine{
		Engine:  NewEngine(),
		culling: NewViewportCulling(),
		enabled: true,
	}
}

// SetCullingEnabled enables or disables viewport culling.
func (ce *CulledEngine) SetCullingEnabled(enabled bool) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.enabled = enabled
}

// IsCullingEnabled returns whether culling is enabled.
func (ce *CulledEngine) IsCullingEnabled() bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.enabled
}

// Culling returns the viewport culling manager.
func (ce *CulledEngine) Culling() *ViewportCulling {
	return ce.culling
}

// TickWithCulling performs a simulation step with viewport culling.
func (ce *CulledEngine) TickWithCulling() {
	ce.mu.RLock()
	enabled := ce.enabled
	ce.mu.RUnlock()

	if !enabled {
		ce.Engine.Tick()
		return
	}

	ce.Engine.mu.Lock()
	defer ce.Engine.mu.Unlock()

	if len(ce.Engine.nodes) == 0 {
		return
	}

	// Update visibility.
	ce.culling.UpdateVisibility(ce.Engine.positions)

	// Compute forces only for active nodes.
	forces := ce.initializeForcesFiltered()
	ce.computeRepulsionForcesFiltered(forces)
	ce.applyCenterGravityFiltered(forces)
	ce.updateNodePositionsFiltered(forces)
	ce.Engine.swapPositionBuffer()
}

// initializeForcesFiltered creates forces only for active nodes.
func (ce *CulledEngine) initializeForcesFiltered() map[string][2]float64 {
	forces := make(map[string][2]float64)
	for id := range ce.Engine.nodes {
		if ce.culling.ShouldComputeForces(id) {
			forces[id] = [2]float64{0, 0}
		}
	}
	return forces
}

// computeRepulsionForcesFiltered computes repulsion only between active nodes.
func (ce *CulledEngine) computeRepulsionForcesFiltered(forces map[string][2]float64) {
	activeCount := len(forces)
	if activeCount == 0 {
		return
	}

	// If many active nodes, use Barnes-Hut on just the active positions.
	if activeCount >= BarnesHutThreshold {
		ce.computeForcesBarnesHutFiltered(forces)
	} else {
		ce.computeForcesNaiveFiltered(forces)
	}
}

// computeForcesNaiveFiltered computes forces between active nodes only.
func (ce *CulledEngine) computeForcesNaiveFiltered(forces map[string][2]float64) {
	ids := make([]string, 0, len(forces))
	for id := range forces {
		ids = append(ids, id)
	}

	for i, id1 := range ids {
		pos1 := ce.Engine.positions[id1]
		for j := i + 1; j < len(ids); j++ {
			id2 := ids[j]
			pos2 := ce.Engine.positions[id2]

			dx := pos2.X - pos1.X
			dy := pos2.Y - pos1.Y
			distSq := dx*dx + dy*dy
			if distSq < 1 {
				distSq = 1
			}

			force := ce.Engine.params.RepulsionConstant / distSq
			dist := math.Sqrt(distSq)
			fx := force * dx / dist
			fy := force * dy / dist

			forces[id1] = [2]float64{forces[id1][0] - fx, forces[id1][1] - fy}
			forces[id2] = [2]float64{forces[id2][0] + fx, forces[id2][1] + fy}
		}
	}

	// Apply spring forces for edges with active endpoints.
	ce.applySpringForcesFiltered(forces)
}

// computeForcesBarnesHutFiltered uses Barnes-Hut on active positions.
func (ce *CulledEngine) computeForcesBarnesHutFiltered(forces map[string][2]float64) {
	// Build quadtree with only active positions.
	activePositions := make(map[string]Position, len(forces))
	for id := range forces {
		activePositions[id] = ce.Engine.positions[id]
	}

	qt := newQuadtree(activePositions, ce.Engine.centerX, ce.Engine.centerY, 2000.0)

	for id := range forces {
		pos := ce.Engine.positions[id]
		fx, fy := qt.computeForce(pos.X, pos.Y, id, ce.Engine.params.RepulsionConstant, 0.5)
		forces[id] = [2]float64{forces[id][0] + fx, forces[id][1] + fy}
	}

	ce.applySpringForcesFiltered(forces)
}

// applySpringForcesFiltered applies spring forces for edges with active nodes.
func (ce *CulledEngine) applySpringForcesFiltered(forces map[string][2]float64) {
	for _, edge := range ce.Engine.edges {
		_, src := forces[edge.SourceID]
		_, tgt := forces[edge.TargetID]

		// Skip if neither endpoint is active.
		if !src && !tgt {
			continue
		}

		pos1, ok1 := ce.Engine.positions[edge.SourceID]
		pos2, ok2 := ce.Engine.positions[edge.TargetID]
		if !ok1 || !ok2 {
			continue
		}

		dx := pos2.X - pos1.X
		dy := pos2.Y - pos1.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 1 {
			dist = 1
		}

		restLength := ce.Engine.params.SpringRestLength * math.Exp(-edge.Age/365.0)
		displacement := dist - restLength
		force := ce.Engine.params.SpringConstant * displacement
		fx := force * dx / dist
		fy := force * dy / dist

		if src {
			forces[edge.SourceID] = [2]float64{
				forces[edge.SourceID][0] + fx,
				forces[edge.SourceID][1] + fy,
			}
		}
		if tgt {
			forces[edge.TargetID] = [2]float64{
				forces[edge.TargetID][0] - fx,
				forces[edge.TargetID][1] - fy,
			}
		}
	}
}

// applyCenterGravityFiltered applies gravity to active nodes.
func (ce *CulledEngine) applyCenterGravityFiltered(forces map[string][2]float64) {
	for id := range forces {
		pos := ce.Engine.positions[id]
		dx := ce.Engine.centerX - pos.X
		dy := ce.Engine.centerY - pos.Y
		forces[id] = [2]float64{
			forces[id][0] + dx*ce.Engine.params.GravityConstant,
			forces[id][1] + dy*ce.Engine.params.GravityConstant,
		}
	}
}

// updateNodePositionsFiltered updates only active node positions.
func (ce *CulledEngine) updateNodePositionsFiltered(forces map[string][2]float64) {
	for id, f := range forces {
		pos := ce.Engine.positions[id]

		pos.VX = (pos.VX + f[0]) * ce.Engine.params.DampingCoefficient
		pos.VY = (pos.VY + f[1]) * ce.Engine.params.DampingCoefficient

		clampVelocity(&pos)

		pos.X += pos.VX
		pos.Y += pos.VY

		ce.Engine.positions[id] = pos
	}
}
