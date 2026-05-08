package keys

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultKeystorePaths(t *testing.T) {
	dataDir := "/test/data"
	paths := DefaultKeystorePaths(dataDir)

	expectedSurface := filepath.Join(dataDir, "surface.keystore")
	expectedSpecter := filepath.Join(dataDir, "specter.keystore")
	expectedFortress := filepath.Join(dataDir, "fortress.keystore")

	if paths.Surface != expectedSurface {
		t.Errorf("Surface path = %s, want %s", paths.Surface, expectedSurface)
	}
	if paths.Specter != expectedSpecter {
		t.Errorf("Specter path = %s, want %s", paths.Specter, expectedSpecter)
	}
	if paths.FortressTransport != expectedFortress {
		t.Errorf("FortressTransport path = %s, want %s", paths.FortressTransport, expectedFortress)
	}
}

func TestSaveAndLoadIdentityBundle(t *testing.T) {
	// Create temp directory for keystores.
	tempDir := t.TempDir()
	paths := DefaultKeystorePaths(tempDir)
	passphrase := "test-passphrase-12345"

	// Generate bundle.
	bundle, err := GenerateIdentityBundle()
	if err != nil {
		t.Fatalf("GenerateIdentityBundle() error = %v", err)
	}
	defer bundle.Zero()

	// Save original public keys for comparison.
	origSurfacePub := make([]byte, len(bundle.Surface.PublicKey))
	copy(origSurfacePub, bundle.Surface.PublicKey)
	origSpecterPub := bundle.Specter.PublicKey

	// Save bundle to separate files.
	if err := SaveIdentityBundle(bundle, paths, passphrase); err != nil {
		t.Fatalf("SaveIdentityBundle() error = %v", err)
	}

	// Verify files exist.
	if !fileExists(paths.Surface) {
		t.Error("Surface keystore file not created")
	}
	if !fileExists(paths.Specter) {
		t.Error("Specter keystore file not created")
	}

	// Load bundle from files.
	loaded, err := LoadIdentityBundle(paths, passphrase)
	if err != nil {
		t.Fatalf("LoadIdentityBundle() error = %v", err)
	}
	defer loaded.Zero()

	// Verify Surface keypair matches.
	if !bytes.Equal(loaded.Surface.PublicKey, origSurfacePub) {
		t.Error("Loaded Surface public key does not match original")
	}

	// Verify Specter keypair matches.
	if loaded.Specter.PublicKey != origSpecterPub {
		t.Error("Loaded Specter public key does not match original")
	}

	// Verify signing works with loaded keys.
	msg := []byte("test message")
	sig := loaded.Surface.Sign(msg)
	if !Verify(loaded.Surface.PublicKey, msg, sig) {
		t.Error("Loaded Surface keypair signature verification failed")
	}
}

func TestSaveAndLoadIdentityBundleWithFortress(t *testing.T) {
	tempDir := t.TempDir()
	paths := DefaultKeystorePaths(tempDir)
	passphrase := "fortress-test-pass"

	// Generate bundle with Fortress transport key.
	bundle, err := GenerateIdentityBundleWithFortress()
	if err != nil {
		t.Fatalf("GenerateIdentityBundleWithFortress() error = %v", err)
	}
	defer bundle.Zero()

	// Save bundle.
	if err := SaveIdentityBundle(bundle, paths, passphrase); err != nil {
		t.Fatalf("SaveIdentityBundle() error = %v", err)
	}

	// Verify all three files exist.
	if !fileExists(paths.Surface) {
		t.Error("Surface keystore not created")
	}
	if !fileExists(paths.Specter) {
		t.Error("Specter keystore not created")
	}
	if !fileExists(paths.FortressTransport) {
		t.Error("FortressTransport keystore not created")
	}

	// Load bundle.
	loaded, err := LoadIdentityBundle(paths, passphrase)
	if err != nil {
		t.Fatalf("LoadIdentityBundle() error = %v", err)
	}
	defer loaded.Zero()

	// Verify all three keypairs loaded.
	if loaded.Surface == nil {
		t.Error("Surface keypair not loaded")
	}
	if loaded.Specter == nil {
		t.Error("Specter keypair not loaded")
	}
	if loaded.FortressTransport == nil {
		t.Error("FortressTransport keypair not loaded")
	}
}

func TestLoadIdentityBundleWrongPassphrase(t *testing.T) {
	tempDir := t.TempDir()
	paths := DefaultKeystorePaths(tempDir)
	correctPass := "correct-passphrase"
	wrongPass := "wrong-passphrase"

	// Generate and save bundle.
	bundle, err := GenerateIdentityBundle()
	if err != nil {
		t.Fatalf("GenerateIdentityBundle() error = %v", err)
	}
	defer bundle.Zero()

	if err := SaveIdentityBundle(bundle, paths, correctPass); err != nil {
		t.Fatalf("SaveIdentityBundle() error = %v", err)
	}

	// Attempt to load with wrong passphrase.
	_, err = LoadIdentityBundle(paths, wrongPass)
	if err == nil {
		t.Error("LoadIdentityBundle() with wrong passphrase should fail")
	}
}

func TestLoadIdentityBundleMissingFiles(t *testing.T) {
	tempDir := t.TempDir()
	paths := DefaultKeystorePaths(tempDir)
	passphrase := "test-pass"

	// Attempt to load non-existent keystore.
	_, err := LoadIdentityBundle(paths, passphrase)
	if err == nil {
		t.Error("LoadIdentityBundle() with missing files should fail")
	}
}

func TestVerifyKeystoreSeparation(t *testing.T) {
	// Valid paths (all different).
	validPaths := KeystorePaths{
		Surface:           "/data/surface.keystore",
		Specter:           "/data/specter.keystore",
		FortressTransport: "/data/fortress.keystore",
	}
	if err := VerifyKeystoreSeparation(validPaths); err != nil {
		t.Errorf("VerifyKeystoreSeparation() with valid paths error = %v", err)
	}

	// Invalid: Surface and Specter same file.
	invalidPaths1 := KeystorePaths{
		Surface: "/data/combined.keystore",
		Specter: "/data/combined.keystore",
	}
	if err := VerifyKeystoreSeparation(invalidPaths1); err == nil {
		t.Error("VerifyKeystoreSeparation() should fail when Surface == Specter")
	}

	// Invalid: Fortress and Surface same file.
	invalidPaths2 := KeystorePaths{
		Surface:           "/data/surface.keystore",
		Specter:           "/data/specter.keystore",
		FortressTransport: "/data/surface.keystore",
	}
	if err := VerifyKeystoreSeparation(invalidPaths2); err == nil {
		t.Error("VerifyKeystoreSeparation() should fail when Fortress == Surface")
	}

	// Invalid: Fortress and Specter same file.
	invalidPaths3 := KeystorePaths{
		Surface:           "/data/surface.keystore",
		Specter:           "/data/specter.keystore",
		FortressTransport: "/data/specter.keystore",
	}
	if err := VerifyKeystoreSeparation(invalidPaths3); err == nil {
		t.Error("VerifyKeystoreSeparation() should fail when Fortress == Specter")
	}
}

func TestSaveAndLoadSurfaceKeyPair(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "surface.keystore")
	passphrase := "surface-pass"

	// Generate keypair.
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	defer kp.ZeroKeyPair()

	origPub := make([]byte, len(kp.PublicKey))
	copy(origPub, kp.PublicKey)

	// Save keypair.
	if err := SaveSurfaceKeyPair(kp, path, passphrase); err != nil {
		t.Fatalf("SaveSurfaceKeyPair() error = %v", err)
	}

	// Load keypair.
	loaded, err := LoadSurfaceKeyPair(path, passphrase)
	if err != nil {
		t.Fatalf("LoadSurfaceKeyPair() error = %v", err)
	}
	defer loaded.ZeroKeyPair()

	// Verify match.
	if !bytes.Equal(loaded.PublicKey, origPub) {
		t.Error("Loaded Surface public key does not match original")
	}
}

func TestSaveAndLoadSpecterKeyPair(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "specter.keystore")
	passphrase := "specter-pass"

	// Generate anonymous keypair.
	kp, err := GenerateAnonymousKeyPair()
	if err != nil {
		t.Fatalf("GenerateAnonymousKeyPair() error = %v", err)
	}
	defer kp.ZeroAnonymousKeyPair()

	origPub := kp.PublicKey

	// Save keypair.
	if err := SaveSpecterKeyPair(kp, path, passphrase); err != nil {
		t.Fatalf("SaveSpecterKeyPair() error = %v", err)
	}

	// Load keypair.
	loaded, err := LoadSpecterKeyPair(path, passphrase)
	if err != nil {
		t.Fatalf("LoadSpecterKeyPair() error = %v", err)
	}
	defer loaded.ZeroAnonymousKeyPair()

	// Verify match.
	if loaded.PublicKey != origPub {
		t.Error("Loaded Specter public key does not match original")
	}
}

func TestKeystoreExists(t *testing.T) {
	tempDir := t.TempDir()
	paths := DefaultKeystorePaths(tempDir)

	// Initially should not exist.
	if KeystoreExists(paths) {
		t.Error("KeystoreExists() should return false for non-existent files")
	}

	// Create Surface keystore only.
	kp, _ := GenerateKeyPair()
	SaveSurfaceKeyPair(kp, paths.Surface, "pass")
	kp.ZeroKeyPair()

	// Should still be false (Specter missing).
	if KeystoreExists(paths) {
		t.Error("KeystoreExists() should return false when Specter is missing")
	}

	// Create Specter keystore.
	akp, _ := GenerateAnonymousKeyPair()
	SaveSpecterKeyPair(akp, paths.Specter, "pass")
	akp.ZeroAnonymousKeyPair()

	// Now should be true.
	if !KeystoreExists(paths) {
		t.Error("KeystoreExists() should return true when both files exist")
	}
}

func TestDeleteKeystore(t *testing.T) {
	tempDir := t.TempDir()
	paths := DefaultKeystorePaths(tempDir)
	passphrase := "delete-test"

	// Create keystore files.
	bundle, err := GenerateIdentityBundleWithFortress()
	if err != nil {
		t.Fatalf("GenerateIdentityBundleWithFortress() error = %v", err)
	}
	defer bundle.Zero()

	if err := SaveIdentityBundle(bundle, paths, passphrase); err != nil {
		t.Fatalf("SaveIdentityBundle() error = %v", err)
	}

	// Verify files exist.
	if !fileExists(paths.Surface) || !fileExists(paths.Specter) || !fileExists(paths.FortressTransport) {
		t.Fatal("Keystore files not created")
	}

	// Delete keystore.
	if err := DeleteKeystore(paths); err != nil {
		t.Fatalf("DeleteKeystore() error = %v", err)
	}

	// Verify files deleted.
	if fileExists(paths.Surface) {
		t.Error("Surface keystore not deleted")
	}
	if fileExists(paths.Specter) {
		t.Error("Specter keystore not deleted")
	}
	if fileExists(paths.FortressTransport) {
		t.Error("FortressTransport keystore not deleted")
	}
}

func TestSaveIdentityBundleEmptyPassphrase(t *testing.T) {
	tempDir := t.TempDir()
	paths := DefaultKeystorePaths(tempDir)

	bundle, err := GenerateIdentityBundle()
	if err != nil {
		t.Fatalf("GenerateIdentityBundle() error = %v", err)
	}
	defer bundle.Zero()

	// Empty passphrase should fail.
	if err := SaveIdentityBundle(bundle, paths, ""); err == nil {
		t.Error("SaveIdentityBundle() with empty passphrase should fail")
	}
}

func TestLoadIdentityBundleEmptyPassphrase(t *testing.T) {
	tempDir := t.TempDir()
	paths := DefaultKeystorePaths(tempDir)

	// Empty passphrase should fail.
	if _, err := LoadIdentityBundle(paths, ""); err == nil {
		t.Error("LoadIdentityBundle() with empty passphrase should fail")
	}
}

func TestSaveIdentityBundleNilBundle(t *testing.T) {
	tempDir := t.TempDir()
	paths := DefaultKeystorePaths(tempDir)

	// Nil bundle should fail.
	if err := SaveIdentityBundle(nil, paths, "pass"); err == nil {
		t.Error("SaveIdentityBundle() with nil bundle should fail")
	}
}

func TestExportImportAnonymousKeyPair(t *testing.T) {
	// Generate anonymous keypair.
	kp, err := GenerateAnonymousKeyPair()
	if err != nil {
		t.Fatalf("GenerateAnonymousKeyPair() error = %v", err)
	}
	defer kp.ZeroAnonymousKeyPair()

	origPriv := kp.PrivateKey
	origPub := kp.PublicKey

	// Export keypair.
	data := exportAnonymousKeyPair(kp)
	defer ZeroBytes(data)

	// Verify export format (32 bytes private + 32 bytes public = 64 bytes).
	if len(data) != 64 {
		t.Errorf("exported data length = %d, want 64", len(data))
	}

	// Import keypair.
	imported, err := importAnonymousKeyPair(data)
	if err != nil {
		t.Fatalf("importAnonymousKeyPair() error = %v", err)
	}
	defer imported.ZeroAnonymousKeyPair()

	// Verify match.
	if imported.PrivateKey != origPriv {
		t.Error("Imported private key does not match original")
	}
	if imported.PublicKey != origPub {
		t.Error("Imported public key does not match original")
	}
}

func TestImportAnonymousKeyPairInvalidSize(t *testing.T) {
	// Too short.
	shortData := make([]byte, 32)
	if _, err := importAnonymousKeyPair(shortData); err == nil {
		t.Error("importAnonymousKeyPair() with 32-byte data should fail")
	}

	// Too long.
	longData := make([]byte, 128)
	if _, err := importAnonymousKeyPair(longData); err == nil {
		t.Error("importAnonymousKeyPair() with 128-byte data should fail")
	}
}

func TestKeystoreFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "test.keystore")

	// Write keystore file.
	data := []byte("test data")
	if err := writeKeystoreFile(path, data); err != nil {
		t.Fatalf("writeKeystoreFile() error = %v", err)
	}

	// Check file permissions (should be 0600).
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}

	perm := info.Mode().Perm()
	expectedPerm := os.FileMode(0o600)
	if perm != expectedPerm {
		t.Errorf("file permissions = %o, want %o", perm, expectedPerm)
	}
}

func TestKeystoreDirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "nested", "dir", "keystore.bin")

	data := []byte("test")
	if err := writeKeystoreFile(nestedPath, data); err != nil {
		t.Fatalf("writeKeystoreFile() with nested path error = %v", err)
	}

	// Verify directory was created.
	if !fileExists(filepath.Dir(nestedPath)) {
		t.Error("Nested directory not created")
	}

	// Verify file was written.
	if !fileExists(nestedPath) {
		t.Error("Keystore file not created in nested directory")
	}
}

func TestIsLegacyKeystore(t *testing.T) {
	tempDir := t.TempDir()

	// Create a small file (single keypair, not legacy).
	singlePath := filepath.Join(tempDir, "single.keystore")
	kp, _ := GenerateKeyPair()
	SaveSurfaceKeyPair(kp, singlePath, "pass")
	kp.ZeroKeyPair()

	isLegacy, err := IsLegacyKeystore(singlePath)
	if err != nil {
		t.Fatalf("IsLegacyKeystore() error = %v", err)
	}
	if isLegacy {
		t.Error("Single keypair keystore detected as legacy")
	}

	// Create a large file (simulating combined keystore).
	largePath := filepath.Join(tempDir, "large.keystore")
	largeData := make([]byte, 200) // >168 bytes minimum for legacy
	os.WriteFile(largePath, largeData, 0o600)

	isLegacy, err = IsLegacyKeystore(largePath)
	if err != nil {
		t.Fatalf("IsLegacyKeystore() error = %v", err)
	}
	if !isLegacy {
		t.Error("Large keystore not detected as legacy")
	}
}

func TestMigrateLegacyKeystoreNotImplemented(t *testing.T) {
	tempDir := t.TempDir()
	legacyPath := filepath.Join(tempDir, "legacy.keystore")
	newPaths := DefaultKeystorePaths(tempDir)

	// Should return "not implemented" error.
	err := MigrateLegacyKeystore(legacyPath, "pass", newPaths)
	if err == nil {
		t.Error("MigrateLegacyKeystore() should return error (not implemented)")
	}
}

func TestLoadIdentityBundleWithoutFortress(t *testing.T) {
	tempDir := t.TempDir()
	paths := DefaultKeystorePaths(tempDir)
	passphrase := "test-pass"

	// Generate bundle without Fortress.
	bundle, err := GenerateIdentityBundle()
	if err != nil {
		t.Fatalf("GenerateIdentityBundle() error = %v", err)
	}
	defer bundle.Zero()

	// Save bundle (no Fortress file will be created).
	if err := SaveIdentityBundle(bundle, paths, passphrase); err != nil {
		t.Fatalf("SaveIdentityBundle() error = %v", err)
	}

	// Verify Fortress file does not exist.
	if fileExists(paths.FortressTransport) {
		t.Error("FortressTransport keystore should not be created for non-Fortress bundle")
	}

	// Load bundle (should succeed even without Fortress file).
	loaded, err := LoadIdentityBundle(paths, passphrase)
	if err != nil {
		t.Fatalf("LoadIdentityBundle() error = %v", err)
	}
	defer loaded.Zero()

	// Verify Fortress is nil.
	if loaded.FortressTransport != nil {
		t.Error("Loaded FortressTransport should be nil")
	}

	// Verify Surface and Specter loaded correctly.
	if loaded.Surface == nil {
		t.Error("Surface keypair not loaded")
	}
	if loaded.Specter == nil {
		t.Error("Specter keypair not loaded")
	}
}

func TestKeySeparationSecurity(t *testing.T) {
	tempDir := t.TempDir()
	paths := DefaultKeystorePaths(tempDir)
	passphrase := "security-test"

	// Generate bundle.
	bundle, err := GenerateIdentityBundle()
	if err != nil {
		t.Fatalf("GenerateIdentityBundle() error = %v", err)
	}
	defer bundle.Zero()

	// Save bundle.
	if err := SaveIdentityBundle(bundle, paths, passphrase); err != nil {
		t.Fatalf("SaveIdentityBundle() error = %v", err)
	}

	// Read raw Surface keystore.
	surfaceData, err := os.ReadFile(paths.Surface)
	if err != nil {
		t.Fatalf("Reading Surface keystore error = %v", err)
	}

	// Read raw Specter keystore.
	specterData, err := os.ReadFile(paths.Specter)
	if err != nil {
		t.Fatalf("Reading Specter keystore error = %v", err)
	}

	// Verify files are different (separation enforced).
	if bytes.Equal(surfaceData, specterData) {
		t.Error("Surface and Specter keystore files have identical content (separation violated)")
	}

	// Verify neither file contains plaintext key material.
	// Check that Surface file doesn't contain Specter public key bytes and vice versa.
	if bytes.Contains(surfaceData, bundle.Specter.PublicKey[:]) {
		t.Error("Surface keystore contains Specter public key (isolation violated)")
	}
	if bytes.Contains(specterData, bundle.Surface.PublicKey) {
		t.Error("Specter keystore contains Surface public key (isolation violated)")
	}
}

// buildLegacyKeystoreFixture creates a legacy combined keystore (Surface+Specter in
// one encrypted file) and returns the fixture path, the original surface/specter public
// keys for verification, and any error.
func buildLegacyKeystoreFixture(t *testing.T, dir, passphrase string) (path string, surfPub []byte, specPub [32]byte) {
	t.Helper()

	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	defer kp.ZeroKeyPair()

	akp, err := GenerateAnonymousKeyPair()
	if err != nil {
		t.Fatalf("GenerateAnonymousKeyPair: %v", err)
	}
	defer akp.ZeroAnonymousKeyPair()

	surfPub = make([]byte, len(kp.PublicKey))
	copy(surfPub, kp.PublicKey)
	specPub = akp.PublicKey

	// Construct legacy 128-byte plaintext.
	plain := make([]byte, 128)
	copy(plain[0:64], kp.PrivateKey)
	copy(plain[64:96], akp.PrivateKey[:])
	copy(plain[96:128], akp.PublicKey[:])
	defer ZeroBytes(plain)

	encrypted, err := EncryptKeystore(plain, passphrase)
	if err != nil {
		t.Fatalf("EncryptKeystore: %v", err)
	}

	path = filepath.Join(dir, "legacy.keystore")
	if err := writeKeystoreFile(path, encrypted); err != nil {
		t.Fatalf("writeKeystoreFile: %v", err)
	}

	return path, surfPub, specPub
}

func TestMigrateLegacyKeystore(t *testing.T) {
	tempDir := t.TempDir()
	passphrase := "migrate-test-pass"

	legacyPath, origSurfPub, origSpecPub := buildLegacyKeystoreFixture(t, tempDir, passphrase)

	newPaths := KeystorePaths{
		Surface: filepath.Join(tempDir, "surface.keystore"),
		Specter: filepath.Join(tempDir, "specter.keystore"),
	}

	if err := MigrateLegacyKeystore(legacyPath, passphrase, newPaths); err != nil {
		t.Fatalf("MigrateLegacyKeystore() error = %v", err)
	}

	// Legacy file should be renamed.
	if fileExists(legacyPath) {
		t.Error("Legacy keystore file should have been renamed after migration")
	}
	if !fileExists(legacyPath + ".bak") {
		t.Error("Legacy keystore backup (.bak) should exist after migration")
	}

	// New keystores should load correctly.
	loaded, err := LoadIdentityBundle(newPaths, passphrase)
	if err != nil {
		t.Fatalf("LoadIdentityBundle() after migration error = %v", err)
	}
	defer loaded.Zero()

	if !bytes.Equal(loaded.Surface.PublicKey, origSurfPub) {
		t.Error("Migrated Surface public key does not match original")
	}
	if loaded.Specter.PublicKey != origSpecPub {
		t.Error("Migrated Specter public key does not match original")
	}
}

func TestMigrateLegacyKeystore_WrongPassphrase(t *testing.T) {
	tempDir := t.TempDir()
	legacyPath, _, _ := buildLegacyKeystoreFixture(t, tempDir, "correct-pass")

	newPaths := KeystorePaths{
		Surface: filepath.Join(tempDir, "surface.keystore"),
		Specter: filepath.Join(tempDir, "specter.keystore"),
	}

	if err := MigrateLegacyKeystore(legacyPath, "wrong-pass", newPaths); err == nil {
		t.Error("MigrateLegacyKeystore() should fail with wrong passphrase")
	}
}

func TestMigrateLegacyKeystore_EmptyPassphrase(t *testing.T) {
	tempDir := t.TempDir()
	legacyPath := filepath.Join(tempDir, "legacy.keystore")
	// File doesn't even need to exist — empty passphrase check is first.
	newPaths := DefaultKeystorePaths(tempDir)

	if err := MigrateLegacyKeystore(legacyPath, "", newPaths); err == nil {
		t.Error("MigrateLegacyKeystore() should fail with empty passphrase")
	}
}

func TestMigrateLegacyKeystore_NotLegacyFormat(t *testing.T) {
	tempDir := t.TempDir()
	passphrase := "test-pass"

	// Save a single-key keystore (Surface only) — plaintext is 64 bytes, not 128.
	kp, _ := GenerateKeyPair()
	defer kp.ZeroKeyPair()
	legacyPath := filepath.Join(tempDir, "single.keystore")
	if err := SaveSurfaceKeyPair(kp, legacyPath, passphrase); err != nil {
		t.Fatalf("SaveSurfaceKeyPair: %v", err)
	}

	newPaths := KeystorePaths{
		Surface: filepath.Join(tempDir, "surface.keystore"),
		Specter: filepath.Join(tempDir, "specter.keystore"),
	}

	// Should fail because plaintext is 64 bytes, not the expected 128.
	if err := MigrateLegacyKeystore(legacyPath, passphrase, newPaths); err == nil {
		t.Error("MigrateLegacyKeystore() should fail for non-combined keystore")
	}
}

