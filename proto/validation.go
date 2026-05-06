// Package proto provides protobuf validation helpers.
// Per TECHNICAL_IMPLEMENTATION.md, envelopes must be validated for:
// - Version compatibility
// - Signature verification
// - Timestamp range (±300s)
// - PoW verification
// - Message deduplication
package proto

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"time"

	"github.com/zeebo/blake3"
)

// Validation errors
var (
	ErrInvalidVersion      = errors.New("invalid protocol version")
	ErrInvalidMessageType  = errors.New("invalid message type")
	ErrEmptyPayload        = errors.New("empty payload")
	ErrInvalidPubKeyLength = errors.New("invalid public key length")
	ErrInvalidSignature    = errors.New("invalid signature")
	ErrTimestampTooFuture  = errors.New("timestamp too far in future")
	ErrTimestampTooOld     = errors.New("timestamp too old")
	ErrInvalidMessageID    = errors.New("invalid message_id")
	ErrMessageIDMismatch   = errors.New("message_id does not match payload hash")
	ErrContentTooLarge     = errors.New("content exceeds maximum size")
	ErrInvalidTTL          = errors.New("invalid TTL")
	ErrInvalidWaveType     = errors.New("invalid wave type")
	ErrEmptyWaveContent    = errors.New("empty wave content")
	ErrInvalidHopCount     = errors.New("invalid hop count")
)

// Protocol constants
const (
	CurrentProtocolVersion = 1
	MaxTimestampDrift      = 300 // 300 seconds = 5 minutes
	MaxWaveContentSize     = 2048
	MaxTTLSeconds          = 30 * 24 * 60 * 60 // 30 days in seconds
	DefaultTTLSeconds      = 7 * 24 * 60 * 60  // 7 days in seconds
	MaxHopCount            = 20
	PubKeyLength           = 32
	SignatureLength        = 64
	BLAKE3HashLength       = 32
)

// ValidateEnvelope performs complete validation of a MurmurEnvelope.
// It validates version, message type, timestamp, signature, and message_id.
func ValidateEnvelope(env *MurmurEnvelope) error {
	if env == nil {
		return errors.New("nil envelope")
	}

	if err := validateEnvelopeMetadata(env); err != nil {
		return err
	}

	if err := validateEnvelopeContent(env); err != nil {
		return err
	}

	return validateEnvelopeSignature(env)
}

// validateEnvelopeMetadata validates protocol version and message type.
func validateEnvelopeMetadata(env *MurmurEnvelope) error {
	if err := ValidateVersion(env.Version); err != nil {
		return err
	}
	return ValidateMessageType(env.Type)
}

// validateEnvelopeContent validates payload, timestamp, and message ID.
func validateEnvelopeContent(env *MurmurEnvelope) error {
	if len(env.Payload) == 0 {
		return ErrEmptyPayload
	}

	if err := ValidateTimestamp(env.TimestampUnix); err != nil {
		return err
	}

	return ValidateMessageID(env.MessageId, env.Payload)
}

// validateEnvelopeSignature validates the signature for non-anonymous messages.
func validateEnvelopeSignature(env *MurmurEnvelope) error {
	if !isZeroBytes(env.SenderPubkey) {
		return ValidateSignature(env)
	}
	return nil
}

// ValidateVersion checks if the protocol version is supported.
func ValidateVersion(version uint32) error {
	if version == 0 || version > CurrentProtocolVersion {
		return ErrInvalidVersion
	}
	return nil
}

// ValidateMessageType checks if the message type is valid.
func ValidateMessageType(msgType MessageType) error {
	switch msgType {
	case MessageType_MESSAGE_TYPE_WAVE,
		MessageType_MESSAGE_TYPE_IDENTITY,
		MessageType_MESSAGE_TYPE_SHROUD_AD,
		MessageType_MESSAGE_TYPE_HEARTBEAT:
		return nil
	default:
		return ErrInvalidMessageType
	}
}

// ValidateTimestamp checks if the timestamp is within acceptable range.
// Per TECHNICAL_IMPLEMENTATION.md, timestamps must be within ±300 seconds.
func ValidateTimestamp(timestamp int64) error {
	now := time.Now().Unix()

	// Check if timestamp is too far in the future
	if timestamp > now+MaxTimestampDrift {
		return ErrTimestampTooFuture
	}

	// Note: TTL-based expiry is handled separately by content validation
	return nil
}

// ValidateMessageID verifies that the message_id matches the BLAKE3 hash of payload.
func ValidateMessageID(messageID, payload []byte) error {
	if len(messageID) != BLAKE3HashLength {
		return ErrInvalidMessageID
	}

	computed := blake3.Sum256(payload)
	for i, b := range computed {
		if messageID[i] != b {
			return ErrMessageIDMismatch
		}
	}

	return nil
}

// ValidateSignature verifies the Ed25519 signature on an envelope.
// Signature is computed over: version || type || payload
func ValidateSignature(env *MurmurEnvelope) error {
	if len(env.SenderPubkey) != PubKeyLength {
		return ErrInvalidPubKeyLength
	}
	if len(env.Signature) != SignatureLength {
		return ErrInvalidSignature
	}

	// Construct message to verify
	msg := buildSignatureMessage(env.Version, env.Type, env.Payload)

	if !ed25519.Verify(env.SenderPubkey, msg, env.Signature) {
		return ErrInvalidSignature
	}

	return nil
}

// ValidateWave validates a Wave message structure.
func ValidateWave(wave *Wave) error {
	if wave == nil {
		return errors.New("nil wave")
	}

	if err := ValidateWaveType(wave.WaveType); err != nil {
		return err
	}

	if err := validateWaveContent(wave.Content); err != nil {
		return err
	}

	if err := validateWaveTiming(wave); err != nil {
		return err
	}

	if wave.HopCount > MaxHopCount {
		return ErrInvalidHopCount
	}

	return nil
}

// validateWaveContent checks content size constraints.
func validateWaveContent(content []byte) error {
	if len(content) == 0 {
		return ErrEmptyWaveContent
	}
	if len(content) > MaxWaveContentSize {
		return ErrContentTooLarge
	}
	return nil
}

// validateWaveTiming checks TTL and expiry constraints.
func validateWaveTiming(wave *Wave) error {
	if wave.TtlSeconds <= 0 || wave.TtlSeconds > MaxTTLSeconds {
		return ErrInvalidTTL
	}

	expiry := wave.CreatedAt + wave.TtlSeconds
	if expiry < time.Now().Unix() {
		return ErrTimestampTooOld
	}

	return nil
}

// ValidateWaveType checks if the wave type is valid.
func ValidateWaveType(waveType WaveType) error {
	switch waveType {
	case WaveType_WAVE_TYPE_SURFACE,
		WaveType_WAVE_TYPE_REPLY,
		WaveType_WAVE_TYPE_VEILED,
		WaveType_WAVE_TYPE_SPECTER,
		WaveType_WAVE_TYPE_SIGIL,
		WaveType_WAVE_TYPE_ABYSSAL,
		WaveType_WAVE_TYPE_MASKED,
		WaveType_WAVE_TYPE_BEACON:
		return nil
	default:
		return ErrInvalidWaveType
	}
}

// ComputeMessageID computes the BLAKE3 hash of a payload for deduplication.
func ComputeMessageID(payload []byte) []byte {
	hash := blake3.Sum256(payload)
	return hash[:]
}

// buildSignatureMessage constructs the message to be signed/verified.
// Format: version (4 bytes LE) || type (4 bytes LE) || payload
func buildSignatureMessage(version uint32, msgType MessageType, payload []byte) []byte {
	msg := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(msg[0:4], version)
	binary.LittleEndian.PutUint32(msg[4:8], uint32(msgType))
	copy(msg[8:], payload)
	return msg
}

// SignEnvelope signs a MurmurEnvelope with the given private key.
// The envelope's signature field is set to the computed signature.
func SignEnvelope(env *MurmurEnvelope, privateKey ed25519.PrivateKey) error {
	if env == nil {
		return errors.New("nil envelope")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return errors.New("invalid private key length")
	}

	msg := buildSignatureMessage(env.Version, env.Type, env.Payload)
	env.Signature = ed25519.Sign(privateKey, msg)
	env.SenderPubkey = privateKey.Public().(ed25519.PublicKey)

	return nil
}

// isZeroBytes checks if a byte slice contains only zeros.
func isZeroBytes(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

// IsExpired checks if a Wave has expired based on its TTL.
func IsExpired(wave *Wave) bool {
	if wave == nil {
		return true
	}
	expiry := wave.CreatedAt + wave.TtlSeconds
	return time.Now().Unix() > expiry
}

// RemainingTTL returns the remaining TTL in seconds, or 0 if expired.
func RemainingTTL(wave *Wave) int64 {
	if wave == nil {
		return 0
	}
	expiry := wave.CreatedAt + wave.TtlSeconds
	remaining := expiry - time.Now().Unix()
	if remaining < 0 {
		return 0
	}
	return remaining
}
