package keys

import (
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/curve25519"
)

// BenchmarkGenerateKeyPair measures Ed25519 keypair generation performance.
// Per SECURITY_PRIVACY.md, Ed25519 is used for Surface Layer signatures.
func BenchmarkGenerateKeyPair(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateKeyPair()
		if err != nil {
			b.Fatalf("GenerateKeyPair failed: %v", err)
		}
	}
}

// BenchmarkGenerateAnonymousKeyPair measures Curve25519 keypair generation.
// Per SECURITY_PRIVACY.md, Curve25519 is used for Anonymous Layer key exchange.
func BenchmarkGenerateAnonymousKeyPair(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateAnonymousKeyPair()
		if err != nil {
			b.Fatalf("GenerateAnonymousKeyPair failed: %v", err)
		}
	}
}

// BenchmarkEd25519Sign measures Ed25519 signature generation.
func BenchmarkEd25519Sign(b *testing.B) {
	kp, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("GenerateKeyPair failed: %v", err)
	}

	message := []byte("benchmark test message for signing performance measurement")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = kp.Sign(message)
	}
}

// BenchmarkEd25519Verify measures Ed25519 signature verification.
func BenchmarkEd25519Verify(b *testing.B) {
	kp, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("GenerateKeyPair failed: %v", err)
	}

	message := []byte("benchmark test message for verification performance measurement")
	signature := kp.Sign(message)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !Verify(kp.PublicKey, message, signature) {
			b.Fatal("signature verification failed")
		}
	}
}

// BenchmarkCurve25519DH measures X25519 Diffie-Hellman key exchange.
func BenchmarkCurve25519DH(b *testing.B) {
	kp1, err := GenerateAnonymousKeyPair()
	if err != nil {
		b.Fatalf("GenerateAnonymousKeyPair failed: %v", err)
	}

	kp2, err := GenerateAnonymousKeyPair()
	if err != nil {
		b.Fatalf("GenerateAnonymousKeyPair failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := kp1.DeriveSharedSecret(kp2.PublicKey)
		if err != nil {
			b.Fatalf("DeriveSharedSecret failed: %v", err)
		}
	}
}

// BenchmarkCurve25519ScalarBaseMult measures base point scalar multiplication.
func BenchmarkCurve25519ScalarBaseMult(b *testing.B) {
	var scalar [32]byte
	rand.Read(scalar[:])

	// Clamp the scalar as per X25519 spec.
	scalar[0] &= 248
	scalar[31] &= 127
	scalar[31] |= 64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result [32]byte
		curve25519.ScalarBaseMult(&result, &scalar)
	}
}

// BenchmarkArgon2idKeyDerivation measures passphrase-based key derivation.
// Per TECHNICAL_IMPLEMENTATION.md §1.4: Argon2id with time=3, memory=64 MiB, threads=4.
func BenchmarkArgon2idKeyDerivation(b *testing.B) {
	passphrase := "benchmark test passphrase with reasonable entropy"
	salt := make([]byte, SaltSize)
	rand.Read(salt)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deriveKeyFromPassphrase(passphrase, salt)
	}
}

// BenchmarkEncryptKeystore measures keystore encryption (Argon2id + XChaCha20-Poly1305).
func BenchmarkEncryptKeystore(b *testing.B) {
	plaintext := make([]byte, 128) // Typical keystore size
	rand.Read(plaintext)
	passphrase := "benchmark test passphrase"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncryptKeystore(plaintext, passphrase)
		if err != nil {
			b.Fatalf("EncryptKeystore failed: %v", err)
		}
	}
}

// BenchmarkDecryptKeystore measures keystore decryption.
func BenchmarkDecryptKeystore(b *testing.B) {
	plaintext := make([]byte, 128)
	rand.Read(plaintext)
	passphrase := "benchmark test passphrase"

	encrypted, err := EncryptKeystore(plaintext, passphrase)
	if err != nil {
		b.Fatalf("EncryptKeystore failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecryptKeystore(encrypted, passphrase)
		if err != nil {
			b.Fatalf("DecryptKeystore failed: %v", err)
		}
	}
}

// BenchmarkGenerateIdentityBundle measures full identity generation (Surface + Specter).
func BenchmarkGenerateIdentityBundle(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bundle, err := GenerateIdentityBundle()
		if err != nil {
			b.Fatalf("GenerateIdentityBundle failed: %v", err)
		}
		// Clean up to avoid memory leaks.
		bundle.Surface.ZeroKeyPair()
		if bundle.Specter != nil {
			bundle.Specter.ZeroAnonymousKeyPair()
		}
	}
}

// BenchmarkGenerateIdentityBundleWithFortress measures Fortress-mode identity generation.
func BenchmarkGenerateIdentityBundleWithFortress(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bundle, err := GenerateIdentityBundleWithFortress()
		if err != nil {
			b.Fatalf("GenerateIdentityBundleWithFortress failed: %v", err)
		}
		// Clean up.
		bundle.Surface.ZeroKeyPair()
		if bundle.Specter != nil {
			bundle.Specter.ZeroAnonymousKeyPair()
		}
	}
}

// BenchmarkZeroBytes measures secure memory zeroing performance.
func BenchmarkZeroBytes(b *testing.B) {
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ZeroBytes(data)
	}
}

// BenchmarkEd25519SignVariableSizes measures signing performance across message sizes.
func BenchmarkEd25519SignVariableSizes(b *testing.B) {
	kp, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("GenerateKeyPair failed: %v", err)
	}

	sizes := []int{64, 256, 1024, 2048, 4096}
	for _, size := range sizes {
		message := make([]byte, size)
		rand.Read(message)

		b.Run(string(rune('0'+size/1024))+"KB", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = kp.Sign(message)
			}
		})
	}
}

// BenchmarkEd25519VerifyVariableSizes measures verification performance across message sizes.
func BenchmarkEd25519VerifyVariableSizes(b *testing.B) {
	kp, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("GenerateKeyPair failed: %v", err)
	}

	sizes := []int{64, 256, 1024, 2048, 4096}
	for _, size := range sizes {
		message := make([]byte, size)
		rand.Read(message)
		signature := kp.Sign(message)

		b.Run(string(rune('0'+size/1024))+"KB", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if !Verify(kp.PublicKey, message, signature) {
					b.Fatal("verification failed")
				}
			}
		})
	}
}
