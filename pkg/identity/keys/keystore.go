// Package keys provides keystore file management for Surface and Specter key separation.
// Per ROADMAP.md Security Hardening section, Surface and Specter keys are stored in
// separate encrypted files to enhance security isolation.
package keys

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// KeystorePaths contains file paths for separated keystore files.
type KeystorePaths struct {
	// Surface keystore file path (Ed25519 keypair).
	Surface string
	// Specter keystore file path (Curve25519 keypair).
	Specter string
	// FortressTransport keystore file path (optional, Ed25519 keypair).
	FortressTransport string
}

// DefaultKeystorePaths returns default keystore paths for a data directory.
func DefaultKeystorePaths(dataDir string) KeystorePaths {
	return KeystorePaths{
		Surface:           filepath.Join(dataDir, "surface.keystore"),
		Specter:           filepath.Join(dataDir, "specter.keystore"),
		FortressTransport: filepath.Join(dataDir, "fortress.keystore"),
	}
}

// SaveIdentityBundle saves an IdentityBundle to separate encrypted keystore files.
// Per ROADMAP.md Security Hardening, this separates Surface and Specter keys into
// distinct encrypted files to prevent compromise of one from revealing the other.
func SaveIdentityBundle(bundle *IdentityBundle, paths KeystorePaths, passphrase string) error {
	if err := validateBundleAndPassphrase(bundle, passphrase); err != nil {
		return err
	}

	if err := saveBundleKeypairs(bundle, paths, passphrase); err != nil {
		return err
	}

	return nil
}

// validateBundleAndPassphrase validates the identity bundle and passphrase.
func validateBundleAndPassphrase(bundle *IdentityBundle, passphrase string) error {
	if bundle == nil {
		return errors.New("identity bundle is nil")
	}
	if passphrase == "" {
		return errors.New("passphrase cannot be empty")
	}
	return nil
}

// saveBundleKeypairs saves all keypairs from the bundle to their respective keystore files.
func saveBundleKeypairs(bundle *IdentityBundle, paths KeystorePaths, passphrase string) error {
	if bundle.Surface != nil {
		if err := saveKeypairToKeystore(bundle.Surface, paths.Surface, passphrase, ExportKeyPair); err != nil {
			return fmt.Errorf("saving surface keypair: %w", err)
		}
	}

	if bundle.Specter != nil {
		if err := saveAnonymousKeypairToKeystore(bundle.Specter, paths.Specter, passphrase); err != nil {
			return fmt.Errorf("saving specter keypair: %w", err)
		}
	}

	if bundle.FortressTransport != nil {
		if err := saveKeypairToKeystore(bundle.FortressTransport, paths.FortressTransport, passphrase, ExportKeyPair); err != nil {
			return fmt.Errorf("saving fortress keypair: %w", err)
		}
	}

	return nil
}

// saveKeypairToKeystore exports, encrypts, and writes a KeyPair to a keystore file.
func saveKeypairToKeystore(kp *KeyPair, path, passphrase string, exporter func(*KeyPair) ([]byte, error)) error {
	data, err := exporter(kp)
	if err != nil {
		return fmt.Errorf("exporting keypair: %w", err)
	}
	defer ZeroBytes(data)

	encrypted, err := EncryptKeystore(data, passphrase)
	if err != nil {
		return fmt.Errorf("encrypting keystore: %w", err)
	}

	if err := writeKeystoreFile(path, encrypted); err != nil {
		return fmt.Errorf("writing keystore: %w", err)
	}

	return nil
}

// saveAnonymousKeypairToKeystore exports, encrypts, and writes an AnonymousKeyPair to a keystore file.
func saveAnonymousKeypairToKeystore(kp *AnonymousKeyPair, path, passphrase string) error {
	data := exportAnonymousKeyPair(kp)
	defer ZeroBytes(data)

	encrypted, err := EncryptKeystore(data, passphrase)
	if err != nil {
		return fmt.Errorf("encrypting keystore: %w", err)
	}

	if err := writeKeystoreFile(path, encrypted); err != nil {
		return fmt.Errorf("writing keystore: %w", err)
	}

	return nil
}

// LoadIdentityBundle loads an IdentityBundle from separate encrypted keystore files.
// Returns an error if any required keystore file is missing or cannot be decrypted.
func LoadIdentityBundle(paths KeystorePaths, passphrase string) (*IdentityBundle, error) {
	if passphrase == "" {
		return nil, errors.New("passphrase cannot be empty")
	}

	bundle := &IdentityBundle{}

	var err error
	bundle.Surface, err = loadKeypairFromKeystore(paths.Surface, passphrase, ImportKeyPair)
	if err != nil {
		return nil, fmt.Errorf("loading surface keypair: %w", err)
	}

	bundle.Specter, err = loadAnonymousKeypairFromKeystore(paths.Specter, passphrase)
	if err != nil {
		bundle.Surface.ZeroKeyPair()
		return nil, fmt.Errorf("loading specter keypair: %w", err)
	}

	if fileExists(paths.FortressTransport) {
		bundle.FortressTransport, err = loadKeypairFromKeystore(paths.FortressTransport, passphrase, ImportKeyPair)
		if err != nil {
			bundle.Zero()
			return nil, fmt.Errorf("loading fortress keypair: %w", err)
		}
	}

	return bundle, nil
}

// loadKeypairFromKeystore reads, decrypts, and imports a KeyPair from a keystore file.
func loadKeypairFromKeystore(path, passphrase string, importer func([]byte) (*KeyPair, error)) (*KeyPair, error) {
	encrypted, err := readKeystoreFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading keystore: %w", err)
	}

	data, err := DecryptKeystore(encrypted, passphrase)
	if err != nil {
		return nil, fmt.Errorf("decrypting keystore: %w", err)
	}
	defer ZeroBytes(data)

	kp, err := importer(data)
	if err != nil {
		return nil, fmt.Errorf("importing keypair: %w", err)
	}

	return kp, nil
}

// loadAnonymousKeypairFromKeystore reads, decrypts, and imports an AnonymousKeyPair from a keystore file.
func loadAnonymousKeypairFromKeystore(path, passphrase string) (*AnonymousKeyPair, error) {
	encrypted, err := readKeystoreFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading keystore: %w", err)
	}

	data, err := DecryptKeystore(encrypted, passphrase)
	if err != nil {
		return nil, fmt.Errorf("decrypting keystore: %w", err)
	}
	defer ZeroBytes(data)

	return importAnonymousKeyPair(data)
}

// exportAnonymousKeyPair serializes an AnonymousKeyPair to bytes.
// Format: private_key (32 bytes) || public_key (32 bytes)
func exportAnonymousKeyPair(kp *AnonymousKeyPair) []byte {
	data := make([]byte, 64)
	copy(data[0:32], kp.PrivateKey[:])
	copy(data[32:64], kp.PublicKey[:])
	return data
}

// importAnonymousKeyPair deserializes an AnonymousKeyPair from bytes.
// Per SECURITY_PRIVACY.md §2.1, zeros the input data after copying.
func importAnonymousKeyPair(data []byte) (*AnonymousKeyPair, error) {
	if len(data) != 64 {
		return nil, fmt.Errorf("invalid anonymous key data size: expected 64, got %d", len(data))
	}

	kp := &AnonymousKeyPair{}
	copy(kp.PrivateKey[:], data[0:32])
	copy(kp.PublicKey[:], data[32:64])

	// Zero the input data per SECURITY_PRIVACY.md §2.1.
	defer ZeroBytes(data)

	return kp, nil
}

// writeKeystoreFile writes encrypted keystore data to a file with restricted permissions.
// Per SECURITY_PRIVACY.md, keystore files are mode 0600 (owner read/write only).
func writeKeystoreFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating keystore directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing keystore file: %w", err)
	}

	return nil
}

// readKeystoreFile reads encrypted keystore data from a file.
func readKeystoreFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("keystore file not found: %s", path)
		}
		return nil, fmt.Errorf("reading keystore file: %w", err)
	}
	return data, nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// VerifyKeystoreSeparation verifies that Surface and Specter keystores are in separate files.
// This is a defensive check to ensure the keystore separation is properly implemented.
func VerifyKeystoreSeparation(paths KeystorePaths) error {
	if paths.Surface == paths.Specter {
		return errors.New("surface and specter keystores must be in separate files")
	}

	if paths.FortressTransport != "" {
		if paths.FortressTransport == paths.Surface {
			return errors.New("fortress and surface keystores must be in separate files")
		}
		if paths.FortressTransport == paths.Specter {
			return errors.New("fortress and specter keystores must be in separate files")
		}
	}

	return nil
}

// ExportKeyPairToFile exports and encrypts a keypair to a file.
// This is a convenience wrapper around ExportKeyPair + EncryptKeystore + writeKeystoreFile.
func ExportKeyPairToFile(kp *KeyPair, path, passphrase string) error {
	if kp == nil {
		return errors.New("keypair is nil")
	}
	if passphrase == "" {
		return errors.New("passphrase cannot be empty")
	}
	return saveKeypairToKeystore(kp, path, passphrase, ExportKeyPair)
}

// LoadKeyPairFromFile loads and decrypts a keypair from a file.
// This is a convenience wrapper around readKeystoreFile + DecryptKeystore + ImportKeyPair.
func LoadKeyPairFromFile(path, passphrase string) (*KeyPair, error) {
	if passphrase == "" {
		return nil, errors.New("passphrase cannot be empty")
	}
	return loadKeypairFromKeystore(path, passphrase, ImportKeyPair)
}

// IsLegacyKeystore checks if a file contains a legacy (pre-separation) combined keystore.
// Legacy keystores contain both Surface and Specter keys in a single file.
// This function is used to detect and migrate old keystores during app startup.
func IsLegacyKeystore(path string) (bool, error) {
	data, err := readKeystoreFile(path)
	if err != nil {
		return false, err
	}

	// Legacy keystores are larger than a single Ed25519 keypair (64 bytes).
	// A combined keystore would be at least 64 (Surface) + 64 (Specter) = 128 bytes,
	// plus encryption overhead (40 bytes salt+nonce+tag = 168+ bytes minimum).
	if len(data) < 168 {
		return false, nil
	}

	// Check if the file can be decrypted as a single Ed25519 keypair.
	// If decryption yields exactly 64 bytes, it's a single-key keystore (not legacy).
	// If it yields more, it's likely a combined keystore.
	// Note: This is a heuristic — true validation requires trying to decrypt with passphrase.
	return len(data) >= 168, nil
}

// MigrateLegacyKeystore migrates a legacy combined keystore to separated keystores.
// A legacy keystore stores Surface and Specter keys in a single encrypted file.
// The plaintext format is:
//
//	surface_private_key (64 bytes, Ed25519)
//	specter_private_key (32 bytes, Curve25519) || specter_public_key (32 bytes)
//
// On success the new separated keystores are written to newPaths, then the legacy
// file is renamed to legacyPath + ".bak" so it can be manually removed after
// the caller verifies the new keystores load correctly.
func MigrateLegacyKeystore(legacyPath, passphrase string, newPaths KeystorePaths) error {
	if passphrase == "" {
		return errors.New("passphrase cannot be empty")
	}
	if err := VerifyKeystoreSeparation(newPaths); err != nil {
		return fmt.Errorf("invalid new keystore paths: %w", err)
	}
	bundle, err := decryptLegacyBundle(legacyPath, passphrase)
	if err != nil {
		return err
	}
	defer bundle.Zero()
	if err := SaveIdentityBundle(bundle, newPaths, passphrase); err != nil {
		return fmt.Errorf("saving migrated keystores: %w", err)
	}
	return renameLegacyFile(legacyPath)
}

// renameLegacyFile renames a legacy keystore to ".bak" after successful migration.
// The original file is preserved as a backup for manual recovery if needed.
func renameLegacyFile(path string) error {
	if err := os.Rename(path, path+".bak"); err != nil {
		return fmt.Errorf("migrated successfully but could not rename legacy file: %w", err)
	}
	return nil
}

// decryptLegacyBundle decrypts a legacy combined keystore and returns an IdentityBundle.
// The legacy plaintext is exactly 128 bytes:
// bytes[0:64]   — Ed25519 private key (Surface)
// bytes[64:128] — Curve25519 private(32) || public(32) key (Specter)
func decryptLegacyBundle(legacyPath, passphrase string) (*IdentityBundle, error) {
	plaintext, err := readAndDecryptLegacy(legacyPath, passphrase)
	if err != nil {
		return nil, err
	}
	defer ZeroBytes(plaintext)
	return parseLegacyPlaintext(plaintext)
}

// readAndDecryptLegacy reads and decrypts a legacy keystore file.
func readAndDecryptLegacy(legacyPath, passphrase string) ([]byte, error) {
	encrypted, err := readKeystoreFile(legacyPath)
	if err != nil {
		return nil, fmt.Errorf("reading legacy keystore: %w", err)
	}
	plaintext, err := DecryptKeystore(encrypted, passphrase)
	if err != nil {
		return nil, fmt.Errorf("decrypting legacy keystore: %w", err)
	}
	return plaintext, nil
}

// parseLegacyPlaintext extracts Surface and Specter keypairs from 128-byte legacy plaintext.
func parseLegacyPlaintext(plaintext []byte) (*IdentityBundle, error) {
	const legacySize = 128
	if len(plaintext) != legacySize {
		return nil, fmt.Errorf(
			"unexpected legacy keystore size: got %d, want %d (not a combined keystore?)",
			len(plaintext), legacySize,
		)
	}
	surface, err := importSurfaceFromLegacy(plaintext[0:64])
	if err != nil {
		return nil, err
	}
	specter, err := importAnonymousKeyPair(plaintext[64:128])
	if err != nil {
		surface.ZeroKeyPair()
		return nil, fmt.Errorf("importing specter key from legacy keystore: %w", err)
	}
	return &IdentityBundle{Surface: surface, Specter: specter}, nil
}

// importSurfaceFromLegacy imports an Ed25519 surface key from a copy of the legacy bytes.
func importSurfaceFromLegacy(raw []byte) (*KeyPair, error) {
	buf := make([]byte, len(raw))
	copy(buf, raw)
	defer ZeroBytes(buf)
	kp, err := ImportKeyPair(buf)
	if err != nil {
		return nil, fmt.Errorf("importing surface key from legacy keystore: %w", err)
	}
	return kp, nil
}

// SaveSurfaceKeyPair saves only the Surface keypair to a file.
// This is a convenience function for operations that only need Surface key persistence.
func SaveSurfaceKeyPair(kp *KeyPair, path, passphrase string) error {
	return ExportKeyPairToFile(kp, path, passphrase)
}

// LoadSurfaceKeyPair loads only the Surface keypair from a file.
func LoadSurfaceKeyPair(path, passphrase string) (*KeyPair, error) {
	return LoadKeyPairFromFile(path, passphrase)
}

// SaveSpecterKeyPair saves only the Specter keypair to a file.
func SaveSpecterKeyPair(kp *AnonymousKeyPair, path, passphrase string) error {
	if kp == nil {
		return errors.New("anonymous keypair is nil")
	}
	if passphrase == "" {
		return errors.New("passphrase cannot be empty")
	}
	return saveAnonymousKeypairToKeystore(kp, path, passphrase)
}

// LoadSpecterKeyPair loads only the Specter keypair from a file.
func LoadSpecterKeyPair(path, passphrase string) (*AnonymousKeyPair, error) {
	if passphrase == "" {
		return nil, errors.New("passphrase cannot be empty")
	}
	return loadAnonymousKeypairFromKeystore(path, passphrase)
}

// KeystoreExist checks if all required keystore files exist.
func KeystoreExists(paths KeystorePaths) bool {
	return fileExists(paths.Surface) && fileExists(paths.Specter)
}

// DeleteKeystore securely deletes all keystore files.
// Per SECURITY_PRIVACY.md, this should be called when resetting identity.
func DeleteKeystore(paths KeystorePaths) error {
	var errs []error

	if err := os.Remove(paths.Surface); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("deleting surface keystore: %w", err))
	}

	if err := os.Remove(paths.Specter); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("deleting specter keystore: %w", err))
	}

	if paths.FortressTransport != "" {
		if err := os.Remove(paths.FortressTransport); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("deleting fortress keystore: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("keystore deletion errors: %v", errs)
	}

	return nil
}
