package resonance

import (
	"testing"
	"time"
)

func TestRankString(t *testing.T) {
	tests := []struct {
		rank Rank
		want string
	}{
		{RankNone, "None"},
		{RankShade, "Shade"},
		{RankWraith, "Wraith"},
		{RankShadeWraith, "Shade-Wraith"},
		{RankPhantom, "Phantom"},
		{RankCouncil, "Council-Eligible"},
		{RankAbyss, "Abyss"},
	}

	for _, tc := range tests {
		if got := tc.rank.String(); got != tc.want {
			t.Errorf("%d.String() = %s, want %s", tc.rank, got, tc.want)
		}
	}
}

func TestRankFromScore(t *testing.T) {
	tests := []struct {
		score int
		want  Rank
	}{
		{0, RankNone},
		{24, RankNone},
		{25, RankShade},
		{49, RankShade},
		{50, RankWraith},
		{74, RankWraith},
		{75, RankShadeWraith},
		{99, RankShadeWraith},
		{100, RankPhantom},
		{199, RankPhantom},
		{200, RankCouncil},
		{499, RankCouncil},
		{500, RankAbyss},
		{1000, RankAbyss},
	}

	for _, tc := range tests {
		if got := RankFromScore(tc.score); got != tc.want {
			t.Errorf("RankFromScore(%d) = %s, want %s", tc.score, got, tc.want)
		}
	}
}

func TestNewScore(t *testing.T) {
	s := NewScore()

	if s == nil {
		t.Fatal("NewScore returned nil")
	}

	if s.Publications != 0 {
		t.Errorf("initial Publications = %d, want 0", s.Publications)
	}

	if s.Compute() < 0 {
		t.Error("initial score should be non-negative")
	}
}

func TestAddPublication(t *testing.T) {
	s := NewScore()

	s.AddPublication()

	if s.Publications != 1 {
		t.Errorf("Publications = %d, want 1", s.Publications)
	}

	// Score should increase.
	score := s.Compute()
	if score <= 0 {
		t.Error("score should increase after publication")
	}
}

func TestAddConsecutiveDay(t *testing.T) {
	s := NewScore()

	for i := 0; i < 5; i++ {
		s.AddConsecutiveDay()
	}

	if s.ConsecutiveDays != 5 {
		t.Errorf("ConsecutiveDays = %d, want 5", s.ConsecutiveDays)
	}
}

func TestResetStreak(t *testing.T) {
	s := NewScore()

	s.AddConsecutiveDay()
	s.AddConsecutiveDay()
	s.ResetStreak()

	if s.ConsecutiveDays != 0 {
		t.Errorf("ConsecutiveDays after reset = %d, want 0", s.ConsecutiveDays)
	}
}

func TestAddPuzzleSolved(t *testing.T) {
	s := NewScore()

	s.AddPuzzleSolved()
	s.AddPuzzleSolved()

	if s.PuzzlesSolved != 2 {
		t.Errorf("PuzzlesSolved = %d, want 2", s.PuzzlesSolved)
	}
}

func TestAddGameResult(t *testing.T) {
	s := NewScore()

	s.AddGameResult(true)
	s.AddGameResult(true)
	s.AddGameResult(false)

	if s.GamesWon != 2 {
		t.Errorf("GamesWon = %d, want 2", s.GamesWon)
	}
	if s.GamesLost != 1 {
		t.Errorf("GamesLost = %d, want 1", s.GamesLost)
	}
}

func TestAddGift(t *testing.T) {
	s := NewScore()

	s.AddGiftGiven()
	s.AddGiftGiven()
	s.AddGiftReceived()

	if s.GiftsGiven != 2 {
		t.Errorf("GiftsGiven = %d, want 2", s.GiftsGiven)
	}
	if s.GiftsReceived != 1 {
		t.Errorf("GiftsReceived = %d, want 1", s.GiftsReceived)
	}
}

func TestAddEndorsement(t *testing.T) {
	s := NewScore()

	s.AddEndorsement(false)
	s.AddEndorsement(true)

	if s.Endorsements != 2 {
		t.Errorf("Endorsements = %d, want 2", s.Endorsements)
	}
	if s.HighTierEndorsements != 1 {
		t.Errorf("HighTierEndorsements = %d, want 1", s.HighTierEndorsements)
	}
}

func TestComputeScoreIncreases(t *testing.T) {
	s := NewScore()

	initial := s.Compute()

	// Add various activities.
	s.AddPublication()
	s.AddPuzzleSolved()
	s.AddGiftGiven()
	s.AddEndorsement(false)

	after := s.Compute()

	if after <= initial {
		t.Errorf("score should increase after activity: %d -> %d", initial, after)
	}
}

func TestScoreCaching(t *testing.T) {
	s := NewScore()

	// First compute should cache.
	score1 := s.Compute()

	// Second compute should use cache.
	s.mu.Lock()
	wasCached := s.cacheValid
	s.mu.Unlock()

	if !wasCached {
		t.Error("score should be cached after Compute")
	}

	// Activity should invalidate cache.
	s.AddPublication()

	s.mu.Lock()
	isStillCached := s.cacheValid
	s.mu.Unlock()

	if isStillCached {
		t.Error("cache should be invalidated after activity")
	}

	score2 := s.Compute()
	if score2 <= score1 {
		t.Error("score should increase after publication")
	}
}

func TestComputeDecay(t *testing.T) {
	s := NewScore()

	// Add some activity.
	for i := 0; i < 10; i++ {
		s.AddPublication()
	}

	activeScore := s.Compute()

	// Simulate old activity by manipulating LastActivity.
	s.mu.Lock()
	s.LastActivity = time.Now().Add(-14 * 24 * time.Hour) // 14 days ago
	s.cacheValid = false
	s.mu.Unlock()

	decayedScore := s.Compute()

	if decayedScore >= activeScore {
		t.Errorf("decayed score (%d) should be less than active score (%d)",
			decayedScore, activeScore)
	}
}

func TestDecayGracePeriod(t *testing.T) {
	s := NewScore()

	for i := 0; i < 10; i++ {
		s.AddPublication()
	}

	activeScore := s.Compute()

	// 2 days inactive (within grace period).
	s.mu.Lock()
	s.LastActivity = time.Now().Add(-2 * 24 * time.Hour)
	s.cacheValid = false
	s.mu.Unlock()

	graceScore := s.Compute()

	if graceScore != activeScore {
		t.Errorf("score within grace period should not decay: %d != %d",
			graceScore, activeScore)
	}
}

func TestRank(t *testing.T) {
	s := NewScore()

	// Initial rank should be None.
	if s.Rank() != RankNone {
		t.Errorf("initial rank = %s, want None", s.Rank())
	}

	// Add enough activity to reach Shade.
	for i := 0; i < 100; i++ {
		s.AddPublication()
		s.AddPuzzleSolved()
		s.AddEndorsement(true)
	}

	rank := s.Rank()
	if rank == RankNone {
		t.Error("rank should be higher than None after significant activity")
	}
}

func TestEchoIndex(t *testing.T) {
	s := NewScore()

	// New specter should have low echo.
	echo := s.EchoIndex()
	if echo <= 0 || echo > 1 {
		t.Errorf("EchoIndex = %f, should be in (0, 1]", echo)
	}

	// Add activity.
	for i := 0; i < 50; i++ {
		s.AddPublication()
	}

	higherEcho := s.EchoIndex()
	if higherEcho <= echo {
		t.Error("EchoIndex should increase with activity")
	}
}

func TestNewScorer(t *testing.T) {
	scorer := NewScorer()

	if scorer == nil {
		t.Fatal("NewScorer returned nil")
	}

	if scorer.Count() != 0 {
		t.Errorf("initial count = %d, want 0", scorer.Count())
	}
}

func TestScorerGetScore(t *testing.T) {
	scorer := NewScorer()

	score := scorer.GetScore("specter-1")

	if score == nil {
		t.Fatal("GetScore returned nil")
	}

	// Same ID should return same score.
	score2 := scorer.GetScore("specter-1")
	if score != score2 {
		t.Error("GetScore should return same Score for same ID")
	}

	if scorer.Count() != 1 {
		t.Errorf("count = %d, want 1", scorer.Count())
	}
}

func TestScorerSetScore(t *testing.T) {
	scorer := NewScorer()

	customScore := NewScore()
	customScore.AddPublication()

	scorer.SetScore("custom-specter", customScore)

	retrieved := scorer.GetScore("custom-specter")
	if retrieved.Publications != 1 {
		t.Error("SetScore should set the provided score")
	}
}

func TestScorerRemoveScore(t *testing.T) {
	scorer := NewScorer()

	scorer.GetScore("to-remove")
	scorer.RemoveScore("to-remove")

	if scorer.Count() != 0 {
		t.Errorf("count after remove = %d, want 0", scorer.Count())
	}
}

func TestScorerTopSpecters(t *testing.T) {
	scorer := NewScorer()

	// Create specters with different scores.
	for i := 0; i < 5; i++ {
		s := scorer.GetScore(string(rune('a' + i)))
		for j := 0; j <= i*10; j++ {
			s.AddPublication()
		}
	}

	top3 := scorer.TopSpecters(3)

	if len(top3) != 3 {
		t.Errorf("TopSpecters(3) returned %d, want 3", len(top3))
	}

	// Highest scorer (e) should be first.
	if len(top3) > 0 && top3[0] != "e" {
		t.Errorf("top specter = %s, want e", top3[0])
	}
}

func TestScorerTopSpectersLessThanN(t *testing.T) {
	scorer := NewScorer()

	scorer.GetScore("only-one")

	top5 := scorer.TopSpecters(5)

	if len(top5) != 1 {
		t.Errorf("TopSpecters(5) with 1 specter returned %d", len(top5))
	}
}

func TestDefaultWeights(t *testing.T) {
	w := DefaultWeights()

	sum := w.PublicationConsistency + w.MiniGameQuality +
		w.GiftActivity + w.CommunityEndorsement

	// Weights should sum to 1.0.
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("weights sum = %f, want ~1.0", sum)
	}
}

func TestNewScoreWithWeights(t *testing.T) {
	customWeights := SignalWeights{
		PublicationConsistency: 0.5,
		MiniGameQuality:        0.2,
		GiftActivity:           0.2,
		CommunityEndorsement:   0.1,
	}

	s := NewScoreWithWeights(customWeights)

	if s.weights.PublicationConsistency != 0.5 {
		t.Error("custom weights not applied")
	}
}

func TestScoreConcurrency(t *testing.T) {
	s := NewScore()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			s.AddPublication()
			s.Compute()
			s.Rank()
			s.EchoIndex()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if s.Publications != 10 {
		t.Errorf("concurrent Publications = %d, want 10", s.Publications)
	}
}

func TestScorerConcurrency(t *testing.T) {
	scorer := NewScorer()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		id := string(rune('a' + i))
		go func(specterID string) {
			score := scorer.GetScore(specterID)
			score.AddPublication()
			score.Compute()
			done <- true
		}(id)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if scorer.Count() != 10 {
		t.Errorf("concurrent count = %d, want 10", scorer.Count())
	}
}

// Claims tests — these use the canonical Ristretto-backed ZKClaim path.
// The legacy ClaimGenerator/ClaimVerifier types are deprecated; see claims.go.

func TestZKClaimGenerateAndVerify(t *testing.T) {
	claim, _, err := NewZKClaim(ZKClaimResonanceRange, "specter-1", 100, 50)
	if err != nil {
		t.Fatalf("NewZKClaim failed: %v", err)
	}
	if claim == nil {
		t.Fatal("claim is nil")
	}
	if claim.Type != ZKClaimResonanceRange {
		t.Errorf("claim type = %v, want ZKClaimResonanceRange", claim.Type)
	}
	if claim.SpecterID != "specter-1" {
		t.Errorf("specter ID = %s, want specter-1", claim.SpecterID)
	}
	if claim.Threshold != 50 {
		t.Errorf("threshold = %d, want 50", claim.Threshold)
	}
	if err := claim.Verify(); err != nil {
		t.Errorf("Verify failed: %v", err)
	}
}

func TestZKClaimThresholdNotMet(t *testing.T) {
	_, _, err := NewZKClaim(ZKClaimResonanceRange, "specter-1", 50, 100)
	if err != ErrThresholdNotMet {
		t.Errorf("expected ErrThresholdNotMet, got %v", err)
	}
}

func TestZKClaimExpired(t *testing.T) {
	claim, _, err := NewZKClaim(ZKClaimResonanceRange, "specter-1", 100, 50)
	if err != nil {
		t.Fatalf("NewZKClaim failed: %v", err)
	}
	claim.Proof.Timestamp = time.Now().Add(-10 * time.Minute).Unix()

	verifier := NewRistrettoClaimVerifier()
	err = verifier.VerifyThresholdProof(&claim.Commitment, &claim.Proof, claim.Threshold)
	if err != ErrClaimExpired {
		t.Errorf("expected ErrClaimExpired, got %v", err)
	}
}

func TestZKClaimReplay(t *testing.T) {
	claim, _, err := NewZKClaim(ZKClaimResonanceRange, "specter-1", 100, 50)
	if err != nil {
		t.Fatalf("NewZKClaim failed: %v", err)
	}

	verifier := NewRistrettoClaimVerifier()
	if err := verifier.VerifyThresholdProof(&claim.Commitment, &claim.Proof, claim.Threshold); err != nil {
		t.Fatalf("first VerifyThresholdProof failed: %v", err)
	}
	if err := verifier.VerifyThresholdProof(&claim.Commitment, &claim.Proof, claim.Threshold); err != ErrReplayDetected {
		t.Errorf("expected ErrReplayDetected, got %v", err)
	}
}

func TestZKClaimFuture(t *testing.T) {
	claim, _, err := NewZKClaim(ZKClaimResonanceRange, "specter-1", 100, 50)
	if err != nil {
		t.Fatalf("NewZKClaim failed: %v", err)
	}
	claim.Proof.Timestamp = time.Now().Add(5 * time.Minute).Unix()

	verifier := NewRistrettoClaimVerifier()
	err = verifier.VerifyThresholdProof(&claim.Commitment, &claim.Proof, claim.Threshold)
	if err != ErrInvalidClaim {
		t.Errorf("expected ErrInvalidClaim for future claim, got %v", err)
	}
}

func TestZKClaimVerifierCleanExpired(t *testing.T) {
	verifier := NewRistrettoClaimVerifier()

	oldTimestamp := time.Now().Add(-10 * time.Minute).Unix()
	var oldNonce [32]byte
	copy(oldNonce[:], []byte("old-nonce"))
	verifier.seenOnce[oldNonce] = oldTimestamp

	recentTimestamp := time.Now().Unix()
	var recentNonce [32]byte
	copy(recentNonce[:], []byte("recent-nonce"))
	verifier.seenOnce[recentNonce] = recentTimestamp

	verifier.CleanExpiredNonces()

	if _, exists := verifier.seenOnce[oldNonce]; exists {
		t.Error("old nonce should be cleaned")
	}
	if _, exists := verifier.seenOnce[recentNonce]; !exists {
		t.Error("recent nonce should not be cleaned")
	}
}

func TestZKClaimInvalidProof(t *testing.T) {
	claim, _, err := NewZKClaim(ZKClaimResonanceRange, "specter-1", 100, 50)
	if err != nil {
		t.Fatalf("NewZKClaim failed: %v", err)
	}
	// Corrupt the challenge scalar.
	var zero [32]byte
	claim.Proof.Challenge.SetBytes(&zero)

	verifier := NewRistrettoClaimVerifier()
	err = verifier.VerifyThresholdProof(&claim.Commitment, &claim.Proof, claim.Threshold)
	if err == nil {
		t.Error("expected error for corrupted proof")
	}
}

func TestZKClaimMilestones(t *testing.T) {
	milestones := []int64{
		int64(MilestoneShade),
		int64(MilestoneWraith),
		int64(MilestoneShadeWraith),
		int64(MilestonePhantom),
		int64(MilestoneCouncil),
		int64(MilestoneAbyss),
	}

	for _, m := range milestones {
		claim, _, err := NewZKClaim(ZKClaimResonanceRange, "specter", m, m)
		if err != nil {
			t.Errorf("failed to claim milestone %d: %v", m, err)
		}
		if claim.Threshold != m {
			t.Errorf("claim threshold = %d, want %d", claim.Threshold, m)
		}
	}
}

func TestClaimMilestone(t *testing.T) {
	tests := []struct {
		score int
		want  int
	}{
		{0, 0},
		{24, 0},
		{25, MilestoneShade},
		{50, MilestoneWraith},
		{75, MilestoneShadeWraith},
		{100, MilestonePhantom},
		{200, MilestoneCouncil},
		{500, MilestoneAbyss},
		{1000, MilestoneAbyss},
	}

	for _, tc := range tests {
		got := ClaimMilestone(tc.score)
		if got != tc.want {
			t.Errorf("ClaimMilestone(%d) = %d, want %d", tc.score, got, tc.want)
		}
	}
}

func TestCanClaimMilestone(t *testing.T) {
	if !CanClaimMilestone(50, MilestoneShade) {
		t.Error("score 50 should be able to claim Shade (25)")
	}

	if CanClaimMilestone(20, MilestoneShade) {
		t.Error("score 20 should not be able to claim Shade (25)")
	}
}
