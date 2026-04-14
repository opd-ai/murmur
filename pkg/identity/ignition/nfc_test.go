// Package ignition implements NFC compact format tests.
package ignition

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// TestGenerateNFCIgnitionData tests basic NFC ignition data generation.
func TestGenerateNFCIgnitionData(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	addrs := []string{"192.168.1.1:4001"}
	data, err := GenerateNFCIgnitionData(priv, addrs, token)
	if err != nil {
		t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	// Verify fields.
	if data.Version != NFCVersion {
		t.Errorf("version = %d, want %d", data.Version, NFCVersion)
	}
	if !bytes.Equal(data.PublicKey[:], pub) {
		t.Error("public key mismatch")
	}
	if data.Token != token {
		t.Error("token mismatch")
	}
	if len(data.Addresses) != 1 {
		t.Errorf("address count = %d, want 1", len(data.Addresses))
	}
	if data.Addresses[0].Type != NFCAddressTypeIPv4 {
		t.Errorf("address type = %d, want %d", data.Addresses[0].Type, NFCAddressTypeIPv4)
	}
}

// TestNFCIgnitionDataEncodeDecode tests round-trip encoding.
func TestNFCIgnitionDataEncodeDecode(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	tests := []struct {
		name  string
		addrs []string
	}{
		{
			name:  "single_ipv4",
			addrs: []string{"192.168.1.1:4001"},
		},
		{
			name:  "single_ipv6",
			addrs: []string{"[::1]:4001"},
		},
		{
			name:  "multiaddr_ipv4",
			addrs: []string{"/ip4/192.168.1.1/tcp/4001"},
		},
		{
			name:  "multiaddr_ipv6",
			addrs: []string{"/ip6/::1/tcp/4001"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := GenerateNFCIgnitionData(priv, tt.addrs, token)
			if err != nil {
				t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
			}

			encoded, err := data.Encode()
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}

			decoded, err := DecodeNFCIgnitionData(encoded)
			if err != nil {
				t.Fatalf("DecodeNFCIgnitionData failed: %v", err)
			}

			if decoded.Version != data.Version {
				t.Errorf("version mismatch: got %d, want %d", decoded.Version, data.Version)
			}
			if !bytes.Equal(decoded.PublicKey[:], data.PublicKey[:]) {
				t.Error("public key mismatch")
			}
			if decoded.Token != data.Token {
				t.Error("token mismatch")
			}
			if !decoded.Verify() {
				t.Error("signature verification failed")
			}
		})
	}
}

// TestNFCPayloadSize verifies payload fits NFC constraints.
func TestNFCPayloadSize(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	tests := []struct {
		name        string
		addrs       []string
		maxExpected int
	}{
		{
			name:        "single_ipv4",
			addrs:       []string{"192.168.1.1:4001"},
			maxExpected: NFCMinPayload + 1, // 117 + 1 for IPv4 address overhead
		},
		{
			name:        "single_ipv6",
			addrs:       []string{"[2001:db8::1]:4001"},
			maxExpected: NFCMinPayload + 19, // 117 + 19 for IPv6 address
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := GenerateNFCIgnitionData(priv, tt.addrs, token)
			if err != nil {
				t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
			}

			size := data.PayloadSize()
			t.Logf("%s payload size: %d bytes", tt.name, size)

			if size > NFCMaxPayload {
				t.Errorf("payload size %d exceeds NFC max %d", size, NFCMaxPayload)
			}
		})
	}
}

// TestNFCAddressIPv4 tests IPv4 address encoding.
func TestNFCAddressIPv4(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	addrs := []string{"10.0.0.1:8080"}

	data, err := GenerateNFCIgnitionData(priv, addrs, token)
	if err != nil {
		t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	if len(data.Addresses) != 1 {
		t.Fatalf("expected 1 address, got %d", len(data.Addresses))
	}

	addr := data.Addresses[0]
	if addr.Type != NFCAddressTypeIPv4 {
		t.Errorf("type = %d, want %d", addr.Type, NFCAddressTypeIPv4)
	}
	if addr.Port != 8080 {
		t.Errorf("port = %d, want 8080", addr.Port)
	}

	expectedIP := net.ParseIP("10.0.0.1").To4()
	if !bytes.Equal(addr.IPv4[:], expectedIP) {
		t.Errorf("IPv4 = %v, want %v", addr.IPv4[:], expectedIP)
	}
}

// TestNFCAddressIPv6 tests IPv6 address encoding.
func TestNFCAddressIPv6(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	addrs := []string{"[2001:db8::1]:4001"}

	data, err := GenerateNFCIgnitionData(priv, addrs, token)
	if err != nil {
		t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	if len(data.Addresses) != 1 {
		t.Fatalf("expected 1 address, got %d", len(data.Addresses))
	}

	addr := data.Addresses[0]
	if addr.Type != NFCAddressTypeIPv6 {
		t.Errorf("type = %d, want %d", addr.Type, NFCAddressTypeIPv6)
	}
	if addr.Port != 4001 {
		t.Errorf("port = %d, want 4001", addr.Port)
	}

	expectedIP := net.ParseIP("2001:db8::1").To16()
	if !bytes.Equal(addr.IPv6[:], expectedIP) {
		t.Errorf("IPv6 = %v, want %v", addr.IPv6[:], expectedIP)
	}
}

// TestNFCAddressMultiaddr tests multiaddr parsing.
func TestNFCAddressMultiaddr(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	tests := []struct {
		name     string
		addr     string
		wantType NFCAddressType
		wantPort uint16
	}{
		{
			name:     "ipv4_tcp",
			addr:     "/ip4/192.168.1.1/tcp/4001",
			wantType: NFCAddressTypeIPv4,
			wantPort: 4001,
		},
		{
			name:     "ipv6_tcp",
			addr:     "/ip6/::1/tcp/5001",
			wantType: NFCAddressTypeIPv6,
			wantPort: 5001,
		},
		{
			name:     "ipv4_udp",
			addr:     "/ip4/10.0.0.1/udp/9000",
			wantType: NFCAddressTypeIPv4,
			wantPort: 9000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := GenerateNFCIgnitionData(priv, []string{tt.addr}, token)
			if err != nil {
				t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
			}

			if len(data.Addresses) != 1 {
				t.Fatalf("expected 1 address, got %d", len(data.Addresses))
			}

			addr := data.Addresses[0]
			if addr.Type != tt.wantType {
				t.Errorf("type = %d, want %d", addr.Type, tt.wantType)
			}
			if addr.Port != tt.wantPort {
				t.Errorf("port = %d, want %d", addr.Port, tt.wantPort)
			}
		})
	}
}

// TestNFCVerifySignature tests signature verification.
func TestNFCVerifySignature(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	data, err := GenerateNFCIgnitionData(priv, []string{"192.168.1.1:4001"}, token)
	if err != nil {
		t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	if !data.Verify() {
		t.Error("valid signature should verify")
	}

	// Tamper with data.
	data.Token[0] ^= 0xFF
	if data.Verify() {
		t.Error("tampered data should not verify")
	}
}

// TestNFCTimestamp tests timestamp encoding.
func TestNFCTimestamp(t *testing.T) {
	// Save and restore time function.
	origTime := timeNow
	defer func() { timeNow = origTime }()

	fixedTime := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return fixedTime }

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	data, err := GenerateNFCIgnitionData(priv, []string{"192.168.1.1:4001"}, token)
	if err != nil {
		t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	expectedTimestamp := uint32(fixedTime.Unix())
	if data.Timestamp != expectedTimestamp {
		t.Errorf("timestamp = %d, want %d", data.Timestamp, expectedTimestamp)
	}
}

// TestNFCToIgnitionData tests conversion to full IgnitionData.
func TestNFCToIgnitionData(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	nfcData, err := GenerateNFCIgnitionData(priv, []string{"192.168.1.1:4001"}, token)
	if err != nil {
		t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	ignitionData := nfcData.ToIgnitionData()
	if ignitionData == nil {
		t.Fatal("ToIgnitionData returned nil")
	}

	if !bytes.Equal(ignitionData.PublicKey[:], nfcData.PublicKey[:]) {
		t.Error("public key mismatch")
	}
	if ignitionData.Token != nfcData.Token {
		t.Error("token mismatch")
	}
}

// TestNFCNDEFRecord tests NDEF record creation.
func TestNFCNDEFRecord(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	data, err := GenerateNFCIgnitionData(priv, []string{"192.168.1.1:4001"}, token)
	if err != nil {
		t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	record, err := data.ToNDEF()
	if err != nil {
		t.Fatalf("ToNDEF failed: %v", err)
	}

	if record.TypeNameFormat != 0x02 {
		t.Errorf("TypeNameFormat = %02x, want 0x02", record.TypeNameFormat)
	}
	if record.Type != "application/vnd.murmur.ignition" {
		t.Errorf("Type = %s, want application/vnd.murmur.ignition", record.Type)
	}
	if len(record.Payload) == 0 {
		t.Error("payload should not be empty")
	}
}

// TestNFCDecodeInvalid tests decoding invalid data.
func TestNFCDecodeInvalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty",
			data: nil,
		},
		{
			name: "too_short",
			data: make([]byte, 50),
		},
		{
			name: "wrong_version",
			data: func() []byte {
				d := make([]byte, NFCMinPayload)
				d[0] = 99 // Wrong version.
				return d
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeNFCIgnitionData(tt.data)
			if err == nil {
				t.Error("expected error for invalid data")
			}
		})
	}
}

// TestNFCAddressString tests address string representation.
func TestNFCAddressString(t *testing.T) {
	tests := []struct {
		name string
		addr NFCAddress
		want string
	}{
		{
			name: "ipv4",
			addr: NFCAddress{
				Type: NFCAddressTypeIPv4,
				IPv4: [4]byte{192, 168, 1, 1},
				Port: 4001,
			},
			want: "/ip4/192.168.1.1/tcp/4001",
		},
		{
			name: "ipv6",
			addr: NFCAddress{
				Type: NFCAddressTypeIPv6,
				IPv6: [16]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
				Port: 4001,
			},
			want: "/ip6/2001:db8::1/tcp/4001",
		},
		{
			name: "peerid",
			addr: NFCAddress{
				Type:   NFCAddressTypePeerID,
				PeerID: [8]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
			},
			want: "/p2p/123456789abcdef0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.addr.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestNFCWriteReadAddressRoundtrip tests address encoding round-trip.
func TestNFCWriteReadAddressRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		addr NFCAddress
	}{
		{
			name: "ipv4",
			addr: NFCAddress{
				Type: NFCAddressTypeIPv4,
				IPv4: [4]byte{10, 0, 0, 1},
				Port: 8080,
			},
		},
		{
			name: "ipv6",
			addr: NFCAddress{
				Type: NFCAddressTypeIPv6,
				IPv6: [16]byte{0xfe, 0x80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
				Port: 9000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writeAddressCompact(&buf, tt.addr)

			// Read it back.
			data := buf.Bytes()
			if len(data) == 0 {
				t.Fatal("writeAddressCompact produced empty output")
			}

			// Note: writeAddressCompact doesn't write the type byte.
			// The type is encoded separately in the address count byte.
			decoded, n, err := readAddressCompact(data, tt.addr.Type)
			if err != nil {
				t.Fatalf("readAddressCompact failed: %v", err)
			}

			if decoded.Type != tt.addr.Type {
				t.Errorf("type = %d, want %d", decoded.Type, tt.addr.Type)
			}
			if decoded.Port != tt.addr.Port {
				t.Errorf("port = %d, want %d", decoded.Port, tt.addr.Port)
			}

			// Verify n matches buffer length.
			if n != len(data) {
				t.Errorf("bytes read = %d, want %d", n, len(data))
			}
		})
	}
}

// TestNFCEmptyAddresses tests behavior with no addresses.
func TestNFCEmptyAddresses(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	_, err = GenerateNFCIgnitionData(priv, nil, token)
	if err == nil {
		t.Error("expected error for empty addresses")
	}
}

// TestNFCInvalidAddress tests behavior with invalid address.
func TestNFCInvalidAddress(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	_, err = GenerateNFCIgnitionData(priv, []string{"not-an-address"}, token)
	if err == nil {
		t.Error("expected error for invalid address")
	}
}

// TestNFCMinimalPayloadSize verifies minimum payload is within spec.
func TestNFCMinimalPayloadSize(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	// Use the smallest possible address (IPv4).
	data, err := GenerateNFCIgnitionData(priv, []string{"0.0.0.1:1"}, token)
	if err != nil {
		t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	encoded, err := data.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	t.Logf("Minimal IPv4 payload: %d bytes (max: %d)", len(encoded), NFCMaxPayload)

	// Should fit in NTAG213 (137 bytes).
	if len(encoded) > NFCMaxPayload {
		t.Errorf("payload %d bytes exceeds NFC max %d", len(encoded), NFCMaxPayload)
	}
}

// BenchmarkNFCEncode benchmarks NFC encoding.
func BenchmarkNFCEncode(b *testing.B) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	data, err := GenerateNFCIgnitionData(priv, []string{"192.168.1.1:4001"}, token)
	if err != nil {
		b.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = data.Encode()
	}
}

// BenchmarkNFCDecode benchmarks NFC decoding.
func BenchmarkNFCDecode(b *testing.B) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	data, err := GenerateNFCIgnitionData(priv, []string{"192.168.1.1:4001"}, token)
	if err != nil {
		b.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	encoded, err := data.Encode()
	if err != nil {
		b.Fatalf("Encode failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeNFCIgnitionData(encoded)
	}
}

// TestNFCDecodeNDEFRoundtrip tests NDEF decode round-trip.
func TestNFCDecodeNDEFRoundtrip(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	original, err := GenerateNFCIgnitionData(priv, []string{"192.168.1.1:4001"}, token)
	if err != nil {
		t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	record, err := original.ToNDEF()
	if err != nil {
		t.Fatalf("ToNDEF failed: %v", err)
	}

	decoded, err := DecodeNFCNDEF(record)
	if err != nil {
		t.Fatalf("DecodeNFCNDEF failed: %v", err)
	}

	if !bytes.Equal(decoded.PublicKey[:], original.PublicKey[:]) {
		t.Error("public key mismatch after NDEF round-trip")
	}
	if decoded.Token != original.Token {
		t.Error("token mismatch after NDEF round-trip")
	}
	if !decoded.Verify() {
		t.Error("signature verification failed after NDEF round-trip")
	}
}

// TestNFCMaxAddresses tests with maximum reasonable addresses.
func TestNFCMaxAddresses(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	// Two IPv4 addresses should still fit.
	addrs := []string{"192.168.1.1:4001", "10.0.0.1:4001"}

	data, err := GenerateNFCIgnitionData(priv, addrs, token)
	if err != nil {
		t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	encoded, err := data.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	t.Logf("Two IPv4 addresses payload: %d bytes", len(encoded))

	// May exceed 137 bytes, log but don't fail.
	if len(encoded) > NFCMaxPayload {
		t.Logf("WARNING: payload %d exceeds NFC max %d, multiple addresses may not fit",
			len(encoded), NFCMaxPayload)
	}
}

// TestNFCBinaryFormat verifies the exact binary format.
func TestNFCBinaryFormat(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	var token [TokenSize]byte
	for i := range token {
		token[i] = byte(i)
	}

	// Fix timestamp for reproducibility.
	origTime := timeNow
	defer func() { timeNow = origTime }()
	fixedTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return fixedTime }

	data, err := GenerateNFCIgnitionData(priv, []string{"192.168.1.1:4001"}, token)
	if err != nil {
		t.Fatalf("GenerateNFCIgnitionData failed: %v", err)
	}

	encoded, err := data.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Verify header.
	if encoded[0] != NFCVersion {
		t.Errorf("byte 0 (version) = %02x, want %02x", encoded[0], NFCVersion)
	}

	// Public key at bytes 1-32.
	pubKey := priv.Public().(ed25519.PublicKey)
	if !bytes.Equal(encoded[1:33], pubKey) {
		t.Error("bytes 1-32 (public key) mismatch")
	}

	// Token at bytes 33-48.
	if !bytes.Equal(encoded[33:49], token[:]) {
		t.Error("bytes 33-48 (token) mismatch")
	}

	// Timestamp at bytes 49-52 (big endian uint32).
	timestamp := binary.BigEndian.Uint32(encoded[49:53])
	expectedTS := uint32(fixedTime.Unix())
	if timestamp != expectedTS {
		t.Errorf("timestamp = %d, want %d", timestamp, expectedTS)
	}

	t.Logf("Total encoded length: %d bytes", len(encoded))
}
