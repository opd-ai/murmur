// Package overlays — Territory Drift Pulse Map visualization stub.
// Per ROADMAP.md line 446: "Pulse Map visualization — translucent watermarks
// with territory boundaries".
//
//go:build test
// +build test

package overlays

// TerritoryState mirrors the state from mechanics/territory.go.
type TerritoryState uint8

const (
	TerritoryNeutral    TerritoryState = iota // No dominant controller.
	TerritoryControlled                       // Single controller.
	TerritoryContested                        // Multiple contenders.
)

// TerritoryVisual contains rendering data for a territory overlay (stub).
type TerritoryVisual struct {
	ID             string
	CentroidX      float64 // World-space center X.
	CentroidY      float64 // World-space center Y.
	Boundary       []Point // Convex hull vertices in world-space.
	State          TerritoryState
	Influence      float64 // Controller's influence for opacity scaling.
	AnimationPhase float64 // Animation time accumulator.
}

// Point is a 2D coordinate.
type Point struct {
	X, Y float64
}

// TerritoryOverlay renders territory visuals on the Pulse Map (stub).
type TerritoryOverlay struct {
	Territories []*TerritoryVisual
	Visible     bool
	Opacity     float32 // Global opacity multiplier (0-1).
}

// NewTerritoryOverlay creates a new territory overlay.
func NewTerritoryOverlay() *TerritoryOverlay {
	return &TerritoryOverlay{
		Territories: make([]*TerritoryVisual, 0),
		Visible:     true,
		Opacity:     0.6,
	}
}

// SetOpacity sets the global opacity for all territory visuals.
func (o *TerritoryOverlay) SetOpacity(opacity float32) {
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	o.Opacity = opacity
}

// AddTerritory adds a territory visual to the overlay.
func (o *TerritoryOverlay) AddTerritory(t *TerritoryVisual) {
	o.Territories = append(o.Territories, t)
}

// RemoveTerritory removes a territory by ID.
func (o *TerritoryOverlay) RemoveTerritory(id string) {
	for i, t := range o.Territories {
		if t.ID == id {
			o.Territories = append(o.Territories[:i], o.Territories[i+1:]...)
			return
		}
	}
}

// ClearTerritories removes all territories.
func (o *TerritoryOverlay) ClearTerritories() {
	o.Territories = o.Territories[:0]
}

// GetTerritory retrieves a territory by ID.
func (o *TerritoryOverlay) GetTerritory(id string) *TerritoryVisual {
	for _, t := range o.Territories {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// Update advances animation phases for all territories (stub).
func (o *TerritoryOverlay) Update(dt float64) {
	for _, t := range o.Territories {
		t.AnimationPhase += dt
	}
}

// Count returns the number of territories in the overlay.
func (o *TerritoryOverlay) Count() int {
	return len(o.Territories)
}

// TerritoryBoundaryCircle returns points for a circular territory boundary.
func TerritoryBoundaryCircle(cx, cy, radius float64, segments int) []Point {
	if segments < 3 {
		segments = 12
	}
	points := make([]Point, segments)
	for i := 0; i < segments; i++ {
		angle := 2 * 3.14159265359 * float64(i) / float64(segments)
		points[i] = Point{
			X: cx + radius*cosApprox(angle),
			Y: cy + radius*sinApprox(angle),
		}
	}
	return points
}

// cosApprox is a simple cosine approximation for stub use.
func cosApprox(angle float64) float64 {
	// Taylor series approximation.
	x := normalizeAngle(angle)
	x2 := x * x
	return 1 - x2/2 + x2*x2/24
}

// sinApprox is a simple sine approximation for stub use.
func sinApprox(angle float64) float64 {
	x := normalizeAngle(angle)
	x3 := x * x * x
	return x - x3/6
}

// normalizeAngle wraps angle to [-pi, pi].
func normalizeAngle(angle float64) float64 {
	const pi = 3.14159265359
	for angle > pi {
		angle -= 2 * pi
	}
	for angle < -pi {
		angle += 2 * pi
	}
	return angle
}

// NewTerritoryVisual creates a new territory visual.
func NewTerritoryVisual(id string, centroidX, centroidY float64) *TerritoryVisual {
	return &TerritoryVisual{
		ID:        id,
		CentroidX: centroidX,
		CentroidY: centroidY,
		State:     TerritoryNeutral,
	}
}

// SetBoundary sets the territory boundary polygon.
func (t *TerritoryVisual) SetBoundary(boundary []Point) {
	t.Boundary = boundary
}

// SetState sets the territory's visual state.
func (t *TerritoryVisual) SetState(state TerritoryState) {
	t.State = state
}

// SetInfluence sets the influence level for opacity scaling.
func (t *TerritoryVisual) SetInfluence(influence float64) {
	t.Influence = influence
}
