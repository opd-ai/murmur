// Package store provides typed accessor methods for MURMUR's Bbolt database.
// Per TECHNICAL_IMPLEMENTATION.md, the database stores protobuf-encoded messages.
package store

import (
	"fmt"

	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// Wave accessors

// GetWave retrieves a Wave by its ID.
func (db *DB) GetWave(waveID []byte) (*pb.Wave, error) {
	data, err := db.Get(BucketWaves, waveID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	wave := &pb.Wave{}
	if err := proto.Unmarshal(data, wave); err != nil {
		return nil, fmt.Errorf("unmarshaling wave: %w", err)
	}
	return wave, nil
}

// PutWave stores a Wave by its ID.
func (db *DB) PutWave(wave *pb.Wave) error {
	if wave == nil {
		return fmt.Errorf("nil wave")
	}
	if len(wave.WaveId) == 0 {
		return fmt.Errorf("wave has no ID")
	}

	data, err := proto.Marshal(wave)
	if err != nil {
		return fmt.Errorf("marshaling wave: %w", err)
	}
	return db.Put(BucketWaves, wave.WaveId, data)
}

// DeleteWave removes a Wave by its ID.
func (db *DB) DeleteWave(waveID []byte) error {
	return db.Delete(BucketWaves, waveID)
}

// ListWaves returns all stored Waves.
func (db *DB) ListWaves() ([]*pb.Wave, error) {
	var waves []*pb.Wave
	err := db.ForEach(BucketWaves, func(_, value []byte) error {
		wave := &pb.Wave{}
		if err := proto.Unmarshal(value, wave); err != nil {
			return fmt.Errorf("unmarshaling wave: %w", err)
		}
		waves = append(waves, wave)
		return nil
	})
	return waves, err
}

// Identity accessors

// GetIdentityDeclaration retrieves an IdentityDeclaration by public key.
func (db *DB) GetIdentityDeclaration(pubKey []byte) (*pb.IdentityDeclaration, error) {
	data, err := db.Get(BucketIdentity, pubKey)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	decl := &pb.IdentityDeclaration{}
	if err := proto.Unmarshal(data, decl); err != nil {
		return nil, fmt.Errorf("unmarshaling identity declaration: %w", err)
	}
	return decl, nil
}

// PutIdentityDeclaration stores an IdentityDeclaration by its public key.
func (db *DB) PutIdentityDeclaration(decl *pb.IdentityDeclaration) error {
	if decl == nil {
		return fmt.Errorf("nil identity declaration")
	}
	if len(decl.PublicKey) == 0 {
		return fmt.Errorf("identity declaration has no public key")
	}

	data, err := proto.Marshal(decl)
	if err != nil {
		return fmt.Errorf("marshaling identity declaration: %w", err)
	}
	return db.Put(BucketIdentity, decl.PublicKey, data)
}

// DeleteIdentityDeclaration removes an IdentityDeclaration by public key.
func (db *DB) DeleteIdentityDeclaration(pubKey []byte) error {
	return db.Delete(BucketIdentity, pubKey)
}

// Peer accessors

// GetPeerRecord retrieves a PeerRecord by peer ID.
func (db *DB) GetPeerRecord(peerID string) (*pb.PeerRecord, error) {
	data, err := db.Get(BucketPeers, []byte(peerID))
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	peer := &pb.PeerRecord{}
	if err := proto.Unmarshal(data, peer); err != nil {
		return nil, fmt.Errorf("unmarshaling peer record: %w", err)
	}
	return peer, nil
}

// PutPeerRecord stores a PeerRecord by its peer ID.
func (db *DB) PutPeerRecord(peer *pb.PeerRecord) error {
	if peer == nil {
		return fmt.Errorf("nil peer record")
	}
	if peer.PeerId == "" {
		return fmt.Errorf("peer record has no peer ID")
	}

	data, err := proto.Marshal(peer)
	if err != nil {
		return fmt.Errorf("marshaling peer record: %w", err)
	}
	return db.Put(BucketPeers, []byte(peer.PeerId), data)
}

// DeletePeerRecord removes a PeerRecord by peer ID.
func (db *DB) DeletePeerRecord(peerID string) error {
	return db.Delete(BucketPeers, []byte(peerID))
}

// ListPeerRecords returns all stored PeerRecords.
func (db *DB) ListPeerRecords() ([]*pb.PeerRecord, error) {
	var peers []*pb.PeerRecord
	err := db.ForEach(BucketPeers, func(_, value []byte) error {
		peer := &pb.PeerRecord{}
		if err := proto.Unmarshal(value, peer); err != nil {
			return fmt.Errorf("unmarshaling peer record: %w", err)
		}
		peers = append(peers, peer)
		return nil
	})
	return peers, err
}

// Resonance accessors

// GetResonanceScore retrieves a ResonanceScore by identity key.
func (db *DB) GetResonanceScore(identityKey []byte) (*pb.ResonanceScore, error) {
	data, err := db.Get(BucketResonance, identityKey)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	score := &pb.ResonanceScore{}
	if err := proto.Unmarshal(data, score); err != nil {
		return nil, fmt.Errorf("unmarshaling resonance score: %w", err)
	}
	return score, nil
}

// PutResonanceScore stores a ResonanceScore by identity key.
func (db *DB) PutResonanceScore(identityKey []byte, score *pb.ResonanceScore) error {
	if score == nil {
		return fmt.Errorf("nil resonance score")
	}
	if len(identityKey) == 0 {
		return fmt.Errorf("empty identity key")
	}

	data, err := proto.Marshal(score)
	if err != nil {
		return fmt.Errorf("marshaling resonance score: %w", err)
	}
	return db.Put(BucketResonance, identityKey, data)
}

// Shroud accessors

// GetRelayAdvertisement retrieves a RelayAdvertisement by node ID.
func (db *DB) GetRelayAdvertisement(nodeID []byte) (*pb.RelayAdvertisement, error) {
	data, err := db.Get(BucketShroud, nodeID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	relay := &pb.RelayAdvertisement{}
	if err := proto.Unmarshal(data, relay); err != nil {
		return nil, fmt.Errorf("unmarshaling relay advertisement: %w", err)
	}
	return relay, nil
}

// PutRelayAdvertisement stores a RelayAdvertisement.
func (db *DB) PutRelayAdvertisement(relay *pb.RelayAdvertisement) error {
	if relay == nil {
		return fmt.Errorf("nil relay advertisement")
	}
	if len(relay.Ed25519Pubkey) == 0 {
		return fmt.Errorf("relay advertisement has no ed25519 pubkey")
	}

	data, err := proto.Marshal(relay)
	if err != nil {
		return fmt.Errorf("marshaling relay advertisement: %w", err)
	}
	return db.Put(BucketShroud, relay.Ed25519Pubkey, data)
}

// ListRelayAdvertisements returns all stored RelayAdvertisements.
func (db *DB) ListRelayAdvertisements() ([]*pb.RelayAdvertisement, error) {
	var relays []*pb.RelayAdvertisement
	err := db.ForEach(BucketShroud, func(_, value []byte) error {
		relay := &pb.RelayAdvertisement{}
		if err := proto.Unmarshal(value, relay); err != nil {
			return fmt.Errorf("unmarshaling relay advertisement: %w", err)
		}
		relays = append(relays, relay)
		return nil
	})
	return relays, err
}

// Config accessors

// GetConfigValue retrieves a config value by key.
func (db *DB) GetConfigValue(key string) ([]byte, error) {
	return db.Get(BucketConfig, []byte(key))
}

// PutConfigValue stores a config value by key.
func (db *DB) PutConfigValue(key string, value []byte) error {
	return db.Put(BucketConfig, []byte(key), value)
}

// DeleteConfigValue removes a config value by key.
func (db *DB) DeleteConfigValue(key string) error {
	return db.Delete(BucketConfig, []byte(key))
}

// Thread accessors (for reply chain indexing)

// GetThreadRoot retrieves the thread root Wave ID for a given Wave.
func (db *DB) GetThreadRoot(waveID []byte) ([]byte, error) {
	return db.Get(BucketThreads, waveID)
}

// PutThreadRoot stores the thread root mapping for a Wave.
func (db *DB) PutThreadRoot(waveID, rootID []byte) error {
	return db.Put(BucketThreads, waveID, rootID)
}

// DeleteThreadRoot removes a thread root mapping.
func (db *DB) DeleteThreadRoot(waveID []byte) error {
	return db.Delete(BucketThreads, waveID)
}
