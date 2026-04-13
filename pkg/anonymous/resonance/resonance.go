// Package resonance provides local reputation computation and rank thresholds.
// Per RESONANCE_SYSTEM.md, Resonance milestones unlock at 25/50/75/100/200/500.
package resonance

// Milestones per RESONANCE_SYSTEM.md.
const (
	MilestoneShade       = 25
	MilestoneWraith      = 50
	MilestoneShadeWraith = 75
	MilestonePhantom     = 100
	MilestoneCouncil     = 200
	MilestoneAbyss       = 500
)

// TODO: Implement Resonance computation per RESONANCE_SYSTEM.md.
