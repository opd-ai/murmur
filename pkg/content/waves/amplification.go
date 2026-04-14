// Package waves provides Wave creation, signing, and validation.
// This file implements Wave amplification per WAVE_PROPAGATION.md.
// Amplification re-broadcasts content with attribution and hop count reset.
package waves

import (
	"bytes"
	"errors"
	"time"

	"github.com/opd-ai/murmur/pkg/content/pow"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	pb "github.com/opd-ai/murmur/proto"
)

// AmplificationMaxComment is the maximum comment size for amplifications.
const AmplificationMaxComment = 280

// Errors for Amplification operations.
var (
	ErrNilOriginalWave      = errors.New("original wave is nil")
	ErrSelfAmplification    = errors.New("cannot amplify own wave")
	ErrAmplificationExpired = errors.New("original wave has expired")
	ErrCommentTooLong       = errors.New("comment exceeds maximum size")
	ErrInvalidAmplification = errors.New("invalid amplification")
	ErrDuplicateAmplifier   = errors.New("already amplified by this user")
)

// AmplificationOptions configures Amplification creation.
type AmplificationOptions struct {
	Comment    []byte // Optional comment (max 280 bytes)
	ResetHops  bool   // If true, reset hop count to 0
	SkipPoW    bool   // If true, don't require additional PoW
	Difficulty uint8  // PoW difficulty if SkipPoW is false
}

// DefaultAmplificationOptions returns default options for amplification.
func DefaultAmplificationOptions() AmplificationOptions {
	return AmplificationOptions{
		Comment:    nil,
		ResetHops:  false,
		SkipPoW:    true, // By default, amplifications are PoW-free
		Difficulty: pow.DefaultDifficulty,
	}
}

// CreateAmplification creates a new Amplification of a Wave.
func CreateAmplification(original *pb.Wave, kp *keys.KeyPair, opts AmplificationOptions) (*pb.Amplification, error) {
	if err := validateAmplificationParams(original, kp, opts); err != nil {
		return nil, err
	}

	amp := buildAmplification(original, kp, opts)
	if err := signAmplification(amp, kp); err != nil {
		return nil, err
	}

	return amp, nil
}

// validateAmplificationParams checks prerequisites for amplification.
func validateAmplificationParams(original *pb.Wave, kp *keys.KeyPair, opts AmplificationOptions) error {
	if original == nil {
		return ErrNilOriginalWave
	}
	if kp == nil {
		return ErrNilKeyPair
	}
	if IsExpired(original) {
		return ErrAmplificationExpired
	}
	if bytes.Equal(original.AuthorPubkey, kp.PublicKey) {
		return ErrSelfAmplification
	}
	if len(opts.Comment) > AmplificationMaxComment {
		return ErrCommentTooLong
	}
	return nil
}

// buildAmplification constructs the Amplification protobuf.
func buildAmplification(original *pb.Wave, kp *keys.KeyPair, opts AmplificationOptions) *pb.Amplification {
	// Clone the original wave to avoid modifying it.
	clonedWave := cloneWave(original)

	// Apply hop count modification if requested.
	if opts.ResetHops {
		clonedWave.HopCount = 0
	} else {
		clonedWave.HopCount++
	}

	return &pb.Amplification{
		OriginalWave:    clonedWave,
		AmplifierPubkey: kp.PublicKey,
		AmplifiedAt:     time.Now().Unix(),
		Comment:         opts.Comment,
	}
}

// cloneWave creates a deep copy of a Wave.
func cloneWave(wave *pb.Wave) *pb.Wave {
	if wave == nil {
		return nil
	}

	clone := &pb.Wave{
		WaveType:     wave.WaveType,
		Content:      append([]byte(nil), wave.Content...),
		AuthorPubkey: append([]byte(nil), wave.AuthorPubkey...),
		Signature:    append([]byte(nil), wave.Signature...),
		CreatedAt:    wave.CreatedAt,
		TtlSeconds:   wave.TtlSeconds,
		PowNonce:     wave.PowNonce,
		ParentHash:   append([]byte(nil), wave.ParentHash...),
		HopCount:     wave.HopCount,
		WaveId:       append([]byte(nil), wave.WaveId...),
	}

	// Copy metadata if present.
	if wave.Metadata != nil {
		clone.Metadata = make(map[string][]byte)
		for k, v := range wave.Metadata {
			clone.Metadata[k] = append([]byte(nil), v...)
		}
	}

	return clone
}

// signAmplification signs the amplification record.
func signAmplification(amp *pb.Amplification, kp *keys.KeyPair) error {
	sigData := amplificationSignatureData(amp)
	amp.Signature = kp.Sign(sigData)
	return nil
}

// amplificationSignatureData returns the data to sign for an amplification.
func amplificationSignatureData(amp *pb.Amplification) []byte {
	var data []byte

	// Sign: original_wave_id + amplifier_pubkey + timestamp + comment.
	if amp.OriginalWave != nil {
		data = append(data, amp.OriginalWave.WaveId...)
	}
	data = append(data, amp.AmplifierPubkey...)

	ts := int64ToBytes(amp.AmplifiedAt)
	data = append(data, ts[:]...)

	if len(amp.Comment) > 0 {
		data = append(data, amp.Comment...)
	}

	return data
}

// ValidateAmplification validates an Amplification record.
func ValidateAmplification(amp *pb.Amplification, difficulty uint8) error {
	if amp == nil {
		return ErrInvalidAmplification
	}

	if amp.OriginalWave == nil {
		return ErrNilOriginalWave
	}

	// Validate the original wave.
	if err := Validate(amp.OriginalWave, difficulty); err != nil {
		return err
	}

	// Validate amplifier's signature.
	sigData := amplificationSignatureData(amp)
	if !keys.Verify(amp.AmplifierPubkey, sigData, amp.Signature) {
		return ErrInvalidSig
	}

	// Validate comment length.
	if len(amp.Comment) > AmplificationMaxComment {
		return ErrCommentTooLong
	}

	return nil
}

// GetAmplifiedWaveID returns the ID of the wave being amplified.
func GetAmplifiedWaveID(amp *pb.Amplification) []byte {
	if amp == nil || amp.OriginalWave == nil {
		return nil
	}
	return amp.OriginalWave.WaveId
}

// GetAmplifierPubkey returns the amplifier's public key.
func GetAmplifierPubkey(amp *pb.Amplification) []byte {
	if amp == nil {
		return nil
	}
	return amp.AmplifierPubkey
}

// GetAmplificationTime returns the amplification timestamp.
func GetAmplificationTime(amp *pb.Amplification) time.Time {
	if amp == nil {
		return time.Time{}
	}
	return time.Unix(amp.AmplifiedAt, 0)
}

// GetAmplificationComment returns the amplification comment.
func GetAmplificationComment(amp *pb.Amplification) []byte {
	if amp == nil {
		return nil
	}
	return amp.Comment
}

// HasComment checks if the amplification has a comment.
func HasComment(amp *pb.Amplification) bool {
	return amp != nil && len(amp.Comment) > 0
}

// AmplificationChain tracks all amplifiers of a Wave.
type AmplificationChain struct {
	WaveID     []byte
	Amplifiers []AmplifierRecord
}

// AmplifierRecord represents a single amplifier in the chain.
type AmplifierRecord struct {
	Pubkey      []byte
	Signature   []byte
	AmplifiedAt time.Time
	Comment     []byte
}

// NewAmplificationChain creates a new chain for a Wave.
func NewAmplificationChain(waveID []byte) *AmplificationChain {
	return &AmplificationChain{
		WaveID:     waveID,
		Amplifiers: make([]AmplifierRecord, 0),
	}
}

// Add adds an amplification to the chain.
func (c *AmplificationChain) Add(amp *pb.Amplification) error {
	if amp == nil || c == nil {
		return ErrInvalidAmplification
	}

	// Check if this is a duplicate amplifier.
	for _, existing := range c.Amplifiers {
		if bytes.Equal(existing.Pubkey, amp.AmplifierPubkey) {
			return ErrDuplicateAmplifier
		}
	}

	c.Amplifiers = append(c.Amplifiers, AmplifierRecord{
		Pubkey:      amp.AmplifierPubkey,
		Signature:   amp.Signature,
		AmplifiedAt: time.Unix(amp.AmplifiedAt, 0),
		Comment:     amp.Comment,
	})

	return nil
}

// Count returns the number of amplifications.
func (c *AmplificationChain) Count() int {
	if c == nil {
		return 0
	}
	return len(c.Amplifiers)
}

// HasAmplifier checks if a public key has already amplified.
func (c *AmplificationChain) HasAmplifier(pubkey []byte) bool {
	if c == nil {
		return false
	}
	for _, amp := range c.Amplifiers {
		if bytes.Equal(amp.Pubkey, pubkey) {
			return true
		}
	}
	return false
}

// GetAmplifiers returns all amplifier public keys.
func (c *AmplificationChain) GetAmplifiers() [][]byte {
	if c == nil {
		return nil
	}
	result := make([][]byte, len(c.Amplifiers))
	for i, amp := range c.Amplifiers {
		result[i] = amp.Pubkey
	}
	return result
}

// CreateAmplificationWithComment creates an amplification with a comment.
func CreateAmplificationWithComment(original *pb.Wave, kp *keys.KeyPair, comment []byte) (*pb.Amplification, error) {
	opts := DefaultAmplificationOptions()
	opts.Comment = comment
	return CreateAmplification(original, kp, opts)
}

// CreateAmplificationWithHopReset creates an amplification that resets hop count.
// This is a special case that may require elevated permissions or additional PoW.
func CreateAmplificationWithHopReset(original *pb.Wave, kp *keys.KeyPair) (*pb.Amplification, error) {
	opts := DefaultAmplificationOptions()
	opts.ResetHops = true
	opts.SkipPoW = false // Hop reset requires PoW
	return CreateAmplification(original, kp, opts)
}

// AmplifyAndIncrement creates an amplification and increments the wave's hop count.
// This is the standard amplification behavior.
func AmplifyAndIncrement(original *pb.Wave, kp *keys.KeyPair) (*pb.Amplification, error) {
	return CreateAmplification(original, kp, DefaultAmplificationOptions())
}
