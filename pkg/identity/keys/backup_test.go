package keys

import (
	"bytes"
	"testing"
)

func TestGenerateBackup(t *testing.T) {
	kp, backup, err := GenerateBackup()
	if err != nil {
		t.Fatalf("GenerateBackup failed: %v", err)
	}

	if kp == nil {
		t.Fatal("keypair is nil")
	}
	if backup == nil {
		t.Fatal("backup is nil")
	}
	if backup.Mnemonic == "" {
		t.Fatal("mnemonic is empty")
	}

	// Verify mnemonic has 24 words.
	words := countWords(backup.Mnemonic)
	if words != 24 {
		t.Errorf("expected 24 words, got %d", words)
	}

	// Verify mnemonic is valid.
	if !ValidateMnemonic(backup.Mnemonic) {
		t.Error("generated mnemonic is not valid")
	}
}

func TestRestoreFromMnemonic(t *testing.T) {
	// Generate a keypair with backup.
	original, backup, err := GenerateBackup()
	if err != nil {
		t.Fatalf("GenerateBackup failed: %v", err)
	}

	// Restore from mnemonic.
	restored, err := RestoreFromMnemonic(backup.Mnemonic)
	if err != nil {
		t.Fatalf("RestoreFromMnemonic failed: %v", err)
	}

	// Verify restored keypair matches original.
	if !bytes.Equal(original.PublicKey, restored.PublicKey) {
		t.Error("restored public key does not match original")
	}
	if !bytes.Equal(original.PrivateKey, restored.PrivateKey) {
		t.Error("restored private key does not match original")
	}
}

func TestRestoreFromMnemonicInvalid(t *testing.T) {
	_, err := RestoreFromMnemonic("invalid mnemonic phrase")
	if err == nil {
		t.Error("expected error for invalid mnemonic")
	}
}

func TestValidateMnemonic(t *testing.T) {
	tests := []struct {
		name     string
		mnemonic string
		valid    bool
	}{
		{
			name:     "empty",
			mnemonic: "",
			valid:    false,
		},
		{
			name:     "too short",
			mnemonic: "abandon abandon abandon",
			valid:    false,
		},
		{
			name:     "invalid words",
			mnemonic: "invalid words here that are not bip39",
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateMnemonic(tt.mnemonic); got != tt.valid {
				t.Errorf("ValidateMnemonic() = %v, want %v", got, tt.valid)
			}
		})
	}

	// Generate and validate a real mnemonic.
	_, backup, err := GenerateBackup()
	if err != nil {
		t.Fatalf("GenerateBackup failed: %v", err)
	}
	if !ValidateMnemonic(backup.Mnemonic) {
		t.Error("generated mnemonic should be valid")
	}
}

func TestExportImportKeyPair(t *testing.T) {
	// Generate a keypair.
	original, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Export.
	data, err := ExportKeyPair(original)
	if err != nil {
		t.Fatalf("ExportKeyPair failed: %v", err)
	}

	// Import.
	imported, err := ImportKeyPair(data)
	if err != nil {
		t.Fatalf("ImportKeyPair failed: %v", err)
	}

	// Verify.
	if !bytes.Equal(original.PublicKey, imported.PublicKey) {
		t.Error("imported public key does not match")
	}
	if !bytes.Equal(original.PrivateKey, imported.PrivateKey) {
		t.Error("imported private key does not match")
	}
}

func TestExportKeyPairNil(t *testing.T) {
	_, err := ExportKeyPair(nil)
	if err == nil {
		t.Error("expected error for nil keypair")
	}
}

func TestImportKeyPairInvalidSize(t *testing.T) {
	_, err := ImportKeyPair([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error for invalid key size")
	}
}

func TestBackupSigningRoundTrip(t *testing.T) {
	// Generate keypair with backup.
	original, backup, err := GenerateBackup()
	if err != nil {
		t.Fatalf("GenerateBackup failed: %v", err)
	}

	// Sign a message with original.
	message := []byte("test message for signing")
	signature := original.Sign(message)

	// Restore keypair from mnemonic.
	restored, err := RestoreFromMnemonic(backup.Mnemonic)
	if err != nil {
		t.Fatalf("RestoreFromMnemonic failed: %v", err)
	}

	// Verify signature with restored public key.
	if !Verify(restored.PublicKey, message, signature) {
		t.Error("signature verification failed with restored key")
	}

	// Sign with restored and verify with original.
	signature2 := restored.Sign(message)
	if !Verify(original.PublicKey, message, signature2) {
		t.Error("signature verification failed with original key")
	}
}

// TestImportKeyPairFromFile validates encrypted key file import.
func TestImportKeyPairFromFile(t *testing.T) {
	// Generate a keypair.
	original, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Export and encrypt.
	exported, err := ExportKeyPair(original)
	if err != nil {
		t.Fatalf("ExportKeyPair failed: %v", err)
	}

	passphrase := "test-passphrase-12345"
	encrypted, err := EncryptKeystore(exported, passphrase)
	if err != nil {
		t.Fatalf("EncryptKeystore failed: %v", err)
	}

	// Import from encrypted file data.
	imported, err := ImportKeyPairFromFile(encrypted, passphrase)
	if err != nil {
		t.Fatalf("ImportKeyPairFromFile failed: %v", err)
	}

	// Verify keys match.
	if !bytes.Equal(original.PublicKey, imported.PublicKey) {
		t.Error("imported public key does not match")
	}
	if !bytes.Equal(original.PrivateKey, imported.PrivateKey) {
		t.Error("imported private key does not match")
	}
}

// TestImportKeyPairFromFileWrongPassphrase validates error on wrong passphrase.
func TestImportKeyPairFromFileWrongPassphrase(t *testing.T) {
	// Generate and encrypt a keypair.
	original, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	exported, err := ExportKeyPair(original)
	if err != nil {
		t.Fatalf("ExportKeyPair failed: %v", err)
	}

	encrypted, err := EncryptKeystore(exported, "correct-passphrase")
	if err != nil {
		t.Fatalf("EncryptKeystore failed: %v", err)
	}

	// Attempt import with wrong passphrase.
	_, err = ImportKeyPairFromFile(encrypted, "wrong-passphrase")
	if err == nil {
		t.Error("expected error with wrong passphrase")
	}
}

func countWords(s string) int {
	if s == "" {
		return 0
	}
	count := 1
	for _, c := range s {
		if c == ' ' {
			count++
		}
	}
	return count
}
