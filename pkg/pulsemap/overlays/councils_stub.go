// Package overlays — Phantom Council visualization stub for non-Ebiten builds.
//
//go:build noebiten
// +build noebiten

package overlays

import "image/color"

// CouncilMemberInfo contains information about a council member for visualization.
type CouncilMemberInfo struct {
	SpecterKey      [32]byte
	X, Y            float64
	IsCommunicating bool
}

// CouncilInfo contains information about a Phantom Council for visualization.
type CouncilInfo struct {
	ID             [32]byte
	Name           string
	Members        []CouncilMemberInfo
	Color          color.RGBA
	IsActive       bool
	AnimationPhase float64
}

// CouncilOverlay is a stub for non-Ebiten builds.
type CouncilOverlay struct {
	Visible bool
	Opacity float32
}

// NewCouncilOverlay creates a stub overlay.
func NewCouncilOverlay() *CouncilOverlay {
	return &CouncilOverlay{Visible: true, Opacity: 0.8}
}

// AddCouncil is a no-op stub.
func (co *CouncilOverlay) AddCouncil(info *CouncilInfo) {}

// RemoveCouncil is a no-op stub.
func (co *CouncilOverlay) RemoveCouncil(councilID [32]byte) {}

// UpdateCouncil is a no-op stub.
func (co *CouncilOverlay) UpdateCouncil(info *CouncilInfo) {}

// Update is a no-op stub.
func (co *CouncilOverlay) Update(dt float64) {}

// GetCouncil returns nil in stub.
func (co *CouncilOverlay) GetCouncil(councilID [32]byte) *CouncilInfo { return nil }

// CouncilCount returns 0 in stub.
func (co *CouncilOverlay) CouncilCount() int { return 0 }

// Clear is a no-op stub.
func (co *CouncilOverlay) Clear() {}

// SetVisible sets visibility in stub.
func (co *CouncilOverlay) SetVisible(visible bool) { co.Visible = visible }

// SetOpacity sets opacity in stub.
func (co *CouncilOverlay) SetOpacity(opacity float32) { co.Opacity = opacity }

// SetCouncilActive is a no-op stub.
func (co *CouncilOverlay) SetCouncilActive(councilID [32]byte, active bool) {}

// SetMemberCommunicating is a no-op stub.
func (co *CouncilOverlay) SetMemberCommunicating(councilID, specterKey [32]byte, communicating bool) {
}

// UpdateMemberPosition is a no-op stub.
func (co *CouncilOverlay) UpdateMemberPosition(councilID, specterKey [32]byte, x, y float64) {}

// Tick is a no-op stub.
func (co *CouncilOverlay) Tick() {}

// GenerateCouncilColor generates a council color from ID.
func GenerateCouncilColor(councilID [32]byte) color.RGBA {
	hue := 200 + float64(councilID[0]%80)
	r, g, b := stubHSVtoRGB(hue, 0.7, 0.8)
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}

func stubHSVtoRGB(h, s, v float64) (r, g, b float64) {
	// Simplified HSV conversion.
	c := v * s
	x := c * (1 - abs(mod(h/60, 2)-1))
	m := v - c

	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return r + m, g + m, b + m
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func mod(a, b float64) float64 {
	for a >= b {
		a -= b
	}
	for a < 0 {
		a += b
	}
	return a
}

// NewCouncilInfo creates a new CouncilInfo.
func NewCouncilInfo(id [32]byte, name string) *CouncilInfo {
	return &CouncilInfo{
		ID:      id,
		Name:    name,
		Members: make([]CouncilMemberInfo, 0),
		Color:   GenerateCouncilColor(id),
	}
}

// AddMember adds a member to the council info.
func (ci *CouncilInfo) AddMember(specterKey [32]byte, x, y float64) {
	ci.Members = append(ci.Members, CouncilMemberInfo{
		SpecterKey: specterKey,
		X:          x,
		Y:          y,
	})
}
