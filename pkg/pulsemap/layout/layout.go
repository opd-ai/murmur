// Package layout provides the force-directed graph engine for the Pulse Map.
// Per PULSE_MAP.md, the layout uses Fruchterman-Reingold with Barnes-Hut
// optimization for graphs exceeding 500 nodes.
package layout

import (
	"math"
	"sync"
	"sync/atomic"
)

// BarnesHutThreshold is the node count above which Barnes-Hut is used.
const BarnesHutThreshold = 500

// DefaultTicksPerSecond is the simulation tick rate per PULSE_MAP.md.
const DefaultTicksPerSecond = 30

// Position represents a 2D position with velocity.
type Position struct {
	X, Y   float64
	VX, VY float64
}

// LayoutParams contains tunable force parameters per PULSE_MAP.md.
type LayoutParams struct {
	RepulsionConstant  float64 // Inverse-square repulsion strength
	SpringConstant     float64 // Spring attraction strength
	SpringRestLength   float64 // Base spring rest length
	GravityConstant    float64 // Center gravity strength
	DampingCoefficient float64 // Velocity damping (0-1)
	TicksPerSecond     int     // Simulation update rate
}

// DefaultParams returns default layout parameters tuned per PULSE_MAP.md.
func DefaultParams() LayoutParams {
	return LayoutParams{
		RepulsionConstant:  10000.0,
		SpringConstant:     0.01,
		SpringRestLength:   100.0,
		GravityConstant:    0.01,
		DampingCoefficient: 0.85,
		TicksPerSecond:     DefaultTicksPerSecond,
	}
}

// Node represents a node in the layout graph.
type Node struct {
	ID          string
	Connections int     // Connection count for sizing
	Activity    float64 // Activity metric for sizing
}

// Edge represents a connection between two nodes.
type Edge struct {
	SourceID string
	TargetID string
	Age      float64 // Connection age in days (affects rest length)
}

// PositionBuffer holds double-buffered node positions for lock-free reads.
// Per TECHNICAL_IMPLEMENTATION.md §8, uses atomic.Pointer for swap.
type PositionBuffer struct {
	positions atomic.Pointer[map[string]Position]
}

// NewPositionBuffer creates a new double-buffered position storage.
func NewPositionBuffer() *PositionBuffer {
	pb := &PositionBuffer{}
	empty := make(map[string]Position)
	pb.positions.Store(&empty)
	return pb
}

// Get returns the current position map (read-only, lock-free).
func (pb *PositionBuffer) Get() map[string]Position {
	return *pb.positions.Load()
}

// Swap atomically swaps the position buffer with new positions.
func (pb *PositionBuffer) Swap(newPositions map[string]Position) {
	pb.positions.Store(&newPositions)
}

// Engine is the force-directed layout engine.
type Engine struct {
	mu          sync.RWMutex
	nodes       map[string]*Node
	edges       []Edge
	positions   map[string]Position
	params      LayoutParams
	frontBuffer *PositionBuffer
	running     bool
	stopCh      chan struct{}
	centerX     float64
	centerY     float64
}

// NewEngine creates a new layout engine with default parameters.
func NewEngine() *Engine {
	return &Engine{
		nodes:       make(map[string]*Node),
		edges:       make([]Edge, 0),
		positions:   make(map[string]Position),
		params:      DefaultParams(),
		frontBuffer: NewPositionBuffer(),
		centerX:     400.0,
		centerY:     300.0,
	}
}

// SetParams updates the layout parameters.
func (e *Engine) SetParams(params LayoutParams) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.params = params
}

// SetCenter sets the center point for gravity.
func (e *Engine) SetCenter(x, y float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.centerX = x
	e.centerY = y
}

// AddNode adds a node to the layout.
func (e *Engine) AddNode(node *Node) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nodes[node.ID] = node
	// Initialize position near center with small random offset
	e.positions[node.ID] = Position{
		X: e.centerX + float64(int(hash(node.ID))%200-100),
		Y: e.centerY + float64(int(hash(node.ID+"y"))%200-100),
	}
}

// RemoveNode removes a node from the layout.
func (e *Engine) RemoveNode(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.nodes, id)
	delete(e.positions, id)
	// Remove edges involving this node
	newEdges := make([]Edge, 0, len(e.edges))
	for _, edge := range e.edges {
		if edge.SourceID != id && edge.TargetID != id {
			newEdges = append(newEdges, edge)
		}
	}
	e.edges = newEdges
}

// AddEdge adds an edge to the layout.
func (e *Engine) AddEdge(edge Edge) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.edges = append(e.edges, edge)
}

// Positions returns the current position buffer for rendering.
func (e *Engine) Positions() *PositionBuffer {
	return e.frontBuffer
}

// NodeCount returns the current number of nodes.
func (e *Engine) NodeCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.nodes)
}

// Tick performs a single simulation step.
func (e *Engine) Tick() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.nodes) == 0 {
		return
	}

	forces := e.initializeForces()
	e.computeRepulsionForces(forces)
	e.applyCenterGravity(forces)
	e.updateNodePositions(forces)
	e.swapPositionBuffer()
}

// initializeForces creates and returns a zeroed force map for all nodes.
func (e *Engine) initializeForces() map[string][2]float64 {
	forces := make(map[string][2]float64)
	for id := range e.nodes {
		forces[id] = [2]float64{0, 0}
	}
	return forces
}

// computeRepulsionForces calculates repulsion using appropriate algorithm.
func (e *Engine) computeRepulsionForces(forces map[string][2]float64) {
	if len(e.nodes) >= BarnesHutThreshold {
		e.computeForcesBarnesHut(forces)
	} else {
		e.computeForcesNaive(forces)
	}
}

// applyCenterGravity applies gravitational pull toward center.
func (e *Engine) applyCenterGravity(forces map[string][2]float64) {
	for id := range e.nodes {
		pos := e.positions[id]
		dx := e.centerX - pos.X
		dy := e.centerY - pos.Y
		forces[id] = [2]float64{
			forces[id][0] + dx*e.params.GravityConstant,
			forces[id][1] + dy*e.params.GravityConstant,
		}
	}
}

// updateNodePositions updates velocities and positions based on forces.
func (e *Engine) updateNodePositions(forces map[string][2]float64) {
	for id := range e.nodes {
		pos := e.positions[id]
		f := forces[id]

		pos.VX = (pos.VX + f[0]) * e.params.DampingCoefficient
		pos.VY = (pos.VY + f[1]) * e.params.DampingCoefficient

		clampVelocity(&pos)

		pos.X += pos.VX
		pos.Y += pos.VY

		e.positions[id] = pos
	}
}

// clampVelocity limits velocity magnitude to prevent instability.
func clampVelocity(pos *Position) {
	const maxVel = 50.0
	speed := math.Sqrt(pos.VX*pos.VX + pos.VY*pos.VY)
	if speed > maxVel {
		pos.VX = pos.VX / speed * maxVel
		pos.VY = pos.VY / speed * maxVel
	}
}

// swapPositionBuffer copies positions to the front buffer atomically.
func (e *Engine) swapPositionBuffer() {
	newPositions := make(map[string]Position, len(e.positions))
	for id, pos := range e.positions {
		newPositions[id] = pos
	}
	e.frontBuffer.Swap(newPositions)
}

// computeForcesNaive computes forces using O(n²) pairwise algorithm.
func (e *Engine) computeForcesNaive(forces map[string][2]float64) {
	ids := make([]string, 0, len(e.nodes))
	for id := range e.nodes {
		ids = append(ids, id)
	}

	// Repulsion between all node pairs
	for i, id1 := range ids {
		pos1 := e.positions[id1]
		for j := i + 1; j < len(ids); j++ {
			id2 := ids[j]
			pos2 := e.positions[id2]

			dx := pos2.X - pos1.X
			dy := pos2.Y - pos1.Y
			distSq := dx*dx + dy*dy
			if distSq < 1 {
				distSq = 1 // Prevent division by zero
			}

			// Inverse-square repulsion
			force := e.params.RepulsionConstant / distSq
			dist := math.Sqrt(distSq)
			fx := force * dx / dist
			fy := force * dy / dist

			forces[id1] = [2]float64{forces[id1][0] - fx, forces[id1][1] - fy}
			forces[id2] = [2]float64{forces[id2][0] + fx, forces[id2][1] + fy}
		}
	}

	// Spring attraction for edges
	e.applySpringForces(forces)
}

// computeForcesBarnesHut uses Barnes-Hut algorithm for O(n log n) performance.
func (e *Engine) computeForcesBarnesHut(forces map[string][2]float64) {
	// Build quadtree
	qt := newQuadtree(e.positions, e.centerX, e.centerY, 2000.0)

	// Compute repulsion using quadtree
	for id := range e.nodes {
		pos := e.positions[id]
		fx, fy := qt.computeForce(pos.X, pos.Y, id, e.params.RepulsionConstant, 0.5)
		forces[id] = [2]float64{forces[id][0] + fx, forces[id][1] + fy}
	}

	// Spring attraction (still O(E))
	e.applySpringForces(forces)
}

// applySpringForces computes spring attraction forces for all edges using Hooke's law.
// Spring rest length decreases with connection age per PULSE_MAP.md.
func (e *Engine) applySpringForces(forces map[string][2]float64) {
	for _, edge := range e.edges {
		pos1, ok1 := e.positions[edge.SourceID]
		pos2, ok2 := e.positions[edge.TargetID]
		if !ok1 || !ok2 {
			continue
		}

		dx := pos2.X - pos1.X
		dy := pos2.Y - pos1.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 1 {
			dist = 1
		}

		// Spring rest length decreases with connection age
		restLength := e.params.SpringRestLength * math.Exp(-edge.Age/365.0)
		displacement := dist - restLength

		// Hooke's law
		force := e.params.SpringConstant * displacement
		fx := force * dx / dist
		fy := force * dy / dist

		forces[edge.SourceID] = [2]float64{
			forces[edge.SourceID][0] + fx,
			forces[edge.SourceID][1] + fy,
		}
		forces[edge.TargetID] = [2]float64{
			forces[edge.TargetID][0] - fx,
			forces[edge.TargetID][1] - fy,
		}
	}
}

// hash returns a deterministic hash value for positioning.
func hash(s string) float64 {
	var h uint64
	for _, c := range s {
		h = h*31 + uint64(c)
	}
	return float64(h % 1000000)
}

// quadtree implements Barnes-Hut spatial partitioning.
type quadtree struct {
	centerX, centerY float64
	size             float64
	mass             float64
	comX, comY       float64 // Center of mass
	nodeID           string  // If leaf with single node
	children         [4]*quadtree
	isLeaf           bool
	depth            int
}

const maxQuadtreeDepth = 20

func newQuadtree(positions map[string]Position, cx, cy, size float64) *quadtree {
	qt := &quadtree{
		centerX: cx,
		centerY: cy,
		size:    size,
		isLeaf:  true,
		depth:   0,
	}

	for id, pos := range positions {
		qt.insert(id, pos.X, pos.Y)
	}

	return qt
}

func (qt *quadtree) insert(id string, x, y float64) {
	// Check if point is within bounds
	halfSize := qt.size / 2
	if x < qt.centerX-halfSize || x > qt.centerX+halfSize ||
		y < qt.centerY-halfSize || y > qt.centerY+halfSize {
		return
	}

	if qt.mass == 0 {
		// Empty node, add point
		qt.mass = 1
		qt.comX = x
		qt.comY = y
		qt.nodeID = id
		qt.isLeaf = true
		return
	}

	// If at max depth, just accumulate mass without subdividing
	if qt.depth >= maxQuadtreeDepth {
		totalMass := qt.mass + 1
		qt.comX = (qt.comX*qt.mass + x) / totalMass
		qt.comY = (qt.comY*qt.mass + y) / totalMass
		qt.mass = totalMass
		return
	}

	if qt.isLeaf {
		// Split into quadrants
		qt.subdivide()
		// Re-insert existing point
		qt.insertIntoChild(qt.nodeID, qt.comX, qt.comY)
		qt.nodeID = ""
	}

	// Insert new point
	qt.insertIntoChild(id, x, y)

	// Update center of mass
	totalMass := qt.mass + 1
	qt.comX = (qt.comX*qt.mass + x) / totalMass
	qt.comY = (qt.comY*qt.mass + y) / totalMass
	qt.mass = totalMass
}

func (qt *quadtree) subdivide() {
	halfSize := qt.size / 2
	quarterSize := qt.size / 4
	childDepth := qt.depth + 1

	qt.children[0] = &quadtree{
		centerX: qt.centerX - quarterSize,
		centerY: qt.centerY - quarterSize,
		size:    halfSize,
		isLeaf:  true,
		depth:   childDepth,
	}
	qt.children[1] = &quadtree{
		centerX: qt.centerX + quarterSize,
		centerY: qt.centerY - quarterSize,
		size:    halfSize,
		isLeaf:  true,
		depth:   childDepth,
	}
	qt.children[2] = &quadtree{
		centerX: qt.centerX - quarterSize,
		centerY: qt.centerY + quarterSize,
		size:    halfSize,
		isLeaf:  true,
		depth:   childDepth,
	}
	qt.children[3] = &quadtree{
		centerX: qt.centerX + quarterSize,
		centerY: qt.centerY + quarterSize,
		size:    halfSize,
		isLeaf:  true,
		depth:   childDepth,
	}
	qt.isLeaf = false
}

func (qt *quadtree) insertIntoChild(id string, x, y float64) {
	// Determine which quadrant the point belongs to
	idx := 0
	if x >= qt.centerX {
		idx += 1
	}
	if y >= qt.centerY {
		idx += 2
	}
	if qt.children[idx] != nil {
		qt.children[idx].insert(id, x, y)
	}
}

func (qt *quadtree) computeForce(x, y float64, excludeID string, k, theta float64) (fx, fy float64) {
	if qt.mass == 0 {
		return 0, 0
	}

	dx, dy, dist := qt.computeDistance(x, y)

	if qt.shouldSkipSelf(excludeID) {
		return 0, 0
	}

	if qt.canApproximateAsPoint(dist, theta) {
		return qt.computePointForce(dx, dy, dist, k)
	}

	return qt.aggregateChildForces(x, y, excludeID, k, theta)
}

// computeDistance calculates displacement and distance from point (x,y) to center of mass.
func (qt *quadtree) computeDistance(x, y float64) (dx, dy, dist float64) {
	dx = qt.comX - x
	dy = qt.comY - y
	distSq := dx*dx + dy*dy
	if distSq < 1 {
		distSq = 1
	}
	return dx, dy, math.Sqrt(distSq)
}

// shouldSkipSelf returns true if this is a leaf node matching the excluded ID.
func (qt *quadtree) shouldSkipSelf(excludeID string) bool {
	return qt.isLeaf && qt.nodeID == excludeID
}

// canApproximateAsPoint applies Barnes-Hut criterion to determine if
// this quadtree region can be treated as a single point mass.
func (qt *quadtree) canApproximateAsPoint(dist, theta float64) bool {
	return qt.isLeaf || qt.size/dist < theta
}

// computePointForce calculates the repulsive force treating this region as a point mass.
func (qt *quadtree) computePointForce(dx, dy, dist, k float64) (fx, fy float64) {
	distSq := dist * dist
	force := k * qt.mass / distSq
	return -force * dx / dist, -force * dy / dist
}

// aggregateChildForces sums forces from all non-nil children.
func (qt *quadtree) aggregateChildForces(x, y float64, excludeID string, k, theta float64) (fx, fy float64) {
	for _, child := range qt.children {
		if child != nil {
			cfx, cfy := child.computeForce(x, y, excludeID, k, theta)
			fx += cfx
			fy += cfy
		}
	}
	return fx, fy
}
