package proto

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/zeebo/blake3"
)

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		name    string
		version uint32
		wantErr bool
	}{
		{"valid current", 1, false},
		{"invalid zero", 0, true},
		{"invalid future", 99, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVersion(%d) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}

func TestValidateMessageType(t *testing.T) {
	tests := []struct {
		name    string
		msgType MessageType
		wantErr bool
	}{
		{"wave", MessageType_MESSAGE_TYPE_WAVE, false},
		{"identity", MessageType_MESSAGE_TYPE_IDENTITY, false},
		{"shroud_ad", MessageType_MESSAGE_TYPE_SHROUD_AD, false},
		{"heartbeat", MessageType_MESSAGE_TYPE_HEARTBEAT, false},
		{"unspecified", MessageType_MESSAGE_TYPE_UNSPECIFIED, true},
		{"invalid", MessageType(99), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessageType(tt.msgType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessageType(%v) error = %v, wantErr %v", tt.msgType, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTimestamp(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name      string
		timestamp int64
		wantErr   error
	}{
		{"current", now, nil},
		{"recent past", now - 60, nil},
		{"near future", now + 60, nil},
		{"too far future", now + 600, ErrTimestampTooFuture},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimestamp(tt.timestamp)
			if err != tt.wantErr {
				t.Errorf("ValidateTimestamp(%d) error = %v, wantErr %v", tt.timestamp, err, tt.wantErr)
			}
		})
	}
}

func TestValidateMessageID(t *testing.T) {
	payload := []byte("test payload")
	correctHash := blake3.Sum256(payload)

	tests := []struct {
		name      string
		messageID []byte
		payload   []byte
		wantErr   error
	}{
		{"valid", correctHash[:], payload, nil},
		{"wrong length", correctHash[:16], payload, ErrInvalidMessageID},
		{"mismatch", make([]byte, 32), payload, ErrMessageIDMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessageID(tt.messageID, tt.payload)
			if err != tt.wantErr {
				t.Errorf("ValidateMessageID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSignature(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	payload := []byte("test payload")

	env := &MurmurEnvelope{
		Version:      1,
		Type:         MessageType_MESSAGE_TYPE_WAVE,
		Payload:      payload,
		SenderPubkey: pubKey,
	}

	// Sign the envelope
	_ = SignEnvelope(env, privKey)

	t.Run("valid signature", func(t *testing.T) {
		err := ValidateSignature(env)
		if err != nil {
			t.Errorf("ValidateSignature() unexpected error: %v", err)
		}
	})

	t.Run("invalid pubkey length", func(t *testing.T) {
		badEnv := &MurmurEnvelope{
			Version:      1,
			Type:         MessageType_MESSAGE_TYPE_WAVE,
			Payload:      payload,
			SenderPubkey: []byte("short"),
		}
		err := ValidateSignature(badEnv)
		if err != ErrInvalidPubKeyLength {
			t.Errorf("ValidateSignature() error = %v, want %v", err, ErrInvalidPubKeyLength)
		}
	})

	t.Run("invalid signature length", func(t *testing.T) {
		badEnv := &MurmurEnvelope{
			Version:      1,
			Type:         MessageType_MESSAGE_TYPE_WAVE,
			Payload:      payload,
			SenderPubkey: pubKey,
			Signature:    []byte("short"),
		}
		err := ValidateSignature(badEnv)
		if err != ErrInvalidSignature {
			t.Errorf("ValidateSignature() error = %v, want %v", err, ErrInvalidSignature)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		badEnv := &MurmurEnvelope{
			Version:      1,
			Type:         MessageType_MESSAGE_TYPE_WAVE,
			Payload:      payload,
			SenderPubkey: pubKey,
			Signature:    make([]byte, 64),
		}
		err := ValidateSignature(badEnv)
		if err != ErrInvalidSignature {
			t.Errorf("ValidateSignature() error = %v, want %v", err, ErrInvalidSignature)
		}
	})
}

func TestValidateWave(t *testing.T) {
	validWave := &Wave{
		WaveType:   WaveType_WAVE_TYPE_SURFACE,
		Content:    []byte("test content"),
		CreatedAt:  time.Now().Unix(),
		TtlSeconds: DefaultTTLSeconds,
		HopCount:   0,
	}

	t.Run("valid wave", func(t *testing.T) {
		err := ValidateWave(validWave)
		if err != nil {
			t.Errorf("ValidateWave() unexpected error: %v", err)
		}
	})

	t.Run("nil wave", func(t *testing.T) {
		err := ValidateWave(nil)
		if err == nil {
			t.Error("ValidateWave(nil) expected error, got nil")
		}
	})

	t.Run("invalid wave type", func(t *testing.T) {
		wave := &Wave{
			WaveType:   WaveType_WAVE_TYPE_UNSPECIFIED,
			Content:    []byte("test"),
			CreatedAt:  time.Now().Unix(),
			TtlSeconds: DefaultTTLSeconds,
		}
		err := ValidateWave(wave)
		if err != ErrInvalidWaveType {
			t.Errorf("ValidateWave() error = %v, want %v", err, ErrInvalidWaveType)
		}
	})

	t.Run("empty content", func(t *testing.T) {
		wave := &Wave{
			WaveType:   WaveType_WAVE_TYPE_SURFACE,
			Content:    []byte{},
			CreatedAt:  time.Now().Unix(),
			TtlSeconds: DefaultTTLSeconds,
		}
		err := ValidateWave(wave)
		if err != ErrEmptyWaveContent {
			t.Errorf("ValidateWave() error = %v, want %v", err, ErrEmptyWaveContent)
		}
	})

	t.Run("content too large", func(t *testing.T) {
		wave := &Wave{
			WaveType:   WaveType_WAVE_TYPE_SURFACE,
			Content:    make([]byte, MaxWaveContentSize+1),
			CreatedAt:  time.Now().Unix(),
			TtlSeconds: DefaultTTLSeconds,
		}
		err := ValidateWave(wave)
		if err != ErrContentTooLarge {
			t.Errorf("ValidateWave() error = %v, want %v", err, ErrContentTooLarge)
		}
	})

	t.Run("invalid TTL", func(t *testing.T) {
		wave := &Wave{
			WaveType:   WaveType_WAVE_TYPE_SURFACE,
			Content:    []byte("test"),
			CreatedAt:  time.Now().Unix(),
			TtlSeconds: MaxTTLSeconds + 1,
		}
		err := ValidateWave(wave)
		if err != ErrInvalidTTL {
			t.Errorf("ValidateWave() error = %v, want %v", err, ErrInvalidTTL)
		}
	})

	t.Run("hop count too high", func(t *testing.T) {
		wave := &Wave{
			WaveType:   WaveType_WAVE_TYPE_SURFACE,
			Content:    []byte("test"),
			CreatedAt:  time.Now().Unix(),
			TtlSeconds: DefaultTTLSeconds,
			HopCount:   MaxHopCount + 1,
		}
		err := ValidateWave(wave)
		if err != ErrInvalidHopCount {
			t.Errorf("ValidateWave() error = %v, want %v", err, ErrInvalidHopCount)
		}
	})

	t.Run("expired wave", func(t *testing.T) {
		wave := &Wave{
			WaveType:   WaveType_WAVE_TYPE_SURFACE,
			Content:    []byte("test"),
			CreatedAt:  time.Now().Unix() - 1000,
			TtlSeconds: 100, // expired
		}
		err := ValidateWave(wave)
		if err != ErrTimestampTooOld {
			t.Errorf("ValidateWave() error = %v, want %v", err, ErrTimestampTooOld)
		}
	})
}

func TestValidateEnvelope(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	payload := []byte("test payload")
	messageID := blake3.Sum256(payload)

	env := &MurmurEnvelope{
		Version:       1,
		Type:          MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		SenderPubkey:  pubKey,
		TimestampUnix: time.Now().Unix(),
		MessageId:     messageID[:],
	}
	_ = SignEnvelope(env, privKey)

	t.Run("valid envelope", func(t *testing.T) {
		err := ValidateEnvelope(env)
		if err != nil {
			t.Errorf("ValidateEnvelope() unexpected error: %v", err)
		}
	})

	t.Run("nil envelope", func(t *testing.T) {
		err := ValidateEnvelope(nil)
		if err == nil {
			t.Error("ValidateEnvelope(nil) expected error, got nil")
		}
	})

	t.Run("empty payload", func(t *testing.T) {
		badEnv := &MurmurEnvelope{
			Version:       1,
			Type:          MessageType_MESSAGE_TYPE_WAVE,
			Payload:       []byte{},
			TimestampUnix: time.Now().Unix(),
		}
		err := ValidateEnvelope(badEnv)
		if err != ErrEmptyPayload {
			t.Errorf("ValidateEnvelope() error = %v, want %v", err, ErrEmptyPayload)
		}
	})
}

func TestSignEnvelope(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	payload := []byte("test payload")

	env := &MurmurEnvelope{
		Version: 1,
		Type:    MessageType_MESSAGE_TYPE_WAVE,
		Payload: payload,
	}

	t.Run("sign and verify", func(t *testing.T) {
		err := SignEnvelope(env, privKey)
		if err != nil {
			t.Fatalf("SignEnvelope() error: %v", err)
		}

		if len(env.Signature) != SignatureLength {
			t.Errorf("signature length = %d, want %d", len(env.Signature), SignatureLength)
		}

		if len(env.SenderPubkey) != PubKeyLength {
			t.Errorf("pubkey length = %d, want %d", len(env.SenderPubkey), PubKeyLength)
		}

		// Verify pubkey matches
		for i, b := range pubKey {
			if env.SenderPubkey[i] != b {
				t.Error("pubkey mismatch")
				break
			}
		}

		// Verify signature is valid
		err = ValidateSignature(env)
		if err != nil {
			t.Errorf("ValidateSignature() after SignEnvelope() error: %v", err)
		}
	})

	t.Run("nil envelope", func(t *testing.T) {
		err := SignEnvelope(nil, privKey)
		if err == nil {
			t.Error("SignEnvelope(nil, ...) expected error, got nil")
		}
	})

	t.Run("invalid private key", func(t *testing.T) {
		err := SignEnvelope(env, []byte("short"))
		if err == nil {
			t.Error("SignEnvelope(..., short) expected error, got nil")
		}
	})
}

func TestComputeMessageID(t *testing.T) {
	payload := []byte("test payload")
	expected := blake3.Sum256(payload)

	result := ComputeMessageID(payload)

	if len(result) != BLAKE3HashLength {
		t.Errorf("ComputeMessageID() length = %d, want %d", len(result), BLAKE3HashLength)
	}

	for i, b := range expected {
		if result[i] != b {
			t.Errorf("ComputeMessageID() mismatch at index %d", i)
			break
		}
	}
}

func TestIsExpired(t *testing.T) {
	t.Run("nil wave", func(t *testing.T) {
		if !IsExpired(nil) {
			t.Error("IsExpired(nil) = false, want true")
		}
	})

	t.Run("expired wave", func(t *testing.T) {
		wave := &Wave{
			CreatedAt:  time.Now().Unix() - 1000,
			TtlSeconds: 100,
		}
		if !IsExpired(wave) {
			t.Error("IsExpired() = false for expired wave")
		}
	})

	t.Run("not expired", func(t *testing.T) {
		wave := &Wave{
			CreatedAt:  time.Now().Unix(),
			TtlSeconds: 3600,
		}
		if IsExpired(wave) {
			t.Error("IsExpired() = true for non-expired wave")
		}
	})
}

func TestRemainingTTL(t *testing.T) {
	t.Run("nil wave", func(t *testing.T) {
		if RemainingTTL(nil) != 0 {
			t.Error("RemainingTTL(nil) != 0")
		}
	})

	t.Run("expired wave", func(t *testing.T) {
		wave := &Wave{
			CreatedAt:  time.Now().Unix() - 1000,
			TtlSeconds: 100,
		}
		if RemainingTTL(wave) != 0 {
			t.Error("RemainingTTL() != 0 for expired wave")
		}
	})

	t.Run("active wave", func(t *testing.T) {
		wave := &Wave{
			CreatedAt:  time.Now().Unix(),
			TtlSeconds: 3600,
		}
		remaining := RemainingTTL(wave)
		if remaining < 3500 || remaining > 3600 {
			t.Errorf("RemainingTTL() = %d, expected ~3600", remaining)
		}
	})
}

func TestValidateWaveType(t *testing.T) {
	validTypes := []WaveType{
		WaveType_WAVE_TYPE_SURFACE,
		WaveType_WAVE_TYPE_REPLY,
		WaveType_WAVE_TYPE_VEILED,
		WaveType_WAVE_TYPE_SPECTER,
		WaveType_WAVE_TYPE_SIGIL,
		WaveType_WAVE_TYPE_ABYSSAL,
		WaveType_WAVE_TYPE_MASKED,
		WaveType_WAVE_TYPE_BEACON,
	}

	for _, wt := range validTypes {
		t.Run(wt.String(), func(t *testing.T) {
			if err := ValidateWaveType(wt); err != nil {
				t.Errorf("ValidateWaveType(%v) unexpected error: %v", wt, err)
			}
		})
	}

	t.Run("unspecified", func(t *testing.T) {
		if err := ValidateWaveType(WaveType_WAVE_TYPE_UNSPECIFIED); err != ErrInvalidWaveType {
			t.Errorf("ValidateWaveType(UNSPECIFIED) error = %v, want %v", err, ErrInvalidWaveType)
		}
	})
}
