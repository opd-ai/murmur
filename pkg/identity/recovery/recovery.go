// Package recovery implements Shamir Secret Sharing based social recovery.
// Per SOCIAL_RECOVERY.md, enables M-of-N threshold recovery for Master Keys.
package recovery

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/proto"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

const (
	MinThreshold   = 2
	MaxTotalShares = 10
	NonceSize      = 24
)

var (
	ErrInvalidThreshold     = errors.New("threshold must be >= 2 and <= total_shares")
	ErrInvalidShareCount    = errors.New("total shares must be <= 10")
	ErrNotEnoughShares      = errors.New("insufficient shares for reconstruction")
	ErrInvalidSignature     = errors.New("enrollment signature verification failed")
	ErrInvalidTimestamp     = errors.New("timestamp outside valid range (±300s)")
	ErrDecryptionFailed     = errors.New("failed to decrypt share")
	ErrReconstructionFailed = errors.New("Shamir reconstruction failed")
	ErrInvalidMasterKey     = errors.New("reconstructed key does not match expected public key")
)

// Contact represents a recovery contact with their public key.
type Contact struct {
	PublicKey ed25519.PublicKey
	X25519Key []byte
	Label     string
}

// EnrollmentResult contains the outcome of enrolling a single contact.
type EnrollmentResult struct {
	Contact    Contact
	ShareIndex uint32
	Success    bool
	Error      error
	Enrollment *proto.RecoveryShareEnrollment
}

// RecoveryResult contains the outcome of a recovery request.
type RecoveryResult struct {
	MasterKey  []byte
	SharesUsed []uint32
	Success    bool
	Error      error
}

// deriveSharedSecret performs X25519 ECDH and derives a symmetric key.
func deriveSharedSecret(privateKey, publicKey []byte) ([]byte, error) {
	if len(privateKey) != curve25519.ScalarSize || len(publicKey) != curve25519.PointSize {
		return nil, fmt.Errorf("invalid key sizes")
	}

	shared, err := curve25519.X25519(privateKey, publicKey)
	if err != nil {
		return nil, fmt.Errorf("X25519 key exchange failed: %w", err)
	}
	// F-CRYPTO-5 fix: Zero shared secret before returning.
	defer keys.ZeroBytes(shared)

	// Per AUDIT.md LOW finding, use a fixed domain-separator salt for HKDF auditability.
	hkdfReader := hkdf.New(sha256.New, shared, []byte("murmur-recovery-salt-v1"), []byte("murmur-recovery-share-v1"))
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := hkdfReader.Read(key); err != nil {
		return nil, fmt.Errorf("HKDF derivation failed: %w", err)
	}

	return key, nil
}

// encryptShare encrypts a Shamir share with XChaCha20-Poly1305.
func encryptShare(share, symmetricKey []byte) (ciphertext, nonce []byte, err error) {
	aead, err := chacha20poly1305.NewX(symmetricKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AEAD: %w", err)
	}

	nonce = make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext = aead.Seal(nil, nonce, share, nil)
	return ciphertext, nonce, nil
}

// decryptShare decrypts a Shamir share with XChaCha20-Poly1305.
func decryptShare(ciphertext, nonce, symmetricKey []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(symmetricKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// validateTimestamp checks if timestamp is within ±300 seconds of now.
func validateTimestamp(timestamp int64) error {
	now := time.Now().Unix()
	delta := now - timestamp
	if delta < -300 || delta > 300 {
		return ErrInvalidTimestamp
	}
	return nil
}

// buildEnrollmentData serializes enrollment fields for signing/verification.
func buildEnrollmentData(enrollment *proto.RecoveryShareEnrollment) []byte {
	data := make([]byte, 0, len(enrollment.MasterPublicKey)+len(enrollment.RecipientPublicKey)+len(enrollment.EncryptedShare)+len(enrollment.Nonce)+20)
	data = append(data, enrollment.MasterPublicKey...)
	data = append(data, enrollment.RecipientPublicKey...)
	data = append(data, enrollment.EncryptedShare...)
	data = append(data, enrollment.Nonce...)

	buf := make([]byte, 4)
	for _, val := range []uint32{enrollment.ShareIndex, enrollment.Threshold, enrollment.TotalShares} {
		buf[0] = byte(val >> 24)
		buf[1] = byte(val >> 16)
		buf[2] = byte(val >> 8)
		buf[3] = byte(val)
		data = append(data, buf...)
	}

	tsBuf := make([]byte, 8)
	ts := uint64(enrollment.TimestampUnix)
	for i := 0; i < 8; i++ {
		tsBuf[7-i] = byte(ts >> (i * 8))
	}
	data = append(data, tsBuf...)

	return data
}

// signEnrollment creates an Ed25519 signature over enrollment fields.
func signEnrollment(enrollment *proto.RecoveryShareEnrollment, privateKey ed25519.PrivateKey) ([]byte, error) {
	data := buildEnrollmentData(enrollment)
	return ed25519.Sign(privateKey, data), nil
}

// verifyEnrollmentSignature verifies the enrollment signature.
func verifyEnrollmentSignature(enrollment *proto.RecoveryShareEnrollment) error {
	if len(enrollment.MasterPublicKey) != ed25519.PublicKeySize {
		return ErrInvalidSignature
	}

	data := buildEnrollmentData(enrollment)

	if !ed25519.Verify(enrollment.MasterPublicKey, data, enrollment.EnrollmentSignature) {
		return ErrInvalidSignature
	}

	return nil
}
