// Package app provides broadcast functionality for publishing content to the network.
// Per TECHNICAL_IMPLEMENTATION.md §3, all outgoing messages are wrapped in MurmurEnvelope.
package app

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"time"

	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
	pb "github.com/opd-ai/murmur/proto"
	"github.com/zeebo/blake3"
	"google.golang.org/protobuf/proto"
)

// Broadcast errors.
var (
	ErrNotInitialized   = errors.New("application not initialized")
	ErrNilWave          = errors.New("wave is nil")
	ErrNilDeclaration   = errors.New("declaration is nil")
	ErrNilHeartbeat     = errors.New("heartbeat is nil")
	ErrNilAdvertisement = errors.New("advertisement is nil")
	ErrNotRelay         = errors.New("node is not configured as relay")
)

// BroadcastWave publishes a Wave to the network via GossipSub.
// The Wave is wrapped in a MurmurEnvelope with signature, timestamp, and message ID.
// Per TECHNICAL_IMPLEMENTATION.md §3, the envelope signature is over (version || type || payload).
func (a *App) BroadcastWave(ctx context.Context, wave *pb.Wave) error {
	if wave == nil {
		return ErrNilWave
	}

	a.mu.RLock()
	if !a.running || a.subsystems == nil || a.subsystems.PubSub == nil {
		a.mu.RUnlock()
		return ErrNotInitialized
	}
	identity := a.subsystems.Identity
	ps := a.subsystems.PubSub
	cache := a.subsystems.WaveCache
	a.mu.RUnlock()

	// Serialize the Wave.
	payload, err := proto.Marshal(wave)
	if err != nil {
		return err
	}

	// Create and sign the envelope.
	envelope, err := createSignedEnvelope(
		pb.MessageType_MESSAGE_TYPE_WAVE,
		payload,
		identity,
	)
	if err != nil {
		return err
	}

	// Serialize the envelope.
	data, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}

	// Store locally first.
	if cache != nil {
		if err := cache.Put(wave); err != nil {
			// Log but don't fail - publishing is more important
		}
	}

	// Publish to network.
	return ps.Publish(ctx, gossip.TopicWaves, data)
}

// BroadcastIdentity publishes an identity declaration to the network.
func (a *App) BroadcastIdentity(ctx context.Context, decl *pb.IdentityDeclaration) error {
	if decl == nil {
		return ErrNilDeclaration
	}

	a.mu.RLock()
	if !a.running || a.subsystems == nil || a.subsystems.PubSub == nil {
		a.mu.RUnlock()
		return ErrNotInitialized
	}
	identity := a.subsystems.Identity
	ps := a.subsystems.PubSub
	a.mu.RUnlock()

	// Sign the declaration if not already signed.
	if len(decl.Signature) == 0 {
		sigData := identityDeclarationSignatureData(decl)
		decl.Signature = ed25519.Sign(identity.PrivateKey, sigData)
	}

	// Serialize and publish.
	payload, err := proto.Marshal(decl)
	if err != nil {
		return err
	}
	return createAndPublishEnvelope(ctx, ps, gossip.TopicIdentity, pb.MessageType_MESSAGE_TYPE_IDENTITY, payload, identity)
}

// BroadcastHeartbeat publishes a heartbeat to the network.
// Per TECHNICAL_IMPLEMENTATION.md, heartbeats are sent every 30 seconds.
func (a *App) BroadcastHeartbeat(ctx context.Context) error {
	a.mu.RLock()
	if !a.running || a.subsystems == nil || a.subsystems.PubSub == nil || a.subsystems.Host == nil {
		a.mu.RUnlock()
		return ErrNotInitialized
	}
	identity := a.subsystems.Identity
	ps := a.subsystems.PubSub
	peerID := a.subsystems.Host.PeerID()
	a.mu.RUnlock()

	// Create heartbeat.
	now := time.Now().Unix()
	hb := &pb.Heartbeat{
		PeerId:    peerID.String(),
		PublicKey: identity.PublicKey,
		Timestamp: now,
		Sequence:  uint64(now), // Use timestamp as monotonic sequence
	}

	// Sign the heartbeat.
	sigData := heartbeatSignatureData(hb)
	hb.Signature = ed25519.Sign(identity.PrivateKey, sigData)

	// Serialize and publish.
	payload, err := proto.Marshal(hb)
	if err != nil {
		return err
	}
	return createAndPublishEnvelope(ctx, ps, gossip.TopicPulse, pb.MessageType_MESSAGE_TYPE_HEARTBEAT, payload, identity)
}

// BroadcastRelayAdvertisement publishes a relay advertisement to the network.
// Per SHADOW_GRADIENT.md, relays advertise availability on /murmur/shroud/1.
// Returns ErrNotRelay if this node is not configured as a relay.
func (a *App) BroadcastRelayAdvertisement(ctx context.Context) error {
	a.mu.RLock()
	if !a.running || a.subsystems == nil || a.subsystems.PubSub == nil || a.subsystems.Host == nil {
		a.mu.RUnlock()
		return ErrNotInitialized
	}
	if a.subsystems.Beacon == nil {
		a.mu.RUnlock()
		return ErrNotRelay
	}
	identity := a.subsystems.Identity
	ps := a.subsystems.PubSub
	host := a.subsystems.Host
	beacon := a.subsystems.Beacon
	a.mu.RUnlock()

	// Get node addresses.
	var addrs []string
	for _, addr := range host.Addrs() {
		addrs = append(addrs, addr.String())
	}

	// Generate advertisement.
	ad := beacon.GenerateAdvertisement(identity.PublicKey, identity.PrivateKey, addrs)
	if ad == nil {
		return ErrNotRelay
	}

	// Serialize and publish.
	payload, err := proto.Marshal(ad)
	if err != nil {
		return err
	}
	return createAndPublishEnvelope(ctx, ps, gossip.TopicShroud, pb.MessageType_MESSAGE_TYPE_SHROUD_AD, payload, identity)
}

// CreateWave creates a new Wave with the application's identity and broadcasts it.
// This is a convenience method that combines Wave creation and broadcasting.
func (a *App) CreateWave(ctx context.Context, content []byte, waveType waves.WaveType, opts waves.CreateOptions) (*pb.Wave, error) {
	a.mu.RLock()
	if !a.running || a.subsystems == nil {
		a.mu.RUnlock()
		return nil, ErrNotInitialized
	}
	identity := a.subsystems.Identity
	a.mu.RUnlock()

	// Create the Wave.
	wave, err := waves.Create(waveType, content, identity, opts)
	if err != nil {
		return nil, err
	}

	// Broadcast to network.
	if err := a.BroadcastWave(ctx, wave); err != nil {
		return nil, err
	}

	return wave, nil
}

// CreateSurfaceWave creates and broadcasts a standard Surface Layer Wave.
func (a *App) CreateSurfaceWave(ctx context.Context, content []byte) (*pb.Wave, error) {
	return a.CreateWave(ctx, content, waves.TypeSurface, waves.DefaultCreateOptions())
}

// CreateReplyWave creates and broadcasts a reply to another Wave.
func (a *App) CreateReplyWave(ctx context.Context, content, parentHash []byte) (*pb.Wave, error) {
	opts := waves.DefaultCreateOptions()
	opts.ParentHash = parentHash
	return a.CreateWave(ctx, content, waves.TypeReply, opts)
}

// createSignedEnvelope creates a MurmurEnvelope with signature.
func createSignedEnvelope(msgType pb.MessageType, payload []byte, kp *keys.KeyPair) (*pb.MurmurEnvelope, error) {
	// Compute message ID (BLAKE3 hash of payload).
	msgID := blake3.Sum256(payload)

	envelope := &pb.MurmurEnvelope{
		Version:       ProtocolVersion,
		Type:          msgType,
		Payload:       payload,
		SenderPubkey:  kp.PublicKey,
		TimestampUnix: time.Now().Unix(),
		MessageId:     msgID[:],
	}

	// Sign: version || type || payload.
	sigData := envelopeSignatureData(envelope)
	envelope.Signature = ed25519.Sign(kp.PrivateKey, sigData)

	return envelope, nil
}

// createAndPublishEnvelope creates a signed envelope and publishes it to the given topic.
// This is a convenience helper that combines envelope creation, marshaling, and publishing.
func createAndPublishEnvelope(
	ctx context.Context,
	ps *gossip.PubSub,
	topic string,
	msgType pb.MessageType,
	payload []byte,
	identity *keys.KeyPair,
) error {
	envelope, err := createSignedEnvelope(msgType, payload, identity)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}

	return ps.Publish(ctx, topic, data)
}

// envelopeSignatureData returns the data to sign for an envelope.
func envelopeSignatureData(env *pb.MurmurEnvelope) []byte {
	var data []byte

	// Version as 4 bytes (big-endian).
	versionBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(versionBytes, env.Version)
	data = append(data, versionBytes...)

	// Type as 4 bytes (big-endian).
	typeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(typeBytes, uint32(env.Type))
	data = append(data, typeBytes...)

	// Payload.
	data = append(data, env.Payload...)

	return data
}

// identityDeclarationSignatureData returns the data to sign for an identity declaration.
func identityDeclarationSignatureData(decl *pb.IdentityDeclaration) []byte {
	var data []byte
	data = append(data, decl.PublicKey...)
	data = append(data, []byte(decl.DisplayName)...)
	data = append(data, []byte(decl.Bio)...)

	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(decl.CreatedAt))
	data = append(data, ts...)

	ver := make([]byte, 4)
	binary.BigEndian.PutUint32(ver, decl.Version)
	data = append(data, ver...)

	data = append(data, decl.SigilPng...)

	mode := make([]byte, 4)
	binary.BigEndian.PutUint32(mode, uint32(decl.PrivacyMode))
	data = append(data, mode...)

	return data
}

// heartbeatSignatureData returns the data to sign for a heartbeat.
func heartbeatSignatureData(hb *pb.Heartbeat) []byte {
	var data []byte
	data = append(data, []byte(hb.PeerId)...)

	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(hb.Timestamp))
	data = append(data, ts...)

	return data
}
