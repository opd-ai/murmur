// Package overlays provides Anonymous Layer overlay and activity heatmap.
// This file is a stub for builds without Ebitengine.
//
//go:build test
// +build test

package overlays

import (
	"github.com/opd-ai/murmur/pkg/anonymous/resonance"
)

// EchoIndexOverlay renders Echo Index color-coding on cluster boundaries.
// This is a stub for non-Ebitengine builds.
type EchoIndexOverlay struct {
	ClusterBoundaries map[string][]float32
	ClusterCenters    map[string][2]float32
	Computer          *resonance.EchoIndexComputer
	IsAnonymousLayer  bool
	BadgeRadius       float32
	ShowBadges        bool
	ShowTint          bool
	TintAlpha         uint8
}

// NewEchoIndexOverlay creates a new Echo Index overlay (stub).
func NewEchoIndexOverlay(computer *resonance.EchoIndexComputer) *EchoIndexOverlay {
	return &EchoIndexOverlay{
		ClusterBoundaries: make(map[string][]float32),
		ClusterCenters:    make(map[string][2]float32),
		Computer:          computer,
		BadgeRadius:       12.0,
		ShowBadges:        true,
		ShowTint:          true,
		TintAlpha:         40,
	}
}

// NewEchoShadowOverlay creates an Echo Shadow overlay (stub).
func NewEchoShadowOverlay(shadow *resonance.EchoShadow) *EchoIndexOverlay {
	return &EchoIndexOverlay{
		ClusterBoundaries: make(map[string][]float32),
		ClusterCenters:    make(map[string][2]float32),
		Computer:          shadow.EchoIndexComputer,
		IsAnonymousLayer:  true,
		BadgeRadius:       10.0,
		ShowBadges:        true,
		ShowTint:          true,
		TintAlpha:         30,
	}
}

// SetClusterBoundary sets the boundary polygon for a cluster (stub).
func (o *EchoIndexOverlay) SetClusterBoundary(clusterID string, vertices []float32) {
	if len(vertices) >= 6 {
		o.ClusterBoundaries[clusterID] = vertices
	}
}

// SetClusterCenter sets the center point for a cluster (stub).
func (o *EchoIndexOverlay) SetClusterCenter(clusterID string, x, y float32) {
	o.ClusterCenters[clusterID] = [2]float32{x, y}
}

// RemoveCluster removes a cluster from the overlay (stub).
func (o *EchoIndexOverlay) RemoveCluster(clusterID string) {
	delete(o.ClusterBoundaries, clusterID)
	delete(o.ClusterCenters, clusterID)
}

// Clear removes all cluster data (stub).
func (o *EchoIndexOverlay) Clear() {
	o.ClusterBoundaries = make(map[string][]float32)
	o.ClusterCenters = make(map[string][2]float32)
}

// ClusterData represents cluster information for rendering.
type ClusterData struct {
	ID        string
	EchoIndex float64
	Category  resonance.EchoCategory
	CenterX   float32
	CenterY   float32
	NodeCount int
	TotalAmps int
}

// GetClusterData returns rendering data for all tracked clusters (stub).
func (o *EchoIndexOverlay) GetClusterData() []ClusterData {
	if o.Computer == nil {
		return nil
	}

	indices := o.Computer.GetAllIndices()
	data := make([]ClusterData, 0, len(indices))

	for clusterID, idx := range indices {
		center := o.ClusterCenters[clusterID]
		data = append(data, ClusterData{
			ID:        clusterID,
			EchoIndex: idx.EchoIndex,
			Category:  idx.Category,
			CenterX:   center[0],
			CenterY:   center[1],
			TotalAmps: idx.TotalAmps,
		})
	}

	return data
}
