// Package waves provides Wave creation, signing, and validation.
// Per WAVES.md, there are 8 Wave types (0x01-0x08) with PoW and TTL.
package waves

import "time"

// WaveType represents the type of a Wave message.
type WaveType uint8

// Wave types per WAVES.md.
const (
	TypeSurface WaveType = 0x01 // Standard Surface Layer Wave
	TypeReply   WaveType = 0x02 // Reply to another Wave
	TypeVeiled  WaveType = 0x03 // Encrypted to specific recipients
	TypeSpecter WaveType = 0x04 // Anonymous Specter Wave
	TypeSigil   WaveType = 0x05 // Sigil update announcement
	TypeAbyssal WaveType = 0x06 // Deep anonymous content
	TypeMasked  WaveType = 0x07 // Partially revealed identity
	TypeBeacon  WaveType = 0x08 // Network coordination signal
)

// MaxContentSize is the maximum Wave content size in bytes.
const MaxContentSize = 2048

// DefaultTTL is the default Time-To-Live for Waves.
const DefaultTTL = 7 * 24 * time.Hour

// MaxTTL is the maximum allowed TTL for any Wave.
const MaxTTL = 30 * 24 * time.Hour

// TODO: Implement Wave struct and methods per PLAN.md Step 4.
