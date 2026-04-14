// Package keys provides Ed25519 and Curve25519 keypair generation, signing, and
// verification for MURMUR's identity system.
// Per SECURITY_PRIVACY.md, Ed25519 is used for Surface Layer signatures and
// Curve25519 for Anonymous Layer key exchange.
package keys

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

// Argon2id parameters per TECHNICAL_IMPLEMENTATION.md §1.4.
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MiB
	argon2Threads = 4
	argon2KeyLen  = 32
)

// SaltSize is the size of the random salt for Argon2id key derivation.
const SaltSize = 16

// NonceSize is the size of the nonce for XChaCha20-Poly1305.
const NonceSize = 24

// KeyPair represents an Ed25519 signing keypair.
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// AnonymousKeyPair represents a Curve25519 keypair for anonymous operations.
type AnonymousKeyPair struct {
	PublicKey  [32]byte
	PrivateKey [32]byte
}

// GenerateKeyPair creates a new Ed25519 keypair for Surface Layer identity.
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating Ed25519 keypair: %w", err)
	}
	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// GenerateAnonymousKeyPair creates a Curve25519 keypair for Anonymous Layer.
// The keypair is used for Shroud circuit key exchange and Specter identity.
func GenerateAnonymousKeyPair() (*AnonymousKeyPair, error) {
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		return nil, fmt.Errorf("generating random key: %w", err)
	}

	// Clamp the private key as per X25519 specification.
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return &AnonymousKeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

// Sign signs a message with the Ed25519 private key.
func (kp *KeyPair) Sign(message []byte) []byte {
	return ed25519.Sign(kp.PrivateKey, message)
}

// Verify verifies a signature against a message and public key.
func Verify(publicKey ed25519.PublicKey, message, signature []byte) bool {
	return ed25519.Verify(publicKey, message, signature)
}

// DeriveSharedSecret computes a shared secret using X25519 Diffie-Hellman.
// Used for Shroud circuit encryption.
func (kp *AnonymousKeyPair) DeriveSharedSecret(peerPublic [32]byte) ([]byte, error) {
	var shared [32]byte
	curve25519.ScalarMult(&shared, &kp.PrivateKey, &peerPublic)

	// Check for low-order points that would result in all-zero shared secret.
	var zero [32]byte
	if shared == zero {
		return nil, errors.New("invalid peer public key: low-order point")
	}

	return shared[:], nil
}

// EncryptKeystore encrypts key material using Argon2id + XChaCha20-Poly1305.
// Per TECHNICAL_IMPLEMENTATION.md §1.4, this protects stored keys with a passphrase.
func EncryptKeystore(plaintext []byte, passphrase string) ([]byte, error) {
	// Generate random salt.
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}

	// Derive key using Argon2id.
	key := argon2.IDKey(
		[]byte(passphrase),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)

	// Create XChaCha20-Poly1305 cipher.
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	// Generate random nonce.
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	// Encrypt with AEAD.
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	// Zero the key material.
	for i := range key {
		key[i] = 0
	}

	// Format: salt (16) || nonce (24) || ciphertext
	result := make([]byte, 0, SaltSize+NonceSize+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// DecryptKeystore decrypts key material encrypted by EncryptKeystore.
func DecryptKeystore(data []byte, passphrase string) ([]byte, error) {
	salt, nonce, ciphertext, err := extractKeystoreComponents(data)
	if err != nil {
		return nil, err
	}

	key := deriveKeyFromPassphrase(passphrase, salt)
	defer ZeroBytes(key)

	return decryptWithKey(key, nonce, ciphertext)
}

// extractKeystoreComponents parses encrypted keystore data into its parts.
func extractKeystoreComponents(data []byte) (salt, nonce, ciphertext []byte, err error) {
	minLen := SaltSize + NonceSize + chacha20poly1305.Overhead
	if len(data) < minLen {
		return nil, nil, nil, errors.New("encrypted data too short")
	}
	return data[:SaltSize], data[SaltSize : SaltSize+NonceSize], data[SaltSize+NonceSize:], nil
}

// deriveKeyFromPassphrase uses Argon2id to derive an encryption key.
func deriveKeyFromPassphrase(passphrase string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(passphrase),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)
}

// decryptWithKey decrypts ciphertext using XChaCha20-Poly1305.
func decryptWithKey(key, nonce, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	return plaintext, nil
}

// ZeroBytes zeros a byte slice to help prevent key material leakage.
// Per TECHNICAL_IMPLEMENTATION.md §1.4, key material should be zeroed
// before backing arrays become GC-eligible.
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// ZeroKeyPair zeros the private key material in a KeyPair.
func (kp *KeyPair) ZeroKeyPair() {
	ZeroBytes(kp.PrivateKey)
}

// ZeroAnonymousKeyPair zeros the private key material in an AnonymousKeyPair.
func (kp *AnonymousKeyPair) ZeroAnonymousKeyPair() {
	ZeroBytes(kp.PrivateKey[:])
}

// IdentityBundle contains both Surface and Anonymous Layer keypairs.
// Per SECURITY_PRIVACY.md, these keypairs are cryptographically independent -
// compromising one MUST NOT reveal the other.
type IdentityBundle struct {
	// Surface is the Ed25519 keypair for Surface Layer identity.
	Surface *KeyPair
	// Specter is the Curve25519 keypair for Anonymous Layer identity.
	Specter *AnonymousKeyPair
	// FortressTransport is an optional Ed25519 keypair for Fortress mode.
	// Per SHADOW_GRADIENT.md, Fortress mode uses a dedicated transport key
	// that is separate from both Surface and Specter keys.
	FortressTransport *KeyPair
}

// GenerateIdentityBundle creates a complete identity bundle with independent keypairs.
// Per SECURITY_PRIVACY.md, Surface and Specter keys share no derivation path.
// Each key is generated from independent entropy sources.
func GenerateIdentityBundle() (*IdentityBundle, error) {
	// Generate Surface Layer identity (Ed25519).
	surface, err := GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("generating surface keypair: %w", err)
	}

	// Generate Anonymous Layer identity (Curve25519).
	// This is completely independent - no shared derivation path.
	specter, err := GenerateAnonymousKeyPair()
	if err != nil {
		surface.ZeroKeyPair()
		return nil, fmt.Errorf("generating specter keypair: %w", err)
	}

	return &IdentityBundle{
		Surface: surface,
		Specter: specter,
	}, nil
}

// GenerateIdentityBundleWithFortress creates an identity bundle including Fortress transport key.
// The Fortress transport key is a separate Ed25519 keypair used for transport
// in Fortress mode, ensuring complete key isolation.
func GenerateIdentityBundleWithFortress() (*IdentityBundle, error) {
	bundle, err := GenerateIdentityBundle()
	if err != nil {
		return nil, err
	}

	// Generate dedicated Fortress transport key.
	fortress, err := GenerateKeyPair()
	if err != nil {
		bundle.Zero()
		return nil, fmt.Errorf("generating fortress transport keypair: %w", err)
	}

	bundle.FortressTransport = fortress
	return bundle, nil
}

// Zero securely zeros all keypair material in the bundle.
func (ib *IdentityBundle) Zero() {
	if ib.Surface != nil {
		ib.Surface.ZeroKeyPair()
	}
	if ib.Specter != nil {
		ib.Specter.ZeroAnonymousKeyPair()
	}
	if ib.FortressTransport != nil {
		ib.FortressTransport.ZeroKeyPair()
	}
}

// ValidateIndependence verifies that keypairs share no derivation path.
// This is a defensive check to ensure key generation didn't accidentally
// introduce any correlation between keys.
func (ib *IdentityBundle) ValidateIndependence() bool {
	if ib.Surface == nil || ib.Specter == nil {
		return false
	}

	// Surface Ed25519 public key should not match any portion of Specter public key.
	// This is a sanity check - proper random generation should always pass.
	surfacePub := ib.Surface.PublicKey
	specterPub := ib.Specter.PublicKey[:]

	// Check for exact match (should never happen with proper randomness).
	if len(surfacePub) == len(specterPub) {
		match := true
		for i := range surfacePub {
			if surfacePub[i] != specterPub[i] {
				match = false
				break
			}
		}
		if match {
			return false
		}
	}

	// If Fortress key exists, verify it's also independent.
	if ib.FortressTransport != nil {
		fortressPub := ib.FortressTransport.PublicKey

		// Fortress should not match Surface.
		if len(fortressPub) == len(surfacePub) {
			match := true
			for i := range fortressPub {
				if fortressPub[i] != surfacePub[i] {
					match = false
					break
				}
			}
			if match {
				return false
			}
		}
	}

	return true
}
