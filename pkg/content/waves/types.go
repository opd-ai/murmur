// Package waves provides Wave creation, signing, and validation.
// Per WAVES.md, there are 8 Wave types (0x01-0x08) with PoW and TTL.
package waves

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/content/pow"
	"github.com/opd-ai/murmur/pkg/encoding"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	pb "github.com/opd-ai/murmur/proto"
	"github.com/zeebo/blake3"
	"golang.org/x/crypto/curve25519"
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

// Signer is an interface for signing data. Both keys.KeyPair and AbyssalKeyPair implement this.
type Signer interface {
	Sign(data []byte) []byte
}

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
	// DeviceKey is the device public key for multi-device identity.
	// If nil, single-device mode is assumed (author_pubkey is both master and device key).
	// Per docs/MULTI_DEVICE_IDENTITY.md, this enables device→master authorization verification.
	DeviceKey []byte
}

// DefaultCreateOptions returns default options for Wave creation.
func DefaultCreateOptions() CreateOptions {
	return CreateOptions{
		TTL:        DefaultTTL,
		ParentHash: nil,
		Difficulty: DefaultDifficulty,
		DeviceKey:  nil, // Single-device mode by default
	}
}

// Create creates a new signed Wave with PoW.
func Create(waveType WaveType, content []byte, kp *keys.KeyPair, opts CreateOptions) (*pb.Wave, error) {
	if wave, handled, err := createSpecializedWave(waveType, content, kp, opts); handled {
		return wave, err
	}

	if err := validateCreateParams(kp, content, opts); err != nil {
		return nil, err
	}

	wave := buildWave(waveType, content, kp, opts)
	if err := signAndComputeWavePoW(wave, kp, opts.Difficulty); err != nil {
		return nil, err
	}

	return wave, nil
}

// createSpecializedWave routes Wave creation to type-specific constructors
// when the selected type has custom semantics.
func createSpecializedWave(waveType WaveType, content []byte, kp *keys.KeyPair, opts CreateOptions) (*pb.Wave, bool, error) {
	switch waveType {
	case TypeVeiled:
		if err := validateCreateParams(kp, content, opts); err != nil {
			return nil, true, err
		}
		// Generate an ephemeral Curve25519 keypair for the DH key exchange.
		// Ephemeral keys are never reused or stored, providing per-Wave forward secrecy:
		// even if a long-term key is later compromised it cannot decrypt past Veiled Waves.
		// Per SECURITY_PRIVACY.md §Veiled Wave encryption.
		anonKP, err := keys.GenerateAnonymousKeyPair()
		if err != nil {
			return nil, true, fmt.Errorf("generating anonymous keypair for Veiled Wave: %w", err)
		}
		veiledOpts := DefaultVeiledOptions()
		veiledOpts.TTL = opts.TTL
		veiledOpts.Difficulty = opts.Difficulty
		signer := keyPairSpecterSigner{kp: kp, dhPrivKey: anonKP.PrivateKey}
		wave, err := CreateVeiled(content, signer, veiledOpts)
		return wave, true, err
	case TypeAbyssal:
		if err := validateCreateParams(kp, content, opts); err != nil {
			return nil, true, err
		}
		abyssalOpts := DefaultAbyssalOptions()
		abyssalOpts.TTL = opts.TTL
		abyssalOpts.Difficulty = opts.Difficulty
		abyssalWave, err := CreateAbyssal(content, kp.PrivateKey, abyssalOpts)
		if err != nil {
			return nil, true, err
		}
		return abyssalWave.Wave, true, nil
	case TypeMasked:
		if err := validateCreateParams(kp, content, opts); err != nil {
			return nil, true, err
		}
		maskedKeypair, err := GenerateMaskedKeypair()
		if err != nil {
			return nil, true, err
		}
		defer maskedKeypair.Dispose()
		maskedOpts := DefaultMaskedOptions(defaultMaskedEventID)
		maskedOpts.TTL = opts.TTL
		maskedOpts.Difficulty = opts.Difficulty
		maskedOpts.ParentHash = opts.ParentHash
		wave, err := CreateMasked(content, maskedKeypair, maskedOpts)
		return wave, true, err
	case TypeBeacon:
		beaconOpts := DefaultBeaconOptions()
		beaconOpts.TTL = opts.TTL
		beaconOpts.Difficulty = opts.Difficulty
		wave, err := CreateBeacon(content, beaconOpts)
		return wave, true, err
	default:
		return nil, false, nil
	}
}

const defaultMaskedEventID = "masked-submit"

type keyPairSpecterSigner struct {
	kp        *keys.KeyPair
	dhPrivKey [32]byte // Curve25519 private key for DH key exchange.
}

func (s keyPairSpecterSigner) Sign(data []byte) []byte {
	return s.kp.Sign(data)
}

func (s keyPairSpecterSigner) SpecterPublicKey() []byte {
	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &s.dhPrivKey)
	return pub[:]
}

// ComputeDHSecret performs X25519 key exchange with peerPubKey.
// Returns the shared secret for use in key wrapping.
func (s keyPairSpecterSigner) ComputeDHSecret(peerPubKey []byte) ([]byte, error) {
	return curve25519.X25519(s.dhPrivKey[:], peerPubKey)
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
		WaveType:        pb.WaveType(waveType),
		Content:         content,
		AuthorPubkey:    kp.PublicKey,
		CreatedAt:       now.Unix(),
		TtlSeconds:      int64(opts.TTL.Seconds()),
		ParentHash:      opts.ParentHash,
		HopCount:        0,
		DevicePublicKey: opts.DeviceKey, // Set device key if provided (multi-device mode)
	}
	wave.WaveId = computeWaveID(wave)
	return wave
}

// signAndComputeWavePoW signs the wave and computes proof of work.
// This is a convenience wrapper that calls signWaveAndComputePoW with a keys.KeyPair.
func signAndComputeWavePoW(wave *pb.Wave, kp *keys.KeyPair, difficulty uint8) error {
	return signWaveAndComputePoW(wave, kp, difficulty)
}

// signWaveAndComputePoW signs the wave using the provided signer and computes proof of work.
// This is the shared implementation used by both Surface and Abyssal Waves.
func signWaveAndComputePoW(wave *pb.Wave, signer Signer, difficulty uint8) error {
	sigData := signatureData(wave)
	wave.Signature = signer.Sign(sigData)

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
// For single-device mode compatibility (device_public_key empty).
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

// DeviceAuthorizer is an interface for checking device authorization.
// Implemented by devices.DeviceStore.
type DeviceAuthorizer interface {
	IsDeviceAuthorizedWithGracePeriod(masterPubKey, devicePubKey []byte, waveTimestamp int64) (bool, error)
}

// ValidateWithDeviceStore checks Wave validity including multi-device authorization.
// Per docs/MULTI_DEVICE_IDENTITY.md, verifies:
// 1. Common checks (size, expiry, PoW)
// 2. Signature verifies against device_public_key (or author_pubkey if single-device)
// 3. If multi-device mode, device_public_key is authorized by master author_pubkey
func ValidateWithDeviceStore(wave *pb.Wave, difficulty uint8, deviceStore DeviceAuthorizer) error {
	if err := validateCommon(wave, difficulty); err != nil {
		return err
	}

	sigData := signatureData(wave)
	signingKey, err := determineSigningKey(wave, deviceStore)
	if err != nil {
		return err
	}

	if !keys.Verify(signingKey, sigData, wave.Signature) {
		return ErrInvalidSig
	}

	return nil
}

// determineSigningKey selects the correct public key for signature verification.
func determineSigningKey(wave *pb.Wave, deviceStore DeviceAuthorizer) ([]byte, error) {
	multiDevice := len(wave.DevicePublicKey) > 0
	if !multiDevice {
		return wave.AuthorPubkey, nil
	}

	if err := verifyDeviceAuthorization(wave, deviceStore); err != nil {
		return nil, err
	}
	return wave.DevicePublicKey, nil
}

// verifyDeviceAuthorization checks if device is authorized by master key.
func verifyDeviceAuthorization(wave *pb.Wave, deviceStore DeviceAuthorizer) error {
	if deviceStore == nil {
		return nil
	}

	authorized, err := deviceStore.IsDeviceAuthorizedWithGracePeriod(
		wave.AuthorPubkey,
		wave.DevicePublicKey,
		wave.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("checking device authorization: %w", err)
	}
	if !authorized {
		return fmt.Errorf("device key not authorized by master key")
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
	if err := validateExtensionWave(wave); err != nil {
		return err
	}
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

	// Manually copy fields to avoid copying the protobuf internal lock state.
	// While slightly more verbose than protobuf Clone, this is safer and
	// preserves all fields including device_public_key.
	return &pb.Wave{
		WaveType:        wave.WaveType,
		Content:         wave.Content,
		AuthorPubkey:    wave.AuthorPubkey,
		Signature:       wave.Signature,
		CreatedAt:       wave.CreatedAt,
		TtlSeconds:      wave.TtlSeconds,
		PowNonce:        wave.PowNonce,
		ParentHash:      wave.ParentHash,
		HopCount:        wave.HopCount + 1,
		WaveId:          wave.WaveId,
		DevicePublicKey: wave.DevicePublicKey,
	}
}

// int64ToBytes converts an int64 to an 8-byte big-endian array.
func int64ToBytes(v int64) [8]byte {
	return encoding.Int64ToBytes(v)
}

// hashWavePrefix hashes the common wave fields: type, content, author, creation time.
func hashWavePrefix(wave *pb.Wave) *blake3.Hasher {
	h := blake3.New()
	h.Write([]byte{byte(wave.WaveType)})
	h.Write(wave.Content)
	h.Write(wave.AuthorPubkey)
	ts := int64ToBytes(wave.CreatedAt)
	h.Write(ts[:])
	return h
}

// computeWaveID generates a BLAKE3 hash of the Wave content and metadata.
func computeWaveID(wave *pb.Wave) []byte {
	h := hashWavePrefix(wave)

	// Include parent hash if present.
	if len(wave.ParentHash) > 0 {
		h.Write(wave.ParentHash)
	}

	return h.Sum(nil)
}

// signatureData returns the data to be signed for a Wave.
func signatureData(wave *pb.Wave) []byte {
	// Pre-allocate: 1 byte (type) + content length + 8 (timestamp) + 8 (ttl)
	data := make([]byte, 0, 1+len(wave.Content)+16)

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
	// Pre-allocate: wave_id (32 bytes) + signature (64 bytes for Ed25519)
	data := make([]byte, 0, len(wave.WaveId)+len(wave.Signature))

	// Include wave ID and signature.
	data = append(data, wave.WaveId...)
	data = append(data, wave.Signature...)

	return data
}
