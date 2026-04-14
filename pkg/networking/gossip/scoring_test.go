package gossip

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func TestPeerScoreTracker_RecordValidMessage(t *testing.T) {
	tracker := NewPeerScoreTracker()
	testPeer := peer.ID("test-peer-1")

	tracker.RecordValidMessage(testPeer)

	score := tracker.GetScore(testPeer)
	if score != WeightValidMessage {
		t.Errorf("expected score %f, got %f", WeightValidMessage, score)
	}

	valid, invalid, dup, _ := tracker.GetStats(testPeer)
	if valid != 1 {
		t.Errorf("expected 1 valid message, got %d", valid)
	}
	if invalid != 0 {
		t.Errorf("expected 0 invalid messages, got %d", invalid)
	}
	if dup != 0 {
		t.Errorf("expected 0 duplicate messages, got %d", dup)
	}
}

func TestPeerScoreTracker_RecordInvalidSignature(t *testing.T) {
	tracker := NewPeerScoreTracker()
	testPeer := peer.ID("test-peer-2")

	tracker.RecordInvalidSignature(testPeer)

	score := tracker.GetScore(testPeer)
	if score != WeightInvalidSignature {
		t.Errorf("expected score %f, got %f", WeightInvalidSignature, score)
	}

	_, invalid, _, _ := tracker.GetStats(testPeer)
	if invalid != 1 {
		t.Errorf("expected 1 invalid message, got %d", invalid)
	}
}

func TestPeerScoreTracker_RecordDuplicateMessage(t *testing.T) {
	tracker := NewPeerScoreTracker()
	testPeer := peer.ID("test-peer-3")

	tracker.RecordDuplicateMessage(testPeer)

	score := tracker.GetScore(testPeer)
	if score != WeightDuplicateMessage {
		t.Errorf("expected score %f, got %f", WeightDuplicateMessage, score)
	}

	_, _, dup, _ := tracker.GetStats(testPeer)
	if dup != 1 {
		t.Errorf("expected 1 duplicate message, got %d", dup)
	}
}

func TestPeerScoreTracker_MultipleMessages(t *testing.T) {
	tracker := NewPeerScoreTracker()
	testPeer := peer.ID("test-peer-4")

	// Record multiple valid messages
	tracker.RecordValidMessage(testPeer)
	tracker.RecordValidMessage(testPeer)
	tracker.RecordValidMessage(testPeer)

	score := tracker.GetScore(testPeer)
	expectedScore := 3 * WeightValidMessage
	if score != expectedScore {
		t.Errorf("expected score %f, got %f", expectedScore, score)
	}

	// Record an invalid message
	tracker.RecordInvalidSignature(testPeer)

	score = tracker.GetScore(testPeer)
	expectedScore = 3*WeightValidMessage + WeightInvalidSignature
	if score != expectedScore {
		t.Errorf("expected score %f, got %f", expectedScore, score)
	}
}

func TestPeerScoreTracker_DecayScores(t *testing.T) {
	tracker := NewPeerScoreTracker()
	testPeer := peer.ID("test-peer-5")

	// Record some messages
	tracker.RecordValidMessage(testPeer)
	initialScore := tracker.GetScore(testPeer)

	// Decay scores
	tracker.DecayScores()

	score := tracker.GetScore(testPeer)
	expectedScore := initialScore * ScoreDecayFactor
	if score != expectedScore {
		t.Errorf("expected score %f, got %f", expectedScore, score)
	}
}

func TestPeerScoreTracker_PruneInactive(t *testing.T) {
	tracker := NewPeerScoreTracker()

	// Record messages for two peers
	peer1 := peer.ID("test-peer-6")
	peer2 := peer.ID("test-peer-7")

	tracker.RecordValidMessage(peer1)
	tracker.RecordValidMessage(peer2)

	if tracker.Size() != 2 {
		t.Errorf("expected 2 peers, got %d", tracker.Size())
	}

	// Prune with a very short maxAge (nothing should be pruned)
	pruned := tracker.PruneInactive(time.Hour)
	if pruned != 0 {
		t.Errorf("expected 0 pruned, got %d", pruned)
	}

	// Manually set LastSeen to the past for peer1
	tracker.mu.Lock()
	tracker.scores[peer1].LastSeen = time.Now().Add(-2 * time.Hour)
	tracker.mu.Unlock()

	// Prune with 1 hour maxAge
	pruned = tracker.PruneInactive(time.Hour)
	if pruned != 1 {
		t.Errorf("expected 1 pruned, got %d", pruned)
	}

	if tracker.Size() != 1 {
		t.Errorf("expected 1 peer, got %d", tracker.Size())
	}
}

func TestPeerScoreTracker_Callback(t *testing.T) {
	tracker := NewPeerScoreTracker()
	testPeer := peer.ID("test-peer-8")

	var callbackCalled bool
	var callbackPeer peer.ID
	var callbackScore float64

	tracker.SetCallback(func(p peer.ID, score float64) {
		callbackCalled = true
		callbackPeer = p
		callbackScore = score
	})

	tracker.RecordValidMessage(testPeer)

	if !callbackCalled {
		t.Error("callback was not called")
	}
	if callbackPeer != testPeer {
		t.Errorf("expected peer %s, got %s", testPeer, callbackPeer)
	}
	if callbackScore != WeightValidMessage {
		t.Errorf("expected score %f, got %f", WeightValidMessage, callbackScore)
	}
}

func TestPeerScoreTracker_AppSpecificScoreFunc(t *testing.T) {
	tracker := NewPeerScoreTracker()
	testPeer := peer.ID("test-peer-9")

	tracker.RecordValidMessage(testPeer)
	tracker.RecordValidMessage(testPeer)

	scoreFunc := tracker.AppSpecificScoreFunc()

	score := scoreFunc(testPeer)
	expectedScore := 2 * WeightValidMessage
	if score != expectedScore {
		t.Errorf("expected score %f, got %f", expectedScore, score)
	}

	// Unknown peer should return 0
	unknownPeer := peer.ID("unknown-peer")
	score = scoreFunc(unknownPeer)
	if score != 0 {
		t.Errorf("expected score 0 for unknown peer, got %f", score)
	}
}

func TestPeerScoreTracker_AllPenaltyTypes(t *testing.T) {
	tracker := NewPeerScoreTracker()
	testPeer := peer.ID("test-peer-10")

	// Test all penalty types
	tracker.RecordInvalidTimestamp(testPeer)
	if score := tracker.GetScore(testPeer); score != WeightInvalidTimestamp {
		t.Errorf("invalid timestamp: expected %f, got %f", WeightInvalidTimestamp, score)
	}

	tracker2 := NewPeerScoreTracker()
	peer2 := peer.ID("test-peer-11")
	tracker2.RecordInvalidPayload(peer2)
	if score := tracker2.GetScore(peer2); score != WeightInvalidPayload {
		t.Errorf("invalid payload: expected %f, got %f", WeightInvalidPayload, score)
	}

	tracker3 := NewPeerScoreTracker()
	peer3 := peer.ID("test-peer-12")
	tracker3.RecordInvalidPoW(peer3)
	if score := tracker3.GetScore(peer3); score != WeightInvalidPoW {
		t.Errorf("invalid PoW: expected %f, got %f", WeightInvalidPoW, score)
	}

	tracker4 := NewPeerScoreTracker()
	peer4 := peer.ID("test-peer-13")
	tracker4.RecordExpiredTTL(peer4)
	if score := tracker4.GetScore(peer4); score != WeightExpiredTTL {
		t.Errorf("expired TTL: expected %f, got %f", WeightExpiredTTL, score)
	}
}

func TestPeerScoreTracker_StartDecayLoop(t *testing.T) {
	tracker := NewPeerScoreTracker()
	testPeer := peer.ID("test-peer-14")

	tracker.RecordValidMessage(testPeer)
	initialScore := tracker.GetScore(testPeer)

	ctx, cancel := context.WithCancel(context.Background())
	tracker.StartDecayLoop(ctx)

	// Wait a bit for decay to happen (if interval is short)
	// For testing, we just verify the goroutine starts and stops cleanly
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Score should still be positive (decay interval is 1 minute)
	score := tracker.GetScore(testPeer)
	if score > initialScore {
		t.Errorf("score should not increase, got %f > %f", score, initialScore)
	}
}

func TestValidatingMessageHandlers_New(t *testing.T) {
	tracker := NewPeerScoreTracker()
	handlers := NewValidatingMessageHandlers(tracker)

	if handlers == nil {
		t.Error("expected non-nil handlers")
	}
	if handlers.scoreTracker != tracker {
		t.Error("score tracker not set correctly")
	}
	if handlers.MessageHandlers == nil {
		t.Error("embedded MessageHandlers is nil")
	}
}

func TestValidatingMessageHandlers_CreateValidatingTopicHandler(t *testing.T) {
	tracker := NewPeerScoreTracker()
	handlers := NewValidatingMessageHandlers(tracker)

	handler := handlers.CreateValidatingTopicHandler(TopicWaves)
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}

func TestPeerScoreTracker_GetStatsUnknownPeer(t *testing.T) {
	tracker := NewPeerScoreTracker()
	unknownPeer := peer.ID("unknown-peer")

	valid, invalid, dup, score := tracker.GetStats(unknownPeer)

	if valid != 0 || invalid != 0 || dup != 0 || score != 0 {
		t.Errorf("expected all zeros for unknown peer, got %d, %d, %d, %f",
			valid, invalid, dup, score)
	}
}

func TestPeerScoreTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewPeerScoreTracker()
	testPeer := peer.ID("concurrent-peer")

	done := make(chan bool)

	// Start multiple goroutines recording messages
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				tracker.RecordValidMessage(testPeer)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	valid, _, _, _ := tracker.GetStats(testPeer)
	if valid != 1000 {
		t.Errorf("expected 1000 valid messages, got %d", valid)
	}
}
