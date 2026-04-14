// Package overlays — Oracle Pool Pulse Map visualization stub.
// Per ROADMAP.md line 461: "Pulse Map visualization — swirling vortex icon at pool location".
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"time"
)

// OraclePoolState represents the current state of an Oracle Pool.
type OraclePoolState uint8

const (
	OraclePoolPending   OraclePoolState = iota // Accepting predictions.
	OraclePoolRevealing                        // Reveal phase (after deadline).
	OraclePoolResolved                         // Outcome determined.
	OraclePoolExpired                          // Pool expired without resolution.
)

// OraclePoolVisual contains rendering data for an Oracle Pool overlay (stub).
type OraclePoolVisual struct {
	PoolID          [32]byte
	X, Y            float64 // World-space position.
	State           OraclePoolState
	Question        string // Pool question.
	Deadline        time.Time
	ResolutionTime  time.Time
	PredictionCount int     // Number of predictions submitted.
	AnimationPhase  float64 // Animation time accumulator.
}

// OraclePoolOverlay manages Oracle Pool visualizations (stub).
type OraclePoolOverlay struct {
	Pools   []*OraclePoolVisual
	Visible bool
	Opacity float32
}

// NewOraclePoolOverlay creates a new Oracle Pool overlay.
func NewOraclePoolOverlay() *OraclePoolOverlay {
	return &OraclePoolOverlay{
		Pools:   make([]*OraclePoolVisual, 0),
		Visible: true,
		Opacity: 0.8,
	}
}

// AddPool adds an Oracle Pool to the overlay.
func (o *OraclePoolOverlay) AddPool(pool *OraclePoolVisual) {
	o.Pools = append(o.Pools, pool)
}

// RemovePool removes an Oracle Pool by ID.
func (o *OraclePoolOverlay) RemovePool(poolID [32]byte) {
	for i, p := range o.Pools {
		if p.PoolID == poolID {
			o.Pools = append(o.Pools[:i], o.Pools[i+1:]...)
			return
		}
	}
}

// ClearPools removes all pools.
func (o *OraclePoolOverlay) ClearPools() {
	o.Pools = o.Pools[:0]
}

// GetPool retrieves a pool by ID.
func (o *OraclePoolOverlay) GetPool(poolID [32]byte) *OraclePoolVisual {
	for _, p := range o.Pools {
		if p.PoolID == poolID {
			return p
		}
	}
	return nil
}

// Count returns the number of pools.
func (o *OraclePoolOverlay) Count() int {
	return len(o.Pools)
}

// Update advances animation phases (stub).
func (o *OraclePoolOverlay) Update(dt float64) {
	for _, p := range o.Pools {
		p.AnimationPhase += dt
	}
}

// SetOpacity sets the overlay opacity.
func (o *OraclePoolOverlay) SetOpacity(opacity float32) {
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	o.Opacity = opacity
}

// NewOraclePoolVisual creates a new Oracle Pool visual.
func NewOraclePoolVisual(poolID [32]byte, x, y float64) *OraclePoolVisual {
	return &OraclePoolVisual{
		PoolID: poolID,
		X:      x,
		Y:      y,
		State:  OraclePoolPending,
	}
}

// SetState sets the pool's visual state.
func (p *OraclePoolVisual) SetState(state OraclePoolState) {
	p.State = state
}

// SetPosition sets the pool's position.
func (p *OraclePoolVisual) SetPosition(x, y float64) {
	p.X = x
	p.Y = y
}

// SetQuestion sets the pool question.
func (p *OraclePoolVisual) SetQuestion(question string) {
	p.Question = question
}

// SetDeadline sets the prediction deadline.
func (p *OraclePoolVisual) SetDeadline(deadline time.Time) {
	p.Deadline = deadline
}

// SetResolutionTime sets when the pool resolves.
func (p *OraclePoolVisual) SetResolutionTime(t time.Time) {
	p.ResolutionTime = t
}

// SetPredictionCount sets the number of predictions.
func (p *OraclePoolVisual) SetPredictionCount(count int) {
	p.PredictionCount = count
}
