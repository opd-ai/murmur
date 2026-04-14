package ignition

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func generateTestKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate keypair: %v", err)
	}
	return pub, priv
}

func TestTokenManager(t *testing.T) {
	tm := NewTokenManager()
	pub, _ := generateTestKeyPair(t)

	// Generate token.
	token, err := tm.GenerateToken(pub)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// Validate token.
	validatedPub, err := tm.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if !bytes.Equal(validatedPub, pub) {
		t.Error("validated public key doesn't match")
	}
}

func TestTokenManagerReplayPrevention(t *testing.T) {
	tm := NewTokenManager()
	pub, _ := generateTestKeyPair(t)

	token, err := tm.GenerateToken(pub)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// First validation should succeed.
	_, err = tm.ValidateToken(token)
	if err != nil {
		t.Fatalf("first ValidateToken failed: %v", err)
	}

	// Second validation should fail (replay).
	_, err = tm.ValidateToken(token)
	if err != ErrTokenAlreadyUsed {
		t.Errorf("expected ErrTokenAlreadyUsed, got %v", err)
	}
}

func TestTokenManagerInvalidToken(t *testing.T) {
	tm := NewTokenManager()

	var fakeToken [TokenSize]byte
	rand.Read(fakeToken[:])

	_, err := tm.ValidateToken(fakeToken)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestTokenManagerCleanup(t *testing.T) {
	tm := NewTokenManager()
	pub, _ := generateTestKeyPair(t)

	// Generate tokens.
	tm.GenerateToken(pub)
	tm.GenerateToken(pub)

	// No expired tokens yet.
	cleaned := tm.CleanExpired()
	if cleaned != 0 {
		t.Errorf("CleanExpired returned %d, expected 0", cleaned)
	}
}

func TestGenerateIgnitionData(t *testing.T) {
	_, priv := generateTestKeyPair(t)
	tm := NewTokenManager()

	pub := priv.Public().(ed25519.PublicKey)
	token, err := tm.GenerateToken(pub)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	addresses := []string{
		"/ip4/192.168.1.100/tcp/4001",
		"/ip4/10.0.0.1/tcp/4001",
	}

	data, err := GenerateIgnitionData(priv, addresses, token)
	if err != nil {
		t.Fatalf("GenerateIgnitionData failed: %v", err)
	}

	if data.Version != Version {
		t.Errorf("Version = %d, want %d", data.Version, Version)
	}

	if !bytes.Equal(data.PublicKey, pub) {
		t.Error("PublicKey mismatch")
	}

	if len(data.Addresses) != len(addresses) {
		t.Errorf("Addresses count = %d, want %d", len(data.Addresses), len(addresses))
	}

	if data.Token != token {
		t.Error("Token mismatch")
	}

	if len(data.Signature) != ed25519.SignatureSize {
		t.Errorf("Signature size = %d, want %d", len(data.Signature), ed25519.SignatureSize)
	}
}

func TestIgnitionDataVerify(t *testing.T) {
	_, priv := generateTestKeyPair(t)
	tm := NewTokenManager()

	pub := priv.Public().(ed25519.PublicKey)
	token, _ := tm.GenerateToken(pub)

	data, err := GenerateIgnitionData(priv, []string{"/ip4/127.0.0.1/tcp/4001"}, token)
	if err != nil {
		t.Fatalf("GenerateIgnitionData failed: %v", err)
	}

	if !data.Verify() {
		t.Error("Verify returned false for valid data")
	}

	// Tamper with data.
	data.Timestamp++
	if data.Verify() {
		t.Error("Verify returned true for tampered data")
	}
}

func TestIgnitionDataEncodeDecode(t *testing.T) {
	_, priv := generateTestKeyPair(t)
	tm := NewTokenManager()

	pub := priv.Public().(ed25519.PublicKey)
	token, _ := tm.GenerateToken(pub)

	addresses := []string{
		"/ip4/192.168.1.100/tcp/4001",
		"/ip6/::1/tcp/4001",
	}

	original, err := GenerateIgnitionData(priv, addresses, token)
	if err != nil {
		t.Fatalf("GenerateIgnitionData failed: %v", err)
	}

	// Encode.
	encoded := original.Encode()
	t.Logf("Encoded length: %d chars", len(encoded))

	// Verify URL scheme.
	if !bytes.HasPrefix([]byte(encoded), []byte(QRScheme)) {
		t.Errorf("encoded string doesn't start with %s", QRScheme)
	}

	// Decode.
	decoded, err := DecodeIgnitionData(encoded)
	if err != nil {
		t.Fatalf("DecodeIgnitionData failed: %v", err)
	}

	// Verify decoded data matches original.
	if decoded.Version != original.Version {
		t.Errorf("Version mismatch: %d != %d", decoded.Version, original.Version)
	}

	if !bytes.Equal(decoded.PublicKey, original.PublicKey) {
		t.Error("PublicKey mismatch")
	}

	if len(decoded.Addresses) != len(original.Addresses) {
		t.Errorf("Addresses count mismatch: %d != %d", len(decoded.Addresses), len(original.Addresses))
	}

	for i, addr := range decoded.Addresses {
		if addr != original.Addresses[i] {
			t.Errorf("Address[%d] mismatch: %s != %s", i, addr, original.Addresses[i])
		}
	}

	if decoded.Token != original.Token {
		t.Error("Token mismatch")
	}

	if decoded.Timestamp != original.Timestamp {
		t.Errorf("Timestamp mismatch: %d != %d", decoded.Timestamp, original.Timestamp)
	}

	if !bytes.Equal(decoded.Signature, original.Signature) {
		t.Error("Signature mismatch")
	}
}

func TestDecodeIgnitionDataInvalid(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
		wantErr error
	}{
		{
			name:    "wrong_scheme",
			encoded: "https://example.com/ignite/abc",
			wantErr: ErrInvalidQRData,
		},
		{
			name:    "invalid_base64",
			encoded: QRScheme + "!!!invalid!!!",
			wantErr: nil, // Will fail on base64 decode
		},
		{
			name:    "too_short",
			encoded: QRScheme + "YWJj", // "abc" in base64
			wantErr: ErrInvalidQRData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeIgnitionData(tt.encoded)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestIgnitionDataIsExpired(t *testing.T) {
	_, priv := generateTestKeyPair(t)
	var token [TokenSize]byte
	rand.Read(token[:])

	data, err := GenerateIgnitionData(priv, []string{"/ip4/127.0.0.1/tcp/4001"}, token)
	if err != nil {
		t.Fatalf("GenerateIgnitionData failed: %v", err)
	}

	// Should not be expired immediately.
	if data.IsExpired() {
		t.Error("IsExpired returned true for fresh data")
	}

	// Set timestamp to past expiry.
	data.Timestamp = time.Now().Add(-2 * TokenExpiry).Unix()
	if !data.IsExpired() {
		t.Error("IsExpired returned false for expired data")
	}
}

func TestIgnitionDataPublicKeyHash(t *testing.T) {
	_, priv := generateTestKeyPair(t)
	var token [TokenSize]byte
	rand.Read(token[:])

	data, err := GenerateIgnitionData(priv, []string{"/ip4/127.0.0.1/tcp/4001"}, token)
	if err != nil {
		t.Fatalf("GenerateIgnitionData failed: %v", err)
	}

	hash := data.PublicKeyHash()
	if len(hash) != 32 {
		t.Errorf("hash length = %d, want 32", len(hash))
	}

	// Same data should produce same hash.
	hash2 := data.PublicKeyHash()
	if !bytes.Equal(hash, hash2) {
		t.Error("same data produced different hashes")
	}
}

func TestQRCodeImage(t *testing.T) {
	_, priv := generateTestKeyPair(t)
	var token [TokenSize]byte
	rand.Read(token[:])

	data, err := GenerateIgnitionData(priv, []string{"/ip4/127.0.0.1/tcp/4001"}, token)
	if err != nil {
		t.Fatalf("GenerateIgnitionData failed: %v", err)
	}

	img, err := data.QRCodeImage(4, 4)
	if err != nil {
		t.Fatalf("QRCodeImage failed: %v", err)
	}

	if img == nil {
		t.Fatal("QRCodeImage returned nil image")
	}

	bounds := img.Bounds()
	if bounds.Dx() < 100 || bounds.Dy() < 100 {
		t.Errorf("image too small: %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestQRCodePNG(t *testing.T) {
	_, priv := generateTestKeyPair(t)
	var token [TokenSize]byte
	rand.Read(token[:])

	data, err := GenerateIgnitionData(priv, []string{"/ip4/127.0.0.1/tcp/4001"}, token)
	if err != nil {
		t.Fatalf("GenerateIgnitionData failed: %v", err)
	}

	pngData, err := data.QRCodePNG(4, 4)
	if err != nil {
		t.Fatalf("QRCodePNG failed: %v", err)
	}

	if len(pngData) < 100 {
		t.Errorf("PNG data too small: %d bytes", len(pngData))
	}

	// Verify PNG signature.
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if !bytes.HasPrefix(pngData, pngSignature) {
		t.Error("invalid PNG signature")
	}
}

func TestIgnitionRecord(t *testing.T) {
	pub, _ := generateTestKeyPair(t)

	record := &IgnitionRecord{
		PeerPublicKey: pub,
		Timestamp:     time.Now(),
		Addresses:     []string{"/ip4/192.168.1.1/tcp/4001"},
		Mutual:        true,
	}

	if !record.Mutual {
		t.Error("Mutual should be true")
	}

	if len(record.Addresses) != 1 {
		t.Errorf("Addresses count = %d, want 1", len(record.Addresses))
	}
}

func TestConstants(t *testing.T) {
	// Verify constants per spec.
	if Version != 1 {
		t.Errorf("Version = %d, want 1", Version)
	}

	if TokenSize != 16 {
		t.Errorf("TokenSize = %d, want 16", TokenSize)
	}

	if TokenExpiry != 5*time.Minute {
		t.Errorf("TokenExpiry = %v, want 5m", TokenExpiry)
	}

	if QRScheme != "murmur://ignite/" {
		t.Errorf("QRScheme = %s, want murmur://ignite/", QRScheme)
	}
}

func TestMultipleAddresses(t *testing.T) {
	_, priv := generateTestKeyPair(t)
	var token [TokenSize]byte
	rand.Read(token[:])

	// Test with multiple addresses (direct + relay).
	addresses := []string{
		"/ip4/192.168.1.100/tcp/4001",
		"/ip4/10.0.0.1/tcp/4001",
		"/p2p-circuit/p2p/QmRelay123",
		"/dns4/relay.murmur.net/tcp/4001",
	}

	data, err := GenerateIgnitionData(priv, addresses, token)
	if err != nil {
		t.Fatalf("GenerateIgnitionData failed: %v", err)
	}

	encoded := data.Encode()
	decoded, err := DecodeIgnitionData(encoded)
	if err != nil {
		t.Fatalf("DecodeIgnitionData failed: %v", err)
	}

	if len(decoded.Addresses) != len(addresses) {
		t.Errorf("decoded address count = %d, want %d", len(decoded.Addresses), len(addresses))
	}

	for i, addr := range addresses {
		if decoded.Addresses[i] != addr {
			t.Errorf("Address[%d] = %s, want %s", i, decoded.Addresses[i], addr)
		}
	}
}

func TestEncodedStringLength(t *testing.T) {
	_, priv := generateTestKeyPair(t)
	var token [TokenSize]byte
	rand.Read(token[:])

	// Per ONBOARDING.md: "approximately 100–150 characters long"
	addresses := []string{"/ip4/192.168.1.100/tcp/4001"}

	data, err := GenerateIgnitionData(priv, addresses, token)
	if err != nil {
		t.Fatalf("GenerateIgnitionData failed: %v", err)
	}

	encoded := data.Encode()
	t.Logf("Encoded string length: %d characters", len(encoded))

	// Should be reasonably compact for QR codes and text sharing.
	if len(encoded) > 500 {
		t.Errorf("encoded string too long: %d chars (expected <500)", len(encoded))
	}
}

func BenchmarkGenerateIgnitionData(b *testing.B) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	pub := priv.Public().(ed25519.PublicKey)
	tm := NewTokenManager()

	addresses := []string{"/ip4/192.168.1.100/tcp/4001"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token, _ := tm.GenerateToken(pub)
		GenerateIgnitionData(priv, addresses, token)
	}
}

func BenchmarkEncodeDecodeIgnitionData(b *testing.B) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	pub := priv.Public().(ed25519.PublicKey)
	tm := NewTokenManager()
	token, _ := tm.GenerateToken(pub)

	data, _ := GenerateIgnitionData(priv, []string{"/ip4/192.168.1.100/tcp/4001"}, token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoded := data.Encode()
		DecodeIgnitionData(encoded)
	}
}
