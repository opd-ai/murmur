package rotation

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

var (
	// ErrInvalidOldKey indicates old public key is invalid (not 32 bytes).
	ErrInvalidOldKey = errors.New("rotation: invalid old public key")
	// ErrInvalidNewKey indicates new public key is invalid (not 32 bytes).
	ErrInvalidNewKey = errors.New("rotation: invalid new public key")
	// ErrMissingOldPrivateKey indicates old private key required for rotation.
	ErrMissingOldPrivateKey = errors.New("rotation: missing old private key")
	// ErrMissingNewPrivateKey indicates new private key required for rotation.
	ErrMissingNewPrivateKey = errors.New("rotation: missing new private key")
	// ErrGracePeriodInvalid indicates grace period out of range (must be 1-14 days).
	ErrGracePeriodInvalid = errors.New("rotation: grace period must be 1-14 days")
)

const (
	// DefaultGracePeriodDays is the standard grace period for key rotation (7 days).
	// Per KEY_ROTATION.md §Security Analysis, 7 days ensures 99% peer propagation.
	DefaultGracePeriodDays = 7

	// MinGracePeriodDays is the minimum grace period (urgent rotation, confirmed breach).
	MinGracePeriodDays = 1

	// MaxGracePeriodDays is the maximum grace period (paranoid rotation, double-check).
	MaxGracePeriodDays = 14

	// Ed25519PublicKeySize is the size of Ed25519 public keys in bytes.
	Ed25519PublicKeySize = 32

	// Ed25519PrivateKeySize is the size of Ed25519 private keys in bytes.
	Ed25519PrivateKeySize = 64
)

// RotateOptions configures key rotation behavior.
type RotateOptions struct {
	// GracePeriodDays specifies how long old key remains valid (1-14 days, default 7).
	GracePeriodDays int64
	// Reason is an optional human-readable rotation reason (max 256 bytes).
	Reason string
}

// DefaultRotateOptions returns RotateOptions with 7-day grace period.
func DefaultRotateOptions() *RotateOptions {
	return &RotateOptions{
		GracePeriodDays: DefaultGracePeriodDays,
		Reason:          "Proactive rotation",
	}
}

// CreateRotation generates a ContinuityDeclaration for key rotation.
// oldPrivateKey and newPrivateKey must be Ed25519 private keys (64 bytes each).
// Returns signed declaration ready for GossipSub broadcast.
func CreateRotation(
	oldPrivateKey ed25519.PrivateKey,
	newPrivateKey ed25519.PrivateKey,
	opts *RotateOptions,
) (*pb.ContinuityDeclaration, error) {
	// Validate inputs
	if len(oldPrivateKey) != Ed25519PrivateKeySize {
		return nil, ErrMissingOldPrivateKey
	}
	if len(newPrivateKey) != Ed25519PrivateKeySize {
		return nil, ErrMissingNewPrivateKey
	}

	if opts == nil {
		opts = DefaultRotateOptions()
	}

	if opts.GracePeriodDays < MinGracePeriodDays || opts.GracePeriodDays > MaxGracePeriodDays {
		return nil, ErrGracePeriodInvalid
	}

	// Truncate reason to 256 bytes
	reason := opts.Reason
	if len(reason) > 256 {
		reason = reason[:256]
	}

	// Extract public keys
	oldPublicKey := oldPrivateKey.Public().(ed25519.PublicKey)
	newPublicKey := newPrivateKey.Public().(ed25519.PublicKey)

	// Current timestamp
	now := time.Now().Unix()

	// Construct declaration (without signatures yet)
	decl := &pb.ContinuityDeclaration{
		OldPublicKey:          oldPublicKey,
		NewPublicKey:          newPublicKey,
		RotationTimestampUnix: now,
		GracePeriodDays:       opts.GracePeriodDays,
		RotationReason:        reason,
	}

	// Generate signature data (per KEY_ROTATION.md §2 Protobuf Specification)
	sigData := buildSignatureData(decl)

	// Sign with old key (proves old key holder authorizes rotation)
	decl.OldKeySignature = ed25519.Sign(oldPrivateKey, sigData)

	// Sign with new key (proves new key holder participated)
	decl.NewKeySignature = ed25519.Sign(newPrivateKey, sigData)

	return decl, nil
}

// buildSignatureData constructs the canonical byte sequence for signing.
// Per KEY_ROTATION.md §2, signature covers:
// old_public_key || new_public_key || rotation_timestamp_unix || grace_period_days || rotation_reason
func buildSignatureData(decl *pb.ContinuityDeclaration) []byte {
	// Allocate buffer: 32 + 32 + 8 + 8 + len(reason)
	buf := make([]byte, 0, 80+len(decl.RotationReason))

	// Append fields in order
	buf = append(buf, decl.OldPublicKey...)
	buf = append(buf, decl.NewPublicKey...)

	// Encode int64 as big-endian 8 bytes
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], uint64(decl.RotationTimestampUnix))
	buf = append(buf, ts[:]...)

	var gp [8]byte
	binary.BigEndian.PutUint64(gp[:], uint64(decl.GracePeriodDays))
	buf = append(buf, gp[:]...)

	buf = append(buf, []byte(decl.RotationReason)...)

	return buf
}

// ValidateDeclaration verifies cryptographic signatures on a continuity declaration.
// Returns nil if both old_key_signature and new_key_signature verify correctly.
func ValidateDeclaration(decl *pb.ContinuityDeclaration) error {
	if decl == nil {
		return errors.New("rotation: nil declaration")
	}

	// Validate key sizes
	if len(decl.OldPublicKey) != Ed25519PublicKeySize {
		return ErrInvalidOldKey
	}
	if len(decl.NewPublicKey) != Ed25519PublicKeySize {
		return ErrInvalidNewKey
	}

	// Validate signatures exist
	if len(decl.OldKeySignature) != ed25519.SignatureSize {
		return errors.New("rotation: invalid old key signature")
	}
	if len(decl.NewKeySignature) != ed25519.SignatureSize {
		return errors.New("rotation: invalid new key signature")
	}

	// Validate grace period
	if decl.GracePeriodDays < MinGracePeriodDays || decl.GracePeriodDays > MaxGracePeriodDays {
		return ErrGracePeriodInvalid
	}

	// Validate timestamp (must be within ±300 seconds, per Wave validation)
	now := time.Now().Unix()
	if decl.RotationTimestampUnix < now-300 || decl.RotationTimestampUnix > now+300 {
		return fmt.Errorf("rotation: timestamp out of range (now=%d, decl=%d)", now, decl.RotationTimestampUnix)
	}

	// Reconstruct signature data
	sigData := buildSignatureData(decl)

	// Verify old key signature
	if !ed25519.Verify(decl.OldPublicKey, sigData, decl.OldKeySignature) {
		return errors.New("rotation: old key signature verification failed")
	}

	// Verify new key signature
	if !ed25519.Verify(decl.NewPublicKey, sigData, decl.NewKeySignature) {
		return errors.New("rotation: new key signature verification failed")
	}

	return nil
}

// IsKeyValidForTimestamp checks if a signing key is valid for a given timestamp.
// key may be either old (within grace period) or new (current active key).
// chain is the continuity chain for the identity.
// Returns true if key is valid at the given timestamp.
func IsKeyValidForTimestamp(key []byte, timestamp int64, chain *pb.ContinuityChain) bool {
	if chain == nil || len(chain.Declarations) == 0 {
		// No rotation history; key must match identity root
		return len(key) == Ed25519PublicKeySize && len(chain.IdentityRootKey) == Ed25519PublicKeySize &&
			bytesEqual(key, chain.IdentityRootKey)
	}

	// Check if key is current active key (fast path)
	if bytesEqual(key, chain.CurrentActiveKey) {
		return true
	}

	// Walk chain to find matching declaration
	for _, decl := range chain.Declarations {
		// Check if key is the new key (always valid after rotation)
		if bytesEqual(key, decl.NewPublicKey) {
			return true
		}

		// Check if key is old key within grace period
		if bytesEqual(key, decl.OldPublicKey) {
			graceExpiry := decl.RotationTimestampUnix + (decl.GracePeriodDays * 86400)
			if timestamp <= graceExpiry {
				return true // Old key still within grace period
			}
		}
	}

	return false
}

// bytesEqual performs constant-time comparison of byte slices.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
