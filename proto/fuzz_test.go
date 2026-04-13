package proto

import (
	"testing"

	"google.golang.org/protobuf/proto"
)

// FuzzMurmurEnvelope tests that parsing arbitrary bytes doesn't panic.
func FuzzMurmurEnvelope(f *testing.F) {
	// Add seed corpus.
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0xFF})

	// Valid envelope bytes (minimal).
	validEnv := &MurmurEnvelope{
		Version: 1,
		Type:    MessageType_MESSAGE_TYPE_WAVE,
		Payload: []byte("test payload"),
	}
	validBytes, _ := proto.Marshal(validEnv)
	f.Add(validBytes)

	// Truncated valid bytes.
	if len(validBytes) > 1 {
		f.Add(validBytes[:len(validBytes)/2])
	}

	// Large payload.
	largePayload := make([]byte, 4096)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}
	largeEnv := &MurmurEnvelope{
		Version: 1,
		Type:    MessageType_MESSAGE_TYPE_WAVE,
		Payload: largePayload,
	}
	largeBytes, _ := proto.Marshal(largeEnv)
	f.Add(largeBytes)

	f.Fuzz(func(t *testing.T, data []byte) {
		// This should never panic.
		env := &MurmurEnvelope{}
		_ = proto.Unmarshal(data, env)

		// If parsing succeeded, verify we can re-marshal.
		if env.GetVersion() != 0 || len(env.GetPayload()) > 0 {
			_, _ = proto.Marshal(env)
		}
	})
}

// FuzzWave tests that parsing arbitrary bytes as Wave doesn't panic.
func FuzzWave(f *testing.F) {
	// Add seed corpus.
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0xFF})

	// Valid wave bytes.
	validWave := &Wave{
		WaveType:   WaveType_WAVE_TYPE_SURFACE,
		Content:    []byte("Hello, World!"),
		TtlSeconds: 604800, // 7 days
	}
	validBytes, _ := proto.Marshal(validWave)
	f.Add(validBytes)

	f.Fuzz(func(t *testing.T, data []byte) {
		// This should never panic.
		wave := &Wave{}
		_ = proto.Unmarshal(data, wave)

		// If parsing succeeded, verify we can re-marshal.
		if wave.GetTtlSeconds() != 0 || len(wave.GetContent()) > 0 {
			_, _ = proto.Marshal(wave)
		}
	})
}

// FuzzHeartbeat tests heartbeat message parsing.
func FuzzHeartbeat(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0xFF})

	validHeartbeat := &Heartbeat{
		Timestamp: 1234567890,
		Signature: make([]byte, 64),
	}
	validBytes, _ := proto.Marshal(validHeartbeat)
	f.Add(validBytes)

	f.Fuzz(func(t *testing.T, data []byte) {
		hb := &Heartbeat{}
		_ = proto.Unmarshal(data, hb)

		if hb.GetTimestamp() != 0 {
			_, _ = proto.Marshal(hb)
		}
	})
}
