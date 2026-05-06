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

// CipherPuzzle accessors

// GetCipherPuzzle retrieves a CipherPuzzle by its ID.
func (db *DB) GetCipherPuzzle(puzzleID []byte) (*pb.CipherPuzzle, error) {
	data, err := db.Get(BucketPuzzles, puzzleID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	puzzle := &pb.CipherPuzzle{}
	if err := proto.Unmarshal(data, puzzle); err != nil {
		return nil, fmt.Errorf("unmarshaling cipher puzzle: %w", err)
	}
	return puzzle, nil
}

// PutCipherPuzzle stores a CipherPuzzle by its ID.
func (db *DB) PutCipherPuzzle(puzzle *pb.CipherPuzzle) error {
	if puzzle == nil {
		return fmt.Errorf("nil cipher puzzle")
	}
	if len(puzzle.Id) == 0 {
		return fmt.Errorf("cipher puzzle has no ID")
	}

	data, err := proto.Marshal(puzzle)
	if err != nil {
		return fmt.Errorf("marshaling cipher puzzle: %w", err)
	}
	return db.Put(BucketPuzzles, puzzle.Id, data)
}

// DeleteCipherPuzzle removes a CipherPuzzle by its ID.
func (db *DB) DeleteCipherPuzzle(puzzleID []byte) error {
	return db.Delete(BucketPuzzles, puzzleID)
}

// ListCipherPuzzles returns all stored CipherPuzzles.
func (db *DB) ListCipherPuzzles() ([]*pb.CipherPuzzle, error) {
	var puzzles []*pb.CipherPuzzle
	err := db.ForEach(BucketPuzzles, func(_, value []byte) error {
		puzzle := &pb.CipherPuzzle{}
		if err := proto.Unmarshal(value, puzzle); err != nil {
			return fmt.Errorf("unmarshaling cipher puzzle: %w", err)
		}
		puzzles = append(puzzles, puzzle)
		return nil
	})
	return puzzles, err
}

// ListActiveCipherPuzzles returns puzzles in ACTIVE state.
func (db *DB) ListActiveCipherPuzzles() ([]*pb.CipherPuzzle, error) {
	var puzzles []*pb.CipherPuzzle
	err := db.ForEach(BucketPuzzles, func(_, value []byte) error {
		puzzle := &pb.CipherPuzzle{}
		if err := proto.Unmarshal(value, puzzle); err != nil {
			return fmt.Errorf("unmarshaling cipher puzzle: %w", err)
		}
		if puzzle.State == pb.PuzzleState_PUZZLE_STATE_ACTIVE {
			puzzles = append(puzzles, puzzle)
		}
		return nil
	})
	return puzzles, err
}

// SpecterHunt accessors

// GetSpecterHunt retrieves a SpecterHunt by its ID.
func (db *DB) GetSpecterHunt(huntID []byte) (*pb.SpecterHunt, error) {
	data, err := db.Get(BucketHunts, huntID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	hunt := &pb.SpecterHunt{}
	if err := proto.Unmarshal(data, hunt); err != nil {
		return nil, fmt.Errorf("unmarshaling specter hunt: %w", err)
	}
	return hunt, nil
}

// PutSpecterHunt stores a SpecterHunt by its ID.
func (db *DB) PutSpecterHunt(hunt *pb.SpecterHunt) error {
	if hunt == nil {
		return fmt.Errorf("nil specter hunt")
	}
	if len(hunt.Id) == 0 {
		return fmt.Errorf("specter hunt has no ID")
	}

	data, err := proto.Marshal(hunt)
	if err != nil {
		return fmt.Errorf("marshaling specter hunt: %w", err)
	}
	return db.Put(BucketHunts, hunt.Id, data)
}

// DeleteSpecterHunt removes a SpecterHunt by its ID.
func (db *DB) DeleteSpecterHunt(huntID []byte) error {
	return db.Delete(BucketHunts, huntID)
}

// ListSpecterHunts returns all stored SpecterHunts.
func (db *DB) ListSpecterHunts() ([]*pb.SpecterHunt, error) {
	var hunts []*pb.SpecterHunt
	err := db.ForEach(BucketHunts, func(_, value []byte) error {
		hunt := &pb.SpecterHunt{}
		if err := proto.Unmarshal(value, hunt); err != nil {
			return fmt.Errorf("unmarshaling specter hunt: %w", err)
		}
		hunts = append(hunts, hunt)
		return nil
	})
	return hunts, err
}

// ListActiveSpecterHunts returns hunts in ACTIVE state.
func (db *DB) ListActiveSpecterHunts() ([]*pb.SpecterHunt, error) {
	var hunts []*pb.SpecterHunt
	err := db.ForEach(BucketHunts, func(_, value []byte) error {
		hunt := &pb.SpecterHunt{}
		if err := proto.Unmarshal(value, hunt); err != nil {
			return fmt.Errorf("unmarshaling specter hunt: %w", err)
		}
		if hunt.State == pb.HuntState_HUNT_STATE_ACTIVE {
			hunts = append(hunts, hunt)
		}
		return nil
	})
	return hunts, err
}

// Territory accessors

// GetTerritory retrieves a Territory by its ID.
func (db *DB) GetTerritory(territoryID []byte) (*pb.Territory, error) {
	data, err := db.Get(BucketTerritories, territoryID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	territory := &pb.Territory{}
	if err := proto.Unmarshal(data, territory); err != nil {
		return nil, fmt.Errorf("unmarshaling territory: %w", err)
	}
	return territory, nil
}

// PutTerritory stores a Territory by its ID.
func (db *DB) PutTerritory(territory *pb.Territory) error {
	if territory == nil {
		return fmt.Errorf("nil territory")
	}
	if len(territory.Id) == 0 {
		return fmt.Errorf("territory has no ID")
	}

	data, err := proto.Marshal(territory)
	if err != nil {
		return fmt.Errorf("marshaling territory: %w", err)
	}
	return db.Put(BucketTerritories, territory.Id, data)
}

// DeleteTerritory removes a Territory by its ID.
func (db *DB) DeleteTerritory(territoryID []byte) error {
	return db.Delete(BucketTerritories, territoryID)
}

// ListTerritories returns all stored Territories.
func (db *DB) ListTerritories() ([]*pb.Territory, error) {
	var territories []*pb.Territory
	err := db.ForEach(BucketTerritories, func(_, value []byte) error {
		territory := &pb.Territory{}
		if err := proto.Unmarshal(value, territory); err != nil {
			return fmt.Errorf("unmarshaling territory: %w", err)
		}
		territories = append(territories, territory)
		return nil
	})
	return territories, err
}

// OraclePool accessors

// GetOraclePool retrieves an OraclePool by its ID.
func (db *DB) GetOraclePool(poolID []byte) (*pb.OraclePool, error) {
	data, err := db.Get(BucketOracles, poolID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	pool := &pb.OraclePool{}
	if err := proto.Unmarshal(data, pool); err != nil {
		return nil, fmt.Errorf("unmarshaling oracle pool: %w", err)
	}
	return pool, nil
}

// PutOraclePool stores an OraclePool by its ID.
func (db *DB) PutOraclePool(pool *pb.OraclePool) error {
	if pool == nil {
		return fmt.Errorf("nil oracle pool")
	}
	if len(pool.Id) == 0 {
		return fmt.Errorf("oracle pool has no ID")
	}

	data, err := proto.Marshal(pool)
	if err != nil {
		return fmt.Errorf("marshaling oracle pool: %w", err)
	}
	return db.Put(BucketOracles, pool.Id, data)
}

// DeleteOraclePool removes an OraclePool by its ID.
func (db *DB) DeleteOraclePool(poolID []byte) error {
	return db.Delete(BucketOracles, poolID)
}

// ListOraclePools returns all stored OraclePools.
func (db *DB) ListOraclePools() ([]*pb.OraclePool, error) {
	var pools []*pb.OraclePool
	err := db.ForEach(BucketOracles, func(_, value []byte) error {
		pool := &pb.OraclePool{}
		if err := proto.Unmarshal(value, pool); err != nil {
			return fmt.Errorf("unmarshaling oracle pool: %w", err)
		}
		pools = append(pools, pool)
		return nil
	})
	return pools, err
}

// ListOpenOraclePools returns pools in OPEN state.
func (db *DB) ListOpenOraclePools() ([]*pb.OraclePool, error) {
	var pools []*pb.OraclePool
	err := db.ForEach(BucketOracles, func(_, value []byte) error {
		pool := &pb.OraclePool{}
		if err := proto.Unmarshal(value, pool); err != nil {
			return fmt.Errorf("unmarshaling oracle pool: %w", err)
		}
		if pool.State == pb.OracleState_ORACLE_STATE_OPEN {
			pools = append(pools, pool)
		}
		return nil
	})
	return pools, err
}

// ForgeProject accessors

// GetForgeProject retrieves a ForgeProject by its ID.
func (db *DB) GetForgeProject(projectID []byte) (*pb.ForgeProject, error) {
	data, err := db.Get(BucketForge, projectID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	project := &pb.ForgeProject{}
	if err := proto.Unmarshal(data, project); err != nil {
		return nil, fmt.Errorf("unmarshaling forge project: %w", err)
	}
	return project, nil
}

// PutForgeProject stores a ForgeProject by its ID.
func (db *DB) PutForgeProject(project *pb.ForgeProject) error {
	if project == nil {
		return fmt.Errorf("nil forge project")
	}
	if len(project.Id) == 0 {
		return fmt.Errorf("forge project has no ID")
	}

	data, err := proto.Marshal(project)
	if err != nil {
		return fmt.Errorf("marshaling forge project: %w", err)
	}
	return db.Put(BucketForge, project.Id, data)
}

// DeleteForgeProject removes a ForgeProject by its ID.
func (db *DB) DeleteForgeProject(projectID []byte) error {
	return db.Delete(BucketForge, projectID)
}

// ListForgeProjects returns all stored ForgeProjects.
func (db *DB) ListForgeProjects() ([]*pb.ForgeProject, error) {
	var projects []*pb.ForgeProject
	err := db.ForEach(BucketForge, func(_, value []byte) error {
		project := &pb.ForgeProject{}
		if err := proto.Unmarshal(value, project); err != nil {
			return fmt.Errorf("unmarshaling forge project: %w", err)
		}
		projects = append(projects, project)
		return nil
	})
	return projects, err
}

// ShadowPlay accessors

// GetShadowPlay retrieves a ShadowPlay by its ID.
func (db *DB) GetShadowPlay(playID []byte) (*pb.ShadowPlay, error) {
	data, err := db.Get(BucketShadowPlay, playID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	play := &pb.ShadowPlay{}
	if err := proto.Unmarshal(data, play); err != nil {
		return nil, fmt.Errorf("unmarshaling shadow play: %w", err)
	}
	return play, nil
}

// PutShadowPlay stores a ShadowPlay by its ID.
func (db *DB) PutShadowPlay(play *pb.ShadowPlay) error {
	if play == nil {
		return fmt.Errorf("nil shadow play")
	}
	if len(play.Id) == 0 {
		return fmt.Errorf("shadow play has no ID")
	}

	data, err := proto.Marshal(play)
	if err != nil {
		return fmt.Errorf("marshaling shadow play: %w", err)
	}
	return db.Put(BucketShadowPlay, play.Id, data)
}

// DeleteShadowPlay removes a ShadowPlay by its ID.
func (db *DB) DeleteShadowPlay(playID []byte) error {
	return db.Delete(BucketShadowPlay, playID)
}

// ListShadowPlays returns all stored ShadowPlays.
func (db *DB) ListShadowPlays() ([]*pb.ShadowPlay, error) {
	var plays []*pb.ShadowPlay
	err := db.ForEach(BucketShadowPlay, func(_, value []byte) error {
		play := &pb.ShadowPlay{}
		if err := proto.Unmarshal(value, play); err != nil {
			return fmt.Errorf("unmarshaling shadow play: %w", err)
		}
		plays = append(plays, play)
		return nil
	})
	return plays, err
}

// PhantomCouncil accessors

// GetPhantomCouncil retrieves a PhantomCouncil by its ID.
func (db *DB) GetPhantomCouncil(councilID []byte) (*pb.PhantomCouncil, error) {
	data, err := db.Get(BucketCouncils, councilID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	council := &pb.PhantomCouncil{}
	if err := proto.Unmarshal(data, council); err != nil {
		return nil, fmt.Errorf("unmarshaling phantom council: %w", err)
	}
	return council, nil
}

// PutPhantomCouncil stores a PhantomCouncil by its ID.
func (db *DB) PutPhantomCouncil(council *pb.PhantomCouncil) error {
	if council == nil {
		return fmt.Errorf("nil phantom council")
	}
	if len(council.Id) == 0 {
		return fmt.Errorf("phantom council has no ID")
	}

	data, err := proto.Marshal(council)
	if err != nil {
		return fmt.Errorf("marshaling phantom council: %w", err)
	}
	return db.Put(BucketCouncils, council.Id, data)
}

// DeletePhantomCouncil removes a PhantomCouncil by its ID.
func (db *DB) DeletePhantomCouncil(councilID []byte) error {
	return db.Delete(BucketCouncils, councilID)
}

// ListPhantomCouncils returns all stored PhantomCouncils.
func (db *DB) ListPhantomCouncils() ([]*pb.PhantomCouncil, error) {
	var councils []*pb.PhantomCouncil
	err := db.ForEach(BucketCouncils, func(_, value []byte) error {
		council := &pb.PhantomCouncil{}
		if err := proto.Unmarshal(value, council); err != nil {
			return fmt.Errorf("unmarshaling phantom council: %w", err)
		}
		councils = append(councils, council)
		return nil
	})
	return councils, err
}

// ListActivePhantomCouncils returns councils in ACTIVE state.
func (db *DB) ListActivePhantomCouncils() ([]*pb.PhantomCouncil, error) {
	var councils []*pb.PhantomCouncil
	err := db.ForEach(BucketCouncils, func(_, value []byte) error {
		council := &pb.PhantomCouncil{}
		if err := proto.Unmarshal(value, council); err != nil {
			return fmt.Errorf("unmarshaling phantom council: %w", err)
		}
		if council.State == pb.CouncilState_COUNCIL_STATE_ACTIVE {
			councils = append(councils, council)
		}
		return nil
	})
	return councils, err
}

// PhantomGift accessors

// GetPhantomGift retrieves a PhantomGift by its ID.
func (db *DB) GetPhantomGift(giftID []byte) (*pb.PhantomGift, error) {
	data, err := db.Get(BucketGifts, giftID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	gift := &pb.PhantomGift{}
	if err := proto.Unmarshal(data, gift); err != nil {
		return nil, fmt.Errorf("unmarshaling phantom gift: %w", err)
	}
	return gift, nil
}

// PutPhantomGift stores a PhantomGift by its ID.
func (db *DB) PutPhantomGift(gift *pb.PhantomGift) error {
	if gift == nil {
		return fmt.Errorf("nil phantom gift")
	}
	if len(gift.Id) == 0 {
		return fmt.Errorf("phantom gift has no ID")
	}

	data, err := proto.Marshal(gift)
	if err != nil {
		return fmt.Errorf("marshaling phantom gift: %w", err)
	}
	return db.Put(BucketGifts, gift.Id, data)
}

// DeletePhantomGift removes a PhantomGift by its ID.
func (db *DB) DeletePhantomGift(giftID []byte) error {
	return db.Delete(BucketGifts, giftID)
}

// ListPhantomGifts returns all stored PhantomGifts.
func (db *DB) ListPhantomGifts() ([]*pb.PhantomGift, error) {
	var gifts []*pb.PhantomGift
	err := db.ForEach(BucketGifts, func(_, value []byte) error {
		gift := &pb.PhantomGift{}
		if err := proto.Unmarshal(value, gift); err != nil {
			return fmt.Errorf("unmarshaling phantom gift: %w", err)
		}
		gifts = append(gifts, gift)
		return nil
	})
	return gifts, err
}

// ListGiftsForRecipient returns gifts for a specific recipient.
func (db *DB) ListGiftsForRecipient(recipientKey []byte) ([]*pb.PhantomGift, error) {
	var gifts []*pb.PhantomGift
	keyStr := string(recipientKey)
	err := db.ForEach(BucketGifts, func(_, value []byte) error {
		gift := &pb.PhantomGift{}
		if err := proto.Unmarshal(value, gift); err != nil {
			return fmt.Errorf("unmarshaling phantom gift: %w", err)
		}
		if string(gift.RecipientPubkey) == keyStr {
			gifts = append(gifts, gift)
		}
		return nil
	})
	return gifts, err
}

// SpecterMark accessors

// GetSpecterMark retrieves a SpecterMark by its ID.
func (db *DB) GetSpecterMark(markID []byte) (*pb.SpecterMark, error) {
	data, err := db.Get(BucketMarks, markID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	mark := &pb.SpecterMark{}
	if err := proto.Unmarshal(data, mark); err != nil {
		return nil, fmt.Errorf("unmarshaling specter mark: %w", err)
	}
	return mark, nil
}

// PutSpecterMark stores a SpecterMark by its ID.
func (db *DB) PutSpecterMark(mark *pb.SpecterMark) error {
	if mark == nil {
		return fmt.Errorf("nil specter mark")
	}
	if len(mark.Id) == 0 {
		return fmt.Errorf("specter mark has no ID")
	}

	data, err := proto.Marshal(mark)
	if err != nil {
		return fmt.Errorf("marshaling specter mark: %w", err)
	}
	return db.Put(BucketMarks, mark.Id, data)
}

// DeleteSpecterMark removes a SpecterMark by its ID.
func (db *DB) DeleteSpecterMark(markID []byte) error {
	return db.Delete(BucketMarks, markID)
}

// ListSpecterMarks returns all stored SpecterMarks.
func (db *DB) ListSpecterMarks() ([]*pb.SpecterMark, error) {
	var marks []*pb.SpecterMark
	err := db.ForEach(BucketMarks, func(_, value []byte) error {
		mark := &pb.SpecterMark{}
		if err := proto.Unmarshal(value, mark); err != nil {
			return fmt.Errorf("unmarshaling specter mark: %w", err)
		}
		marks = append(marks, mark)
		return nil
	})
	return marks, err
}

// ListMarksForTarget returns marks on a specific target identity.
func (db *DB) ListMarksForTarget(targetKey []byte) ([]*pb.SpecterMark, error) {
	var marks []*pb.SpecterMark
	keyStr := string(targetKey)
	err := db.ForEach(BucketMarks, func(_, value []byte) error {
		mark := &pb.SpecterMark{}
		if err := proto.Unmarshal(value, mark); err != nil {
			return fmt.Errorf("unmarshaling specter mark: %w", err)
		}
		if string(mark.TargetPubkey) == keyStr {
			marks = append(marks, mark)
		}
		return nil
	})
	return marks, err
}

// Cross-Layer Artifact Query Methods
// Per PLAN.md Step 2: methods for querying anonymous artifacts visible on Surface Layer.

// GetActiveGiftsForRecipient returns non-expired gifts for a specific recipient.
func (db *DB) GetActiveGiftsForRecipient(recipientKey []byte, nowUnix int64) ([]*pb.PhantomGift, error) {
	var gifts []*pb.PhantomGift
	keyStr := string(recipientKey)
	err := db.ForEach(BucketGifts, func(_, value []byte) error {
		gift := &pb.PhantomGift{}
		if err := proto.Unmarshal(value, gift); err != nil {
			return fmt.Errorf("unmarshaling phantom gift: %w", err)
		}
		if string(gift.RecipientPubkey) == keyStr && gift.ExpiresAt > nowUnix {
			gifts = append(gifts, gift)
		}
		return nil
	})
	return gifts, err
}

// GetActivePuzzlesNearNode returns active puzzles (spatial query placeholder).
// Currently returns all active puzzles; spatial filtering deferred to future work.
func (db *DB) GetActivePuzzlesNearNode(_ []byte, _ float64) ([]*pb.CipherPuzzle, error) {
	return db.ListActiveCipherPuzzles()
}

// GetActiveHuntsWithFragmentsNear returns active hunts (spatial query placeholder).
// Currently returns all active hunts; spatial filtering deferred to future work.
func (db *DB) GetActiveHuntsWithFragmentsNear(_ []byte, _ float64) ([]*pb.SpecterHunt, error) {
	var hunts []*pb.SpecterHunt
	err := db.ForEach(BucketHunts, func(_, value []byte) error {
		hunt := &pb.SpecterHunt{}
		if err := proto.Unmarshal(value, hunt); err != nil {
			return fmt.Errorf("unmarshaling specter hunt: %w", err)
		}
		if hunt.State == pb.HuntState_HUNT_STATE_ACTIVE {
			hunts = append(hunts, hunt)
		}
		return nil
	})
	return hunts, err
}

// GetTerritoryInfluenceAt returns territory state at a node (spatial query placeholder).
// Currently returns the first territory; proper spatial lookup deferred to future work.
func (db *DB) GetTerritoryInfluenceAt(_ []byte) (*pb.Territory, error) {
	territories, err := db.ListTerritories()
	if err != nil || len(territories) == 0 {
		return nil, err
	}
	return territories[0], nil
}

// GetActiveOraclePoolsNearNode returns open oracle pools (spatial query placeholder).
// Currently returns all open pools; spatial filtering deferred to future work.
func (db *DB) GetActiveOraclePoolsNearNode(_ []byte, _ float64) ([]*pb.OraclePool, error) {
	return db.ListOpenOraclePools()
}

// GetActiveForgeEventsNearNode returns active forge projects (spatial query placeholder).
// Currently returns all collecting projects; spatial filtering deferred to future work.
func (db *DB) GetActiveForgeEventsNearNode(_ []byte, _ float64) ([]*pb.ForgeProject, error) {
	var projects []*pb.ForgeProject
	err := db.ForEach(BucketForge, func(_, value []byte) error {
		project := &pb.ForgeProject{}
		if err := proto.Unmarshal(value, project); err != nil {
			return fmt.Errorf("unmarshaling forge project: %w", err)
		}
		if project.State == pb.ForgeState_FORGE_STATE_COLLECTING {
			projects = append(projects, project)
		}
		return nil
	})
	return projects, err
}

// GetActiveShadowPlayNearNode returns active shadow plays (spatial query placeholder).
// Currently returns all performing plays; spatial filtering deferred to future work.
func (db *DB) GetActiveShadowPlayNearNode(_ []byte, _ float64) ([]*pb.ShadowPlay, error) {
	var plays []*pb.ShadowPlay
	err := db.ForEach(BucketShadowPlay, func(_, value []byte) error {
		play := &pb.ShadowPlay{}
		if err := proto.Unmarshal(value, play); err != nil {
			return fmt.Errorf("unmarshaling shadow play: %w", err)
		}
		if play.State == pb.ShadowPlayState_SHADOW_PLAY_STATE_PERFORMING {
			plays = append(plays, play)
		}
		return nil
	})
	return plays, err
}

// GetMaskedEventsNearNode returns masked events (spatial query placeholder).
// Masked events use custom StoredMaskedEvent type; use MaskedEventStore for queries.
// This method is a placeholder for cross-layer rendering integration.
func (db *DB) GetMaskedEventsNearNode(_ []byte, _ float64) ([]StoredMaskedEvent, error) {
	// Placeholder: return empty slice. Masked events require MaskedEventStore for proper access.
	return []StoredMaskedEvent{}, nil
}

// GetCouncilsWithMember returns councils containing a specific member.
func (db *DB) GetCouncilsWithMember(memberKey []byte) ([]*pb.PhantomCouncil, error) {
	var councils []*pb.PhantomCouncil
	keyStr := string(memberKey)
	err := db.ForEach(BucketCouncils, func(_, value []byte) error {
		council := &pb.PhantomCouncil{}
		if err := proto.Unmarshal(value, council); err != nil {
			return fmt.Errorf("unmarshaling phantom council: %w", err)
		}
		for _, member := range council.Members {
			if string(member.SpecterPubkey) == keyStr {
				councils = append(councils, council)
				break
			}
		}
		return nil
	})
	return councils, err
}

// Device accessors for multi-device identity

// GetDeviceList retrieves the authorized device list for a master identity.
func (db *DB) GetDeviceList(masterPubkey []byte) (*pb.DeviceList, error) {
	data, err := db.Get(BucketDevices, masterPubkey)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return &pb.DeviceList{Devices: []*pb.AuthorizedDevice{}}, nil
	}

	list := &pb.DeviceList{}
	if err := proto.Unmarshal(data, list); err != nil {
		return nil, fmt.Errorf("unmarshaling device list: %w", err)
	}
	return list, nil
}

// PutDeviceList stores the authorized device list for a master identity.
func (db *DB) PutDeviceList(masterPubkey []byte, list *pb.DeviceList) error {
	if list == nil {
		return fmt.Errorf("nil device list")
	}
	if len(masterPubkey) == 0 {
		return fmt.Errorf("empty master pubkey")
	}

	data, err := proto.Marshal(list)
	if err != nil {
		return fmt.Errorf("marshaling device list: %w", err)
	}
	return db.Put(BucketDevices, masterPubkey, data)
}

// DeleteDeviceList removes the device list for a master identity.
func (db *DB) DeleteDeviceList(masterPubkey []byte) error {
	return db.Delete(BucketDevices, masterPubkey)
}
