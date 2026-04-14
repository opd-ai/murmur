// Package pow provides SHA-256 Proof of Work for Wave validation.
// This file contains configurable difficulty settings per WAVES.md §PoW
// and TECHNICAL_IMPLEMENTATION.md §2.1.

package pow

import (
	"sync/atomic"
)

// DifficultyConfig holds the local node's PoW difficulty settings.
// Per WAVES.md, PoW difficulty is not globally fixed - each node maintains
// local difficulty parameters that it applies when validating incoming Waves.
type DifficultyConfig struct {
	// Standard is the difficulty for standard Waves (default: 20).
	Standard atomic.Uint32

	// Beacon is the difficulty for Beacon Waves (default: 24).
	Beacon atomic.Uint32

	// Abyssal is the difficulty for Abyssal Waves (default: 22).
	Abyssal atomic.Uint32

	// MinAcceptable is the minimum difficulty a node will accept (default: 16).
	// Waves below this difficulty are always rejected.
	MinAcceptable atomic.Uint32

	// MaxAcceptable is the maximum difficulty a node expects (default: 32).
	// Waves claiming higher difficulty are suspicious but not rejected.
	MaxAcceptable atomic.Uint32
}

// Default difficulty values per WAVES.md and TECHNICAL_IMPLEMENTATION.md.
const (
	// StandardDifficulty is the default for standard Waves (20 leading zero bits).
	StandardDifficulty uint8 = 20

	// BeaconDifficulty is the elevated difficulty for Beacon Waves (24 bits).
	BeaconDifficultyDefault uint8 = 24

	// AbyssalDifficulty is the elevated difficulty for Abyssal Waves (22 bits).
	AbyssalDifficultyDefault uint8 = 22

	// MinAcceptableDifficulty is the floor below which Waves are always rejected.
	MinAcceptableDifficulty uint8 = 16

	// MaxAcceptableDifficulty is the ceiling above which is suspicious.
	MaxAcceptableDifficulty uint8 = 32
)

// DefaultConfig returns a DifficultyConfig with default values.
func DefaultConfig() *DifficultyConfig {
	cfg := &DifficultyConfig{}
	cfg.Standard.Store(uint32(StandardDifficulty))
	cfg.Beacon.Store(uint32(BeaconDifficultyDefault))
	cfg.Abyssal.Store(uint32(AbyssalDifficultyDefault))
	cfg.MinAcceptable.Store(uint32(MinAcceptableDifficulty))
	cfg.MaxAcceptable.Store(uint32(MaxAcceptableDifficulty))
	return cfg
}

// GetStandard returns the current standard difficulty.
func (c *DifficultyConfig) GetStandard() uint8 {
	return uint8(c.Standard.Load())
}

// SetStandard sets the standard difficulty.
// Returns false if the value is outside acceptable range.
func (c *DifficultyConfig) SetStandard(d uint8) bool {
	if d < uint8(c.MinAcceptable.Load()) || d > uint8(c.MaxAcceptable.Load()) {
		return false
	}
	c.Standard.Store(uint32(d))
	return true
}

// GetBeacon returns the current Beacon difficulty.
func (c *DifficultyConfig) GetBeacon() uint8 {
	return uint8(c.Beacon.Load())
}

// SetBeacon sets the Beacon difficulty.
// Returns false if the value is outside acceptable range.
func (c *DifficultyConfig) SetBeacon(d uint8) bool {
	if d < uint8(c.MinAcceptable.Load()) || d > uint8(c.MaxAcceptable.Load()) {
		return false
	}
	c.Beacon.Store(uint32(d))
	return true
}

// GetAbyssal returns the current Abyssal difficulty.
func (c *DifficultyConfig) GetAbyssal() uint8 {
	return uint8(c.Abyssal.Load())
}

// SetAbyssal sets the Abyssal difficulty.
// Returns false if the value is outside acceptable range.
func (c *DifficultyConfig) SetAbyssal(d uint8) bool {
	if d < uint8(c.MinAcceptable.Load()) || d > uint8(c.MaxAcceptable.Load()) {
		return false
	}
	c.Abyssal.Store(uint32(d))
	return true
}

// GetMinAcceptable returns the minimum acceptable difficulty.
func (c *DifficultyConfig) GetMinAcceptable() uint8 {
	return uint8(c.MinAcceptable.Load())
}

// GetMaxAcceptable returns the maximum acceptable difficulty.
func (c *DifficultyConfig) GetMaxAcceptable() uint8 {
	return uint8(c.MaxAcceptable.Load())
}

// ValidateDifficulty checks if a given difficulty meets the local node's requirements.
// Returns true if the difficulty is within the acceptable range.
func (c *DifficultyConfig) ValidateDifficulty(d uint8) bool {
	min := uint8(c.MinAcceptable.Load())
	max := uint8(c.MaxAcceptable.Load())
	return d >= min && d <= max
}

// globalConfig is the singleton difficulty configuration for the node.
var globalConfig = DefaultConfig()

// GetGlobalConfig returns the global difficulty configuration.
// Per WAVES.md, each node maintains local difficulty parameters.
func GetGlobalConfig() *DifficultyConfig {
	return globalConfig
}

// ResetGlobalConfig resets the global configuration to defaults.
// Primarily used for testing.
func ResetGlobalConfig() {
	globalConfig = DefaultConfig()
}
