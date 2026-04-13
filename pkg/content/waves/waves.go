// Package waves provides Wave creation, signing, and validation.
// Per WAVES.md, there are 8 Wave types (0x01-0x08) with PoW and TTL.
package waves

import (
	"crypto/ed25519"
	"errors"
	"time"

	"github.com/opd-ai/murmur/pkg/content/pow"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	pb "github.com/opd-ai/murmur/proto"
	"github.com/zeebo/blake3"
)

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

// DefaultDifficulty is the default PoW difficulty for Waves.
const DefaultDifficulty = pow.DefaultDifficulty

// Errors for Wave operations.
var (
	ErrContentTooLarge = errors.New("content exceeds maximum size")
	ErrTTLTooLong      = errors.New("TTL exceeds maximum allowed")
	ErrInvalidTTL      = errors.New("TTL must be positive")
	ErrExpired         = errors.New("wave has expired")
	ErrInvalidPoW      = errors.New("invalid proof of work")
	ErrInvalidSig      = errors.New("invalid signature")
	ErrNilKeyPair      = errors.New("keypair is nil")
)

// CreateOptions configures Wave creation.
type CreateOptions struct {
	TTL        time.Duration
	ParentHash []byte
	Difficulty uint8
}

// DefaultCreateOptions returns default options for Wave creation.
func DefaultCreateOptions() CreateOptions {
	return CreateOptions{
		TTL:        DefaultTTL,
		ParentHash: nil,
		Difficulty: DefaultDifficulty,
	}
}

// Create creates a new signed Wave with PoW.
func Create(waveType WaveType, content []byte, kp *keys.KeyPair, opts CreateOptions) (*pb.Wave, error) {
	if err := validateCreateParams(kp, content, opts); err != nil {
		return nil, err
	}

	wave := buildWave(waveType, content, kp, opts)
	if err := signAndComputeWavePoW(wave, kp, opts.Difficulty); err != nil {
		return nil, err
	}

	return wave, nil
}

// validateCreateParams checks prerequisites for Wave creation.
func validateCreateParams(kp *keys.KeyPair, content []byte, opts CreateOptions) error {
	if kp == nil {
		return ErrNilKeyPair
	}
	if len(content) > MaxContentSize {
		return ErrContentTooLarge
	}
	if opts.TTL <= 0 {
		return ErrInvalidTTL
	}
	if opts.TTL > MaxTTL {
		return ErrTTLTooLong
	}
	return nil
}

// buildWave constructs the protobuf Wave with ID computed.
func buildWave(waveType WaveType, content []byte, kp *keys.KeyPair, opts CreateOptions) *pb.Wave {
	now := time.Now()
	wave := &pb.Wave{
		WaveType:     pb.WaveType(waveType),
		Content:      content,
		AuthorPubkey: kp.PublicKey,
		CreatedAt:    now.Unix(),
		TtlSeconds:   int64(opts.TTL.Seconds()),
		ParentHash:   opts.ParentHash,
		HopCount:     0,
	}
	wave.WaveId = computeWaveID(wave)
	return wave
}

// signAndComputeWavePoW signs the wave and computes proof of work.
func signAndComputeWavePoW(wave *pb.Wave, kp *keys.KeyPair, difficulty uint8) error {
	sigData := signatureData(wave)
	wave.Signature = kp.Sign(sigData)

	powInput := powData(wave)
	work, err := pow.Compute(powInput, difficulty)
	if err != nil {
		return err
	}
	wave.PowNonce = work.Nonce
	return nil
}

// CreateSurface creates a standard Surface Layer Wave.
func CreateSurface(content []byte, kp *keys.KeyPair) (*pb.Wave, error) {
	return Create(TypeSurface, content, kp, DefaultCreateOptions())
}

// CreateReply creates a reply to another Wave.
func CreateReply(content, parentHash []byte, kp *keys.KeyPair) (*pb.Wave, error) {
	opts := DefaultCreateOptions()
	opts.ParentHash = parentHash
	return Create(TypeReply, content, kp, opts)
}

// Validate checks if a Wave is valid (signature, PoW, TTL).
func Validate(wave *pb.Wave, difficulty uint8) error {
	if err := validateCommon(wave, difficulty); err != nil {
		return err
	}
	// Use keys.Verify for standard waves (same as ed25519.Verify)
	sigData := signatureData(wave)
	if !keys.Verify(wave.AuthorPubkey, sigData, wave.Signature) {
		return ErrInvalidSig
	}
	return nil
}

// validateCommon performs common validation checks shared by all wave types.
// Checks content size, expiration, public key size, and PoW.
func validateCommon(wave *pb.Wave, difficulty uint8) error {
	if wave == nil {
		return errors.New("wave is nil")
	}

	// Check content size.
	if len(wave.Content) > MaxContentSize {
		return ErrContentTooLarge
	}

	// Check expiration.
	if IsExpired(wave) {
		return ErrExpired
	}

	// Verify public key size.
	if len(wave.AuthorPubkey) != ed25519.PublicKeySize {
		return ErrInvalidSig
	}

	// Verify PoW.
	pd := powData(wave)
	if !pow.Verify(pd, wave.PowNonce, difficulty) {
		return ErrInvalidPoW
	}

	return nil
}

// IsExpired checks if a Wave has exceeded its TTL.
func IsExpired(wave *pb.Wave) bool {
	if wave == nil {
		return true
	}

	created := time.Unix(wave.CreatedAt, 0)
	ttl := time.Duration(wave.TtlSeconds) * time.Second
	expiration := created.Add(ttl)

	return time.Now().After(expiration)
}

// ExpiresAt returns the expiration time of a Wave.
func ExpiresAt(wave *pb.Wave) time.Time {
	if wave == nil {
		return time.Time{}
	}

	created := time.Unix(wave.CreatedAt, 0)
	ttl := time.Duration(wave.TtlSeconds) * time.Second
	return created.Add(ttl)
}

// IncrementHop increments the hop count on a Wave.
// Returns a new Wave with incremented hop count.
func IncrementHop(wave *pb.Wave) *pb.Wave {
	if wave == nil {
		return nil
	}

	// Create a new Wave with incremented hop count.
	// We manually copy fields to avoid copying the protobuf internal state.
	return &pb.Wave{
		WaveType:     wave.WaveType,
		Content:      wave.Content,
		AuthorPubkey: wave.AuthorPubkey,
		Signature:    wave.Signature,
		CreatedAt:    wave.CreatedAt,
		TtlSeconds:   wave.TtlSeconds,
		PowNonce:     wave.PowNonce,
		ParentHash:   wave.ParentHash,
		HopCount:     wave.HopCount + 1,
		WaveId:       wave.WaveId,
	}
}

// int64ToBytes converts an int64 to an 8-byte big-endian array.
func int64ToBytes(v int64) [8]byte {
	var b [8]byte
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
	return b
}

// computeWaveID generates a BLAKE3 hash of the Wave content and metadata.
func computeWaveID(wave *pb.Wave) []byte {
	h := blake3.New()

	// Include type, content, author, and creation time.
	h.Write([]byte{byte(wave.WaveType)})
	h.Write(wave.Content)
	h.Write(wave.AuthorPubkey)

	// Include creation time as bytes.
	ts := int64ToBytes(wave.CreatedAt)
	h.Write(ts[:])

	// Include parent hash if present.
	if len(wave.ParentHash) > 0 {
		h.Write(wave.ParentHash)
	}

	return h.Sum(nil)
}

// signatureData returns the data to be signed for a Wave.
func signatureData(wave *pb.Wave) []byte {
	var data []byte

	// wave_type || content || created_at || ttl
	data = append(data, byte(wave.WaveType))
	data = append(data, wave.Content...)

	ts := int64ToBytes(wave.CreatedAt)
	data = append(data, ts[:]...)

	ttl := int64ToBytes(wave.TtlSeconds)
	data = append(data, ttl[:]...)

	return data
}

// powData returns the data to be used for PoW computation.
func powData(wave *pb.Wave) []byte {
	var data []byte

	// Include wave ID and signature.
	data = append(data, wave.WaveId...)
	data = append(data, wave.Signature...)

	return data
}
