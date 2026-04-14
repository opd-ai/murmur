// Package overlays - Surface Spark overlay stub for non-Ebiten builds.
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"image/color"
	"time"
)

// SparkType identifies the type of Surface Spark.
type SparkType uint8

const (
	SparkWaveRelay SparkType = iota + 1
	SparkEchoRace
)

// SparkState represents the lifecycle state of a Spark.
type SparkState uint8

const (
	SparkActive SparkState = iota + 1
	SparkCompleted
	SparkExpired
	SparkCancelled
)

// SparkInfo contains information about a Spark for visualization.
type SparkInfo struct {
	ID          [32]byte
	Type        SparkType
	State       SparkState
	X, Y        float64
	Prompt      string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	WinnerKey   [32]byte
	Responses   int
	InitiatorID [32]byte
}

// CrownHolder represents a user holding a winner crown.
type CrownHolder struct {
	UserKey   [32]byte
	X, Y      float64
	ExpiresAt time.Time
}

// SparkOverlay is a stub for non-Ebiten builds.
type SparkOverlay struct {
	visible      bool
	sparks       map[[32]byte]*SparkInfo
	crownHolders map[[32]byte]*CrownHolder
}

// NewSparkOverlay creates a new stub overlay.
func NewSparkOverlay() *SparkOverlay {
	return &SparkOverlay{
		visible:      true,
		sparks:       make(map[[32]byte]*SparkInfo),
		crownHolders: make(map[[32]byte]*CrownHolder),
	}
}

// SetVisible controls visibility.
func (o *SparkOverlay) SetVisible(visible bool) { o.visible = visible }

// IsVisible returns visibility status.
func (o *SparkOverlay) IsVisible() bool { return o.visible }

// SetSpark adds or updates a spark.
func (o *SparkOverlay) SetSpark(spark *SparkInfo) {
	if spark != nil {
		o.sparks[spark.ID] = spark
	}
}

// RemoveSpark removes a spark by ID.
func (o *SparkOverlay) RemoveSpark(id [32]byte) { delete(o.sparks, id) }

// GetSpark returns a spark by ID.
func (o *SparkOverlay) GetSpark(id [32]byte) *SparkInfo { return o.sparks[id] }

// GetAllSparks returns all sparks.
func (o *SparkOverlay) GetAllSparks() []*SparkInfo {
	sparks := make([]*SparkInfo, 0, len(o.sparks))
	for _, s := range o.sparks {
		sparks = append(sparks, s)
	}
	return sparks
}

// GetActiveSparks returns only active sparks.
func (o *SparkOverlay) GetActiveSparks() []*SparkInfo {
	var active []*SparkInfo
	for _, s := range o.sparks {
		if s.State == SparkActive {
			active = append(active, s)
		}
	}
	return active
}

// SetCrownHolder adds or updates a crown holder.
func (o *SparkOverlay) SetCrownHolder(holder *CrownHolder) {
	if holder != nil {
		o.crownHolders[holder.UserKey] = holder
	}
}

// RemoveCrownHolder removes a crown holder.
func (o *SparkOverlay) RemoveCrownHolder(userKey [32]byte) { delete(o.crownHolders, userKey) }

// GetCrownHolder returns a crown holder by key.
func (o *SparkOverlay) GetCrownHolder(userKey [32]byte) *CrownHolder { return o.crownHolders[userKey] }

// HasCrown checks if a user has an active crown.
func (o *SparkOverlay) HasCrown(userKey [32]byte) bool {
	holder, ok := o.crownHolders[userKey]
	if !ok {
		return false
	}
	return time.Now().Before(holder.ExpiresAt)
}

// Update is a no-op stub.
func (o *SparkOverlay) Update(dt float64) {}

// SparkCount returns the total number of sparks.
func (o *SparkOverlay) SparkCount() int { return len(o.sparks) }

// ActiveSparkCount returns the number of active sparks.
func (o *SparkOverlay) ActiveSparkCount() int {
	count := 0
	for _, s := range o.sparks {
		if s.State == SparkActive {
			count++
		}
	}
	return count
}

// CrownCount returns the number of active crown holders.
func (o *SparkOverlay) CrownCount() int {
	now := time.Now()
	count := 0
	for _, h := range o.crownHolders {
		if now.Before(h.ExpiresAt) {
			count++
		}
	}
	return count
}

// ClearExpired removes expired sparks.
func (o *SparkOverlay) ClearExpired(maxAge time.Duration) int {
	now := time.Now()
	removed := 0
	for id, spark := range o.sparks {
		if spark.State != SparkActive {
			age := now.Sub(spark.ExpiresAt)
			if age > maxAge {
				delete(o.sparks, id)
				removed++
			}
		}
	}
	return removed
}

// UpdateCrownPosition updates the position of a crown holder.
func (o *SparkOverlay) UpdateCrownPosition(userKey [32]byte, x, y float64) {
	if holder, ok := o.crownHolders[userKey]; ok {
		holder.X = x
		holder.Y = y
	}
}

// UpdateSparkPosition updates the position of a spark.
func (o *SparkOverlay) UpdateSparkPosition(id [32]byte, x, y float64) {
	if spark, ok := o.sparks[id]; ok {
		spark.X = x
		spark.Y = y
	}
}

// SetSparkResponses updates the response count for a spark.
func (o *SparkOverlay) SetSparkResponses(id [32]byte, count int) {
	if spark, ok := o.sparks[id]; ok {
		spark.Responses = count
	}
}

// SetSparkState updates the state of a spark.
func (o *SparkOverlay) SetSparkState(id [32]byte, state SparkState) {
	if spark, ok := o.sparks[id]; ok {
		spark.State = state
	}
}

// SparkTypeString returns a human-readable name.
func SparkTypeString(st SparkType) string {
	switch st {
	case SparkWaveRelay:
		return "Wave Relay"
	case SparkEchoRace:
		return "Echo Race"
	default:
		return "Unknown"
	}
}

// SparkStateString returns a human-readable name.
func SparkStateString(ss SparkState) string {
	switch ss {
	case SparkActive:
		return "Active"
	case SparkCompleted:
		return "Completed"
	case SparkExpired:
		return "Expired"
	case SparkCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Stub types and functions for non-Ebiten builds.
type color_RGBA = color.RGBA
