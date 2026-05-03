package sparks

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

func TestNewSparkStore(t *testing.T) {
	store := NewSparkStore()
	if store == nil {
		t.Fatal("NewSparkStore returned nil")
	}
	if store.CountTotalSparks() != 0 {
		t.Error("new store should have no sparks")
	}
}

func TestCreateSpark_WaveRelay(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	prompt := "Describe your day in exactly 7 words"

	spark, err := store.CreateSpark(SparkWaveRelay, initiatorPub, prompt, initiatorPriv)
	if err != nil {
		t.Fatalf("CreateSpark failed: %v", err)
	}
	if spark == nil {
		t.Fatal("spark is nil")
	}
	if spark.Type != SparkWaveRelay {
		t.Errorf("expected SparkWaveRelay, got %v", spark.Type)
	}
	if spark.Prompt != prompt {
		t.Errorf("prompt mismatch")
	}
	if spark.State != SparkActive {
		t.Error("new spark should be active")
	}
	if len(spark.Signature) == 0 {
		t.Error("expected signature")
	}

	// Verify signature.
	if !VerifySpark(spark, initiatorPub) {
		t.Error("signature verification failed")
	}
}

func TestCreateSpark_EchoRace(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)

	// EchoRace doesn't require prompt.
	spark, err := store.CreateSpark(SparkEchoRace, initiatorPub, "", initiatorPriv)
	if err != nil {
		t.Fatalf("CreateSpark failed: %v", err)
	}
	if spark.Type != SparkEchoRace {
		t.Errorf("expected SparkEchoRace, got %v", spark.Type)
	}
}

func TestCreateSpark_InvalidType(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)

	_, err := store.CreateSpark(SparkType(99), initiatorPub, "", initiatorPriv)
	if err != ErrSparkInvalidType {
		t.Errorf("expected ErrSparkInvalidType, got %v", err)
	}
}

func TestCreateSpark_WaveRelayRequiresPrompt(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)

	_, err := store.CreateSpark(SparkWaveRelay, initiatorPub, "", initiatorPriv)
	if err != ErrInvalidPrompt {
		t.Errorf("expected ErrInvalidPrompt, got %v", err)
	}
}

func TestRespondToSpark_EchoRace(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	responderPub, responderPriv, _ := ed25519.GenerateKey(rand.Reader)

	spark, _ := store.CreateSpark(SparkEchoRace, initiatorPub, "", initiatorPriv)

	var waveID [32]byte
	rand.Read(waveID[:])

	response, err := store.RespondToSpark(spark.ID, responderPub, waveID, responderPriv)
	if err != nil {
		t.Fatalf("RespondToSpark failed: %v", err)
	}
	if response == nil {
		t.Fatal("response is nil")
	}

	// Verify signature.
	if !VerifySparkResponse(response, responderPub) {
		t.Error("response signature verification failed")
	}

	// Check winner was set.
	spark, _ = store.GetSpark(spark.ID)
	if spark.WinnerID == nil {
		t.Error("expected winner to be set")
	}
	if mechanics.KeyToHex(spark.WinnerID) != mechanics.KeyToHex(responderPub) {
		t.Error("winner should be responder")
	}
	if spark.State != SparkCompleted {
		t.Error("spark should be completed")
	}

	// Check crown was granted.
	if !store.HasCrown(responderPub) {
		t.Error("responder should have crown")
	}

	// Check result was recorded.
	result := store.GetResult(spark.ID)
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.TotalResponses != 1 {
		t.Errorf("expected 1 response, got %d", result.TotalResponses)
	}
}

func TestRespondToSpark_AlreadyWon(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	responder1Pub, responder1Priv, _ := ed25519.GenerateKey(rand.Reader)
	responder2Pub, responder2Priv, _ := ed25519.GenerateKey(rand.Reader)

	spark, _ := store.CreateSpark(SparkEchoRace, initiatorPub, "", initiatorPriv)

	var waveID1, waveID2 [32]byte
	rand.Read(waveID1[:])
	rand.Read(waveID2[:])

	// First response wins.
	store.RespondToSpark(spark.ID, responder1Pub, waveID1, responder1Priv)

	// Second response should fail.
	_, err := store.RespondToSpark(spark.ID, responder2Pub, waveID2, responder2Priv)
	if err != ErrSparkAlreadyWon {
		t.Errorf("expected ErrSparkAlreadyWon, got %v", err)
	}
}

func TestRespondToSpark_SelfResponse(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)

	spark, _ := store.CreateSpark(SparkEchoRace, initiatorPub, "", initiatorPriv)

	var waveID [32]byte
	rand.Read(waveID[:])

	_, err := store.RespondToSpark(spark.ID, initiatorPub, waveID, initiatorPriv)
	if err != ErrSparkSelfResponse {
		t.Errorf("expected ErrSparkSelfResponse, got %v", err)
	}
}

func TestRespondToSpark_WaveRelay(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)

	prompt := "Test prompt"
	spark, _ := store.CreateSpark(SparkWaveRelay, initiatorPub, prompt, initiatorPriv)

	// Multiple responses allowed for WaveRelay.
	for i := 0; i < 3; i++ {
		responderPub, responderPriv, _ := ed25519.GenerateKey(rand.Reader)
		var waveID [32]byte
		rand.Read(waveID[:])

		_, err := store.RespondToSpark(spark.ID, responderPub, waveID, responderPriv)
		if err != nil {
			t.Errorf("response %d failed: %v", i, err)
		}
	}

	responses := store.GetResponses(spark.ID)
	if len(responses) != 3 {
		t.Errorf("expected 3 responses, got %d", len(responses))
	}

	// Spark should still be active (no winner concept for WaveRelay).
	spark, _ = store.GetSpark(spark.ID)
	if spark.State != SparkActive {
		t.Error("WaveRelay spark should remain active until expiry")
	}
}

func TestRespondToSpark_NotFound(t *testing.T) {
	store := NewSparkStore()

	responderPub, responderPriv, _ := ed25519.GenerateKey(rand.Reader)

	var badSparkID, waveID [32]byte
	rand.Read(badSparkID[:])
	rand.Read(waveID[:])

	_, err := store.RespondToSpark(badSparkID, responderPub, waveID, responderPriv)
	if err != ErrSparkNotFound {
		t.Errorf("expected ErrSparkNotFound, got %v", err)
	}
}

func TestGetSpark(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	spark, _ := store.CreateSpark(SparkEchoRace, initiatorPub, "", initiatorPriv)

	retrieved, err := store.GetSpark(spark.ID)
	if err != nil {
		t.Errorf("GetSpark failed: %v", err)
	}
	if retrieved.ID != spark.ID {
		t.Error("ID mismatch")
	}

	// Non-existent spark.
	var badID [32]byte
	rand.Read(badID[:])
	_, err = store.GetSpark(badID)
	if err != ErrSparkNotFound {
		t.Errorf("expected ErrSparkNotFound, got %v", err)
	}
}

func TestGetActiveSparks(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)

	// Create 3 sparks.
	for i := 0; i < 3; i++ {
		store.CreateSpark(SparkEchoRace, initiatorPub, "", initiatorPriv)
	}

	active := store.GetActiveSparks()
	if len(active) != 3 {
		t.Errorf("expected 3 active sparks, got %d", len(active))
	}
}

func TestGetSparksByInitiator(t *testing.T) {
	store := NewSparkStore()

	initiator1Pub, initiator1Priv, _ := ed25519.GenerateKey(rand.Reader)
	initiator2Pub, initiator2Priv, _ := ed25519.GenerateKey(rand.Reader)

	// Initiator 1 creates 2 sparks.
	store.CreateSpark(SparkEchoRace, initiator1Pub, "", initiator1Priv)
	store.CreateSpark(SparkEchoRace, initiator1Pub, "", initiator1Priv)

	// Initiator 2 creates 1 spark.
	store.CreateSpark(SparkEchoRace, initiator2Pub, "", initiator2Priv)

	sparks1 := store.GetSparksByInitiator(initiator1Pub)
	if len(sparks1) != 2 {
		t.Errorf("expected 2 sparks for initiator1, got %d", len(sparks1))
	}

	sparks2 := store.GetSparksByInitiator(initiator2Pub)
	if len(sparks2) != 1 {
		t.Errorf("expected 1 spark for initiator2, got %d", len(sparks2))
	}
}

func TestHasCrown(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	responderPub, responderPriv, _ := ed25519.GenerateKey(rand.Reader)

	// Initially no crown.
	if store.HasCrown(responderPub) {
		t.Error("should not have crown initially")
	}

	spark, _ := store.CreateSpark(SparkEchoRace, initiatorPub, "", initiatorPriv)

	var waveID [32]byte
	rand.Read(waveID[:])
	store.RespondToSpark(spark.ID, responderPub, waveID, responderPriv)

	// Now has crown.
	if !store.HasCrown(responderPub) {
		t.Error("should have crown after winning")
	}

	// Check expiry.
	expiry := store.GetCrownExpiry(responderPub)
	if expiry.IsZero() {
		t.Error("crown expiry should not be zero")
	}
	if expiry.Before(time.Now()) {
		t.Error("crown should not be expired yet")
	}
}

func TestExpireSparks(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)

	spark, _ := store.CreateSpark(SparkWaveRelay, initiatorPub, "test", initiatorPriv)

	// Manually expire for testing.
	spark.ExpiresAt = time.Now().Add(-time.Hour)

	expired := store.ExpireSparks()
	if expired != 1 {
		t.Errorf("expected 1 expired, got %d", expired)
	}

	spark, _ = store.GetSpark(spark.ID)
	if spark.State != SparkExpired {
		t.Error("spark should be expired")
	}

	// Result should be recorded.
	result := store.GetResult(spark.ID)
	if result == nil {
		t.Error("result should exist for expired WaveRelay")
	}
}

func TestPurgeExpiredCrowns(t *testing.T) {
	store := NewSparkStore()

	// Manually add expired crown.
	store.mu.Lock()
	store.crownHolders["test-key"] = time.Now().Add(-time.Hour)
	store.mu.Unlock()

	if store.CountCrownHolders() != 0 {
		t.Error("expired crown should not be counted")
	}

	purged := store.PurgeExpiredCrowns()
	if purged != 1 {
		t.Errorf("expected 1 purged, got %d", purged)
	}
}

func TestCountFunctions(t *testing.T) {
	store := NewSparkStore()

	if store.CountActiveSparks() != 0 {
		t.Error("should have 0 active sparks")
	}
	if store.CountTotalSparks() != 0 {
		t.Error("should have 0 total sparks")
	}
	if store.CountCrownHolders() != 0 {
		t.Error("should have 0 crown holders")
	}

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)
	store.CreateSpark(SparkEchoRace, initiatorPub, "", initiatorPriv)

	if store.CountActiveSparks() != 1 {
		t.Error("should have 1 active spark")
	}
	if store.CountTotalSparks() != 1 {
		t.Error("should have 1 total spark")
	}
}

func TestSparkTypeString(t *testing.T) {
	tests := []struct {
		t    SparkType
		want string
	}{
		{SparkWaveRelay, "Wave Relay"},
		{SparkEchoRace, "Echo Race"},
		{SparkType(99), "Unknown"},
	}

	for _, tt := range tests {
		got := SparkTypeString(tt.t)
		if got != tt.want {
			t.Errorf("SparkTypeString(%v) = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestSparkIsExpired(t *testing.T) {
	spark := &Spark{
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if spark.IsExpired() {
		t.Error("future spark should not be expired")
	}

	spark.ExpiresAt = time.Now().Add(-time.Hour)
	if !spark.IsExpired() {
		t.Error("past spark should be expired")
	}
}

func TestSparkIsActive(t *testing.T) {
	spark := &Spark{
		State:     SparkActive,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if !spark.IsActive() {
		t.Error("should be active")
	}

	spark.State = SparkCompleted
	if spark.IsActive() {
		t.Error("completed spark should not be active")
	}

	spark.State = SparkActive
	spark.ExpiresAt = time.Now().Add(-time.Hour)
	if spark.IsActive() {
		t.Error("expired spark should not be active")
	}
}

func TestGetSparksByType(t *testing.T) {
	store := NewSparkStore()

	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)

	// Create 2 EchoRace and 1 WaveRelay.
	store.CreateSpark(SparkEchoRace, initiatorPub, "", initiatorPriv)
	store.CreateSpark(SparkEchoRace, initiatorPub, "", initiatorPriv)
	store.CreateSpark(SparkWaveRelay, initiatorPub, "prompt", initiatorPriv)

	echoRaces := store.GetSparksByType(SparkEchoRace)
	if len(echoRaces) != 2 {
		t.Errorf("expected 2 EchoRace sparks, got %d", len(echoRaces))
	}

	waveRelays := store.GetSparksByType(SparkWaveRelay)
	if len(waveRelays) != 1 {
		t.Errorf("expected 1 WaveRelay spark, got %d", len(waveRelays))
	}
}

func TestVerifySpark_NilCases(t *testing.T) {
	initiatorPub, _, _ := ed25519.GenerateKey(rand.Reader)

	// Nil spark.
	if VerifySpark(nil, initiatorPub) {
		t.Error("nil spark should fail verification")
	}

	// Empty signature.
	spark := &Spark{}
	if VerifySpark(spark, initiatorPub) {
		t.Error("empty signature should fail verification")
	}
}

func TestVerifySparkResponse_NilCases(t *testing.T) {
	responderPub, _, _ := ed25519.GenerateKey(rand.Reader)

	// Nil response.
	if VerifySparkResponse(nil, responderPub) {
		t.Error("nil response should fail verification")
	}

	// Empty signature.
	response := &SparkResponse{}
	if VerifySparkResponse(response, responderPub) {
		t.Error("empty signature should fail verification")
	}
}

func BenchmarkCreateSpark(b *testing.B) {
	store := NewSparkStore()
	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.CreateSpark(SparkEchoRace, initiatorPub, "", initiatorPriv)
	}
}

func BenchmarkRespondToSpark(b *testing.B) {
	store := NewSparkStore()
	initiatorPub, initiatorPriv, _ := ed25519.GenerateKey(rand.Reader)

	// Create sparks for each iteration.
	sparks := make([]*Spark, b.N)
	for i := 0; i < b.N; i++ {
		sparks[i], _ = store.CreateSpark(SparkWaveRelay, initiatorPub, "prompt", initiatorPriv)
	}

	responderPub, responderPriv, _ := ed25519.GenerateKey(rand.Reader)
	var waveID [32]byte
	rand.Read(waveID[:])

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.RespondToSpark(sparks[i].ID, responderPub, waveID, responderPriv)
	}
}
