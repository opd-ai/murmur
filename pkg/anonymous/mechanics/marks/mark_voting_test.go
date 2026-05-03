package marks

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestNewMarkVoteStore(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	if voteStore == nil {
		t.Fatal("NewMarkVoteStore returned nil")
	}
	if voteStore.markStore != markStore {
		t.Error("markStore reference not set")
	}
}

func TestCanVote(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	// Create a marker and a mark.
	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, privKey)

	// Test insufficient resonance.
	var voterKey [32]byte
	rand.Read(voterKey[:])
	err := voteStore.CanVote(voterKey, mark.ID, 40)
	if err != ErrVoteInsufficientResonance {
		t.Errorf("expected ErrVoteInsufficientResonance, got %v", err)
	}

	// Test valid voter.
	err = voteStore.CanVote(voterKey, mark.ID, 60)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Test voting on own mark.
	err = voteStore.CanVote(markerKey, mark.ID, 100)
	if err != ErrVoteOnOwnMark {
		t.Errorf("expected ErrVoteOnOwnMark, got %v", err)
	}

	// Test non-existent mark.
	var nonExistentMarkID [32]byte
	rand.Read(nonExistentMarkID[:])
	err = voteStore.CanVote(voterKey, nonExistentMarkID, 60)
	if err != ErrMarkNotFound {
		t.Errorf("expected ErrMarkNotFound, got %v", err)
	}
}

func TestCastVote(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	markerPub, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
	_ = markerPub
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

	var voterKey [32]byte
	rand.Read(voterKey[:])
	voterPub, voterPriv, _ := ed25519.GenerateKey(rand.Reader)
	_ = voterPub

	// Cast an endorsement.
	vote, err := voteStore.CastVote(voterKey, mark.ID, MarkVoteEndorse, voterPriv)
	if err != nil {
		t.Fatalf("CastVote failed: %v", err)
	}
	if vote == nil {
		t.Fatal("vote is nil")
	}
	if vote.VoteType != MarkVoteEndorse {
		t.Errorf("expected MarkVoteEndorse, got %v", vote.VoteType)
	}
	if vote.VoterKey != voterKey {
		t.Error("voter key mismatch")
	}
	if vote.MarkID != mark.ID {
		t.Error("mark ID mismatch")
	}
	if len(vote.Signature) == 0 {
		t.Error("expected signature")
	}

	// Try to vote again (should fail).
	_, err = voteStore.CastVote(voterKey, mark.ID, MarkVoteEndorse, voterPriv)
	if err != ErrVoteAlreadyCast {
		t.Errorf("expected ErrVoteAlreadyCast, got %v", err)
	}

	// Invalid vote type.
	var voter2Key [32]byte
	rand.Read(voter2Key[:])
	_, voter2Priv, _ := ed25519.GenerateKey(rand.Reader)
	_, err = voteStore.CastVote(voter2Key, mark.ID, MarkVoteType(99), voter2Priv)
	if err != ErrInvalidMarkVoteType {
		t.Errorf("expected ErrInvalidMarkVoteType, got %v", err)
	}
}

func TestGetVote(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

	var voterKey [32]byte
	rand.Read(voterKey[:])
	_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)

	vote, _ := voteStore.CastVote(voterKey, mark.ID, MarkVoteEndorse, voterPriv)

	// Get existing vote.
	retrieved := voteStore.GetVote(vote.ID)
	if retrieved == nil {
		t.Fatal("GetVote returned nil")
	}
	if retrieved.ID != vote.ID {
		t.Error("vote ID mismatch")
	}

	// Get non-existent vote.
	var badID [32]byte
	rand.Read(badID[:])
	retrieved = voteStore.GetVote(badID)
	if retrieved != nil {
		t.Error("expected nil for non-existent vote")
	}
}

func TestGetVotesOnMark(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

	// Cast multiple votes.
	for i := 0; i < 5; i++ {
		var voterKey [32]byte
		rand.Read(voterKey[:])
		_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)

		voteType := MarkVoteEndorse
		if i%2 == 0 {
			voteType = MarkVoteChallenge
		}
		voteStore.CastVote(voterKey, mark.ID, voteType, voterPriv)
	}

	votes := voteStore.GetVotesOnMark(mark.ID)
	if len(votes) != 5 {
		t.Errorf("expected 5 votes, got %d", len(votes))
	}
}

func TestGetVotesByVoter(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	var voterKey [32]byte
	rand.Read(voterKey[:])
	_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)

	// Create multiple marks and vote on them.
	for i := 0; i < 3; i++ {
		var markerKey [32]byte
		rand.Read(markerKey[:])
		targetKey := make([]byte, 32)
		rand.Read(targetKey)

		_, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
		mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

		voteStore.CastVote(voterKey, mark.ID, MarkVoteEndorse, voterPriv)
	}

	votes := voteStore.GetVotesByVoter(voterKey)
	if len(votes) != 3 {
		t.Errorf("expected 3 votes, got %d", len(votes))
	}
}

func TestMarkScore(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

	// Cast 3 endorsements.
	for i := 0; i < 3; i++ {
		var voterKey [32]byte
		rand.Read(voterKey[:])
		_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)
		voteStore.CastVote(voterKey, mark.ID, MarkVoteEndorse, voterPriv)
	}

	// Cast 1 challenge.
	var challengerKey [32]byte
	rand.Read(challengerKey[:])
	_, challengerPriv, _ := ed25519.GenerateKey(rand.Reader)
	voteStore.CastVote(challengerKey, mark.ID, MarkVoteChallenge, challengerPriv)

	score := voteStore.GetMarkScore(mark.ID)
	if score == nil {
		t.Fatal("score is nil")
	}
	if score.Endorsements != 3 {
		t.Errorf("expected 3 endorsements, got %d", score.Endorsements)
	}
	if score.Challenges != 1 {
		t.Errorf("expected 1 challenge, got %d", score.Challenges)
	}
	if score.NetScore != 2 {
		t.Errorf("expected net score 2, got %d", score.NetScore)
	}
	if score.IsHidden {
		t.Error("mark should not be hidden")
	}
}

func TestMarkHiding(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

	// Cast enough challenges to hide the mark (threshold is 5).
	for i := 0; i < 7; i++ {
		var voterKey [32]byte
		rand.Read(voterKey[:])
		_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)
		voteStore.CastVote(voterKey, mark.ID, MarkVoteChallenge, voterPriv)
	}

	if !voteStore.IsMarkHidden(mark.ID) {
		t.Error("mark should be hidden with 7 challenges")
	}

	// Add endorsement.
	var endorserKey [32]byte
	rand.Read(endorserKey[:])
	_, endorserPriv, _ := ed25519.GenerateKey(rand.Reader)
	voteStore.CastVote(endorserKey, mark.ID, MarkVoteEndorse, endorserPriv)

	// Net score is now -6, still hidden.
	if !voteStore.IsMarkHidden(mark.ID) {
		t.Error("mark should still be hidden")
	}

	hiddenMarks := voteStore.GetHiddenMarks()
	found := false
	for _, id := range hiddenMarks {
		if id == mark.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("mark not in hidden marks list")
	}
}

func TestEffectiveVisibility(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

	// Initial visibility is 1.0 (fresh mark).
	baseVis := mark.CurrentVisibility()
	if baseVis < 0.99 || baseVis > 1.01 {
		t.Errorf("expected base visibility ~1.0, got %f", baseVis)
	}

	// Effective visibility without votes matches base (with tolerance for time drift).
	effVis := voteStore.GetEffectiveVisibility(mark.ID)
	if effVis < baseVis-0.0001 || effVis > baseVis+0.0001 {
		t.Errorf("expected effective visibility ~%f, got %f", baseVis, effVis)
	}

	// Add 3 endorsements: +30% boost.
	for i := 0; i < 3; i++ {
		var voterKey [32]byte
		rand.Read(voterKey[:])
		_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)
		voteStore.CastVote(voterKey, mark.ID, MarkVoteEndorse, voterPriv)
	}

	effVis = voteStore.GetEffectiveVisibility(mark.ID)
	expected := baseVis * 1.3 // 1.0 + 3*0.1
	if effVis < expected-0.01 || effVis > expected+0.01 {
		t.Errorf("expected effective visibility ~%f, got %f", expected, effVis)
	}

	// Test hidden mark returns 0.
	voteStore2 := NewMarkVoteStore(markStore)
	for i := 0; i < 7; i++ {
		var voterKey [32]byte
		rand.Read(voterKey[:])
		_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)
		voteStore2.CastVote(voterKey, mark.ID, MarkVoteChallenge, voterPriv)
	}
	effVis = voteStore2.GetEffectiveVisibility(mark.ID)
	if effVis != 0.0 {
		t.Errorf("expected 0 for hidden mark, got %f", effVis)
	}
}

func TestRemoveVote(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

	var voterKey [32]byte
	rand.Read(voterKey[:])
	_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)

	vote, _ := voteStore.CastVote(voterKey, mark.ID, MarkVoteEndorse, voterPriv)

	// Verify vote exists.
	if voteStore.GetVote(vote.ID) == nil {
		t.Fatal("vote should exist")
	}
	if !voteStore.HasVoted(voterKey, mark.ID) {
		t.Error("HasVoted should return true")
	}

	// Remove vote.
	err := voteStore.RemoveVote(vote.ID)
	if err != nil {
		t.Errorf("RemoveVote failed: %v", err)
	}

	// Verify vote removed.
	if voteStore.GetVote(vote.ID) != nil {
		t.Error("vote should be removed")
	}
	if voteStore.HasVoted(voterKey, mark.ID) {
		t.Error("HasVoted should return false after removal")
	}

	// Try to remove again.
	err = voteStore.RemoveVote(vote.ID)
	if err != ErrVoteNotFound {
		t.Errorf("expected ErrVoteNotFound, got %v", err)
	}
}

func TestPurgeExpiredVotes(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

	// Create expired vote directly.
	var voterKey [32]byte
	rand.Read(voterKey[:])
	_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)
	vote, _ := voteStore.CastVote(voterKey, mark.ID, MarkVoteEndorse, voterPriv)

	// Manually expire the vote for testing.
	vote.ExpiresAt = time.Now().Add(-time.Hour)

	purged := voteStore.PurgeExpiredVotes()
	if purged != 1 {
		t.Errorf("expected 1 purged vote, got %d", purged)
	}
}

func TestCountVotesByType(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

	// Cast 5 endorsements and 3 challenges.
	for i := 0; i < 5; i++ {
		var voterKey [32]byte
		rand.Read(voterKey[:])
		_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)
		voteStore.CastVote(voterKey, mark.ID, MarkVoteEndorse, voterPriv)
	}
	for i := 0; i < 3; i++ {
		var voterKey [32]byte
		rand.Read(voterKey[:])
		_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)
		voteStore.CastVote(voterKey, mark.ID, MarkVoteChallenge, voterPriv)
	}

	endorsements, challenges := voteStore.CountVotesByType(mark.ID)
	if endorsements != 5 {
		t.Errorf("expected 5 endorsements, got %d", endorsements)
	}
	if challenges != 3 {
		t.Errorf("expected 3 challenges, got %d", challenges)
	}
}

func TestCountTotalVotes(t *testing.T) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	if voteStore.CountTotalVotes() != 0 {
		t.Error("expected 0 initial votes")
	}

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

	for i := 0; i < 10; i++ {
		var voterKey [32]byte
		rand.Read(voterKey[:])
		_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)
		voteStore.CastVote(voterKey, mark.ID, MarkVoteEndorse, voterPriv)
	}

	if voteStore.CountTotalVotes() != 10 {
		t.Errorf("expected 10 votes, got %d", voteStore.CountTotalVotes())
	}
}

func TestMarkVoteTypeString(t *testing.T) {
	tests := []struct {
		vt   MarkVoteType
		want string
	}{
		{MarkVoteEndorse, "Endorse"},
		{MarkVoteChallenge, "Challenge"},
		{MarkVoteType(99), "Unknown"},
	}

	for _, tt := range tests {
		got := MarkVoteTypeString(tt.vt)
		if got != tt.want {
			t.Errorf("MarkVoteTypeString(%v) = %q, want %q", tt.vt, got, tt.want)
		}
	}
}

func TestVoteIsExpired(t *testing.T) {
	vote := &MarkVote{
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if vote.IsExpired() {
		t.Error("fresh vote should not be expired")
	}

	vote.ExpiresAt = time.Now().Add(-time.Hour)
	if !vote.IsExpired() {
		t.Error("old vote should be expired")
	}
}

func TestNilMarkStore(t *testing.T) {
	// Test with nil mark store.
	voteStore := NewMarkVoteStore(nil)

	var voterKey [32]byte
	rand.Read(voterKey[:])
	var markID [32]byte
	rand.Read(markID[:])

	// CanVote should still work (skip mark validation).
	err := voteStore.CanVote(voterKey, markID, 60)
	if err != nil {
		t.Errorf("expected nil error with nil mark store, got %v", err)
	}

	// GetEffectiveVisibility with nil mark store.
	vis := voteStore.GetEffectiveVisibility(markID)
	if vis != 0.0 {
		t.Errorf("expected 0 visibility with nil mark store, got %f", vis)
	}
}

func BenchmarkCastVote(b *testing.B) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var voterKey [32]byte
		rand.Read(voterKey[:])
		_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)
		voteStore.CastVote(voterKey, mark.ID, MarkVoteEndorse, voterPriv)
	}
}

func BenchmarkGetEffectiveVisibility(b *testing.B) {
	markStore := NewMarkStore()
	voteStore := NewMarkVoteStore(markStore)

	var markerKey [32]byte
	rand.Read(markerKey[:])
	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, markerPriv, _ := ed25519.GenerateKey(rand.Reader)
	mark, _ := markStore.PlaceMark(markerKey, targetKey, MarkAlly, "", 150, markerPriv)

	// Add some votes.
	for i := 0; i < 20; i++ {
		var voterKey [32]byte
		rand.Read(voterKey[:])
		_, voterPriv, _ := ed25519.GenerateKey(rand.Reader)
		voteStore.CastVote(voterKey, mark.ID, MarkVoteEndorse, voterPriv)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		voteStore.GetEffectiveVisibility(mark.ID)
	}
}
