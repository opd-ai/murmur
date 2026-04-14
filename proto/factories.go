// Package proto provides message factories for creating MurmurEnvelopes.
// Per TECHNICAL_IMPLEMENTATION.md, each GossipSub message is wrapped in a MurmurEnvelope.
package proto

import (
	"crypto/ed25519"
	"time"

	"google.golang.org/protobuf/proto"
)

// NewWaveEnvelope creates a MurmurEnvelope containing a Wave message.
// The envelope is signed with the provided private key.
func NewWaveEnvelope(wave *Wave, privateKey ed25519.PrivateKey) (*MurmurEnvelope, error) {
	payload, err := proto.Marshal(wave)
	if err != nil {
		return nil, err
	}

	env := &MurmurEnvelope{
		Version:       CurrentProtocolVersion,
		Type:          MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		TimestampUnix: time.Now().Unix(),
		MessageId:     ComputeMessageID(payload),
	}

	if err := SignEnvelope(env, privateKey); err != nil {
		return nil, err
	}

	return env, nil
}

// NewIdentityEnvelope creates a MurmurEnvelope containing an IdentityDeclaration.
func NewIdentityEnvelope(decl *IdentityDeclaration, privateKey ed25519.PrivateKey) (*MurmurEnvelope, error) {
	payload, err := proto.Marshal(decl)
	if err != nil {
		return nil, err
	}

	env := &MurmurEnvelope{
		Version:       CurrentProtocolVersion,
		Type:          MessageType_MESSAGE_TYPE_IDENTITY,
		Payload:       payload,
		TimestampUnix: time.Now().Unix(),
		MessageId:     ComputeMessageID(payload),
	}

	if err := SignEnvelope(env, privateKey); err != nil {
		return nil, err
	}

	return env, nil
}

// NewConnectionAnnouncementEnvelope creates a MurmurEnvelope for a ConnectionAnnouncement.
func NewConnectionAnnouncementEnvelope(announcement *ConnectionAnnouncement, privateKey ed25519.PrivateKey) (*MurmurEnvelope, error) {
	payload, err := proto.Marshal(announcement)
	if err != nil {
		return nil, err
	}

	env := &MurmurEnvelope{
		Version:       CurrentProtocolVersion,
		Type:          MessageType_MESSAGE_TYPE_IDENTITY,
		Payload:       payload,
		TimestampUnix: time.Now().Unix(),
		MessageId:     ComputeMessageID(payload),
	}

	if err := SignEnvelope(env, privateKey); err != nil {
		return nil, err
	}

	return env, nil
}

// NewShroudAdEnvelope creates a MurmurEnvelope containing a RelayAdvertisement.
func NewShroudAdEnvelope(relay *RelayAdvertisement, privateKey ed25519.PrivateKey) (*MurmurEnvelope, error) {
	payload, err := proto.Marshal(relay)
	if err != nil {
		return nil, err
	}

	env := &MurmurEnvelope{
		Version:       CurrentProtocolVersion,
		Type:          MessageType_MESSAGE_TYPE_SHROUD_AD,
		Payload:       payload,
		TimestampUnix: time.Now().Unix(),
		MessageId:     ComputeMessageID(payload),
	}

	if err := SignEnvelope(env, privateKey); err != nil {
		return nil, err
	}

	return env, nil
}

// NewHeartbeatEnvelope creates a MurmurEnvelope containing a Heartbeat.
// Per TECHNICAL_IMPLEMENTATION.md, heartbeats are 64-byte signed timestamps.
func NewHeartbeatEnvelope(heartbeat *Heartbeat, privateKey ed25519.PrivateKey) (*MurmurEnvelope, error) {
	payload, err := proto.Marshal(heartbeat)
	if err != nil {
		return nil, err
	}

	env := &MurmurEnvelope{
		Version:       CurrentProtocolVersion,
		Type:          MessageType_MESSAGE_TYPE_HEARTBEAT,
		Payload:       payload,
		TimestampUnix: time.Now().Unix(),
		MessageId:     ComputeMessageID(payload),
	}

	if err := SignEnvelope(env, privateKey); err != nil {
		return nil, err
	}

	return env, nil
}

// NewAnonymousWaveEnvelope creates a MurmurEnvelope for anonymous (Specter) Waves.
// The sender_pubkey is zeroed and no signature is attached.
func NewAnonymousWaveEnvelope(wave *Wave) (*MurmurEnvelope, error) {
	payload, err := proto.Marshal(wave)
	if err != nil {
		return nil, err
	}

	env := &MurmurEnvelope{
		Version:       CurrentProtocolVersion,
		Type:          MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		SenderPubkey:  make([]byte, PubKeyLength), // zeroed
		Signature:     nil,                        // no signature
		TimestampUnix: time.Now().Unix(),
		MessageId:     ComputeMessageID(payload),
	}

	return env, nil
}

// NewReplyEnvelope creates a MurmurEnvelope containing a Reply message.
func NewReplyEnvelope(reply *Reply, privateKey ed25519.PrivateKey) (*MurmurEnvelope, error) {
	payload, err := proto.Marshal(reply)
	if err != nil {
		return nil, err
	}

	env := &MurmurEnvelope{
		Version:       CurrentProtocolVersion,
		Type:          MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		TimestampUnix: time.Now().Unix(),
		MessageId:     ComputeMessageID(payload),
	}

	if err := SignEnvelope(env, privateKey); err != nil {
		return nil, err
	}

	return env, nil
}

// NewAmplificationEnvelope creates a MurmurEnvelope for an Amplification.
func NewAmplificationEnvelope(amp *Amplification, privateKey ed25519.PrivateKey) (*MurmurEnvelope, error) {
	payload, err := proto.Marshal(amp)
	if err != nil {
		return nil, err
	}

	env := &MurmurEnvelope{
		Version:       CurrentProtocolVersion,
		Type:          MessageType_MESSAGE_TYPE_WAVE,
		Payload:       payload,
		TimestampUnix: time.Now().Unix(),
		MessageId:     ComputeMessageID(payload),
	}

	if err := SignEnvelope(env, privateKey); err != nil {
		return nil, err
	}

	return env, nil
}

// UnwrapWave extracts and validates a Wave from a MurmurEnvelope.
func UnwrapWave(env *MurmurEnvelope) (*Wave, error) {
	if err := ValidateEnvelope(env); err != nil {
		return nil, err
	}

	if env.Type != MessageType_MESSAGE_TYPE_WAVE {
		return nil, ErrInvalidMessageType
	}

	wave := &Wave{}
	if err := proto.Unmarshal(env.Payload, wave); err != nil {
		return nil, err
	}

	if err := ValidateWave(wave); err != nil {
		return nil, err
	}

	return wave, nil
}

// UnwrapIdentityDeclaration extracts an IdentityDeclaration from a MurmurEnvelope.
func UnwrapIdentityDeclaration(env *MurmurEnvelope) (*IdentityDeclaration, error) {
	if err := ValidateEnvelope(env); err != nil {
		return nil, err
	}

	if env.Type != MessageType_MESSAGE_TYPE_IDENTITY {
		return nil, ErrInvalidMessageType
	}

	decl := &IdentityDeclaration{}
	if err := proto.Unmarshal(env.Payload, decl); err != nil {
		return nil, err
	}

	return decl, nil
}

// UnwrapRelayAdvertisement extracts a RelayAdvertisement from a MurmurEnvelope.
func UnwrapRelayAdvertisement(env *MurmurEnvelope) (*RelayAdvertisement, error) {
	if err := ValidateEnvelope(env); err != nil {
		return nil, err
	}

	if env.Type != MessageType_MESSAGE_TYPE_SHROUD_AD {
		return nil, ErrInvalidMessageType
	}

	relay := &RelayAdvertisement{}
	if err := proto.Unmarshal(env.Payload, relay); err != nil {
		return nil, err
	}

	return relay, nil
}

// UnwrapHeartbeat extracts a Heartbeat from a MurmurEnvelope.
func UnwrapHeartbeat(env *MurmurEnvelope) (*Heartbeat, error) {
	if err := ValidateEnvelope(env); err != nil {
		return nil, err
	}

	if env.Type != MessageType_MESSAGE_TYPE_HEARTBEAT {
		return nil, ErrInvalidMessageType
	}

	heartbeat := &Heartbeat{}
	if err := proto.Unmarshal(env.Payload, heartbeat); err != nil {
		return nil, err
	}

	return heartbeat, nil
}

// CreateSurfaceWave creates a new Surface Wave with default TTL.
func CreateSurfaceWave(content, authorPubkey []byte) *Wave {
	return &Wave{
		WaveType:     WaveType_WAVE_TYPE_SURFACE,
		Content:      content,
		AuthorPubkey: authorPubkey,
		CreatedAt:    time.Now().Unix(),
		TtlSeconds:   DefaultTTLSeconds,
		HopCount:     0,
	}
}

// CreateReplyWave creates a Reply Wave linked to a parent.
func CreateReplyWave(content, authorPubkey, parentHash []byte) *Wave {
	return &Wave{
		WaveType:     WaveType_WAVE_TYPE_REPLY,
		Content:      content,
		AuthorPubkey: authorPubkey,
		ParentHash:   parentHash,
		CreatedAt:    time.Now().Unix(),
		TtlSeconds:   DefaultTTLSeconds,
		HopCount:     0,
	}
}

// CreateSpecterWave creates an anonymous Specter Wave.
func CreateSpecterWave(content []byte) *Wave {
	return &Wave{
		WaveType:     WaveType_WAVE_TYPE_SPECTER,
		Content:      content,
		AuthorPubkey: nil, // anonymous
		CreatedAt:    time.Now().Unix(),
		TtlSeconds:   DefaultTTLSeconds,
		HopCount:     0,
	}
}

// CreateBeaconWave creates a Beacon Wave (requires elevated PoW).
func CreateBeaconWave(content, authorPubkey []byte) *Wave {
	return &Wave{
		WaveType:     WaveType_WAVE_TYPE_BEACON,
		Content:      content,
		AuthorPubkey: authorPubkey,
		CreatedAt:    time.Now().Unix(),
		TtlSeconds:   DefaultTTLSeconds,
		HopCount:     0,
	}
}

// CreateHeartbeat creates a new Heartbeat message.
func CreateHeartbeat(peerID string) *Heartbeat {
	return &Heartbeat{
		PeerId:    peerID,
		Timestamp: time.Now().Unix(),
		Sequence:  0,
	}
}
