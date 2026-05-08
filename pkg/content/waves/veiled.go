// Package waves provides Wave creation, signing, and validation.
// This file implements Veiled Wave creation — cross-layer Waves authored
// by Specters with symmetric key wrapping for optional encryption.
// Per WAVES.md, Veiled Waves (type 0x03) propagate through both layers.
package waves

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"time"

	"github.com/opd-ai/murmur/pkg/content/pow"
	pb "github.com/opd-ai/murmur/proto"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

// Veiled Wave constants.
const (
	// VeiledMetadataKey is the metadata key indicating a Veiled Wave.
	VeiledMetadataKey = "veil"

	// VeiledMetadataValue is the value indicating a Veiled Wave.
	VeiledMetadataValue = "true"

	// EncryptedContentKey is the metadata key for encrypted content indicator.
	EncryptedContentKey = "encrypted"

	// WrappedKeyKey is the metadata key for the wrapped symmetric key.
	WrappedKeyKey = "wrapped_key"

	// NonceKey is the metadata key for the encryption nonce.
	NonceKey = "nonce"

	// SymmetricKeySize is the size of the symmetric key for content encryption.
	SymmetricKeySize = 32
)

// VeiledWaveErrors.
var (
	ErrSpecterKeyRequired = errors.New("specter keypair required for Veiled Wave")
	ErrInvalidWrappedKey  = errors.New("invalid wrapped key")
	ErrDecryptionFailed   = errors.New("decryption failed")
)

// SpecterSigner is an interface for Specter-specific signing operations.
type SpecterSigner interface {
	Signer
	// SpecterPublicKey returns the Specter's Curve25519 public key (32 bytes).
	SpecterPublicKey() []byte
	// ComputeDHSecret performs X25519 key exchange with peerPubKey and returns
	// the shared secret. Used for key wrapping in Veiled Wave encryption.
	// Per TECHNICAL_IMPLEMENTATION.md, uses Curve25519 for Anonymous Layer.
	ComputeDHSecret(peerPubKey []byte) ([]byte, error)
}

// VeiledOptions configures Veiled Wave creation.
type VeiledOptions struct {
	// TTL is the time-to-live for the Wave.
	TTL time.Duration

	// Difficulty is the PoW difficulty for the Wave.
	Difficulty uint8

	// Encrypted indicates if content should be encrypted.
	Encrypted bool

	// RecipientPubKey is the recipient's public key for encryption (32 bytes).
	// Only used if Encrypted is true.
	RecipientPubKey []byte
}

// DefaultVeiledOptions returns default options for Veiled Wave creation.
func DefaultVeiledOptions() VeiledOptions {
	return VeiledOptions{
		TTL:        DefaultTTL,
		Difficulty: pow.DefaultDifficulty,
		Encrypted:  false,
	}
}

// CreateVeiled creates a Veiled Wave authored by a Specter.
// The Wave propagates through both Anonymous and Surface layers.
// Per WAVES.md, Veiled Waves are the primary mechanism for anonymous
// voices to reach the broader network.
func CreateVeiled(content []byte, specter SpecterSigner, opts VeiledOptions) (*pb.Wave, error) {
	if specter == nil {
		return nil, ErrSpecterKeyRequired
	}
	if len(content) > MaxContentSize {
		return nil, ErrContentTooLarge
	}
	if opts.TTL <= 0 {
		return nil, ErrInvalidTTL
	}
	if opts.TTL > MaxTTL {
		return nil, ErrTTLTooLong
	}

	// Build the Veiled Wave.
	wave, err := buildVeiledWave(content, specter, opts)
	if err != nil {
		return nil, err
	}

	// Sign and compute PoW.
	if err := signWaveAndComputePoW(wave, specter, opts.Difficulty); err != nil {
		return nil, err
	}

	return wave, nil
}

// buildVeiledWave constructs the Veiled Wave structure.
func buildVeiledWave(content []byte, specter SpecterSigner, opts VeiledOptions) (*pb.Wave, error) {
	now := time.Now()

	// Initialize metadata with veil flag.
	metadata := map[string][]byte{
		VeiledMetadataKey: []byte(VeiledMetadataValue),
	}

	// Handle encryption if requested.
	finalContent := content
	if opts.Encrypted && len(opts.RecipientPubKey) > 0 {
		encContent, nonce, wrappedKey, err := encryptVeiledContent(
			content,
			specter,
			opts.RecipientPubKey,
		)
		if err != nil {
			return nil, err
		}
		finalContent = encContent
		metadata[EncryptedContentKey] = []byte("true")
		metadata[WrappedKeyKey] = wrappedKey
		metadata[NonceKey] = nonce
	}

	wave := &pb.Wave{
		WaveType:     pb.WaveType(TypeVeiled),
		Content:      finalContent,
		AuthorPubkey: specter.SpecterPublicKey(),
		CreatedAt:    now.Unix(),
		TtlSeconds:   int64(opts.TTL.Seconds()),
		HopCount:     0,
		Metadata:     metadata,
	}

	wave.WaveId = computeWaveID(wave)
	return wave, nil
}

// encryptVeiledContent encrypts content using XChaCha20-Poly1305.
// The symmetric key is wrapped using X25519 DH + HKDF-SHA-256, so only
// the holder of the recipient's Curve25519 private key can unwrap it.
// Per TECHNICAL_IMPLEMENTATION.md, uses Curve25519 for Anonymous Layer
// key exchange and HKDF-SHA-256 for key derivation.
// Returns encrypted content, nonce, and wrapped symmetric key.
func encryptVeiledContent(content []byte, specter SpecterSigner, recipientPubKey []byte) ([]byte, []byte, []byte, error) {
	// Generate a random symmetric key.
	symmetricKey := make([]byte, SymmetricKeySize)
	if _, err := rand.Read(symmetricKey); err != nil {
		return nil, nil, nil, err
	}

	// Create XChaCha20-Poly1305 cipher.
	aead, err := chacha20poly1305.NewX(symmetricKey)
	if err != nil {
		return nil, nil, nil, err
	}

	// Generate random nonce.
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, nil, err
	}

	// Encrypt content with author public key as additional data.
	encrypted := aead.Seal(nil, nonce, content, specter.SpecterPublicKey())

	// Wrap symmetric key with DH-derived wrap key.
	// sharedSecret = X25519(senderPrivKey, recipientPubKey)
	sharedSecret, err := specter.ComputeDHSecret(recipientPubKey)
	if err != nil {
		return nil, nil, nil, err
	}

	wrapKey, err := deriveVeiledWrapKey(sharedSecret)
	if err != nil {
		return nil, nil, nil, err
	}

	wrappedKey := xorBytes(symmetricKey, wrapKey)
	return encrypted, nonce, wrappedKey, nil
}

// deriveVeiledWrapKey derives a 32-byte wrap key from a DH shared secret
// using HKDF-SHA-256 with the "murmur-veil-wrap-v1" info string.
// Per TECHNICAL_IMPLEMENTATION.md, HKDF-SHA-256 is used for key derivation.
func deriveVeiledWrapKey(dhSharedSecret []byte) ([]byte, error) {
	kdf := hkdf.New(sha256.New, dhSharedSecret, nil, []byte("murmur-veil-wrap-v1"))
	key := make([]byte, SymmetricKeySize)
	if _, err := io.ReadFull(kdf, key); err != nil {
		return nil, err
	}
	return key, nil
}

// xorBytes XORs two equal-length byte slices into a new slice.
func xorBytes(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// UnwrapSymmetricKey unwraps a symmetric key using X25519 DH + HKDF-SHA-256.
// The recipient uses their Curve25519 private key and the author's public key
// to reproduce the same shared secret and derive the wrap key.
// Per TECHNICAL_IMPLEMENTATION.md, uses Curve25519 for Anonymous Layer.
func UnwrapSymmetricKey(wrappedKey, authorPubKey, recipientPrivKey []byte) ([]byte, error) {
	if len(wrappedKey) != SymmetricKeySize {
		return nil, ErrInvalidWrappedKey
	}

	// sharedSecret = X25519(recipientPrivKey, authorPubKey)
	sharedSecret, err := curve25519.X25519(recipientPrivKey, authorPubKey)
	if err != nil {
		return nil, ErrInvalidWrappedKey
	}

	wrapKey, err := deriveVeiledWrapKey(sharedSecret)
	if err != nil {
		return nil, ErrInvalidWrappedKey
	}

	return xorBytes(wrappedKey, wrapKey), nil
}

// DecryptVeiledContent decrypts a Veiled Wave's content.
// recipientPrivKey is the recipient's Curve25519 private key (32 bytes),
// used with the wave's AuthorPubkey to derive the symmetric wrap key via DH.
func DecryptVeiledContent(wave *pb.Wave, recipientPrivKey []byte) ([]byte, error) {
	if wave == nil {
		return nil, errors.New("wave is nil")
	}

	if !isWaveEncrypted(wave) {
		return wave.Content, nil
	}

	wrappedKey, nonce, err := extractEncryptionMetadata(wave)
	if err != nil {
		return nil, err
	}

	symmetricKey, err := UnwrapSymmetricKey(wrappedKey, wave.AuthorPubkey, recipientPrivKey)
	if err != nil {
		return nil, err
	}

	return decryptContent(symmetricKey, nonce, wave.Content, wave.AuthorPubkey)
}

// isWaveEncrypted checks if the wave has encrypted content.
func isWaveEncrypted(wave *pb.Wave) bool {
	encrypted, ok := wave.Metadata[EncryptedContentKey]
	return ok && string(encrypted) == "true"
}

// extractEncryptionMetadata extracts wrapped key and nonce from wave metadata.
func extractEncryptionMetadata(wave *pb.Wave) ([]byte, []byte, error) {
	wrappedKey, ok := wave.Metadata[WrappedKeyKey]
	if !ok {
		return nil, nil, ErrInvalidWrappedKey
	}

	nonce, ok := wave.Metadata[NonceKey]
	if !ok || len(nonce) != chacha20poly1305.NonceSizeX {
		return nil, nil, ErrDecryptionFailed
	}

	return wrappedKey, nonce, nil
}

// decryptContent decrypts content using XChaCha20-Poly1305.
func decryptContent(symmetricKey, nonce, ciphertext, aad []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(symmetricKey)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// IsVeiled checks if a Wave is a Veiled Wave.
func IsVeiled(wave *pb.Wave) bool {
	if wave == nil {
		return false
	}
	if wave.WaveType != pb.WaveType(TypeVeiled) {
		return false
	}
	veil, ok := wave.Metadata[VeiledMetadataKey]
	return ok && string(veil) == VeiledMetadataValue
}

// IsEncryptedVeiled checks if a Veiled Wave has encrypted content.
func IsEncryptedVeiled(wave *pb.Wave) bool {
	if !IsVeiled(wave) {
		return false
	}
	encrypted, ok := wave.Metadata[EncryptedContentKey]
	return ok && string(encrypted) == "true"
}

// ValidateVeiled validates a Veiled Wave.
func ValidateVeiled(wave *pb.Wave, difficulty uint8) error {
	if wave == nil {
		return errors.New("wave is nil")
	}
	if wave.WaveType != pb.WaveType(TypeVeiled) {
		return errors.New("not a Veiled Wave")
	}
	if !IsVeiled(wave) {
		return errors.New("missing veil metadata")
	}

	// Common validation.
	if err := validateCommon(wave, difficulty); err != nil {
		return err
	}

	return nil
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
