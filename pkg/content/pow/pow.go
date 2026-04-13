// Package pow provides SHA-256 Proof of Work for Wave validation.
// Per TECHNICAL_IMPLEMENTATION.md §2.1, PoW targets 2-5 seconds
// computation time with difficulty 20 (leading zero bits).
package pow

// DefaultDifficulty is the number of leading zero bits required.
const DefaultDifficulty = 20

// MinComputeTime is the minimum target computation time.
const MinComputeTime = 2 // seconds

// MaxComputeTime is the maximum target computation time.
const MaxComputeTime = 5 // seconds

// TODO: Implement PoW computation per PLAN.md Step 4.
