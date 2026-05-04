// Package keys provides Ed25519 and Curve25519 keypair generation, signing, and
// verification for MURMUR's identity system.
// This file implements BIP-39 mnemonic backup and recovery per ROADMAP.md Priority 2.
package keys

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	"github.com/tyler-smith/go-bip39"
)

// MnemonicBitSize is the entropy size for 24-word mnemonics (256 bits).
const MnemonicBitSize = 256

// Backup represents a mnemonic backup of a keypair.
type Backup struct {
	Mnemonic string
}

// GenerateBackup creates a new keypair with its BIP-39 mnemonic backup.
// Returns the keypair and a 24-word mnemonic phrase.
// Per DESIGN_DOCUMENT.md, mnemonic backups enable keypair recovery without file exports.
func GenerateBackup() (*KeyPair, *Backup, error) {
	// Generate 256 bits of entropy for 24-word mnemonic.
	entropy, err := bip39.NewEntropy(MnemonicBitSize)
	if err != nil {
		return nil, nil, fmt.Errorf("generating entropy: %w", err)
	}

	// Create mnemonic from entropy.
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, nil, fmt.Errorf("creating mnemonic: %w", err)
	}

	// Derive seed from mnemonic (no passphrase for now).
	seed := bip39.NewSeed(mnemonic, "")

	// Use first 32 bytes as Ed25519 seed (Ed25519 uses 32-byte seeds).
	edSeed := seed[:ed25519.SeedSize]
	privateKey := ed25519.NewKeyFromSeed(edSeed)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	// Zero the seed bytes.
	ZeroBytes(seed)
	ZeroBytes(entropy)

	return &KeyPair{
			PublicKey:  publicKey,
			PrivateKey: privateKey,
		}, &Backup{
			Mnemonic: mnemonic,
		}, nil
}

// RestoreFromMnemonic recovers a keypair from a BIP-39 mnemonic phrase.
// Per DESIGN_DOCUMENT.md, this enables identity recovery on new devices.
func RestoreFromMnemonic(mnemonic string) (*KeyPair, error) {
	// Validate mnemonic.
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("invalid mnemonic phrase")
	}

	// Derive seed from mnemonic.
	seed := bip39.NewSeed(mnemonic, "")

	// Use first 32 bytes as Ed25519 seed.
	edSeed := seed[:ed25519.SeedSize]
	privateKey := ed25519.NewKeyFromSeed(edSeed)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	// Zero the seed bytes.
	ZeroBytes(seed)

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

// ValidateMnemonic checks if a mnemonic phrase is valid BIP-39.
func ValidateMnemonic(mnemonic string) bool {
	return bip39.IsMnemonicValid(mnemonic)
}

// ExportKeyPair serializes a keypair for device migration.
// The exported data can be encrypted with EncryptKeystore before storage.
func ExportKeyPair(kp *KeyPair) ([]byte, error) {
	if kp == nil || len(kp.PrivateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid keypair")
	}

	// Export format: private key bytes (64 bytes).
	// Public key can be derived from private key.
	exported := make([]byte, ed25519.PrivateKeySize)
	copy(exported, kp.PrivateKey)

	return exported, nil
}

// ImportKeyPair deserializes a keypair from exported data.
// Per SECURITY_PRIVACY.md §2.1, zeros the input data after copying.
func ImportKeyPair(data []byte) (*KeyPair, error) {
	if len(data) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid key data size: expected %d, got %d",
			ed25519.PrivateKeySize, len(data))
	}

	privateKey := make(ed25519.PrivateKey, ed25519.PrivateKeySize)
	copy(privateKey, data)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	// Zero the input data per SECURITY_PRIVACY.md §2.1.
	// Caller's responsibility if data is a slice of a larger buffer.
	defer ZeroBytes(data)

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}
