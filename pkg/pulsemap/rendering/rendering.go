// Package rendering provides Ebitengine-based node/edge drawing for the Pulse Map.
// Per TECHNICAL_IMPLEMENTATION.md §1.2, rendering uses Ebitengine v2.7+
// with Kage shaders for glow and ripple effects.
package rendering

// TargetFPS is the target rendering frame rate.
const TargetFPS = 60

// NOTE: This package is the only one under pkg/ that imports Ebitengine.
// Per TECHNICAL_IMPLEMENTATION.md §2, no other package should import Ebitengine.

// TODO: Implement rendering pipeline per PULSE_MAP.md.
