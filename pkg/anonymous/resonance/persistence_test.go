// Package resonance provides local reputation computation and rank thresholds.
package resonance

import (
	"os"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/store"
)

func createTestDB(t *testing.T) (*store.DB, func()) {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "resonance-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	db, err := store.Open(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		t.Fatalf("failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(tmpPath)
	}

	return db, cleanup
}

func TestPersistentScorerBasic(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	scorer, err := NewPersistentScorer(db)
	if err != nil {
		t.Fatalf("NewPersistentScorer failed: %v", err)
	}

	// Get score for new specter (should create it).
	score := scorer.GetScore("specter1")
	if score == nil {
		t.Fatal("expected non-nil score")
	}

	// Modify the score.
	score.AddPublication()
	score.AddPuzzleSolved()

	// Force persist.
	scorer.SetScore("specter1", score)

	// Verify count.
	if scorer.Count() != 1 {
		t.Errorf("expected count 1, got %d", scorer.Count())
	}
}

func TestPersistentScorerPersistence(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	// Create scorer and add data.
	scorer1, err := NewPersistentScorer(db)
	if err != nil {
		t.Fatalf("NewPersistentScorer failed: %v", err)
	}

	score := scorer1.GetScore("specter1")
	score.Publications = 10
	score.PuzzlesSolved = 5
	score.GamesWon = 3
	scorer1.SetScore("specter1", score)

	// Create new scorer from same database.
	scorer2, err := NewPersistentScorer(db)
	if err != nil {
		t.Fatalf("NewPersistentScorer 2 failed: %v", err)
	}

	// Verify data was loaded.
	loadedScore := scorer2.GetScore("specter1")
	if loadedScore.Publications != 10 {
		t.Errorf("expected Publications=10, got %d", loadedScore.Publications)
	}
	if loadedScore.PuzzlesSolved != 5 {
		t.Errorf("expected PuzzlesSolved=5, got %d", loadedScore.PuzzlesSolved)
	}
	if loadedScore.GamesWon != 3 {
		t.Errorf("expected GamesWon=3, got %d", loadedScore.GamesWon)
	}
}

func TestPersistentScorerRemove(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	scorer, err := NewPersistentScorer(db)
	if err != nil {
		t.Fatalf("NewPersistentScorer failed: %v", err)
	}

	// Add and then remove.
	scorer.GetScore("specter1")
	scorer.GetScore("specter2")

	if scorer.Count() != 2 {
		t.Errorf("expected count 2, got %d", scorer.Count())
	}

	scorer.RemoveScore("specter1")

	if scorer.Count() != 1 {
		t.Errorf("expected count 1, got %d", scorer.Count())
	}

	// Create new scorer and verify removal persisted.
	scorer2, err := NewPersistentScorer(db)
	if err != nil {
		t.Fatalf("NewPersistentScorer 2 failed: %v", err)
	}

	if scorer2.Count() != 1 {
		t.Errorf("expected count 1 after reload, got %d", scorer2.Count())
	}
}

func TestPersistentScorerTopSpecters(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	scorer, err := NewPersistentScorer(db)
	if err != nil {
		t.Fatalf("NewPersistentScorer failed: %v", err)
	}

	// Add specters with different scores.
	s1 := scorer.GetScore("low")
	s1.Publications = 1

	s2 := scorer.GetScore("medium")
	s2.Publications = 50
	s2.PuzzlesSolved = 10

	s3 := scorer.GetScore("high")
	s3.Publications = 100
	s3.PuzzlesSolved = 50
	s3.GiftsGiven = 20

	scorer.SetScore("low", s1)
	scorer.SetScore("medium", s2)
	scorer.SetScore("high", s3)

	// Get top 2.
	top := scorer.TopSpecters(2)
	if len(top) != 2 {
		t.Fatalf("expected 2 top specters, got %d", len(top))
	}

	// High should be first.
	if top[0] != "high" {
		t.Errorf("expected 'high' first, got %s", top[0])
	}
	if top[1] != "medium" {
		t.Errorf("expected 'medium' second, got %s", top[1])
	}
}

func TestPersistentScorerUpdateScore(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	scorer, err := NewPersistentScorer(db)
	if err != nil {
		t.Fatalf("NewPersistentScorer failed: %v", err)
	}

	// Update with function.
	scorer.UpdateScore("specter1", func(s *Score) {
		s.AddPublication()
		s.AddPuzzleSolved()
		s.AddGiftGiven()
	})

	// Verify changes.
	score := scorer.GetScore("specter1")
	if score.Publications != 1 {
		t.Errorf("expected Publications=1, got %d", score.Publications)
	}
	if score.PuzzlesSolved != 1 {
		t.Errorf("expected PuzzlesSolved=1, got %d", score.PuzzlesSolved)
	}
	if score.GiftsGiven != 1 {
		t.Errorf("expected GiftsGiven=1, got %d", score.GiftsGiven)
	}

	// Verify persistence.
	scorer2, err := NewPersistentScorer(db)
	if err != nil {
		t.Fatalf("NewPersistentScorer 2 failed: %v", err)
	}

	loadedScore := scorer2.GetScore("specter1")
	if loadedScore.Publications != 1 {
		t.Errorf("expected Publications=1 after reload, got %d", loadedScore.Publications)
	}
}

func TestPersistentScorerFlush(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	scorer, err := NewPersistentScorer(db)
	if err != nil {
		t.Fatalf("NewPersistentScorer failed: %v", err)
	}

	// Modify scores directly without SetScore.
	s1 := scorer.GetScore("specter1")
	s1.mu.Lock()
	s1.Publications = 99
	s1.mu.Unlock()

	// Flush all.
	if err := scorer.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Verify persistence.
	scorer2, err := NewPersistentScorer(db)
	if err != nil {
		t.Fatalf("NewPersistentScorer 2 failed: %v", err)
	}

	loadedScore := scorer2.GetScore("specter1")
	if loadedScore.Publications != 99 {
		t.Errorf("expected Publications=99, got %d", loadedScore.Publications)
	}
}

func TestPersistentScorerNilDB(t *testing.T) {
	// Should work without database (pure in-memory).
	scorer, err := NewPersistentScorer(nil)
	if err != nil {
		t.Fatalf("NewPersistentScorer(nil) failed: %v", err)
	}

	score := scorer.GetScore("specter1")
	score.AddPublication()
	scorer.SetScore("specter1", score)

	if scorer.Count() != 1 {
		t.Errorf("expected count 1, got %d", scorer.Count())
	}
}

func TestPersistentScorerTimeFields(t *testing.T) {
	db, cleanup := createTestDB(t)
	defer cleanup()

	scorer, err := NewPersistentScorer(db)
	if err != nil {
		t.Fatalf("NewPersistentScorer failed: %v", err)
	}

	// Create score with specific times.
	before := time.Now().Truncate(time.Second)
	score := scorer.GetScore("specter1")
	score.AddPublication()

	// Verify times are set.
	if score.CreatedAt.Before(before) {
		t.Error("CreatedAt should be after test start")
	}
	if score.LastActivity.Before(before) {
		t.Error("LastActivity should be after test start")
	}

	scorer.SetScore("specter1", score)

	// Reload and verify times persisted.
	scorer2, err := NewPersistentScorer(db)
	if err != nil {
		t.Fatalf("NewPersistentScorer 2 failed: %v", err)
	}

	loadedScore := scorer2.GetScore("specter1")
	// Times should be within a second of original (due to Unix timestamp precision).
	if loadedScore.CreatedAt.Unix() != score.CreatedAt.Unix() {
		t.Errorf("CreatedAt mismatch: %v vs %v", loadedScore.CreatedAt, score.CreatedAt)
	}
	if loadedScore.LastActivity.Unix() != score.LastActivity.Unix() {
		t.Errorf("LastActivity mismatch: %v vs %v", loadedScore.LastActivity, score.LastActivity)
	}
}
