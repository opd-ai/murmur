package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

func TestWaveAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	wave := &pb.Wave{
		WaveId:     []byte("test-wave-id-12345678901234567890"),
		WaveType:   pb.WaveType_WAVE_TYPE_SURFACE,
		Content:    []byte("hello world"),
		CreatedAt:  time.Now().Unix(),
		TtlSeconds: 7 * 24 * 60 * 60,
	}

	t.Run("put and get wave", func(t *testing.T) {
		if err := db.PutWave(wave); err != nil {
			t.Fatalf("PutWave() error: %v", err)
		}

		got, err := db.GetWave(wave.WaveId)
		if err != nil {
			t.Fatalf("GetWave() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetWave() returned nil")
		}
		if string(got.Content) != "hello world" {
			t.Errorf("Content = %q, want %q", string(got.Content), "hello world")
		}
	})

	t.Run("get non-existent wave", func(t *testing.T) {
		got, err := db.GetWave([]byte("non-existent"))
		if err != nil {
			t.Fatalf("GetWave() error: %v", err)
		}
		if got != nil {
			t.Error("GetWave() expected nil for non-existent wave")
		}
	})

	t.Run("list waves", func(t *testing.T) {
		waves, err := db.ListWaves()
		if err != nil {
			t.Fatalf("ListWaves() error: %v", err)
		}
		if len(waves) != 1 {
			t.Errorf("ListWaves() count = %d, want 1", len(waves))
		}
	})

	t.Run("delete wave", func(t *testing.T) {
		if err := db.DeleteWave(wave.WaveId); err != nil {
			t.Fatalf("DeleteWave() error: %v", err)
		}
		got, _ := db.GetWave(wave.WaveId)
		if got != nil {
			t.Error("Wave should be deleted")
		}
	})

	t.Run("put nil wave", func(t *testing.T) {
		if err := db.PutWave(nil); err == nil {
			t.Error("PutWave(nil) expected error")
		}
	})

	t.Run("put wave without ID", func(t *testing.T) {
		badWave := &pb.Wave{Content: []byte("test")}
		if err := db.PutWave(badWave); err == nil {
			t.Error("PutWave() without ID expected error")
		}
	})
}

func TestIdentityAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	pubKey := make([]byte, 32)
	for i := range pubKey {
		pubKey[i] = byte(i)
	}

	decl := &pb.IdentityDeclaration{
		PublicKey:   pubKey,
		DisplayName: "Test User",
		Bio:         "Hello, I am a test user",
	}

	t.Run("put and get identity", func(t *testing.T) {
		if err := db.PutIdentityDeclaration(decl); err != nil {
			t.Fatalf("PutIdentityDeclaration() error: %v", err)
		}

		got, err := db.GetIdentityDeclaration(pubKey)
		if err != nil {
			t.Fatalf("GetIdentityDeclaration() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetIdentityDeclaration() returned nil")
		}
		if got.DisplayName != "Test User" {
			t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Test User")
		}
	})

	t.Run("delete identity", func(t *testing.T) {
		if err := db.DeleteIdentityDeclaration(pubKey); err != nil {
			t.Fatalf("DeleteIdentityDeclaration() error: %v", err)
		}
		got, _ := db.GetIdentityDeclaration(pubKey)
		if got != nil {
			t.Error("Identity should be deleted")
		}
	})

	t.Run("put nil identity", func(t *testing.T) {
		if err := db.PutIdentityDeclaration(nil); err == nil {
			t.Error("PutIdentityDeclaration(nil) expected error")
		}
	})
}

func TestPeerAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	peer := &pb.PeerRecord{
		PeerId:      "12D3KooWTestPeerID",
		DisplayName: "Test Peer",
		TrustScore:  0.8,
	}

	t.Run("put and get peer", func(t *testing.T) {
		if err := db.PutPeerRecord(peer); err != nil {
			t.Fatalf("PutPeerRecord() error: %v", err)
		}

		got, err := db.GetPeerRecord("12D3KooWTestPeerID")
		if err != nil {
			t.Fatalf("GetPeerRecord() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetPeerRecord() returned nil")
		}
		if got.DisplayName != "Test Peer" {
			t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Test Peer")
		}
	})

	t.Run("list peers", func(t *testing.T) {
		peers, err := db.ListPeerRecords()
		if err != nil {
			t.Fatalf("ListPeerRecords() error: %v", err)
		}
		if len(peers) != 1 {
			t.Errorf("ListPeerRecords() count = %d, want 1", len(peers))
		}
	})

	t.Run("delete peer", func(t *testing.T) {
		if err := db.DeletePeerRecord("12D3KooWTestPeerID"); err != nil {
			t.Fatalf("DeletePeerRecord() error: %v", err)
		}
		got, _ := db.GetPeerRecord("12D3KooWTestPeerID")
		if got != nil {
			t.Error("Peer should be deleted")
		}
	})

	t.Run("put nil peer", func(t *testing.T) {
		if err := db.PutPeerRecord(nil); err == nil {
			t.Error("PutPeerRecord(nil) expected error")
		}
	})
}

func TestResonanceAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	identityKey := make([]byte, 32)
	score := &pb.ResonanceScore{
		Score:     100.5,
		Milestone: pb.ResonanceMilestone_RESONANCE_MILESTONE_PHANTOM,
	}

	t.Run("put and get resonance", func(t *testing.T) {
		if err := db.PutResonanceScore(identityKey, score); err != nil {
			t.Fatalf("PutResonanceScore() error: %v", err)
		}

		got, err := db.GetResonanceScore(identityKey)
		if err != nil {
			t.Fatalf("GetResonanceScore() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetResonanceScore() returned nil")
		}
		if got.Score != 100.5 {
			t.Errorf("Score = %f, want 100.5", got.Score)
		}
	})

	t.Run("put nil resonance", func(t *testing.T) {
		if err := db.PutResonanceScore(identityKey, nil); err == nil {
			t.Error("PutResonanceScore(nil) expected error")
		}
	})
}

func TestRelayAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	pubKey := make([]byte, 32)
	for i := range pubKey {
		pubKey[i] = byte(i)
	}

	relay := &pb.RelayAdvertisement{
		Ed25519Pubkey:    pubKey,
		Curve25519Pubkey: make([]byte, 32),
		Bandwidth:        1000,
	}

	t.Run("put and get relay", func(t *testing.T) {
		if err := db.PutRelayAdvertisement(relay); err != nil {
			t.Fatalf("PutRelayAdvertisement() error: %v", err)
		}

		got, err := db.GetRelayAdvertisement(pubKey)
		if err != nil {
			t.Fatalf("GetRelayAdvertisement() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetRelayAdvertisement() returned nil")
		}
		if got.Bandwidth != 1000 {
			t.Errorf("Bandwidth = %d, want 1000", got.Bandwidth)
		}
	})

	t.Run("list relays", func(t *testing.T) {
		relays, err := db.ListRelayAdvertisements()
		if err != nil {
			t.Fatalf("ListRelayAdvertisements() error: %v", err)
		}
		if len(relays) != 1 {
			t.Errorf("ListRelayAdvertisements() count = %d, want 1", len(relays))
		}
	})

	t.Run("put nil relay", func(t *testing.T) {
		if err := db.PutRelayAdvertisement(nil); err == nil {
			t.Error("PutRelayAdvertisement(nil) expected error")
		}
	})
}

func TestConfigAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	t.Run("put and get config", func(t *testing.T) {
		if err := db.PutConfigValue("test-key", []byte("test-value")); err != nil {
			t.Fatalf("PutConfigValue() error: %v", err)
		}

		got, err := db.GetConfigValue("test-key")
		if err != nil {
			t.Fatalf("GetConfigValue() error: %v", err)
		}
		if string(got) != "test-value" {
			t.Errorf("value = %q, want %q", string(got), "test-value")
		}
	})

	t.Run("delete config", func(t *testing.T) {
		if err := db.DeleteConfigValue("test-key"); err != nil {
			t.Fatalf("DeleteConfigValue() error: %v", err)
		}
		got, _ := db.GetConfigValue("test-key")
		if got != nil {
			t.Error("Config value should be deleted")
		}
	})
}

func TestThreadAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	waveID := []byte("wave-id-123")
	rootID := []byte("root-id-456")

	t.Run("put and get thread root", func(t *testing.T) {
		if err := db.PutThreadRoot(waveID, rootID); err != nil {
			t.Fatalf("PutThreadRoot() error: %v", err)
		}

		got, err := db.GetThreadRoot(waveID)
		if err != nil {
			t.Fatalf("GetThreadRoot() error: %v", err)
		}
		if string(got) != string(rootID) {
			t.Errorf("root = %q, want %q", string(got), string(rootID))
		}
	})

	t.Run("delete thread root", func(t *testing.T) {
		if err := db.DeleteThreadRoot(waveID); err != nil {
			t.Fatalf("DeleteThreadRoot() error: %v", err)
		}
		got, _ := db.GetThreadRoot(waveID)
		if got != nil {
			t.Error("Thread root should be deleted")
		}
	})
}

func TestCipherPuzzleAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	puzzle := &pb.CipherPuzzle{
		Id:               []byte("puzzle-id-12345678901234567890123456"),
		CreatorPubkey:    []byte("creator-pubkey-12345678901234567890"),
		EncryptedContent: []byte("encrypted content here"),
		Hint:             []byte("think carefully"),
		Difficulty:       5,
		CreatedAt:        now,
		ExpiresAt:        now + 3600,
		State:            pb.PuzzleState_PUZZLE_STATE_ACTIVE,
		SolutionHash:     []byte("solution-hash-123456789012345678901"),
	}

	t.Run("put and get puzzle", func(t *testing.T) {
		if err := db.PutCipherPuzzle(puzzle); err != nil {
			t.Fatalf("PutCipherPuzzle() error: %v", err)
		}

		got, err := db.GetCipherPuzzle(puzzle.Id)
		if err != nil {
			t.Fatalf("GetCipherPuzzle() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetCipherPuzzle() returned nil")
		}
		if string(got.Hint) != "think carefully" {
			t.Errorf("Hint = %q, want %q", string(got.Hint), "think carefully")
		}
		if got.Difficulty != 5 {
			t.Errorf("Difficulty = %d, want 5", got.Difficulty)
		}
	})

	t.Run("get non-existent puzzle", func(t *testing.T) {
		got, err := db.GetCipherPuzzle([]byte("non-existent"))
		if err != nil {
			t.Fatalf("GetCipherPuzzle() error: %v", err)
		}
		if got != nil {
			t.Error("GetCipherPuzzle() expected nil for non-existent puzzle")
		}
	})

	t.Run("list puzzles", func(t *testing.T) {
		puzzles, err := db.ListCipherPuzzles()
		if err != nil {
			t.Fatalf("ListCipherPuzzles() error: %v", err)
		}
		if len(puzzles) != 1 {
			t.Errorf("ListCipherPuzzles() count = %d, want 1", len(puzzles))
		}
	})

	t.Run("list active puzzles", func(t *testing.T) {
		active, err := db.ListActiveCipherPuzzles()
		if err != nil {
			t.Fatalf("ListActiveCipherPuzzles() error: %v", err)
		}
		if len(active) != 1 {
			t.Errorf("ListActiveCipherPuzzles() count = %d, want 1", len(active))
		}

		// Add a solved puzzle.
		solvedPuzzle := &pb.CipherPuzzle{
			Id:           []byte("solved-puzzle-id-1234567890123456789"),
			Difficulty:   3,
			State:        pb.PuzzleState_PUZZLE_STATE_SOLVED,
			CreatedAt:    now,
			WinnerPubkey: []byte("winner-key"),
		}
		if err := db.PutCipherPuzzle(solvedPuzzle); err != nil {
			t.Fatalf("PutCipherPuzzle() error: %v", err)
		}

		// Should still only return the active one.
		active, err = db.ListActiveCipherPuzzles()
		if err != nil {
			t.Fatalf("ListActiveCipherPuzzles() error: %v", err)
		}
		if len(active) != 1 {
			t.Errorf("ListActiveCipherPuzzles() count = %d, want 1", len(active))
		}

		// Total should be 2.
		all, _ := db.ListCipherPuzzles()
		if len(all) != 2 {
			t.Errorf("ListCipherPuzzles() count = %d, want 2", len(all))
		}
	})

	t.Run("delete puzzle", func(t *testing.T) {
		if err := db.DeleteCipherPuzzle(puzzle.Id); err != nil {
			t.Fatalf("DeleteCipherPuzzle() error: %v", err)
		}
		got, _ := db.GetCipherPuzzle(puzzle.Id)
		if got != nil {
			t.Error("Puzzle should be deleted")
		}
	})

	t.Run("put nil puzzle", func(t *testing.T) {
		if err := db.PutCipherPuzzle(nil); err == nil {
			t.Error("PutCipherPuzzle(nil) expected error")
		}
	})

	t.Run("put puzzle without ID", func(t *testing.T) {
		badPuzzle := &pb.CipherPuzzle{Difficulty: 1}
		if err := db.PutCipherPuzzle(badPuzzle); err == nil {
			t.Error("PutCipherPuzzle() without ID expected error")
		}
	})
}

func TestMain(m *testing.M) {
	// Clean test environment
	os.Exit(m.Run())
}

func TestSpecterHuntAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	hunt := &pb.SpecterHunt{
		Id:              []byte("hunt-id-123456789012345678901234567"),
		OrganizerPubkey: []byte("organizer-key-1234567890123456789"),
		Title:           "The Great Specter Hunt",
		Description:     "Find the hidden Specters",
		StartTime:       now,
		EndTime:         now + 3600,
		State:           pb.HuntState_HUNT_STATE_ACTIVE,
		MaxParticipants: 20,
	}

	t.Run("put and get hunt", func(t *testing.T) {
		if err := db.PutSpecterHunt(hunt); err != nil {
			t.Fatalf("PutSpecterHunt() error: %v", err)
		}

		got, err := db.GetSpecterHunt(hunt.Id)
		if err != nil {
			t.Fatalf("GetSpecterHunt() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetSpecterHunt() returned nil")
		}
		if got.Title != "The Great Specter Hunt" {
			t.Errorf("Title = %q, want %q", got.Title, "The Great Specter Hunt")
		}
	})

	t.Run("list active hunts", func(t *testing.T) {
		active, err := db.ListActiveSpecterHunts()
		if err != nil {
			t.Fatalf("ListActiveSpecterHunts() error: %v", err)
		}
		if len(active) != 1 {
			t.Errorf("ListActiveSpecterHunts() count = %d, want 1", len(active))
		}
	})

	t.Run("delete hunt", func(t *testing.T) {
		if err := db.DeleteSpecterHunt(hunt.Id); err != nil {
			t.Fatalf("DeleteSpecterHunt() error: %v", err)
		}
		got, _ := db.GetSpecterHunt(hunt.Id)
		if got != nil {
			t.Error("Hunt should be deleted")
		}
	})

	t.Run("put nil hunt", func(t *testing.T) {
		if err := db.PutSpecterHunt(nil); err == nil {
			t.Error("PutSpecterHunt(nil) expected error")
		}
	})
}

func TestTerritoryAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	territory := &pb.Territory{
		Id:               []byte("territory-id-12345678901234567890"),
		Name:             "Shadow Realm",
		ControllerPubkey: []byte("controller-key-12345678901234567"),
		Influence:        75,
		LastUpdated:      time.Now().Unix(),
	}

	t.Run("put and get territory", func(t *testing.T) {
		if err := db.PutTerritory(territory); err != nil {
			t.Fatalf("PutTerritory() error: %v", err)
		}

		got, err := db.GetTerritory(territory.Id)
		if err != nil {
			t.Fatalf("GetTerritory() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetTerritory() returned nil")
		}
		if got.Name != "Shadow Realm" {
			t.Errorf("Name = %q, want %q", got.Name, "Shadow Realm")
		}
		if got.Influence != 75 {
			t.Errorf("Influence = %d, want 75", got.Influence)
		}
	})

	t.Run("list territories", func(t *testing.T) {
		territories, err := db.ListTerritories()
		if err != nil {
			t.Fatalf("ListTerritories() error: %v", err)
		}
		if len(territories) != 1 {
			t.Errorf("ListTerritories() count = %d, want 1", len(territories))
		}
	})

	t.Run("delete territory", func(t *testing.T) {
		if err := db.DeleteTerritory(territory.Id); err != nil {
			t.Fatalf("DeleteTerritory() error: %v", err)
		}
		got, _ := db.GetTerritory(territory.Id)
		if got != nil {
			t.Error("Territory should be deleted")
		}
	})
}

func TestOraclePoolAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	pool := &pb.OraclePool{
		Id:            []byte("pool-id-1234567890123456789012345678"),
		CreatorPubkey: []byte("creator-key-12345678901234567890"),
		Question:      "Will MURMUR reach 1000 nodes?",
		CreatedAt:     now,
		ClosesAt:      now + 86400,
		ResolvesAt:    now + 172800,
		State:         pb.OracleState_ORACLE_STATE_OPEN,
	}

	t.Run("put and get pool", func(t *testing.T) {
		if err := db.PutOraclePool(pool); err != nil {
			t.Fatalf("PutOraclePool() error: %v", err)
		}

		got, err := db.GetOraclePool(pool.Id)
		if err != nil {
			t.Fatalf("GetOraclePool() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetOraclePool() returned nil")
		}
		if got.Question != "Will MURMUR reach 1000 nodes?" {
			t.Errorf("Question = %q, want %q", got.Question, "Will MURMUR reach 1000 nodes?")
		}
	})

	t.Run("list open pools", func(t *testing.T) {
		pools, err := db.ListOpenOraclePools()
		if err != nil {
			t.Fatalf("ListOpenOraclePools() error: %v", err)
		}
		if len(pools) != 1 {
			t.Errorf("ListOpenOraclePools() count = %d, want 1", len(pools))
		}
	})

	t.Run("delete pool", func(t *testing.T) {
		if err := db.DeleteOraclePool(pool.Id); err != nil {
			t.Fatalf("DeleteOraclePool() error: %v", err)
		}
		got, _ := db.GetOraclePool(pool.Id)
		if got != nil {
			t.Error("Pool should be deleted")
		}
	})
}

func TestForgeProjectAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	project := &pb.ForgeProject{
		Id:            []byte("forge-id-123456789012345678901234567"),
		CreatorPubkey: []byte("creator-key-12345678901234567890"),
		Name:          "Phoenix Sigil",
		Seed:          []byte("random-seed-data"),
		CreatedAt:     now,
		Deadline:      now + 86400,
		State:         pb.ForgeState_FORGE_STATE_COLLECTING,
	}

	t.Run("put and get project", func(t *testing.T) {
		if err := db.PutForgeProject(project); err != nil {
			t.Fatalf("PutForgeProject() error: %v", err)
		}

		got, err := db.GetForgeProject(project.Id)
		if err != nil {
			t.Fatalf("GetForgeProject() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetForgeProject() returned nil")
		}
		if got.Name != "Phoenix Sigil" {
			t.Errorf("Name = %q, want %q", got.Name, "Phoenix Sigil")
		}
	})

	t.Run("list projects", func(t *testing.T) {
		projects, err := db.ListForgeProjects()
		if err != nil {
			t.Fatalf("ListForgeProjects() error: %v", err)
		}
		if len(projects) != 1 {
			t.Errorf("ListForgeProjects() count = %d, want 1", len(projects))
		}
	})

	t.Run("delete project", func(t *testing.T) {
		if err := db.DeleteForgeProject(project.Id); err != nil {
			t.Fatalf("DeleteForgeProject() error: %v", err)
		}
		got, _ := db.GetForgeProject(project.Id)
		if got != nil {
			t.Error("Project should be deleted")
		}
	})
}

func TestShadowPlayAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	play := &pb.ShadowPlay{
		Id:              []byte("play-id-1234567890123456789012345678"),
		DirectorPubkey:  []byte("director-key-1234567890123456789"),
		Title:           "The Phantom's Dance",
		Script:          "A tale of shadows and light...",
		ScheduledTime:   now + 3600,
		DurationSeconds: 1800,
		State:           pb.ShadowPlayState_SHADOW_PLAY_STATE_CASTING,
	}

	t.Run("put and get play", func(t *testing.T) {
		if err := db.PutShadowPlay(play); err != nil {
			t.Fatalf("PutShadowPlay() error: %v", err)
		}

		got, err := db.GetShadowPlay(play.Id)
		if err != nil {
			t.Fatalf("GetShadowPlay() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetShadowPlay() returned nil")
		}
		if got.Title != "The Phantom's Dance" {
			t.Errorf("Title = %q, want %q", got.Title, "The Phantom's Dance")
		}
	})

	t.Run("list plays", func(t *testing.T) {
		plays, err := db.ListShadowPlays()
		if err != nil {
			t.Fatalf("ListShadowPlays() error: %v", err)
		}
		if len(plays) != 1 {
			t.Errorf("ListShadowPlays() count = %d, want 1", len(plays))
		}
	})

	t.Run("delete play", func(t *testing.T) {
		if err := db.DeleteShadowPlay(play.Id); err != nil {
			t.Fatalf("DeleteShadowPlay() error: %v", err)
		}
		got, _ := db.GetShadowPlay(play.Id)
		if got != nil {
			t.Error("Play should be deleted")
		}
	})
}

func TestPhantomCouncilAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	council := &pb.PhantomCouncil{
		Id:            []byte("council-id-123456789012345678901234"),
		Name:          "Shadow Council",
		FounderPubkey: []byte("founder-key-123456789012345678901"),
		CreatedAt:     now,
		MinResonance:  200,
		Quorum:        3,
		State:         pb.CouncilState_COUNCIL_STATE_ACTIVE,
	}

	t.Run("put and get council", func(t *testing.T) {
		if err := db.PutPhantomCouncil(council); err != nil {
			t.Fatalf("PutPhantomCouncil() error: %v", err)
		}

		got, err := db.GetPhantomCouncil(council.Id)
		if err != nil {
			t.Fatalf("GetPhantomCouncil() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetPhantomCouncil() returned nil")
		}
		if got.Name != "Shadow Council" {
			t.Errorf("Name = %q, want %q", got.Name, "Shadow Council")
		}
		if got.MinResonance != 200 {
			t.Errorf("MinResonance = %d, want 200", got.MinResonance)
		}
	})

	t.Run("list active councils", func(t *testing.T) {
		councils, err := db.ListActivePhantomCouncils()
		if err != nil {
			t.Fatalf("ListActivePhantomCouncils() error: %v", err)
		}
		if len(councils) != 1 {
			t.Errorf("ListActivePhantomCouncils() count = %d, want 1", len(councils))
		}
	})

	t.Run("delete council", func(t *testing.T) {
		if err := db.DeletePhantomCouncil(council.Id); err != nil {
			t.Fatalf("DeletePhantomCouncil() error: %v", err)
		}
		got, _ := db.GetPhantomCouncil(council.Id)
		if got != nil {
			t.Error("Council should be deleted")
		}
	})
}

func TestPhantomGiftAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	gift := &pb.PhantomGift{
		Id:              []byte("gift-id-1234567890123456789012345678"),
		SenderPubkey:    []byte("sender-key-12345678901234567890123"),
		RecipientPubkey: []byte("recipient-key-123456789012345678901"),
		EffectType:      3,
		CreatedAt:       now,
		ExpiresAt:       now + 86400*7,
	}

	t.Run("put and get gift", func(t *testing.T) {
		if err := db.PutPhantomGift(gift); err != nil {
			t.Fatalf("PutPhantomGift() error: %v", err)
		}

		got, err := db.GetPhantomGift(gift.Id)
		if err != nil {
			t.Fatalf("GetPhantomGift() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetPhantomGift() returned nil")
		}
		if got.EffectType != 3 {
			t.Errorf("EffectType = %d, want 3", got.EffectType)
		}
	})

	t.Run("list gifts for recipient", func(t *testing.T) {
		gifts, err := db.ListGiftsForRecipient(gift.RecipientPubkey)
		if err != nil {
			t.Fatalf("ListGiftsForRecipient() error: %v", err)
		}
		if len(gifts) != 1 {
			t.Errorf("ListGiftsForRecipient() count = %d, want 1", len(gifts))
		}
	})

	t.Run("delete gift", func(t *testing.T) {
		if err := db.DeletePhantomGift(gift.Id); err != nil {
			t.Fatalf("DeletePhantomGift() error: %v", err)
		}
		got, _ := db.GetPhantomGift(gift.Id)
		if got != nil {
			t.Error("Gift should be deleted")
		}
	})
}

func TestSpecterMarkAccessors(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	now := time.Now().Unix()
	mark := &pb.SpecterMark{
		Id:            []byte("mark-id-1234567890123456789012345678"),
		SpecterPubkey: []byte("specter-key-123456789012345678901"),
		TargetPubkey:  []byte("target-key-12345678901234567890123"),
		Content:       "A shadowy observation",
		CreatedAt:     now,
		ExpiresAt:     now + 86400*30,
	}

	t.Run("put and get mark", func(t *testing.T) {
		if err := db.PutSpecterMark(mark); err != nil {
			t.Fatalf("PutSpecterMark() error: %v", err)
		}

		got, err := db.GetSpecterMark(mark.Id)
		if err != nil {
			t.Fatalf("GetSpecterMark() error: %v", err)
		}
		if got == nil {
			t.Fatal("GetSpecterMark() returned nil")
		}
		if got.Content != "A shadowy observation" {
			t.Errorf("Content = %q, want %q", got.Content, "A shadowy observation")
		}
	})

	t.Run("list marks for target", func(t *testing.T) {
		marks, err := db.ListMarksForTarget(mark.TargetPubkey)
		if err != nil {
			t.Fatalf("ListMarksForTarget() error: %v", err)
		}
		if len(marks) != 1 {
			t.Errorf("ListMarksForTarget() count = %d, want 1", len(marks))
		}
	})

	t.Run("delete mark", func(t *testing.T) {
		if err := db.DeleteSpecterMark(mark.Id); err != nil {
			t.Fatalf("DeleteSpecterMark() error: %v", err)
		}
		got, _ := db.GetSpecterMark(mark.Id)
		if got != nil {
			t.Error("Mark should be deleted")
		}
	})
}
