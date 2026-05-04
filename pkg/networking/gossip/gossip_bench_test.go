// Package gossip provides GossipSub configuration, topic management, and peer scoring.
// This file implements performance benchmarks per TECHNICAL_IMPLEMENTATION.md.
package gossip

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"
)

// BenchmarkValidateEnvelope measures envelope validation performance.
// Per TECHNICAL_IMPLEMENTATION.md, target is <500ms propagation across 3 hops.
func BenchmarkValidateEnvelope(b *testing.B) {
	// Generate test keypair
	pubkey, privkey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}

	// Create test payload
	payload := make([]byte, 2048)
	rand.Read(payload)

	// Compute message ID
	hasher := blake3.New()
	hasher.Write(payload)
	messageID := hasher.Sum(nil)

	// Create signature data
	timestamp := time.Now().Unix()
	sigData := make([]byte, 4+4+len(payload))
	copy(sigData[0:4], []byte{0, 0, 0, 1}) // version
	copy(sigData[4:8], []byte{0, 0, 0, 1}) // type
	copy(sigData[8:], payload)

	signature := ed25519.Sign(privkey, sigData)

	// Create protobuf envelope
	envelope := &pb.MurmurEnvelope{
		Version:       1,
		Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		SenderPubkey:  pubkey,
		Signature:     signature,
		TimestampUnix: timestamp,
		MessageId:     messageID,
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Parse envelope
		var env pb.MurmurEnvelope
		if err := proto.Unmarshal(data, &env); err != nil {
			b.Fatal(err)
		}

		// Verify signature
		sigData := make([]byte, 4+4+len(env.Payload))
		copy(sigData[0:4], []byte{0, 0, 0, byte(env.Version)})
		copy(sigData[4:8], []byte{0, 0, 0, byte(env.Type)})
		copy(sigData[8:], env.Payload)

		if !ed25519.Verify(env.SenderPubkey, sigData, env.Signature) {
			b.Fatal("signature verification failed")
		}

		// Verify message ID
		hasher := blake3.New()
		hasher.Write(env.Payload)
		computedID := hasher.Sum(nil)

		for j := 0; j < len(computedID); j++ {
			if computedID[j] != env.MessageId[j] {
				b.Fatal("message ID mismatch")
			}
		}
	}
}

// BenchmarkSignatureVerification measures Ed25519 signature verification only.
func BenchmarkSignatureVerification(b *testing.B) {
	pubkey, privkey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}

	message := make([]byte, 2048)
	rand.Read(message)
	signature := ed25519.Sign(privkey, message)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !ed25519.Verify(pubkey, message, signature) {
			b.Fatal("signature verification failed")
		}
	}
}

// BenchmarkMessageIDComputation measures BLAKE3 hashing for message IDs.
func BenchmarkMessageIDComputation(b *testing.B) {
	payload := make([]byte, 2048)
	rand.Read(payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasher := blake3.New()
		hasher.Write(payload)
		_ = hasher.Sum(nil)
	}
}

// BenchmarkEnvelopeMarshaling measures protobuf serialization performance.
func BenchmarkEnvelopeMarshaling(b *testing.B) {
	pubkey, privkey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}

	payload := make([]byte, 2048)
	rand.Read(payload)

	hasher := blake3.New()
	hasher.Write(payload)
	messageID := hasher.Sum(nil)

	timestamp := time.Now().Unix()
	sigData := make([]byte, 4+4+len(payload))
	signature := ed25519.Sign(privkey, sigData)

	envelope := &pb.MurmurEnvelope{
		Version:       1,
		Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		SenderPubkey:  pubkey,
		Signature:     signature,
		TimestampUnix: timestamp,
		MessageId:     messageID,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proto.Marshal(envelope)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEnvelopeUnmarshaling measures protobuf deserialization performance.
func BenchmarkEnvelopeUnmarshaling(b *testing.B) {
	pubkey, privkey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}

	payload := make([]byte, 2048)
	rand.Read(payload)

	hasher := blake3.New()
	hasher.Write(payload)
	messageID := hasher.Sum(nil)

	timestamp := time.Now().Unix()
	sigData := make([]byte, 4+4+len(payload))
	signature := ed25519.Sign(privkey, sigData)

	envelope := &pb.MurmurEnvelope{
		Version:       1,
		Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		SenderPubkey:  pubkey,
		Signature:     signature,
		TimestampUnix: timestamp,
		MessageId:     messageID,
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var env pb.MurmurEnvelope
		if err := proto.Unmarshal(data, &env); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTimestampValidation measures timestamp drift check performance.
func BenchmarkTimestampValidation(b *testing.B) {
	now := time.Now()
	maxDrift := 300 * time.Second

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timestamp := time.Unix(now.Unix(), 0)
		drift := now.Sub(timestamp)
		if drift < 0 {
			drift = -drift
		}
		if drift > maxDrift {
			b.Fatal("timestamp out of range")
		}
	}
}
