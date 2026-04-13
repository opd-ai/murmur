// Package security provides security audit and verification utilities.
// Per ROADMAP.md Priority 12, this package implements cryptographic
// implementation review helpers, key storage security validation,
// and dependency vulnerability checking interfaces.
package security

import (
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

// Cryptographic audit errors.
var (
	ErrWeakKey            = errors.New("weak or predictable key detected")
	ErrKeyLengthInvalid   = errors.New("invalid key length")
	ErrSignatureInvalid   = errors.New("signature verification failed")
	ErrEncryptionFailed   = errors.New("encryption operation failed")
	ErrDecryptionFailed   = errors.New("decryption operation failed")
	ErrRandomSourceWeak   = errors.New("random source may be weak")
	ErrKeyMaterialExposed = errors.New("key material may be exposed")
)

// AuditResult contains the result of a security audit.
type AuditResult struct {
	Category    string
	Passed      bool
	Description string
	Details     string
}

// CryptoAuditor verifies cryptographic implementations.
type CryptoAuditor struct {
	results []AuditResult
}

// NewCryptoAuditor creates a new cryptographic auditor.
func NewCryptoAuditor() *CryptoAuditor {
	return &CryptoAuditor{
		results: make([]AuditResult, 0),
	}
}

// AuditEd25519 verifies Ed25519 implementation correctness.
func (a *CryptoAuditor) AuditEd25519() error {
	// Test key generation.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		a.addResult("Ed25519", false, "Key generation failed", err.Error())
		return fmt.Errorf("Ed25519 key generation: %w", err)
	}
	a.addResult("Ed25519", true, "Key generation", "Generated valid keypair")

	// Test key length.
	if len(pub) != ed25519.PublicKeySize {
		a.addResult("Ed25519", false, "Public key size", fmt.Sprintf("expected %d, got %d", ed25519.PublicKeySize, len(pub)))
		return ErrKeyLengthInvalid
	}
	if len(priv) != ed25519.PrivateKeySize {
		a.addResult("Ed25519", false, "Private key size", fmt.Sprintf("expected %d, got %d", ed25519.PrivateKeySize, len(priv)))
		return ErrKeyLengthInvalid
	}
	a.addResult("Ed25519", true, "Key sizes", "Public and private keys are correct size")

	// Test signing and verification.
	message := []byte("test message for Ed25519")
	sig := ed25519.Sign(priv, message)

	if !ed25519.Verify(pub, message, sig) {
		a.addResult("Ed25519", false, "Signature verification", "Valid signature failed verification")
		return ErrSignatureInvalid
	}
	a.addResult("Ed25519", true, "Signature verification", "Sign and verify roundtrip successful")

	// Test tampered message detection.
	tampered := append([]byte{}, message...)
	tampered[0] ^= 0xFF
	if ed25519.Verify(pub, tampered, sig) {
		a.addResult("Ed25519", false, "Tampering detection", "Tampered message accepted")
		return ErrSignatureInvalid
	}
	a.addResult("Ed25519", true, "Tampering detection", "Tampered message correctly rejected")

	return nil
}

// AuditCurve25519 verifies Curve25519 implementation correctness.
func (a *CryptoAuditor) AuditCurve25519() error {
	// Generate two keypairs.
	var priv1, priv2 [32]byte
	if _, err := rand.Read(priv1[:]); err != nil {
		a.addResult("Curve25519", false, "Key generation", err.Error())
		return fmt.Errorf("Curve25519 key generation: %w", err)
	}
	if _, err := rand.Read(priv2[:]); err != nil {
		a.addResult("Curve25519", false, "Key generation", err.Error())
		return fmt.Errorf("Curve25519 key generation: %w", err)
	}

	// Clamp keys per spec.
	clampScalar(&priv1)
	clampScalar(&priv2)

	var pub1, pub2 [32]byte
	curve25519.ScalarBaseMult(&pub1, &priv1)
	curve25519.ScalarBaseMult(&pub2, &priv2)
	a.addResult("Curve25519", true, "Key generation", "Generated valid keypairs")

	// Test key exchange.
	var shared1, shared2 [32]byte
	curve25519.ScalarMult(&shared1, &priv1, &pub2)
	curve25519.ScalarMult(&shared2, &priv2, &pub1)

	if shared1 != shared2 {
		a.addResult("Curve25519", false, "Key exchange", "Shared secrets do not match")
		return fmt.Errorf("Curve25519 key exchange failed")
	}
	a.addResult("Curve25519", true, "Key exchange", "DH key exchange produces matching shared secrets")

	// Test non-zero shared secret.
	var zero [32]byte
	if shared1 == zero {
		a.addResult("Curve25519", false, "Shared secret", "Shared secret is all zeros")
		return ErrWeakKey
	}
	a.addResult("Curve25519", true, "Shared secret", "Non-zero shared secret")

	return nil
}

// AuditChaCha20Poly1305 verifies XChaCha20-Poly1305 implementation.
func (a *CryptoAuditor) AuditChaCha20Poly1305() error {
	key, err := a.generateChaChaKey()
	if err != nil {
		return err
	}

	cipher, err := a.createChaCha20Cipher(key)
	if err != nil {
		return err
	}

	plaintext, nonce, ciphertext, err := a.testChaChaRoundtrip(cipher)
	if err != nil {
		return err
	}

	return a.testChaChaAuthTag(cipher, nonce, ciphertext, plaintext)
}

// generateChaChaKey generates and validates a random key for XChaCha20-Poly1305.
func (a *CryptoAuditor) generateChaChaKey() ([]byte, error) {
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		a.addResult("XChaCha20-Poly1305", false, "Key generation", err.Error())
		return nil, err
	}
	a.addResult("XChaCha20-Poly1305", true, "Key generation", "Generated valid key")
	return key, nil
}

// createChaCha20Cipher creates and validates an XChaCha20-Poly1305 cipher.
func (a *CryptoAuditor) createChaCha20Cipher(key []byte) (cipher.AEAD, error) {
	c, err := chacha20poly1305.NewX(key)
	if err != nil {
		a.addResult("XChaCha20-Poly1305", false, "Cipher creation", err.Error())
		return nil, err
	}
	a.addResult("XChaCha20-Poly1305", true, "Cipher creation", "Created XChaCha20-Poly1305 cipher")
	return c, nil
}

// testChaChaRoundtrip tests encryption and decryption roundtrip.
func (a *CryptoAuditor) testChaChaRoundtrip(c cipher.AEAD) ([]byte, []byte, []byte, error) {
	plaintext := []byte("test message for ChaCha20-Poly1305")
	nonce := make([]byte, c.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		a.addResult("XChaCha20-Poly1305", false, "Nonce generation", err.Error())
		return nil, nil, nil, err
	}

	ciphertext := c.Seal(nil, nonce, plaintext, nil)
	decrypted, err := c.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		a.addResult("XChaCha20-Poly1305", false, "Decryption", err.Error())
		return nil, nil, nil, ErrDecryptionFailed
	}

	if string(decrypted) != string(plaintext) {
		a.addResult("XChaCha20-Poly1305", false, "Roundtrip", "Decrypted text doesn't match")
		return nil, nil, nil, ErrDecryptionFailed
	}
	a.addResult("XChaCha20-Poly1305", true, "Roundtrip", "Encryption/decryption roundtrip successful")
	return plaintext, nonce, ciphertext, nil
}

// testChaChaAuthTag verifies authentication tag integrity by testing tampering detection.
func (a *CryptoAuditor) testChaChaAuthTag(c cipher.AEAD, nonce, ciphertext, _ []byte) error {
	tampered := append([]byte{}, ciphertext...)
	tampered[0] ^= 0xFF
	if _, err := c.Open(nil, nonce, tampered, nil); err == nil {
		a.addResult("XChaCha20-Poly1305", false, "Tampering detection", "Tampered ciphertext accepted")
		return ErrDecryptionFailed
	}
	a.addResult("XChaCha20-Poly1305", true, "Tampering detection", "Tampered ciphertext correctly rejected")
	return nil
}

// AuditRandom verifies random source quality.
func (a *CryptoAuditor) AuditRandom() error {
	sample := make([]byte, 1024)
	if err := a.checkRandomReadable(sample); err != nil {
		return err
	}
	if err := a.checkRandomEntropy(sample); err != nil {
		return err
	}
	if err := a.checkRandomDistribution(sample); err != nil {
		return err
	}
	return nil
}

// checkRandomReadable verifies the random source can be read.
func (a *CryptoAuditor) checkRandomReadable(sample []byte) error {
	if _, err := rand.Read(sample); err != nil {
		a.addResult("Random", false, "Read", err.Error())
		return err
	}
	a.addResult("Random", true, "Read", "Random source is readable")
	return nil
}

// checkRandomEntropy verifies the sample has non-zero entropy.
func (a *CryptoAuditor) checkRandomEntropy(sample []byte) error {
	if isAllZero(sample) {
		a.addResult("Random", false, "Entropy", "Random source returned all zeros")
		return ErrRandomSourceWeak
	}
	a.addResult("Random", true, "Entropy", "Random source has entropy")
	return nil
}

// isAllZero checks if all bytes in the slice are zero.
func isAllZero(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// checkRandomDistribution verifies reasonable byte distribution.
func (a *CryptoAuditor) checkRandomDistribution(sample []byte) error {
	maxCount := computeMaxByteCount(sample)
	if maxCount > len(sample)/2 {
		a.addResult("Random", false, "Distribution", "Single byte value dominates sample")
		return ErrRandomSourceWeak
	}
	a.addResult("Random", true, "Distribution", "Reasonable byte distribution")
	return nil
}

// computeMaxByteCount returns the highest frequency of any byte value.
func computeMaxByteCount(data []byte) int {
	counts := make([]int, 256)
	for _, b := range data {
		counts[b]++
	}
	maxCount := 0
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}
	return maxCount
}

// RunFullAudit performs all cryptographic audits.
func (a *CryptoAuditor) RunFullAudit() ([]AuditResult, error) {
	var errs []error

	if err := a.AuditRandom(); err != nil {
		errs = append(errs, fmt.Errorf("random: %w", err))
	}

	if err := a.AuditEd25519(); err != nil {
		errs = append(errs, fmt.Errorf("Ed25519: %w", err))
	}

	if err := a.AuditCurve25519(); err != nil {
		errs = append(errs, fmt.Errorf("Curve25519: %w", err))
	}

	if err := a.AuditChaCha20Poly1305(); err != nil {
		errs = append(errs, fmt.Errorf("ChaCha20-Poly1305: %w", err))
	}

	if len(errs) > 0 {
		return a.results, fmt.Errorf("audit failed with %d errors: %v", len(errs), errs)
	}

	return a.results, nil
}

// Results returns all audit results.
func (a *CryptoAuditor) Results() []AuditResult {
	return a.results
}

// PassedCount returns the number of passed audits.
func (a *CryptoAuditor) PassedCount() int {
	count := 0
	for _, r := range a.results {
		if r.Passed {
			count++
		}
	}
	return count
}

// FailedCount returns the number of failed audits.
func (a *CryptoAuditor) FailedCount() int {
	return len(a.results) - a.PassedCount()
}

// addResult adds an audit result.
func (a *CryptoAuditor) addResult(category string, passed bool, description, details string) {
	a.results = append(a.results, AuditResult{
		Category:    category,
		Passed:      passed,
		Description: description,
		Details:     details,
	})
}

// clampScalar applies the Curve25519 scalar clamping operation.
func clampScalar(s *[32]byte) {
	s[0] &= 248
	s[31] &= 127
	s[31] |= 64
}

// KeyStorageAuditor verifies key storage security.
type KeyStorageAuditor struct {
	results []AuditResult
}

// NewKeyStorageAuditor creates a key storage auditor.
func NewKeyStorageAuditor() *KeyStorageAuditor {
	return &KeyStorageAuditor{
		results: make([]AuditResult, 0),
	}
}

// AuditKeyZeroization verifies that key material can be zeroed.
func (a *KeyStorageAuditor) AuditKeyZeroization() error {
	// Create a key and verify it can be zeroed.
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		a.addResult("Zeroization", false, "Key generation", err.Error())
		return err
	}

	// Store original hash.
	originalHash := sha256.Sum256(key)

	// Zero the key.
	ZeroBytes(key)

	// Verify it's zeroed.
	for i, b := range key {
		if b != 0 {
			a.addResult("Zeroization", false, "Zero check", fmt.Sprintf("byte %d not zeroed", i))
			return ErrKeyMaterialExposed
		}
	}

	// Verify hash changed.
	zeroHash := sha256.Sum256(key)
	if originalHash == zeroHash {
		a.addResult("Zeroization", false, "Hash check", "Hash unchanged after zeroization")
		return ErrKeyMaterialExposed
	}

	a.addResult("Zeroization", true, "Key zeroization", "Key material successfully zeroed")
	return nil
}

// Results returns all audit results.
func (a *KeyStorageAuditor) Results() []AuditResult {
	return a.results
}

// addResult adds an audit result.
func (a *KeyStorageAuditor) addResult(category string, passed bool, description, details string) {
	a.results = append(a.results, AuditResult{
		Category:    category,
		Passed:      passed,
		Description: description,
		Details:     details,
	})
}

// ZeroBytes zeroes a byte slice.
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// ZeroKey zeroes a 32-byte key.
func ZeroKey(k *[32]byte) {
	for i := range k {
		k[i] = 0
	}
}

// SecureReader wraps an io.Reader to provide secure random bytes.
type SecureReader struct {
	reader io.Reader
}

// NewSecureReader creates a SecureReader using crypto/rand.
func NewSecureReader() *SecureReader {
	return &SecureReader{reader: rand.Reader}
}

// Read fills b with random bytes.
func (r *SecureReader) Read(b []byte) (int, error) {
	return r.reader.Read(b)
}

// GenerateRandomBytes generates n random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
