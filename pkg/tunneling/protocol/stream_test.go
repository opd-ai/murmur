package protocol

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/tunneling"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	payload := []byte("hello")
	if err := WriteFrame(&buf, FrameTypeData, payload); err != nil {
		t.Fatalf("WriteFrame failed: %v", err)
	}

	ft, got, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}
	if ft != FrameTypeData {
		t.Fatalf("frame type mismatch: got %d", ft)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload mismatch: got %q", string(got))
	}
}

func TestRegisterCellVerify(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	payload, err := EncodeRegisterCell(tunneling.TunnelID("demo-abcdefghijkl"), pub, priv, 1024)
	if err != nil {
		t.Fatalf("EncodeRegisterCell failed: %v", err)
	}

	if _, err := DecodeAndVerifyRegisterCell(payload, time.Now()); err != nil {
		t.Fatalf("DecodeAndVerifyRegisterCell failed: %v", err)
	}
}

func TestRegisterCellSkew(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	payload, err := EncodeRegisterCell(tunneling.TunnelID("demo-abcdefghijkl"), pub, priv, 1024)
	if err != nil {
		t.Fatalf("EncodeRegisterCell failed: %v", err)
	}

	if _, err := DecodeAndVerifyRegisterCell(payload, time.Now().Add(10*time.Minute)); err == nil {
		t.Fatal("expected timestamp skew validation failure")
	}
}
