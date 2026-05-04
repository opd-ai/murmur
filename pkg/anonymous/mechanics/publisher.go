// Package mechanics - Shared publisher definitions for anonymous game mechanics.
package mechanics

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// TopicAnonymousMechanics is the GossipSub topic for all anonymous mechanics events.
// Per TECHNICAL_IMPLEMENTATION.md, all mechanics events go to /murmur/anonymous/mechanics/1.0.
const TopicAnonymousMechanics = "/murmur/anonymous/mechanics/1.0"

// Publisher provides an interface for publishing to GossipSub.
// This abstracts the networking layer from the mechanics package.
type Publisher interface {
	Publish(ctx context.Context, topicName string, data []byte) error
}

// Publication errors.
var (
	ErrPublisherNotSet   = errors.New("publisher not set")
	ErrMissingSignature  = errors.New("missing signature")
	ErrSignatureFailed   = errors.New("signature verification failed")
	ErrMissingPrivateKey = errors.New("private key required for signing")
)

// KeyToHex converts a byte slice to hexadecimal string for logging/debugging.
// This is exported so subpackages can use it.
func KeyToHex(key []byte) string {
	return fmt.Sprintf("%x", key)
}

// HexToKey converts a hex string to a byte array.
// This is exported so subpackages can use it.
func HexToKey(hex string, dst []byte) {
	for i := 0; i < len(dst) && i*2+1 < len(hex); i++ {
		dst[i] = hexDigit(hex[i*2])<<4 | hexDigit(hex[i*2+1])
	}
}

func hexDigit(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}

// VerifyEd25519Signature is a helper for verifying Ed25519 signatures in mechanics events.
// It checks that event signature exists, nested signature exists, pubkey is valid size,
// and performs Ed25519 verification. Returns nil on success.
func VerifyEd25519Signature(
	eventSig []byte,
	nestedSig []byte,
	pubkey []byte,
	sigData []byte,
) error {
	if len(eventSig) == 0 {
		return ErrMissingSignature
	}

	if len(nestedSig) == 0 {
		return ErrMissingSignature
	}

	if len(pubkey) != ed25519.PublicKeySize {
		return ErrSignatureFailed
	}

	if ed25519.Verify(pubkey, sigData, eventSig) {
		return nil
	}

	return ErrSignatureFailed
}

// EventExtractor extracts a specific event type from a GossipMessage.
type EventExtractor[T any] func(*pb.GossipMessage) T

// EventVerifier verifies an event's signature.
type EventVerifier[T any] func(T) error

// EventProcessor processes a verified event.
type EventProcessor[T any] func(T) error

// ProcessGossipEvent unmarshals a GossipMessage, extracts a typed event,
// verifies its signature, and processes it. This consolidates the common
// pattern used across all mechanics receivers.
func ProcessGossipEvent[T any](
	data []byte,
	extract EventExtractor[T],
	verify EventVerifier[T],
	process EventProcessor[T],
) error {
	var gossipMsg pb.GossipMessage
	if err := proto.Unmarshal(data, &gossipMsg); err != nil {
		return fmt.Errorf("failed to unmarshal gossip message: %w", err)
	}

	event := extract(&gossipMsg)
	var zero T
	if any(event) == any(zero) {
		return nil // Event type not present in message.
	}

	if err := verify(event); err != nil {
		return err
	}

	return process(event)
}
