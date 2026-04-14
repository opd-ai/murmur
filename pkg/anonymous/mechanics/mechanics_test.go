package mechanics

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

// --- Gift Tests ---

func TestGiftCatalogAvailableEffects(t *testing.T) {
	catalog := &GiftCatalog{}

	// Test no effects below Resonance 25.
	effects := catalog.AvailableEffects(20)
	if len(effects) != 0 {
		t.Errorf("Expected 0 effects at Resonance 20, got %d", len(effects))
	}

	// Test basic effects at Resonance 25.
	effects = catalog.AvailableEffects(25)
	if len(effects) != 5 {
		t.Errorf("Expected 5 basic effects at Resonance 25, got %d", len(effects))
	}

	// Test expanded effects at Resonance 50.
	effects = catalog.AvailableEffects(50)
	if len(effects) != 15 {
		t.Errorf("Expected 15 effects at Resonance 50, got %d", len(effects))
	}

	// Test premium effects at Resonance 100.
	effects = catalog.AvailableEffects(100)
	if len(effects) != 25 {
		t.Errorf("Expected 25 effects at Resonance 100, got %d", len(effects))
	}
}

func TestRequiredResonance(t *testing.T) {
	tests := []struct {
		effect   EffectType
		expected int
	}{
		{EffectSoftGlowPulse, GiftTierBasic},
		{EffectFaintHaloRing, GiftTierBasic},
		{EffectOrbitingGeometric, GiftTierExpanded},
		{EffectAuroraColorShift, GiftTierExpanded},
		{EffectMultiParticleSystem, GiftTierPremium},
		{EffectFluidSimulation, GiftTierPremium},
	}

	for _, tt := range tests {
		got := RequiredResonance(tt.effect)
		if got != tt.expected {
			t.Errorf("RequiredResonance(%d) = %d, want %d", tt.effect, got, tt.expected)
		}
	}
}

func TestGiftStoreCreateAndGet(t *testing.T) {
	store := NewGiftStore()

	var senderKey [32]byte
	rand.Read(senderKey[:])

	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	// Create a gift.
	gift, err := store.CreateGift(senderKey, recipientKey, EffectSoftGlowPulse, 30, nil)
	if err != nil {
		t.Fatalf("CreateGift failed: %v", err)
	}

	if gift == nil {
		t.Fatal("Expected gift, got nil")
	}

	// Get the gift.
	retrieved, err := store.GetGift(gift.ID)
	if err != nil {
		t.Fatalf("GetGift failed: %v", err)
	}

	if retrieved.Effect != EffectSoftGlowPulse {
		t.Errorf("Expected effect %d, got %d", EffectSoftGlowPulse, retrieved.Effect)
	}
}

func TestGiftStoreInsufficientResonance(t *testing.T) {
	store := NewGiftStore()

	var senderKey [32]byte
	rand.Read(senderKey[:])

	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	// Try to create a premium effect with low resonance.
	_, err := store.CreateGift(senderKey, recipientKey, EffectMultiParticleSystem, 30, nil)
	if err != ErrInsufficientResonance {
		t.Errorf("Expected ErrInsufficientResonance, got %v", err)
	}
}

func TestGiftStoreDailyLimit(t *testing.T) {
	store := NewGiftStore()

	var senderKey [32]byte
	rand.Read(senderKey[:])

	// Send 3 gifts (the daily limit).
	for i := 0; i < MaxGiftsPerDay; i++ {
		recipientKey := make([]byte, 32)
		rand.Read(recipientKey)

		_, err := store.CreateGift(senderKey, recipientKey, EffectSoftGlowPulse, 30, nil)
		if err != nil {
			t.Fatalf("CreateGift %d failed: %v", i, err)
		}
	}

	// Try to send a 4th gift.
	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	_, err := store.CreateGift(senderKey, recipientKey, EffectSoftGlowPulse, 30, nil)
	if err != ErrDailyLimitExceeded {
		t.Errorf("Expected ErrDailyLimitExceeded, got %v", err)
	}
}

func TestGiftExpiration(t *testing.T) {
	gift := &Gift{
		CreatedAt: time.Now().Add(-8 * 24 * time.Hour), // 8 days ago
		ExpiresAt: time.Now().Add(-1 * 24 * time.Hour), // 1 day ago
	}

	if !gift.IsExpired() {
		t.Error("Expected gift to be expired")
	}

	gift2 := &Gift{
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if gift2.IsExpired() {
		t.Error("Expected gift to not be expired")
	}
}

func TestGiftVerification(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	store := NewGiftStore()

	var senderKey [32]byte
	rand.Read(senderKey[:])

	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	gift, err := store.CreateGift(senderKey, recipientKey, EffectSoftGlowPulse, 30, priv)
	if err != nil {
		t.Fatalf("CreateGift failed: %v", err)
	}

	if !VerifyGift(gift, pub) {
		t.Error("Expected gift verification to succeed")
	}

	// Tamper with the gift.
	gift.Effect = EffectEmberTrails
	if VerifyGift(gift, pub) {
		t.Error("Expected gift verification to fail after tampering")
	}
}

func TestGiftStoreGarbageCollect(t *testing.T) {
	store := NewGiftStore()

	var senderKey [32]byte
	rand.Read(senderKey[:])

	// Create an expired gift manually.
	recipientKey := make([]byte, 32)
	rand.Read(recipientKey)

	gift := &Gift{
		SenderPubKey: senderKey,
		RecipientKey: recipientKey,
		Effect:       EffectSoftGlowPulse,
		CreatedAt:    time.Now().Add(-8 * 24 * time.Hour),
		ExpiresAt:    time.Now().Add(-1 * 24 * time.Hour),
	}
	rand.Read(gift.ID[:])

	store.mu.Lock()
	store.gifts[gift.ID] = gift
	store.mu.Unlock()

	removed := store.GarbageCollect()
	if removed != 1 {
		t.Errorf("Expected 1 removed, got %d", removed)
	}
}

func TestEffectName(t *testing.T) {
	name := EffectName(EffectSoftGlowPulse)
	if name != "Soft Glow Pulse" {
		t.Errorf("Expected 'Soft Glow Pulse', got '%s'", name)
	}

	name = EffectName(EffectMultiParticleSystem)
	if name != "Multi-Particle System" {
		t.Errorf("Expected 'Multi-Particle System', got '%s'", name)
	}
}

// --- Mark Tests ---

func TestMarkStoreCreateAndGet(t *testing.T) {
	store := NewMarkStore()

	var markerKey [32]byte
	rand.Read(markerKey[:])

	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	mark, err := store.PlaceMark(markerKey, targetKey, MarkWatcher, "test note", 60, nil)
	if err != nil {
		t.Fatalf("PlaceMark failed: %v", err)
	}

	if mark == nil {
		t.Fatal("Expected mark, got nil")
	}

	retrieved, err := store.GetMark(mark.ID)
	if err != nil {
		t.Fatalf("GetMark failed: %v", err)
	}

	if retrieved.Category != MarkWatcher {
		t.Errorf("Expected category %d, got %d", MarkWatcher, retrieved.Category)
	}
}

func TestMarkStoreInsufficientResonance(t *testing.T) {
	store := NewMarkStore()

	var markerKey [32]byte
	rand.Read(markerKey[:])

	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	_, err := store.PlaceMark(markerKey, targetKey, MarkWatcher, "", 30, nil)
	if err != ErrMarkInsufficientResonance {
		t.Errorf("Expected ErrMarkInsufficientResonance, got %v", err)
	}
}

func TestMarkStoreDuplicatePrevention(t *testing.T) {
	store := NewMarkStore()

	var markerKey [32]byte
	rand.Read(markerKey[:])

	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	// Place first mark.
	_, err := store.PlaceMark(markerKey, targetKey, MarkWatcher, "", 60, nil)
	if err != nil {
		t.Fatalf("PlaceMark failed: %v", err)
	}

	// Try to place second mark on same target.
	_, err = store.PlaceMark(markerKey, targetKey, MarkAlly, "", 60, nil)
	if err != ErrMarkAlreadyPlaced {
		t.Errorf("Expected ErrMarkAlreadyPlaced, got %v", err)
	}
}

func TestMarkVisibilityDecay(t *testing.T) {
	mark := &Mark{
		CreatedAt: time.Now().Add(-15 * 24 * time.Hour), // 15 days ago
		ExpiresAt: time.Now().Add(15 * 24 * time.Hour),  // 15 days from now
	}

	visibility := mark.CurrentVisibility()
	if visibility < 0.45 || visibility > 0.55 {
		t.Errorf("Expected visibility ~0.5, got %f", visibility)
	}
}

func TestMarkVerification(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	store := NewMarkStore()

	var markerKey [32]byte
	rand.Read(markerKey[:])

	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	mark, err := store.PlaceMark(markerKey, targetKey, MarkAlly, "test", 60, priv)
	if err != nil {
		t.Fatalf("PlaceMark failed: %v", err)
	}

	if !VerifyMark(mark, pub) {
		t.Error("Expected mark verification to succeed")
	}
}

func TestMarkCategories(t *testing.T) {
	store := NewMarkStore()

	targetKey := make([]byte, 32)
	rand.Read(targetKey)

	// Place marks from different Specters.
	categories := []MarkCategory{MarkWatcher, MarkAlly, MarkAlly, MarkRival}
	for _, cat := range categories {
		var markerKey [32]byte
		rand.Read(markerKey[:])

		_, err := store.PlaceMark(markerKey, targetKey, cat, "", 60, nil)
		if err != nil {
			t.Fatalf("PlaceMark failed: %v", err)
		}
	}

	counts := store.CountMarksByCategory(targetKey)
	if counts[MarkAlly] != 2 {
		t.Errorf("Expected 2 Ally marks, got %d", counts[MarkAlly])
	}

	dominant := store.GetDominantMark(targetKey)
	if dominant != MarkAlly {
		t.Errorf("Expected dominant mark Ally, got %s", CategoryString(dominant))
	}
}

// --- Territory Tests ---

func TestTerritoryInfluence(t *testing.T) {
	territory := NewTerritory("test-1", 100.0, 100.0)

	var specterKey [32]byte
	rand.Read(specterKey[:])

	// Add some influence.
	territory.AddInfluence(specterKey, InfluenceWaveAmplified, 1.0)
	territory.AddInfluence(specterKey, InfluenceConnection, 1.0)
	territory.AddInfluence(specterKey, InfluenceMechanic, 1.0)

	territory.ComputeInfluence()

	influence := territory.GetInfluence(specterKey)
	// Expected: 8 * ln(1 + 3) = 8 * 1.386 ≈ 11.09
	if influence < 10.0 || influence > 12.0 {
		t.Errorf("Expected influence ~11.09, got %f", influence)
	}
}

func TestTerritoryControl(t *testing.T) {
	territory := NewTerritory("test-2", 50.0, 50.0)

	var controller [32]byte
	rand.Read(controller[:])

	// Add significant influence.
	for i := 0; i < 10; i++ {
		territory.AddInfluence(controller, InfluenceWaveAmplified, 1.0)
	}

	territory.ComputeInfluence()

	if territory.State != TerritoryControlled {
		t.Errorf("Expected TerritoryControlled, got %d", territory.State)
	}

	ctrl := territory.GetController()
	if ctrl == nil {
		t.Error("Expected controller, got nil")
	}
}

func TestTerritoryContested(t *testing.T) {
	territory := NewTerritory("test-3", 75.0, 75.0)

	var specter1, specter2 [32]byte
	rand.Read(specter1[:])
	rand.Read(specter2[:])

	// Add similar influence to both.
	for i := 0; i < 10; i++ {
		territory.AddInfluence(specter1, InfluenceWaveAmplified, 1.0)
		territory.AddInfluence(specter2, InfluenceWaveAmplified, 1.0)
	}

	territory.ComputeInfluence()

	if !territory.IsContested() {
		t.Error("Expected territory to be contested")
	}
}

func TestTerritoryReset(t *testing.T) {
	territory := NewTerritory("test-4", 25.0, 25.0)

	var specterKey [32]byte
	rand.Read(specterKey[:])

	territory.AddInfluence(specterKey, InfluenceWaveAmplified, 1.0)
	territory.ComputeInfluence()

	if territory.GetInfluence(specterKey) == 0 {
		t.Error("Expected non-zero influence before reset")
	}

	territory.Reset()

	if territory.GetInfluence(specterKey) != 0 {
		t.Error("Expected zero influence after reset")
	}
}

func TestTerritoryManager(t *testing.T) {
	manager := NewTerritoryManager()

	t1 := NewTerritory("t1", 0, 0)
	t2 := NewTerritory("t2", 100, 100)

	manager.AddTerritory(t1)
	manager.AddTerritory(t2)

	if manager.Count() != 2 {
		t.Errorf("Expected 2 territories, got %d", manager.Count())
	}

	retrieved := manager.GetTerritory("t1")
	if retrieved == nil {
		t.Error("Expected territory t1, got nil")
	}
}

func TestTerritoryScore(t *testing.T) {
	manager := NewTerritoryManager()

	var specterKey [32]byte
	rand.Read(specterKey[:])

	// Create controlled territory.
	t1 := NewTerritory("t1", 0, 0)
	for i := 0; i < 10; i++ {
		t1.AddInfluence(specterKey, InfluenceWaveAmplified, 1.0)
	}
	t1.ComputeInfluence()
	manager.AddTerritory(t1)

	// Create contested territory.
	var other [32]byte
	rand.Read(other[:])

	t2 := NewTerritory("t2", 100, 100)
	for i := 0; i < 10; i++ {
		t2.AddInfluence(specterKey, InfluenceWaveAmplified, 1.0)
		t2.AddInfluence(other, InfluenceWaveAmplified, 1.0)
	}
	t2.ComputeInfluence()
	manager.AddTerritory(t2)

	score := manager.ComputeTerritoryScore(specterKey)
	// Expected: 3 * ln(1 + 1 + 0.5*1) = 3 * ln(2.5) ≈ 2.75
	if score < 2.0 || score > 4.0 {
		t.Errorf("Expected territory score ~2.75, got %f", score)
	}
}

// --- Puzzle Tests ---

func TestNewPuzzle(t *testing.T) {
	var seed [32]byte
	var initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	puzzle, err := NewPuzzle(PuzzleFragment, seed, 20, PuzzleDuration30Min, initiator)
	if err != nil {
		t.Fatalf("NewPuzzle failed: %v", err)
	}

	if puzzle.Type != PuzzleFragment {
		t.Errorf("Expected PuzzleFragment, got %d", puzzle.Type)
	}

	if puzzle.State != PuzzleActive {
		t.Errorf("Expected PuzzleActive, got %d", puzzle.State)
	}
}

func TestPuzzleInvalidDuration(t *testing.T) {
	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	_, err := NewPuzzle(PuzzleFragment, seed, 20, 45*time.Minute, initiator)
	if err != ErrInvalidPuzzleDuration {
		t.Errorf("Expected ErrInvalidPuzzleDuration, got %v", err)
	}
}

func TestFragmentPuzzleSolution(t *testing.T) {
	var seed [32]byte
	var initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	// Use very low difficulty for testing.
	puzzle, _ := NewPuzzle(PuzzleFragment, seed, 8, PuzzleDuration30Min, initiator)

	generator := &FragmentGenerator{Difficulty: 8}

	// Find a valid solution by brute force.
	var solution []byte
	for nonce := uint64(0); nonce < 1000000; nonce++ {
		testSolution := make([]byte, 8)
		for i := 0; i < 8; i++ {
			testSolution[i] = byte(nonce >> (i * 8))
		}
		if generator.Verify(puzzle, testSolution) {
			solution = testSolution
			break
		}
	}

	if solution == nil {
		t.Skip("Could not find solution within search limit")
	}

	var solver [32]byte
	rand.Read(solver[:])

	err := puzzle.SubmitSolution(solver, solution, generator)
	if err != nil {
		t.Fatalf("SubmitSolution failed: %v", err)
	}

	if !puzzle.IsSolved() {
		t.Error("Expected puzzle to be solved")
	}

	if puzzle.WinnerKey == nil {
		t.Error("Expected winner key")
	}
}

func TestPuzzleExpiration(t *testing.T) {
	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	puzzle := &Puzzle{
		Type:      PuzzleFragment,
		Seed:      seed,
		CreatedAt: time.Now().Add(-1 * time.Hour),
		ExpiresAt: time.Now().Add(-30 * time.Minute),
		State:     PuzzleActive,
	}

	if !puzzle.IsExpired() {
		t.Error("Expected puzzle to be expired")
	}
}

func TestPuzzleStore(t *testing.T) {
	store := NewPuzzleStore()

	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	puzzle, _ := NewPuzzle(PuzzleFragment, seed, 20, PuzzleDuration30Min, initiator)
	store.AddPuzzle(puzzle)

	if store.Count() != 1 {
		t.Errorf("Expected 1 puzzle, got %d", store.Count())
	}

	retrieved := store.GetPuzzle(puzzle.ID)
	if retrieved == nil {
		t.Error("Expected puzzle, got nil")
	}

	active := store.GetActivePuzzles()
	if len(active) != 1 {
		t.Errorf("Expected 1 active puzzle, got %d", len(active))
	}
}

func TestComputePuzzleBonus(t *testing.T) {
	bonus := ComputePuzzleBonus(20, 10)
	// Expected: 4 * ln(1 + 1.0 * 10) = 4 * ln(11) ≈ 9.59
	if bonus < 9.0 || bonus > 10.0 {
		t.Errorf("Expected bonus ~9.59, got %f", bonus)
	}
}

func TestHasLeadingZeros(t *testing.T) {
	tests := []struct {
		hash     []byte
		bits     int
		expected bool
	}{
		{[]byte{0x00, 0x00, 0xFF}, 16, true},
		{[]byte{0x00, 0x00, 0xFF}, 17, false},
		{[]byte{0x00, 0x01, 0xFF}, 15, true},
		{[]byte{0x00, 0x01, 0xFF}, 16, false},
		{[]byte{0x0F, 0xFF, 0xFF}, 4, true},
		{[]byte{0x0F, 0xFF, 0xFF}, 5, false},
	}

	for _, tt := range tests {
		got := hasLeadingZeros(tt.hash, tt.bits)
		if got != tt.expected {
			t.Errorf("hasLeadingZeros(%x, %d) = %v, want %v",
				tt.hash, tt.bits, got, tt.expected)
		}
	}
}

func TestPuzzleTypeString(t *testing.T) {
	if PuzzleTypeString(PuzzleFragment) != "Fragment" {
		t.Error("Expected 'Fragment'")
	}
	if PuzzleTypeString(PuzzleMosaic) != "Mosaic" {
		t.Error("Expected 'Mosaic'")
	}
	if PuzzleTypeString(PuzzleCascade) != "Cascade" {
		t.Error("Expected 'Cascade'")
	}
}

func TestPuzzleStateString(t *testing.T) {
	if PuzzleStateString(PuzzleActive) != "Active" {
		t.Error("Expected 'Active'")
	}
	if PuzzleStateString(PuzzleSolved) != "Solved" {
		t.Error("Expected 'Solved'")
	}
}

// --- Hunt Tests ---

func TestNewHunt(t *testing.T) {
	var seed [32]byte
	var initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	hunt, err := NewHunt("Test Hunt", seed, initiator, HuntDuration60Min, 10, HuntMinResonance)
	if err != nil {
		t.Fatalf("NewHunt failed: %v", err)
	}

	if hunt.Theme != "Test Hunt" {
		t.Errorf("Expected theme 'Test Hunt', got '%s'", hunt.Theme)
	}

	if hunt.State != HuntActive {
		t.Errorf("Expected HuntActive, got %d", hunt.State)
	}

	if len(hunt.Fragments) != 10 {
		t.Errorf("Expected 10 fragments, got %d", len(hunt.Fragments))
	}
}

func TestHuntInvalidDuration(t *testing.T) {
	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	_, err := NewHunt("Test", seed, initiator, 45*time.Minute, 10, HuntMinResonance)
	if err != ErrInvalidHuntDuration {
		t.Errorf("Expected ErrInvalidHuntDuration, got %v", err)
	}
}

func TestHuntInvalidFragmentCount(t *testing.T) {
	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	_, err := NewHunt("Test", seed, initiator, HuntDuration60Min, 3, HuntMinResonance) // Below minimum of 5.
	if err != ErrInvalidFragmentCount {
		t.Errorf("Expected ErrInvalidFragmentCount, got %v", err)
	}

	_, err = NewHunt("Test", seed, initiator, HuntDuration60Min, 25, HuntMinResonance) // Above maximum of 20.
	if err != ErrInvalidFragmentCount {
		t.Errorf("Expected ErrInvalidFragmentCount, got %v", err)
	}
}

func TestHuntInsufficientResonance(t *testing.T) {
	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	// Resonance below minimum (75).
	_, err := NewHunt("Test", seed, initiator, HuntDuration60Min, 10, HuntMinResonance-1)
	if err != ErrHuntInsufficientRes {
		t.Errorf("Expected ErrHuntInsufficientRes, got %v", err)
	}

	// Exactly at minimum should succeed.
	_, err = NewHunt("Test", seed, initiator, HuntDuration60Min, 10, HuntMinResonance)
	if err != nil {
		t.Errorf("Expected success at minimum resonance, got %v", err)
	}
}

func TestHuntFragmentGeneration(t *testing.T) {
	var seed [32]byte
	var initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiator, HuntDuration30Min, 5, HuntMinResonance)

	// Verify all fragments have unique location hashes.
	locations := make(map[[32]byte]bool)
	for _, f := range hunt.Fragments {
		if locations[f.LocationHash] {
			t.Error("Duplicate fragment location hash found")
		}
		locations[f.LocationHash] = true

		// Verify each fragment has clues.
		if len(f.Clues) != 4 {
			t.Errorf("Expected 4 clues, got %d", len(f.Clues))
		}
	}
}

func TestHuntClaimFragment(t *testing.T) {
	var seed, initiator, claimer [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])
	rand.Read(claimer[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiator, HuntDuration60Min, 5, HuntMinResonance)

	// Create a valid proximity proof.
	proof := ProximityProof{
		ClaimerPeerID:  "test-peer-id",
		ConnectedPeers: []string{"peer1", "peer2"},
		HopDistances:   []int{2}, // Within 3 hops.
	}

	err := hunt.ClaimFragment(0, claimer, proof)
	if err != nil {
		t.Fatalf("ClaimFragment failed: %v", err)
	}

	fragment := hunt.GetFragment(0)
	if !fragment.Claimed {
		t.Error("Expected fragment to be claimed")
	}

	if fragment.ClaimerKey == nil {
		t.Error("Expected claimer key to be set")
	}
}

func TestHuntClaimAlreadyClaimed(t *testing.T) {
	var seed, initiator, claimer1, claimer2 [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])
	rand.Read(claimer1[:])
	rand.Read(claimer2[:])

	hunt, _ := NewHunt("Test", seed, initiator, HuntDuration60Min, 5, HuntMinResonance)

	proof := ProximityProof{HopDistances: []int{1}}

	// First claim should succeed.
	hunt.ClaimFragment(0, claimer1, proof)

	// Second claim should fail.
	err := hunt.ClaimFragment(0, claimer2, proof)
	if err != ErrFragmentClaimed {
		t.Errorf("Expected ErrFragmentClaimed, got %v", err)
	}
}

func TestHuntNotInProximity(t *testing.T) {
	var seed, initiator, claimer [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])
	rand.Read(claimer[:])

	hunt, _ := NewHunt("Test", seed, initiator, HuntDuration60Min, 5, HuntMinResonance)

	// Proof with too many hops.
	proof := ProximityProof{HopDistances: []int{5}} // More than 3 hops.

	err := hunt.ClaimFragment(0, claimer, proof)
	if err != ErrNotInProximity {
		t.Errorf("Expected ErrNotInProximity, got %v", err)
	}
}

func TestHuntCompletion(t *testing.T) {
	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	hunt, _ := NewHunt("Test", seed, initiator, HuntDuration60Min, 5, HuntMinResonance)

	proof := ProximityProof{HopDistances: []int{1}}

	// Claim all fragments.
	for i := 0; i < 5; i++ {
		var claimer [32]byte
		rand.Read(claimer[:])
		hunt.ClaimFragment(i, claimer, proof)
	}

	if !hunt.IsCompleted() {
		t.Error("Expected hunt to be completed")
	}

	if hunt.State != HuntCompleted {
		t.Errorf("Expected HuntCompleted, got %d", hunt.State)
	}
}

func TestHuntExpiration(t *testing.T) {
	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	hunt := &Hunt{
		Seed:          seed,
		InitiatorKey:  initiator,
		CreatedAt:     time.Now().Add(-2 * time.Hour),
		ExpiresAt:     time.Now().Add(-1 * time.Hour), // Expired.
		State:         HuntActive,
		FragmentCount: 5,
	}

	if !hunt.IsExpired() {
		t.Error("Expected hunt to be expired")
	}
}

func TestHuntRevealClues(t *testing.T) {
	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	// Create a hunt that started 25 minutes ago.
	hunt, _ := NewHunt("Test", seed, initiator, HuntDuration60Min, 5, HuntMinResonance)
	hunt.CreatedAt = time.Now().Add(-25 * time.Minute)

	hunt.RevealClues()

	// After 25 minutes (2 intervals of 10 minutes), should have 2 clues.
	clues := hunt.GetVisibleClues(0)
	if len(clues) != 2 {
		t.Errorf("Expected 2 visible clues, got %d", len(clues))
	}
}

func TestHuntLeaderboard(t *testing.T) {
	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	hunt, _ := NewHunt("Test", seed, initiator, HuntDuration60Min, 5, HuntMinResonance)

	var claimer1, claimer2 [32]byte
	rand.Read(claimer1[:])
	rand.Read(claimer2[:])

	proof := ProximityProof{HopDistances: []int{1}}

	// Claimer1 gets 3 fragments.
	hunt.ClaimFragment(0, claimer1, proof)
	hunt.ClaimFragment(1, claimer1, proof)
	hunt.ClaimFragment(2, claimer1, proof)

	// Claimer2 gets 2 fragments.
	hunt.ClaimFragment(3, claimer2, proof)
	hunt.ClaimFragment(4, claimer2, proof)

	leaderboard := hunt.GetLeaderboard()

	if len(leaderboard) != 2 {
		t.Errorf("Expected 2 participants, got %d", len(leaderboard))
	}

	if leaderboard[0].Claims != 3 {
		t.Errorf("Expected first place to have 3 claims, got %d", leaderboard[0].Claims)
	}

	if leaderboard[1].Claims != 2 {
		t.Errorf("Expected second place to have 2 claims, got %d", leaderboard[1].Claims)
	}
}

func TestComputeHuntBonus(t *testing.T) {
	bonus := ComputeHuntBonus(5, 10)
	// Expected: 5.0 * (5/10) * 5 = 5.0 * 0.5 * 5 = 12.5
	if bonus != 12.5 {
		t.Errorf("Expected bonus 12.5, got %f", bonus)
	}

	// Zero total should return 0.
	bonus = ComputeHuntBonus(5, 0)
	if bonus != 0 {
		t.Errorf("Expected bonus 0, got %f", bonus)
	}
}

func TestHuntStore(t *testing.T) {
	store := NewHuntStore()

	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	hunt, _ := NewHunt("Test Hunt", seed, initiator, HuntDuration60Min, 5, HuntMinResonance)
	store.AddHunt(hunt)

	if store.Count() != 1 {
		t.Errorf("Expected 1 hunt, got %d", store.Count())
	}

	retrieved := store.GetHunt(hunt.ID)
	if retrieved == nil {
		t.Error("Expected hunt, got nil")
	}

	active := store.GetActiveHunts()
	if len(active) != 1 {
		t.Errorf("Expected 1 active hunt, got %d", len(active))
	}
}

func TestHuntStoreUpdateStates(t *testing.T) {
	store := NewHuntStore()

	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	// Create an expired hunt.
	hunt := &Hunt{
		Seed:          seed,
		InitiatorKey:  initiator,
		CreatedAt:     time.Now().Add(-2 * time.Hour),
		ExpiresAt:     time.Now().Add(-1 * time.Hour),
		State:         HuntActive,
		FragmentCount: 5,
	}
	rand.Read(hunt.ID[:])
	store.hunts[hunt.ID] = hunt
	store.active = append(store.active, hunt)

	store.UpdateHuntStates()

	if hunt.State != HuntExpired {
		t.Errorf("Expected HuntExpired, got %d", hunt.State)
	}

	active := store.GetActiveHunts()
	if len(active) != 0 {
		t.Errorf("Expected 0 active hunts, got %d", len(active))
	}
}

func TestHuntStateString(t *testing.T) {
	if HuntStateString(HuntActive) != "Active" {
		t.Error("Expected 'Active'")
	}
	if HuntStateString(HuntCompleted) != "Completed" {
		t.Error("Expected 'Completed'")
	}
	if HuntStateString(HuntExpired) != "Expired" {
		t.Error("Expected 'Expired'")
	}
}

// --- Oracle Pool Tests ---

func TestNewOraclePool(t *testing.T) {
	var creator [32]byte
	rand.Read(creator[:])

	deadline := time.Now().Add(1 * time.Hour)
	resolution := time.Now().Add(2 * time.Hour)

	pool, err := NewOraclePool(
		"Will daily message count exceed 1000?",
		OraclePredictionBoolean,
		"gossip_message_count",
		creator,
		deadline,
		resolution,
	)
	if err != nil {
		t.Fatalf("NewOraclePool failed: %v", err)
	}

	if pool.State != OraclePoolOpen {
		t.Errorf("Expected OraclePoolOpen, got %d", pool.State)
	}

	if !pool.IsOpen() {
		t.Error("Expected pool to be open")
	}
}

func TestOraclePoolQuestionTooLong(t *testing.T) {
	var creator [32]byte
	rand.Read(creator[:])

	longQuestion := make([]byte, OracleMaxQuestionLength+1)
	for i := range longQuestion {
		longQuestion[i] = 'a'
	}

	_, err := NewOraclePool(
		string(longQuestion),
		OraclePredictionBoolean,
		"test",
		creator,
		time.Now().Add(1*time.Hour),
		time.Now().Add(2*time.Hour),
	)

	if err != ErrOracleQuestionTooLong {
		t.Errorf("Expected ErrOracleQuestionTooLong, got %v", err)
	}
}

func TestOracleCommitmentAndReveal(t *testing.T) {
	var creator, predictor [32]byte
	var nonce [32]byte
	rand.Read(creator[:])
	rand.Read(predictor[:])
	rand.Read(nonce[:])

	deadline := time.Now().Add(-1 * time.Minute) // Already past.
	resolution := time.Now().Add(1 * time.Hour)

	pool, _ := NewOraclePool(
		"Test question?",
		OraclePredictionBoolean,
		"test",
		creator,
		time.Now().Add(1*time.Hour), // Temporarily set future deadline.
		resolution,
	)

	// Compute commitment.
	value := 1.0 // Predicting true.
	commitment := ComputeCommitmentHash(value, nonce)

	// Submit commitment.
	err := pool.SubmitCommitment(predictor, commitment)
	if err != nil {
		t.Fatalf("SubmitCommitment failed: %v", err)
	}

	// Verify commitment was stored.
	stored := pool.GetCommitment(predictor)
	if stored == nil {
		t.Fatal("Expected stored commitment")
	}

	// Move to reveal period.
	pool.Deadline = deadline

	// Reveal prediction.
	err = pool.RevealPrediction(predictor, value, nonce)
	if err != nil {
		t.Fatalf("RevealPrediction failed: %v", err)
	}

	pred := pool.GetPrediction(predictor)
	if pred == nil {
		t.Fatal("Expected prediction")
	}
	if pred.Value != value {
		t.Errorf("Expected value %f, got %f", value, pred.Value)
	}
}

func TestOracleInvalidReveal(t *testing.T) {
	var creator, predictor [32]byte
	var nonce, wrongNonce [32]byte
	rand.Read(creator[:])
	rand.Read(predictor[:])
	rand.Read(nonce[:])
	rand.Read(wrongNonce[:])

	pool, _ := NewOraclePool(
		"Test?",
		OraclePredictionBoolean,
		"test",
		creator,
		time.Now().Add(-1*time.Minute),
		time.Now().Add(1*time.Hour),
	)

	// Make deadline in the past so we can reveal.
	pool.Deadline = time.Now().Add(-1 * time.Minute)

	value := 1.0
	commitment := ComputeCommitmentHash(value, nonce)
	pool.commitments[keyToHex(predictor[:])] = &Commitment{
		SpecterKey: predictor,
		Hash:       commitment,
	}

	// Try to reveal with wrong nonce.
	err := pool.RevealPrediction(predictor, value, wrongNonce)
	if err != ErrOracleInvalidReveal {
		t.Errorf("Expected ErrOracleInvalidReveal, got %v", err)
	}
}

func TestOraclePoolResolution(t *testing.T) {
	var creator [32]byte
	rand.Read(creator[:])

	pool, _ := NewOraclePool(
		"Numeric prediction test",
		OraclePredictionNumeric,
		"test_metric",
		creator,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-1*time.Hour),
	)

	// Add some predictions.
	predictions := []float64{100, 105, 110, 95, 200}
	for i, predValue := range predictions {
		var predictor [32]byte
		rand.Read(predictor[:])

		pool.predictions[keyToHex(predictor[:])] = &Prediction{
			SpecterKey: predictor,
			Value:      predValue,
		}
		_ = i
	}

	// Resolve with actual outcome.
	outcome := 102.0
	err := pool.Resolve(outcome)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if !pool.IsResolved() {
		t.Error("Expected pool to be resolved")
	}

	// Check winners.
	winners := pool.GetWinners()
	if len(winners) == 0 {
		t.Error("Expected at least one winner")
	}

	// Top 25% of 5 = 2 winners (ceil).
	if len(winners) < 1 || len(winners) > 2 {
		t.Errorf("Expected 1-2 winners, got %d", len(winners))
	}
}

func TestOracleBooleanResolution(t *testing.T) {
	var creator [32]byte
	rand.Read(creator[:])

	pool, _ := NewOraclePool(
		"Will it happen?",
		OraclePredictionBoolean,
		"test",
		creator,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-1*time.Hour),
	)

	// Add predictions: 3 true, 2 false.
	for i := 0; i < 5; i++ {
		var predictor [32]byte
		rand.Read(predictor[:])

		value := 1.0 // True.
		if i >= 3 {
			value = 0.0 // False.
		}

		pool.predictions[keyToHex(predictor[:])] = &Prediction{
			SpecterKey: predictor,
			Value:      value,
		}
	}

	// Outcome is true.
	pool.Resolve(1.0)

	// All true predictors should have accuracy 1.0.
	allPreds := pool.GetAllPredictions()
	trueCount := 0
	for _, pred := range allPreds {
		if pred.Value >= 0.5 && pred.Accuracy == 1.0 {
			trueCount++
		}
	}

	if trueCount != 3 {
		t.Errorf("Expected 3 correct predictions, got %d", trueCount)
	}
}

func TestComputeOracleBonus(t *testing.T) {
	// Test with 10 participants, rank 1.
	bonus := ComputeOracleBonus(10, 1)
	// Expected: 3 * ln(1 + 10/1) = 3 * ln(11) ≈ 7.19
	if bonus < 7.0 || bonus > 7.5 {
		t.Errorf("Expected bonus ~7.19, got %f", bonus)
	}

	// Test with 10 participants, rank 5.
	bonus = ComputeOracleBonus(10, 5)
	// Expected: 3 * ln(1 + 10/5) = 3 * ln(3) ≈ 3.30
	if bonus < 3.0 || bonus > 3.5 {
		t.Errorf("Expected bonus ~3.30, got %f", bonus)
	}
}

func TestOraclePoolStore(t *testing.T) {
	store := NewOraclePoolStore()

	var creator [32]byte
	rand.Read(creator[:])

	pool, _ := NewOraclePool(
		"Test pool",
		OraclePredictionBoolean,
		"test",
		creator,
		time.Now().Add(1*time.Hour),
		time.Now().Add(2*time.Hour),
	)

	store.AddPool(pool)

	if store.Count() != 1 {
		t.Errorf("Expected 1 pool, got %d", store.Count())
	}

	retrieved := store.GetPool(pool.ID)
	if retrieved == nil {
		t.Error("Expected pool, got nil")
	}

	open := store.GetOpenPools()
	if len(open) != 1 {
		t.Errorf("Expected 1 open pool, got %d", len(open))
	}
}

func TestOraclePoolStateUpdate(t *testing.T) {
	var creator [32]byte
	rand.Read(creator[:])

	pool, _ := NewOraclePool(
		"Test",
		OraclePredictionBoolean,
		"test",
		creator,
		time.Now().Add(-1*time.Minute), // Past deadline.
		time.Now().Add(1*time.Hour),
	)

	pool.UpdateState()

	if pool.State != OraclePoolPending {
		t.Errorf("Expected OraclePoolPending, got %d", pool.State)
	}
}

func TestOraclePoolStateStrings(t *testing.T) {
	if OraclePoolStateString(OraclePoolOpen) != "Open" {
		t.Error("Expected 'Open'")
	}
	if OraclePoolStateString(OraclePoolResolved) != "Resolved" {
		t.Error("Expected 'Resolved'")
	}
	if OraclePredictionTypeString(OraclePredictionBoolean) != "Boolean" {
		t.Error("Expected 'Boolean'")
	}
	if OraclePredictionTypeString(OraclePredictionNumeric) != "Numeric" {
		t.Error("Expected 'Numeric'")
	}
}

// --- Sigil Forge Tests ---

func TestNewSigilForge(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	forge, err := NewSigilForge(
		ForgeSigilArt,
		"Create a sigil representing hope",
		initiator,
		ForgeDuration30Min, ForgeMinResonance,
	)
	if err != nil {
		t.Fatalf("NewSigilForge failed: %v", err)
	}

	if forge.State != ForgeActive {
		t.Errorf("Expected ForgeActive, got %d", forge.State)
	}

	if !forge.IsActive() {
		t.Error("Expected forge to be active")
	}

	if forge.Type != ForgeSigilArt {
		t.Errorf("Expected ForgeSigilArt, got %d", forge.Type)
	}
}

func TestForgeInvalidType(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	_, err := NewSigilForge(
		ForgeType(99),
		"Invalid forge",
		initiator,
		ForgeDuration30Min,
		ForgeMinResonance,
	)
	if err != ErrForgeInvalidType {
		t.Errorf("Expected ErrForgeInvalidType, got %v", err)
	}
}

func TestForgePromptTooLong(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	longPrompt := make([]byte, ForgeMaxPromptLength+1)
	for i := range longPrompt {
		longPrompt[i] = 'a'
	}

	_, err := NewSigilForge(
		ForgeSigilArt,
		string(longPrompt),
		initiator,
		ForgeDuration30Min,
		ForgeMinResonance,
	)
	if err != ErrForgePromptTooLong {
		t.Errorf("Expected ErrForgePromptTooLong, got %v", err)
	}
}

func TestForgeInvalidDuration(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	_, err := NewSigilForge(
		ForgeSigilArt,
		"Test",
		initiator,
		15*time.Minute, // Invalid duration.
		ForgeMinResonance,
	)
	if err != ErrForgeInvalidDuration {
		t.Errorf("Expected ErrForgeInvalidDuration, got %v", err)
	}
}

func TestForgeInsufficientResonance(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	// Resonance below minimum (50).
	_, err := NewSigilForge(
		ForgeSigilArt,
		"Test",
		initiator,
		ForgeDuration30Min,
		ForgeMinResonance-1,
	)
	if err != ErrForgeInsufficientResonance {
		t.Errorf("Expected ErrForgeInsufficientResonance, got %v", err)
	}

	// Exactly at minimum should succeed.
	_, err = NewSigilForge(
		ForgeSigilArt,
		"Test",
		initiator,
		ForgeDuration30Min,
		ForgeMinResonance,
	)
	if err != nil {
		t.Errorf("Expected success at minimum resonance, got %v", err)
	}
}

func TestForgeSubmitEntry(t *testing.T) {
	var initiator, specter [32]byte
	rand.Read(initiator[:])
	rand.Read(specter[:])

	forge, _ := NewSigilForge(
		ForgeMicroFiction,
		"Write a story about shadows",
		initiator,
		ForgeDuration60Min, ForgeMinResonance,
	)

	content := []byte("In the darkness, a shadow found its light...")
	entry, err := forge.SubmitEntry(specter, content, [32]byte{})
	if err != nil {
		t.Fatalf("SubmitEntry failed: %v", err)
	}

	if entry.SpecterKey != specter {
		t.Error("Entry specter key mismatch")
	}

	if forge.EntryCount() != 1 {
		t.Errorf("Expected 1 entry, got %d", forge.EntryCount())
	}
}

func TestForgeDuplicateEntry(t *testing.T) {
	var initiator, specter [32]byte
	rand.Read(initiator[:])
	rand.Read(specter[:])

	forge, _ := NewSigilForge(
		ForgeSigilArt,
		"Test",
		initiator,
		ForgeDuration30Min, ForgeMinResonance,
	)

	forge.SubmitEntry(specter, []byte("First entry"), [32]byte{})

	_, err := forge.SubmitEntry(specter, []byte("Second entry"), [32]byte{})
	if err != ErrForgeDuplicateEntry {
		t.Errorf("Expected ErrForgeDuplicateEntry, got %v", err)
	}
}

func TestForgeEntryTooLarge(t *testing.T) {
	var initiator, specter [32]byte
	rand.Read(initiator[:])
	rand.Read(specter[:])

	forge, _ := NewSigilForge(
		ForgeMicroFiction,
		"Test",
		initiator,
		ForgeDuration30Min, ForgeMinResonance,
	)

	largeContent := make([]byte, ForgeMaxEntrySize+1)
	_, err := forge.SubmitEntry(specter, largeContent, [32]byte{})
	if err != ErrForgeEntryTooLarge {
		t.Errorf("Expected ErrForgeEntryTooLarge, got %v", err)
	}
}

func TestForgeAmplification(t *testing.T) {
	var initiator, specter1, specter2, amplifier [32]byte
	rand.Read(initiator[:])
	rand.Read(specter1[:])
	rand.Read(specter2[:])
	rand.Read(amplifier[:])

	forge, _ := NewSigilForge(
		ForgeSigilArt,
		"Test",
		initiator,
		ForgeDuration30Min, ForgeMinResonance,
	)

	entry1, _ := forge.SubmitEntry(specter1, []byte("Art 1"), [32]byte{})
	forge.SubmitEntry(specter2, []byte("Art 2"), [32]byte{})

	// Amplify first entry with Resonance 50.
	err := forge.AmplifyEntry(entry1.ID, amplifier, 50.0)
	if err != nil {
		t.Fatalf("AmplifyEntry failed: %v", err)
	}

	// Check amplification was recorded.
	retrieved := forge.GetEntry(entry1.ID)
	// Expected weight: 1.0 + (50/100) = 1.5
	if retrieved.Amplifications < 1.4 || retrieved.Amplifications > 1.6 {
		t.Errorf("Expected amplification ~1.5, got %f", retrieved.Amplifications)
	}

	// Duplicate amplification should be ignored.
	forge.AmplifyEntry(entry1.ID, amplifier, 50.0)
	if retrieved.Amplifications < 1.4 || retrieved.Amplifications > 1.6 {
		t.Errorf("Duplicate amplification should be ignored")
	}
}

func TestForgeEvaluation(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	forge, _ := NewSigilForge(
		ForgeSigilArt,
		"Test",
		initiator,
		ForgeDuration30Min, ForgeMinResonance,
	)

	// Add entries with different amplifications.
	for i := 0; i < 5; i++ {
		var specter [32]byte
		rand.Read(specter[:])

		entry, _ := forge.SubmitEntry(specter, []byte("Entry"), [32]byte{})

		// Add amplifications (more for earlier entries).
		for j := 0; j < 5-i; j++ {
			var amp [32]byte
			rand.Read(amp[:])
			forge.AmplifyEntry(entry.ID, amp, 25.0)
		}
	}

	// Move to evaluating state.
	forge.State = ForgeEvaluating

	err := forge.Evaluate()
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if !forge.IsCompleted() {
		t.Error("Expected forge to be completed")
	}

	winner := forge.GetWinner()
	if winner == nil {
		t.Fatal("Expected winner")
	}

	if winner.Rank != 1 {
		t.Errorf("Expected winner rank 1, got %d", winner.Rank)
	}
}

func TestForgeRemixChain(t *testing.T) {
	var initiator, specter1, specter2, specter3 [32]byte
	rand.Read(initiator[:])
	rand.Read(specter1[:])
	rand.Read(specter2[:])
	rand.Read(specter3[:])

	forge, _ := NewSigilForge(
		ForgeRemixChain,
		"Create a remix chain",
		initiator,
		ForgeDuration60Min, ForgeMinResonance,
	)

	// First entry (root).
	root, _ := forge.SubmitEntry(specter1, []byte("Original"), [32]byte{})

	// Remix of root.
	remix1, _ := forge.SubmitEntry(specter2, []byte("Remix 1"), root.ID)

	// Remix of remix1.
	remix2, _ := forge.SubmitEntry(specter3, []byte("Remix 2"), remix1.ID)

	// Add amplifications.
	for i := 0; i < 3; i++ {
		var amp [32]byte
		rand.Read(amp[:])
		forge.AmplifyEntry(root.ID, amp, 50.0)
		forge.AmplifyEntry(remix1.ID, amp, 50.0)
		forge.AmplifyEntry(remix2.ID, amp, 50.0)
	}

	// Move to evaluating state.
	forge.State = ForgeEvaluating
	forge.Evaluate()

	// In remix chains, scores should be shared.
	// All three entries should have similar scores.
	leaderboard := forge.GetLeaderboard()
	if len(leaderboard) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(leaderboard))
	}
}

func TestForgeNoEntries(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	forge, _ := NewSigilForge(
		ForgeSigilArt,
		"Test",
		initiator,
		ForgeDuration30Min, ForgeMinResonance,
	)

	forge.State = ForgeEvaluating

	err := forge.Evaluate()
	if err != ErrForgeNoEntries {
		t.Errorf("Expected ErrForgeNoEntries, got %v", err)
	}

	if forge.State != ForgeExpired {
		t.Errorf("Expected ForgeExpired state, got %d", forge.State)
	}
}

func TestComputeForgeBonus(t *testing.T) {
	// Winner bonus: 4 * ln(1 + 10) = 4 * ln(11) ≈ 9.59
	winnerBonus := ComputeForgeWinnerBonus(10.0)
	if winnerBonus < 9.5 || winnerBonus > 9.7 {
		t.Errorf("Expected winner bonus ~9.59, got %f", winnerBonus)
	}

	// Participation bonus: 2 * ln(1 + 5) = 2 * ln(6) ≈ 3.58
	partBonus := ComputeForgeParticipationBonus(5.0)
	if partBonus < 3.5 || partBonus > 3.7 {
		t.Errorf("Expected participation bonus ~3.58, got %f", partBonus)
	}
}

func TestForgeStore(t *testing.T) {
	store := NewForgeStore()

	var initiator [32]byte
	rand.Read(initiator[:])

	forge, _ := NewSigilForge(
		ForgeSigilArt,
		"Test",
		initiator,
		ForgeDuration30Min, ForgeMinResonance,
	)

	store.AddForge(forge)

	if store.Count() != 1 {
		t.Errorf("Expected 1 forge, got %d", store.Count())
	}

	retrieved := store.GetForge(forge.ID)
	if retrieved == nil {
		t.Error("Expected forge, got nil")
	}

	active := store.GetActiveForges()
	if len(active) != 1 {
		t.Errorf("Expected 1 active forge, got %d", len(active))
	}

	byType := store.GetForgesByType(ForgeSigilArt)
	if len(byType) != 1 {
		t.Errorf("Expected 1 forge of type, got %d", len(byType))
	}
}

func TestForgeStateUpdate(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	forge, _ := NewSigilForge(
		ForgeSigilArt,
		"Test",
		initiator,
		ForgeDuration30Min, ForgeMinResonance,
	)

	// Set deadline to past.
	forge.Deadline = time.Now().Add(-1 * time.Minute)

	forge.UpdateState()

	// No entries, so should be expired.
	if forge.State != ForgeExpired {
		t.Errorf("Expected ForgeExpired, got %d", forge.State)
	}
}

func TestForgeStateStrings(t *testing.T) {
	if ForgeTypeString(ForgeSigilArt) != "Sigil Art" {
		t.Error("Expected 'Sigil Art'")
	}
	if ForgeTypeString(ForgeMicroFiction) != "Micro Fiction" {
		t.Error("Expected 'Micro Fiction'")
	}
	if ForgeTypeString(ForgeRemixChain) != "Remix Chain" {
		t.Error("Expected 'Remix Chain'")
	}
	if ForgeStateString(ForgeActive) != "Active" {
		t.Error("Expected 'Active'")
	}
	if ForgeStateString(ForgeCompleted) != "Completed" {
		t.Error("Expected 'Completed'")
	}
}

// --- Shadow Play Tests ---

func TestNewShadowPlay(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	game, err := NewShadowPlay(initiator, ShadowPlayDuration30Min, 7)
	if err != nil {
		t.Fatalf("NewShadowPlay failed: %v", err)
	}

	if game.State != ShadowPlayWaiting {
		t.Errorf("Expected ShadowPlayWaiting, got %d", game.State)
	}

	if !game.IsWaiting() {
		t.Error("Expected game to be waiting")
	}

	if game.MaxPlayers != 7 {
		t.Errorf("Expected 7 max players, got %d", game.MaxPlayers)
	}
}

func TestShadowPlayInvalidSize(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	// Too few players.
	_, err := NewShadowPlay(initiator, ShadowPlayDuration30Min, 3)
	if err != ErrShadowPlayInvalidSize {
		t.Errorf("Expected ErrShadowPlayInvalidSize, got %v", err)
	}

	// Too many players.
	_, err = NewShadowPlay(initiator, ShadowPlayDuration30Min, 20)
	if err != ErrShadowPlayInvalidSize {
		t.Errorf("Expected ErrShadowPlayInvalidSize, got %v", err)
	}
}

func TestShadowPlayInvalidDuration(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	_, err := NewShadowPlay(initiator, 15*time.Minute, 7)
	if err != ErrShadowPlayInvalidDuration {
		t.Errorf("Expected ErrShadowPlayInvalidDuration, got %v", err)
	}
}

func TestShadowPlayJoin(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)

	// Join 5 players.
	for i := 0; i < 5; i++ {
		var player [32]byte
		rand.Read(player[:])
		err := game.Join(player)
		if err != nil {
			t.Fatalf("Join failed: %v", err)
		}
	}

	if game.PlayerCount() != 5 {
		t.Errorf("Expected 5 players, got %d", game.PlayerCount())
	}

	// Try to join when full.
	var extraPlayer [32]byte
	rand.Read(extraPlayer[:])
	err := game.Join(extraPlayer)
	if err != ErrShadowPlayFull {
		t.Errorf("Expected ErrShadowPlayFull, got %v", err)
	}
}

func TestShadowPlayStart(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)

	// Join 5 players.
	for i := 0; i < 5; i++ {
		var player [32]byte
		rand.Read(player[:])
		game.Join(player)
	}

	err := game.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !game.IsActive() {
		t.Error("Expected game to be active")
	}

	if game.CurrentRound != 1 {
		t.Errorf("Expected round 1, got %d", game.CurrentRound)
	}
}

func TestShadowPlayRoleAssignment(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 7)

	// Join 7 players.
	var players [][32]byte
	for i := 0; i < 7; i++ {
		var player [32]byte
		rand.Read(player[:])
		players = append(players, player)
		game.Join(player)
	}

	game.Start()

	// Count roles.
	var echoCount, shadeCount int
	for _, player := range players {
		role, _ := game.DeriveRole(player)
		if role == RoleEcho {
			echoCount++
		} else {
			shadeCount++
		}
	}

	// With 7 players, should have 1 Shade.
	if shadeCount != 1 {
		t.Errorf("Expected 1 Shade, got %d", shadeCount)
	}
	if echoCount != 6 {
		t.Errorf("Expected 6 Echoes, got %d", echoCount)
	}
}

func TestShadowPlayVoting(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)

	// Join 5 players.
	var players [][32]byte
	for i := 0; i < 5; i++ {
		var player [32]byte
		rand.Read(player[:])
		players = append(players, player)
		game.Join(player)
	}

	game.Start()
	game.StartVoting()

	if game.State != ShadowPlayVoting {
		t.Errorf("Expected ShadowPlayVoting, got %d", game.State)
	}

	// All players vote for player 0.
	for i := 1; i < 5; i++ {
		err := game.Vote(players[i], players[0])
		if err != nil {
			t.Fatalf("Vote failed: %v", err)
		}
	}

	// Tally votes.
	eliminated, _, err := game.TallyVotes()
	if err != nil {
		t.Fatalf("TallyVotes failed: %v", err)
	}

	if eliminated == nil {
		t.Fatal("Expected someone to be eliminated")
	}

	if eliminated.SpecterKey != players[0] {
		t.Error("Wrong player eliminated")
	}
}

func TestShadowPlayEchoesWin(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)

	// Join 5 players.
	var players [][32]byte
	for i := 0; i < 5; i++ {
		var player [32]byte
		rand.Read(player[:])
		players = append(players, player)
		game.Join(player)
	}

	game.Start()

	// Find the Shade.
	shades := game.GetShades()
	if len(shades) != 1 {
		t.Fatalf("Expected 1 Shade, got %d", len(shades))
	}
	shade := shades[0]

	// Vote to eliminate the Shade.
	game.StartVoting()
	for _, player := range players {
		if player != shade.SpecterKey {
			game.Vote(player, shade.SpecterKey)
		}
	}

	_, gameOver, _ := game.TallyVotes()

	if !gameOver {
		t.Error("Expected game to be over")
	}

	if game.State != ShadowPlayEchoesWin {
		t.Errorf("Expected ShadowPlayEchoesWin, got %d", game.State)
	}
}

func TestShadowPlayShadesWin(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)

	// Join 5 players.
	var players [][32]byte
	for i := 0; i < 5; i++ {
		var player [32]byte
		rand.Read(player[:])
		players = append(players, player)
		game.Join(player)
	}

	game.Start()

	// Find Echoes.
	var echoes [][32]byte
	for _, player := range players {
		p := game.GetPlayer(player)
		if p.Role == RoleEcho {
			echoes = append(echoes, player)
		}
	}

	// Eliminate Echoes until Shades win.
	// With 5 players, 1 Shade, 4 Echoes.
	// Shades win when Shades >= Echoes.
	// Need to eliminate 3 Echoes (leaving 1 Echo, 1 Shade).
	for i := 0; i < 3; i++ {
		game.StartVoting()
		target := echoes[i]

		// All active players vote for target.
		for _, player := range players {
			p := game.GetPlayer(player)
			if !p.IsEliminated {
				game.Vote(player, target)
			}
		}

		_, gameOver, _ := game.TallyVotes()
		if i < 2 && gameOver {
			t.Fatalf("Game ended too early at elimination %d", i)
		}
		if i == 2 && !gameOver {
			t.Error("Game should be over after 3rd elimination")
		}
	}

	if game.State != ShadowPlayShadesWin {
		t.Errorf("Expected ShadowPlayShadesWin, got %d", game.State)
	}
}

func TestComputeShadowPlayBonus(t *testing.T) {
	// Win bonus: 5 * ln(1 + 7) = 5 * ln(8) ≈ 10.4
	winBonus := ComputeShadowPlayWinBonus(7)
	if winBonus < 10.0 || winBonus > 11.0 {
		t.Errorf("Expected win bonus ~10.4, got %f", winBonus)
	}

	// Lose bonus: 2 * ln(1 + 7) = 2 * ln(8) ≈ 4.16
	loseBonus := ComputeShadowPlayLoseBonus(7)
	if loseBonus < 4.0 || loseBonus > 4.5 {
		t.Errorf("Expected lose bonus ~4.16, got %f", loseBonus)
	}
}

func TestShadowPlayStore(t *testing.T) {
	store := NewShadowPlayStore()

	var initiator [32]byte
	rand.Read(initiator[:])

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)
	store.AddGame(game)

	if store.Count() != 1 {
		t.Errorf("Expected 1 game, got %d", store.Count())
	}

	retrieved := store.GetGame(game.ID)
	if retrieved == nil {
		t.Error("Expected game, got nil")
	}

	waiting := store.GetWaitingGames()
	if len(waiting) != 1 {
		t.Errorf("Expected 1 waiting game, got %d", len(waiting))
	}
}

func TestShadowPlayStateStrings(t *testing.T) {
	if ShadowPlayStateString(ShadowPlayWaiting) != "Waiting" {
		t.Error("Expected 'Waiting'")
	}
	if ShadowPlayStateString(ShadowPlayActive) != "Active" {
		t.Error("Expected 'Active'")
	}
	if ShadowPlayStateString(ShadowPlayEchoesWin) != "Echoes Win" {
		t.Error("Expected 'Echoes Win'")
	}
	if PlayerRoleString(RoleEcho) != "Echo" {
		t.Error("Expected 'Echo'")
	}
	if PlayerRoleString(RoleShade) != "Shade" {
		t.Error("Expected 'Shade'")
	}
}

// --- Phantom Council Tests ---

func TestNewPhantomCouncil(t *testing.T) {
	var creator [32]byte
	rand.Read(creator[:])

	council, err := NewPhantomCouncil(
		creator,
		"The Inner Circle",
		"Discussion of governance matters",
		200.0,
		7,
		CouncilMinResonance,
	)
	if err != nil {
		t.Fatalf("NewPhantomCouncil failed: %v", err)
	}

	// Council starts dormant until 3+ members.
	if council.State != CouncilDormant {
		t.Errorf("Expected CouncilDormant, got %d", council.State)
	}

	// Creator is first member.
	if council.ActiveMemberCount() != 1 {
		t.Errorf("Expected 1 member, got %d", council.ActiveMemberCount())
	}

	if !council.IsMember(creator) {
		t.Error("Expected creator to be member")
	}
}

func TestCouncilInvalidSize(t *testing.T) {
	var creator [32]byte
	rand.Read(creator[:])

	// Too few max members.
	_, err := NewPhantomCouncil(creator, "Test", "Test", 200.0, 2, CouncilMinResonance)
	if err != ErrCouncilInvalidSize {
		t.Errorf("Expected ErrCouncilInvalidSize, got %v", err)
	}

	// Too many max members.
	_, err = NewPhantomCouncil(creator, "Test", "Test", 200.0, 20, CouncilMinResonance)
	if err != ErrCouncilInvalidSize {
		t.Errorf("Expected ErrCouncilInvalidSize, got %v", err)
	}
}

func TestCouncilInvalidResonance(t *testing.T) {
	var creator [32]byte
	rand.Read(creator[:])

	_, err := NewPhantomCouncil(creator, "Test", "Test", 100.0, 5, CouncilMinResonance)
	if err != ErrCouncilInvalidMinResonance {
		t.Errorf("Expected ErrCouncilInvalidMinResonance, got %v", err)
	}
}

func TestCouncilInsufficientCreatorResonance(t *testing.T) {
	var creator [32]byte
	rand.Read(creator[:])

	// Creator resonance below minimum (200).
	_, err := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance-1)
	if err != ErrCouncilInsufficientResonance {
		t.Errorf("Expected ErrCouncilInsufficientResonance, got %v", err)
	}

	// Exactly at minimum should succeed.
	_, err = NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)
	if err != nil {
		t.Errorf("Expected success at minimum resonance, got %v", err)
	}
}

func TestCouncilNameTooLong(t *testing.T) {
	var creator [32]byte
	rand.Read(creator[:])

	longName := make([]byte, CouncilMaxNameLength+1)
	for i := range longName {
		longName[i] = 'a'
	}

	_, err := NewPhantomCouncil(creator, string(longName), "Test", 200.0, 5, CouncilMinResonance)
	if err != ErrCouncilNameTooLong {
		t.Errorf("Expected ErrCouncilNameTooLong, got %v", err)
	}
}

func TestCouncilApplication(t *testing.T) {
	var creator, applicant [32]byte
	rand.Read(creator[:])
	rand.Read(applicant[:])

	council, _ := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)

	err := council.Apply(applicant, []byte("zk_proof_placeholder"))
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	pending := council.GetPendingApplications()
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending application, got %d", len(pending))
	}
}

func TestCouncilAdmission(t *testing.T) {
	var creator, applicant1, applicant2 [32]byte
	rand.Read(creator[:])
	rand.Read(applicant1[:])
	rand.Read(applicant2[:])

	council, _ := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)

	// Apply and admit first applicant.
	council.Apply(applicant1, nil)
	err := council.VoteOnApplication(creator, applicant1, VoteFor)
	if err != nil {
		t.Fatalf("VoteOnApplication failed: %v", err)
	}

	// Should be admitted.
	if !council.IsMember(applicant1) {
		t.Error("Expected applicant1 to be member")
	}

	// Apply and admit second applicant.
	council.Apply(applicant2, nil)
	council.VoteOnApplication(creator, applicant2, VoteFor)
	council.VoteOnApplication(applicant1, applicant2, VoteFor)

	// Now should have 3 members and be active.
	if council.ActiveMemberCount() != 3 {
		t.Errorf("Expected 3 members, got %d", council.ActiveMemberCount())
	}

	if !council.IsActive() {
		t.Error("Expected council to be active")
	}
}

func TestCouncilRejection(t *testing.T) {
	var creator, applicant [32]byte
	rand.Read(creator[:])
	rand.Read(applicant[:])

	council, _ := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)

	council.Apply(applicant, nil)
	council.VoteOnApplication(creator, applicant, VoteAgainst)

	// Should be rejected.
	if council.IsMember(applicant) {
		t.Error("Expected applicant to be rejected")
	}

	apps := council.GetPendingApplications()
	if len(apps) != 0 {
		t.Errorf("Expected 0 pending applications, got %d", len(apps))
	}
}

func TestCouncilExpulsion(t *testing.T) {
	var creator, member1, member2 [32]byte
	rand.Read(creator[:])
	rand.Read(member1[:])
	rand.Read(member2[:])

	council, _ := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)

	// Admit two members.
	council.Apply(member1, nil)
	council.VoteOnApplication(creator, member1, VoteFor)
	council.Apply(member2, nil)
	council.VoteOnApplication(creator, member2, VoteFor)
	council.VoteOnApplication(member1, member2, VoteFor)

	// Now we have 3 members. Initiate expulsion of member2.
	err := council.InitiateExpulsion(creator, member2)
	if err != nil {
		t.Fatalf("InitiateExpulsion failed: %v", err)
	}

	// Creator already voted for. member1 needs to vote for expulsion.
	// Threshold is 2/3 of 2 voting members (creator + member1) = 2.
	council.VoteOnExpulsion(member1, member2, VoteFor)

	// member2 should be expelled.
	if council.IsMember(member2) {
		t.Error("Expected member2 to be expelled")
	}
}

func TestCouncilLeave(t *testing.T) {
	var creator, member1 [32]byte
	rand.Read(creator[:])
	rand.Read(member1[:])

	council, _ := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)

	council.Apply(member1, nil)
	council.VoteOnApplication(creator, member1, VoteFor)

	err := council.Leave(member1)
	if err != nil {
		t.Fatalf("Leave failed: %v", err)
	}

	if council.IsMember(member1) {
		t.Error("Expected member1 to have left")
	}
}

func TestCouncilProposal(t *testing.T) {
	var creator, member1, member2 [32]byte
	rand.Read(creator[:])
	rand.Read(member1[:])
	rand.Read(member2[:])

	council, _ := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)

	// Build council with 3 members.
	council.Apply(member1, nil)
	council.VoteOnApplication(creator, member1, VoteFor)
	council.Apply(member2, nil)
	council.VoteOnApplication(creator, member2, VoteFor)
	council.VoteOnApplication(member1, member2, VoteFor)

	// Create a proposal.
	proposal, err := council.CreateProposal(creator, "Should we expand?")
	if err != nil {
		t.Fatalf("CreateProposal failed: %v", err)
	}

	// Vote on proposal.
	council.VoteOnProposal(creator, proposal.ID, VoteFor)
	council.VoteOnProposal(member1, proposal.ID, VoteFor)
	council.VoteOnProposal(member2, proposal.ID, VoteAgainst)

	// Should pass (2 for, 1 against).
	if !proposal.Resolved {
		t.Error("Expected proposal to be resolved")
	}
	if !proposal.Passed {
		t.Error("Expected proposal to pass")
	}
}

func TestCouncilProposalFail(t *testing.T) {
	var creator, member1, member2 [32]byte
	rand.Read(creator[:])
	rand.Read(member1[:])
	rand.Read(member2[:])

	council, _ := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)

	// Build council.
	council.Apply(member1, nil)
	council.VoteOnApplication(creator, member1, VoteFor)
	council.Apply(member2, nil)
	council.VoteOnApplication(creator, member2, VoteFor)
	council.VoteOnApplication(member1, member2, VoteFor)

	proposal, _ := council.CreateProposal(creator, "Bad idea")

	// Vote against.
	council.VoteOnProposal(creator, proposal.ID, VoteAgainst)
	council.VoteOnProposal(member1, proposal.ID, VoteAgainst)
	council.VoteOnProposal(member2, proposal.ID, VoteFor)

	if !proposal.Resolved {
		t.Error("Expected proposal to be resolved")
	}
	if proposal.Passed {
		t.Error("Expected proposal to fail")
	}
}

func TestCouncilStore(t *testing.T) {
	store := NewCouncilStore()

	var creator [32]byte
	rand.Read(creator[:])

	council, _ := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)
	store.AddCouncil(council)

	if store.Count() != 1 {
		t.Errorf("Expected 1 council, got %d", store.Count())
	}

	retrieved := store.GetCouncil(council.ID)
	if retrieved == nil {
		t.Error("Expected council, got nil")
	}

	memberCouncils := store.GetCouncilsForMember(creator)
	if len(memberCouncils) != 1 {
		t.Errorf("Expected 1 council for member, got %d", len(memberCouncils))
	}
}

func TestCouncilStateStrings(t *testing.T) {
	if CouncilStateString(CouncilActive) != "Active" {
		t.Error("Expected 'Active'")
	}
	if CouncilStateString(CouncilDormant) != "Dormant" {
		t.Error("Expected 'Dormant'")
	}
	if MemberStatusString(MemberActive) != "Active" {
		t.Error("Expected 'Active'")
	}
	if VoteValueString(VoteFor) != "For" {
		t.Error("Expected 'For'")
	}
	if VoteValueString(VoteAgainst) != "Against" {
		t.Error("Expected 'Against'")
	}
}

// --- Resonance Gating Tests ---

// mockResonanceGate is a test implementation of ResonanceGate.
type mockResonanceGate struct {
	scores map[[32]byte]int
}

func (m *mockResonanceGate) GetResonance(specterKey [32]byte) (int, error) {
	if score, ok := m.scores[specterKey]; ok {
		return score, nil
	}
	return 0, nil
}

func newMockGate(specterKey [32]byte, resonance int) *mockResonanceGate {
	return &mockResonanceGate{
		scores: map[[32]byte]int{specterKey: resonance},
	}
}

func TestCheckResonanceGate(t *testing.T) {
	var key [32]byte
	rand.Read(key[:])

	// Test with nil gate (should allow).
	err := CheckResonanceGate(nil, key, 50)
	if err != nil {
		t.Errorf("Expected nil error with nil gate, got %v", err)
	}

	// Test with gate meeting requirement.
	gate := newMockGate(key, 75)
	err = CheckResonanceGate(gate, key, 50)
	if err != nil {
		t.Errorf("Expected nil error with sufficient Resonance, got %v", err)
	}

	// Test with gate not meeting requirement.
	gate = newMockGate(key, 25)
	err = CheckResonanceGate(gate, key, 50)
	if err != ErrResonanceRequirementNotMet {
		t.Errorf("Expected ErrResonanceRequirementNotMet, got %v", err)
	}

	// Test at exact threshold.
	gate = newMockGate(key, 50)
	err = CheckResonanceGate(gate, key, 50)
	if err != nil {
		t.Errorf("Expected nil error at exact threshold, got %v", err)
	}
}

func TestNewPuzzleGated(t *testing.T) {
	var seed, initiator [32]byte
	rand.Read(seed[:])
	rand.Read(initiator[:])

	// Test with sufficient Resonance.
	gate := newMockGate(initiator, PuzzleMinResonance)
	puzzle, err := NewPuzzleGated(PuzzleFragment, seed, 20, PuzzleDuration30Min, initiator, gate)
	if err != nil {
		t.Fatalf("NewPuzzleGated failed with sufficient Resonance: %v", err)
	}
	if puzzle == nil {
		t.Fatal("Expected puzzle to be created")
	}

	// Test with insufficient Resonance.
	lowGate := newMockGate(initiator, PuzzleMinResonance-1)
	_, err = NewPuzzleGated(PuzzleFragment, seed, 20, PuzzleDuration30Min, initiator, lowGate)
	if err != ErrPuzzleInsufficientRes {
		t.Errorf("Expected ErrPuzzleInsufficientRes, got %v", err)
	}

	// Test with nil gate (permissionless mode).
	puzzle, err = NewPuzzleGated(PuzzleFragment, seed, 20, PuzzleDuration30Min, initiator, nil)
	if err != nil {
		t.Fatalf("NewPuzzleGated failed with nil gate: %v", err)
	}
	if puzzle == nil {
		t.Fatal("Expected puzzle to be created with nil gate")
	}
}

// --- ZK Claim Verification Tests ---

// mockZKVerifier is a test implementation of ZKClaimVerifier.
type mockZKVerifier struct {
	shouldPass bool
}

func (m *mockZKVerifier) VerifyResonanceClaim(proof []byte, minResonance int64) error {
	if !m.shouldPass {
		return ErrInvalidZKClaim
	}
	return nil
}

func TestCouncilZKVerification(t *testing.T) {
	var creator, applicant [32]byte
	rand.Read(creator[:])
	rand.Read(applicant[:])

	// Create council with ZK verifier.
	council, _ := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)
	council.SetZKVerifier(&mockZKVerifier{shouldPass: true})

	// Apply with valid ZK proof.
	err := council.Apply(applicant, []byte("valid_proof"))
	if err != nil {
		t.Fatalf("Apply should succeed with valid ZK proof: %v", err)
	}

	pending := council.GetPendingApplications()
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending application, got %d", len(pending))
	}
}

func TestCouncilZKVerificationFailure(t *testing.T) {
	var creator, applicant [32]byte
	rand.Read(creator[:])
	rand.Read(applicant[:])

	// Create council with failing ZK verifier.
	council, _ := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)
	council.SetZKVerifier(&mockZKVerifier{shouldPass: false})

	// Apply with invalid ZK proof.
	err := council.Apply(applicant, []byte("invalid_proof"))
	if err != ErrInvalidZKClaim {
		t.Errorf("Expected ErrInvalidZKClaim, got %v", err)
	}

	// Should have no pending applications.
	pending := council.GetPendingApplications()
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending applications, got %d", len(pending))
	}
}

func TestCouncilZKVerificationMissingProof(t *testing.T) {
	var creator, applicant [32]byte
	rand.Read(creator[:])
	rand.Read(applicant[:])

	// Create council with ZK verifier that requires proof.
	council, _ := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)
	council.SetZKVerifier(&mockZKVerifier{shouldPass: true})

	// Apply without ZK proof.
	err := council.Apply(applicant, nil)
	if err != ErrMissingZKClaim {
		t.Errorf("Expected ErrMissingZKClaim, got %v", err)
	}

	// Also test with empty proof.
	err = council.Apply(applicant, []byte{})
	if err != ErrMissingZKClaim {
		t.Errorf("Expected ErrMissingZKClaim for empty proof, got %v", err)
	}
}

func TestCouncilWithoutZKVerifier(t *testing.T) {
	var creator, applicant [32]byte
	rand.Read(creator[:])
	rand.Read(applicant[:])

	// Create council without ZK verifier (backward compatible).
	council, _ := NewPhantomCouncil(creator, "Test", "Test", 200.0, 5, CouncilMinResonance)

	// Apply without ZK proof should succeed.
	err := council.Apply(applicant, nil)
	if err != nil {
		t.Fatalf("Apply should succeed without ZK verifier: %v", err)
	}

	pending := council.GetPendingApplications()
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending application, got %d", len(pending))
	}
}

// --- Oracle Pool Gating Tests ---

func TestNewOraclePoolGated(t *testing.T) {
	var creator [32]byte
	rand.Read(creator[:])

	deadline := time.Now().Add(24 * time.Hour)
	resolution := time.Now().Add(48 * time.Hour)

	// Test with sufficient Resonance.
	gate := newMockGate(creator, 150) // Above OracleMinResonance (100).
	pool, err := NewOraclePoolGated(
		"Will MURMUR have 1000 nodes by end of month?",
		OraclePredictionBoolean,
		"DHT node count",
		creator,
		deadline,
		resolution,
		gate,
	)
	if err != nil {
		t.Fatalf("NewOraclePoolGated with sufficient Resonance failed: %v", err)
	}
	if pool == nil {
		t.Fatal("Expected non-nil pool")
	}
	if pool.Question != "Will MURMUR have 1000 nodes by end of month?" {
		t.Error("Pool question mismatch")
	}

	// Test with insufficient Resonance.
	gate = newMockGate(creator, 50) // Below OracleMinResonance (100).
	pool, err = NewOraclePoolGated(
		"Test question",
		OraclePredictionBoolean,
		"test",
		creator,
		deadline,
		resolution,
		gate,
	)
	if err != ErrOracleInsufficientRes {
		t.Errorf("Expected ErrOracleInsufficientRes, got %v", err)
	}
	if pool != nil {
		t.Error("Expected nil pool with insufficient Resonance")
	}

	// Test at exact threshold.
	gate = newMockGate(creator, OracleMinResonance)
	pool, err = NewOraclePoolGated(
		"Threshold test",
		OraclePredictionNumeric,
		"test",
		creator,
		deadline,
		resolution,
		gate,
	)
	if err != nil {
		t.Errorf("NewOraclePoolGated at exact threshold failed: %v", err)
	}
	if pool == nil {
		t.Error("Expected non-nil pool at exact threshold")
	}
}

// --- Shadow Play Gating Tests ---

func TestNewShadowPlayGated(t *testing.T) {
	var initiator [32]byte
	rand.Read(initiator[:])

	// Test with sufficient Resonance.
	gate := newMockGate(initiator, 250) // Above ShadowPlayMinResonance (200).
	game, err := NewShadowPlayGated(
		initiator,
		ShadowPlayDuration30Min,
		8,
		gate,
	)
	if err != nil {
		t.Fatalf("NewShadowPlayGated with sufficient Resonance failed: %v", err)
	}
	if game == nil {
		t.Fatal("Expected non-nil game")
	}
	if game.MaxPlayers != 8 {
		t.Errorf("Expected MaxPlayers 8, got %d", game.MaxPlayers)
	}
	if game.Duration != ShadowPlayDuration30Min {
		t.Error("Duration mismatch")
	}

	// Test with insufficient Resonance.
	gate = newMockGate(initiator, 100) // Below ShadowPlayMinResonance (200).
	game, err = NewShadowPlayGated(
		initiator,
		ShadowPlayDuration60Min,
		10,
		gate,
	)
	if err != ErrShadowPlayInsufficientResonance {
		t.Errorf("Expected ErrShadowPlayInsufficientResonance, got %v", err)
	}
	if game != nil {
		t.Error("Expected nil game with insufficient Resonance")
	}

	// Test at exact threshold.
	gate = newMockGate(initiator, ShadowPlayMinResonance)
	game, err = NewShadowPlayGated(
		initiator,
		ShadowPlayDuration30Min,
		5,
		gate,
	)
	if err != nil {
		t.Errorf("NewShadowPlayGated at exact threshold failed: %v", err)
	}
	if game == nil {
		t.Error("Expected non-nil game at exact threshold")
	}
}
