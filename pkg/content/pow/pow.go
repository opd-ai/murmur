// Package pow provides SHA-256 Proof of Work for Wave validation.
// Per TECHNICAL_IMPLEMENTATION.md §2.1, PoW targets 2-5 seconds
// computation time with difficulty 20 (leading zero bits).
package pow

import (
	"crypto/sha256"
	"encoding/binary"
	"math/bits"
)

// DefaultDifficulty is the number of leading zero bits required.
const DefaultDifficulty = 20

// MinComputeTime is the minimum target computation time.
const MinComputeTime = 2 // seconds

// MaxComputeTime is the maximum target computation time.
const MaxComputeTime = 5 // seconds

// MaxNonce is the maximum nonce value before giving up.
const MaxNonce = ^uint64(0)

// Work represents a proof of work result.
type Work struct {
	Nonce      uint64
	Hash       [32]byte
	Difficulty uint8
}

// Compute performs SHA-256 PoW on the given data with the specified difficulty.
// Returns the nonce that satisfies the difficulty requirement.
// Difficulty is the number of leading zero bits required in the hash.
func Compute(data []byte, difficulty uint8) (*Work, error) {
	input := make([]byte, len(data)+8)
	copy(input, data)

	for nonce := uint64(0); nonce < MaxNonce; nonce++ {
		binary.BigEndian.PutUint64(input[len(data):], nonce)

		hash := sha256.Sum256(input)

		if checkDifficulty(hash[:], difficulty) {
			return &Work{
				Nonce:      nonce,
				Hash:       hash,
				Difficulty: difficulty,
			}, nil
		}
	}

	return nil, ErrMaxNonceReached
}

// Verify checks if the given nonce produces a valid hash for the data.
func Verify(data []byte, nonce uint64, difficulty uint8) bool {
	input := make([]byte, len(data)+8)
	copy(input, data)
	binary.BigEndian.PutUint64(input[len(data):], nonce)

	hash := sha256.Sum256(input)
	return checkDifficulty(hash[:], difficulty)
}

// VerifyWork verifies a Work result against the original data.
func VerifyWork(data []byte, work *Work) bool {
	if work == nil {
		return false
	}
	return Verify(data, work.Nonce, work.Difficulty)
}

// checkDifficulty checks if a hash has at least the required leading zero bits.
func checkDifficulty(hash []byte, difficulty uint8) bool {
	if difficulty == 0 {
		return true
	}

	zerosNeeded := int(difficulty)
	zerosFound := 0

	for _, b := range hash {
		if b == 0 {
			zerosFound += 8
			if zerosFound >= zerosNeeded {
				return true
			}
		} else {
			zerosFound += bits.LeadingZeros8(b)
			return zerosFound >= zerosNeeded
		}
	}

	return zerosFound >= zerosNeeded
}

// LeadingZeros counts the number of leading zero bits in a hash.
func LeadingZeros(hash []byte) int {
	zeros := 0
	for _, b := range hash {
		if b == 0 {
			zeros += 8
		} else {
			zeros += bits.LeadingZeros8(b)
			break
		}
	}
	return zeros
}

// Error types for PoW operations.
type Error string

func (e Error) Error() string { return string(e) }

const (
	// ErrMaxNonceReached is returned when the maximum nonce is reached.
	ErrMaxNonceReached Error = "maximum nonce reached without finding valid proof"
)
