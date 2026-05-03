package shadowplay

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics"
)

func TestDiscussionPhaseStateString(t *testing.T) {
	tests := []struct {
		state DiscussionPhaseState
		want  string
	}{
		{DiscussionInactive, "Inactive"},
		{DiscussionActive, "Active"},
		{DiscussionEnding, "Ending"},
		{DiscussionEnded, "Ended"},
		{DiscussionPhaseState(99), "Unknown"},
	}

	for _, tc := range tests {
		got := DiscussionPhaseStateString(tc.state)
		if got != tc.want {
			t.Errorf("DiscussionPhaseStateString(%d) = %q, want %q", tc.state, got, tc.want)
		}
	}
}

func TestNewDiscussionPhase(t *testing.T) {
	gameID := [32]byte{1, 2, 3}
	players := [][32]byte{
		{1}, {2}, {3},
	}

	dp := NewDiscussionPhase(gameID, 1, players)

	if dp.GameID != gameID {
		t.Errorf("GameID mismatch")
	}
	if dp.Round != 1 {
		t.Errorf("Round = %d, want 1", dp.Round)
	}
	if dp.State != DiscussionInactive {
		t.Errorf("Initial state = %v, want DiscussionInactive", dp.State)
	}
	if dp.ParticipantCount() != 3 {
		t.Errorf("ParticipantCount = %d, want 3", dp.ParticipantCount())
	}
}

func TestDiscussionPhase_Start(t *testing.T) {
	dp := NewDiscussionPhase([32]byte{1}, 1, [][32]byte{{1}, {2}})

	dp.Start()

	if dp.State != DiscussionActive {
		t.Errorf("State after Start = %v, want DiscussionActive", dp.State)
	}
	if !dp.IsActive() {
		t.Error("IsActive() should be true after Start")
	}
}

func TestDiscussionPhase_SendMessage(t *testing.T) {
	players := [][32]byte{{1}, {2}}
	dp := NewDiscussionPhase([32]byte{1}, 1, players)
	dp.Start()

	msg, err := dp.SendMessage([32]byte{1}, "Hello world")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if msg == nil {
		t.Fatal("SendMessage returned nil message")
	}
	if msg.Content != "Hello world" {
		t.Errorf("Message content = %q, want %q", msg.Content, "Hello world")
	}
	if msg.Round != 1 {
		t.Errorf("Message round = %d, want 1", msg.Round)
	}
	if msg.SequenceNum != 0 {
		t.Errorf("SequenceNum = %d, want 0", msg.SequenceNum)
	}
}

func TestDiscussionPhase_SendMessage_NotActive(t *testing.T) {
	players := [][32]byte{{1}}
	dp := NewDiscussionPhase([32]byte{1}, 1, players)

	// Don't start - should fail.
	_, err := dp.SendMessage([32]byte{1}, "Test")
	if err != ErrDiscussionNotActive {
		t.Errorf("Expected ErrDiscussionNotActive, got %v", err)
	}
}

func TestDiscussionPhase_SendMessage_NotPlayer(t *testing.T) {
	players := [][32]byte{{1}}
	dp := NewDiscussionPhase([32]byte{1}, 1, players)
	dp.Start()

	// Non-participant should fail.
	_, err := dp.SendMessage([32]byte{99}, "Test")
	if err != ErrDiscussionNotPlayer {
		t.Errorf("Expected ErrDiscussionNotPlayer, got %v", err)
	}
}

func TestDiscussionPhase_SendMessage_EmptyContent(t *testing.T) {
	players := [][32]byte{{1}}
	dp := NewDiscussionPhase([32]byte{1}, 1, players)
	dp.Start()

	_, err := dp.SendMessage([32]byte{1}, "")
	if err != ErrDiscussionMessageEmpty {
		t.Errorf("Expected ErrDiscussionMessageEmpty, got %v", err)
	}
}

func TestDiscussionPhase_SendMessage_TooLong(t *testing.T) {
	players := [][32]byte{{1}}
	dp := NewDiscussionPhase([32]byte{1}, 1, players)
	dp.Start()

	longContent := make([]byte, MaxMessageLength+1)
	for i := range longContent {
		longContent[i] = 'x'
	}

	_, err := dp.SendMessage([32]byte{1}, string(longContent))
	if err != ErrDiscussionMessageLong {
		t.Errorf("Expected ErrDiscussionMessageLong, got %v", err)
	}
}

func TestDiscussionPhase_SendMessage_RateLimit(t *testing.T) {
	players := [][32]byte{{1}}
	dp := NewDiscussionPhase([32]byte{1}, 1, players)
	dp.Start()

	// First message should succeed.
	_, err := dp.SendMessage([32]byte{1}, "First")
	if err != nil {
		t.Fatalf("First message failed: %v", err)
	}

	// Immediate second message should be rate limited.
	_, err = dp.SendMessage([32]byte{1}, "Second")
	if err != ErrDiscussionRateLimit {
		t.Errorf("Expected ErrDiscussionRateLimit, got %v", err)
	}
}

func TestDiscussionPhase_SendMessage_MessageLimit(t *testing.T) {
	player := [32]byte{1}
	players := [][32]byte{player}
	dp := NewDiscussionPhase([32]byte{1}, 1, players)
	dp.Start()

	// Manually set message count to limit using the full 32-byte key.
	dp.mu.Lock()
	dp.playerMessages[mechanics.KeyToHex(player[:])] = MaxMessagesPerPhase
	dp.mu.Unlock()

	_, err := dp.SendMessage(player, "Test")
	if err != ErrDiscussionLimitReached {
		t.Errorf("Expected ErrDiscussionLimitReached, got %v", err)
	}
}

func TestDiscussionPhase_GetMessages(t *testing.T) {
	players := [][32]byte{{1}, {2}}
	dp := NewDiscussionPhase([32]byte{1}, 1, players)
	dp.Start()

	// Send messages from different players.
	dp.SendMessage([32]byte{1}, "Hello from 1")
	time.Sleep(MessageRateLimit + 10*time.Millisecond)
	dp.SendMessage([32]byte{2}, "Hello from 2")

	messages := dp.GetMessages()
	if len(messages) != 2 {
		t.Errorf("MessageCount = %d, want 2", len(messages))
	}
}

func TestDiscussionPhase_GetMessagesSince(t *testing.T) {
	players := [][32]byte{{1}}
	dp := NewDiscussionPhase([32]byte{1}, 1, players)
	dp.Start()

	dp.SendMessage([32]byte{1}, "First")
	time.Sleep(MessageRateLimit + 10*time.Millisecond)
	dp.SendMessage([32]byte{1}, "Second")
	time.Sleep(MessageRateLimit + 10*time.Millisecond)
	dp.SendMessage([32]byte{1}, "Third")

	// Get messages since sequence 0 (should get 1, 2).
	since := dp.GetMessagesSince(0)
	if len(since) != 2 {
		t.Errorf("GetMessagesSince(0) returned %d messages, want 2", len(since))
	}
}

func TestDiscussionPhase_TimeRemaining(t *testing.T) {
	dp := NewDiscussionPhase([32]byte{1}, 1, [][32]byte{{1}})
	dp.Start()

	remaining := dp.TimeRemaining()
	if remaining <= 0 || remaining > DiscussionDuration {
		t.Errorf("TimeRemaining = %v, expected in (0, %v]", remaining, DiscussionDuration)
	}
}

func TestDiscussionPhase_End(t *testing.T) {
	dp := NewDiscussionPhase([32]byte{1}, 1, [][32]byte{{1}})
	dp.Start()

	dp.End()

	if dp.GetState() != DiscussionEnded {
		t.Errorf("State after End = %v, want DiscussionEnded", dp.GetState())
	}
	if dp.IsActive() {
		t.Error("IsActive() should be false after End")
	}
}

func TestDiscussionPhase_Reset(t *testing.T) {
	dp := NewDiscussionPhase([32]byte{1}, 1, [][32]byte{{1}})
	dp.Start()
	dp.SendMessage([32]byte{1}, "Test")
	dp.End()

	dp.Reset(2)

	if dp.Round != 2 {
		t.Errorf("Round after Reset = %d, want 2", dp.Round)
	}
	if dp.GetState() != DiscussionInactive {
		t.Errorf("State after Reset = %v, want DiscussionInactive", dp.GetState())
	}
	if dp.MessageCount() != 0 {
		t.Errorf("MessageCount after Reset = %d, want 0", dp.MessageCount())
	}
}

func TestDiscussionPhase_AddRemoveParticipant(t *testing.T) {
	dp := NewDiscussionPhase([32]byte{1}, 1, [][32]byte{{1}})

	if dp.ParticipantCount() != 1 {
		t.Errorf("Initial ParticipantCount = %d, want 1", dp.ParticipantCount())
	}

	dp.AddParticipant([32]byte{2})
	if dp.ParticipantCount() != 2 {
		t.Errorf("ParticipantCount after Add = %d, want 2", dp.ParticipantCount())
	}

	dp.RemoveParticipant([32]byte{1})
	if dp.ParticipantCount() != 1 {
		t.Errorf("ParticipantCount after Remove = %d, want 1", dp.ParticipantCount())
	}
}

func TestDiscussionPhase_OnMessage(t *testing.T) {
	dp := NewDiscussionPhase([32]byte{1}, 1, [][32]byte{{1}})

	var received int32
	dp.OnMessage(func(msg *DiscussionMessage) {
		atomic.AddInt32(&received, 1)
	})

	dp.Start()
	dp.SendMessage([32]byte{1}, "Test")

	if atomic.LoadInt32(&received) != 1 {
		t.Error("OnMessage callback should have been called")
	}
}

func TestDiscussionPhase_OnPhaseChange(t *testing.T) {
	dp := NewDiscussionPhase([32]byte{1}, 1, [][32]byte{{1}})

	var changes int32
	dp.OnPhaseChange(func(old, new DiscussionPhaseState) {
		atomic.AddInt32(&changes, 1)
	})

	dp.Start() // Inactive -> Active
	dp.End()   // Active -> Ended

	if atomic.LoadInt32(&changes) != 2 {
		t.Errorf("OnPhaseChange called %d times, want 2", atomic.LoadInt32(&changes))
	}
}

func TestDiscussionPhase_Update(t *testing.T) {
	dp := NewDiscussionPhase([32]byte{1}, 1, [][32]byte{{1}})

	// Update on inactive should not panic.
	dp.Update()

	dp.Start()
	dp.Update()

	// State should still be active (not enough time passed).
	if dp.GetState() != DiscussionActive {
		t.Errorf("State = %v, want DiscussionActive", dp.GetState())
	}
}

func TestNewShadowPlayCommunication(t *testing.T) {
	var initiator [32]byte
	copy(initiator[:], "test-initiator-12345678901234567")

	game, err := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)
	if err != nil {
		t.Fatalf("NewShadowPlay failed: %v", err)
	}

	spc := NewShadowPlayCommunication(game)
	if spc == nil {
		t.Fatal("NewShadowPlayCommunication returned nil")
	}
	if spc.CurrentDiscussion() != nil {
		t.Error("CurrentDiscussion should be nil initially")
	}
}

func TestShadowPlayCommunication_StartDiscussion(t *testing.T) {
	var initiator [32]byte
	copy(initiator[:], "test-initiator-12345678901234567")

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)

	// Add some players.
	game.Join([32]byte{1})
	game.Join([32]byte{2})
	game.Join([32]byte{3})

	spc := NewShadowPlayCommunication(game)
	dp := spc.StartDiscussion()

	if dp == nil {
		t.Fatal("StartDiscussion returned nil")
	}
	if !dp.IsActive() {
		t.Error("Discussion should be active after StartDiscussion")
	}
	if spc.CurrentDiscussion() != dp {
		t.Error("CurrentDiscussion should return started discussion")
	}
}

func TestShadowPlayCommunication_GetDiscussionForRound(t *testing.T) {
	var initiator [32]byte
	copy(initiator[:], "test-initiator-12345678901234567")

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)
	game.Join([32]byte{1})

	spc := NewShadowPlayCommunication(game)
	dp := spc.StartDiscussion()

	// Game starts with CurrentRound = 0, so discussion is for round 0.
	retrieved := spc.GetDiscussionForRound(0)
	if retrieved != dp {
		t.Error("GetDiscussionForRound(0) should return the started discussion")
	}

	nilRound := spc.GetDiscussionForRound(99)
	if nilRound != nil {
		t.Error("GetDiscussionForRound(99) should return nil")
	}
}

func TestShadowPlayCommunication_EndDiscussion(t *testing.T) {
	var initiator [32]byte
	copy(initiator[:], "test-initiator-12345678901234567")

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)
	game.Join([32]byte{1})

	spc := NewShadowPlayCommunication(game)
	spc.StartDiscussion()

	spc.EndDiscussion()

	if spc.CurrentDiscussion().IsActive() {
		t.Error("Discussion should not be active after EndDiscussion")
	}
}

func TestShadowPlayCommunication_OnElimination(t *testing.T) {
	var initiator [32]byte
	copy(initiator[:], "test-initiator-12345678901234567")

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)
	game.Join([32]byte{1})
	game.Join([32]byte{2})

	spc := NewShadowPlayCommunication(game)
	dp := spc.StartDiscussion()

	initialCount := dp.ParticipantCount()

	spc.OnElimination([32]byte{1})

	if dp.ParticipantCount() != initialCount-1 {
		t.Errorf("ParticipantCount = %d, want %d", dp.ParticipantCount(), initialCount-1)
	}
}

func TestShadowPlayCommunication_Update(t *testing.T) {
	var initiator [32]byte
	copy(initiator[:], "test-initiator-12345678901234567")

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)

	spc := NewShadowPlayCommunication(game)

	// Update without active discussion should not panic.
	spc.Update()

	spc.StartDiscussion()
	spc.Update()
}

func TestShadowPlayCommunication_TotalMessageCount(t *testing.T) {
	var initiator [32]byte
	copy(initiator[:], "test-initiator-12345678901234567")

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)
	game.Join([32]byte{1})

	spc := NewShadowPlayCommunication(game)
	spc.StartDiscussion()

	// Send a message.
	spc.CurrentDiscussion().SendMessage([32]byte{1}, "Test")

	if spc.TotalMessageCount() != 1 {
		t.Errorf("TotalMessageCount = %d, want 1", spc.TotalMessageCount())
	}
}

func TestShadowPlayCommunication_GetAllMessages(t *testing.T) {
	var initiator [32]byte
	copy(initiator[:], "test-initiator-12345678901234567")

	game, _ := NewShadowPlay(initiator, ShadowPlayDuration30Min, 5)
	game.Join([32]byte{1})

	spc := NewShadowPlayCommunication(game)
	spc.StartDiscussion()
	spc.CurrentDiscussion().SendMessage([32]byte{1}, "Round 1 message")

	msgs := spc.GetAllMessages()
	if len(msgs) != 1 {
		t.Errorf("GetAllMessages returned %d messages, want 1", len(msgs))
	}
}

func TestShadowPlayCommunication_NilGame(t *testing.T) {
	spc := NewShadowPlayCommunication(nil)

	dp := spc.StartDiscussion()
	if dp != nil {
		t.Error("StartDiscussion with nil game should return nil")
	}
}
