// Package overlays — Shadow Play overlay tests.
//
//go:build noebiten
// +build noebiten

package overlays

import (
	"testing"
	"time"
)

func TestNewShadowPlayOverlay(t *testing.T) {
	overlay := NewShadowPlayOverlay()
	if overlay == nil {
		t.Fatal("NewShadowPlayOverlay returned nil")
	}
	if overlay.GameCount() != 0 {
		t.Errorf("GameCount = %d, want 0", overlay.GameCount())
	}
}

func TestShadowPlayOverlay_AddGame(t *testing.T) {
	overlay := NewShadowPlayOverlay()

	gameID := [32]byte{1, 2, 3}
	info := &ShadowPlayInfo{
		GameID:    gameID,
		State:     ShadowPlayActive,
		X:         100,
		Y:         200,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(30 * time.Minute),
		Players: []ShadowPlayer{
			{SpecterKey: [32]byte{1}, Role: ShadowRoleEcho, X: 90, Y: 190},
			{SpecterKey: [32]byte{2}, Role: ShadowRoleShade, X: 110, Y: 210},
		},
	}

	overlay.AddGame(info)

	if overlay.GameCount() != 1 {
		t.Errorf("GameCount = %d, want 1", overlay.GameCount())
	}

	retrieved := overlay.GetGame(gameID)
	if retrieved == nil {
		t.Fatal("GetGame returned nil")
	}
	if retrieved.State != ShadowPlayActive {
		t.Errorf("State = %v, want ShadowPlayActive", retrieved.State)
	}
}

func TestShadowPlayOverlay_AddGame_Nil(t *testing.T) {
	overlay := NewShadowPlayOverlay()
	overlay.AddGame(nil) // Should not panic.

	if overlay.GameCount() != 0 {
		t.Errorf("GameCount = %d, want 0 after adding nil", overlay.GameCount())
	}
}

func TestShadowPlayOverlay_RemoveGame(t *testing.T) {
	overlay := NewShadowPlayOverlay()

	gameID := [32]byte{1, 2, 3}
	info := &ShadowPlayInfo{
		GameID: gameID,
		State:  ShadowPlayActive,
	}

	overlay.AddGame(info)
	if overlay.GameCount() != 1 {
		t.Fatalf("GameCount = %d after add, want 1", overlay.GameCount())
	}

	overlay.RemoveGame(gameID)
	if overlay.GameCount() != 0 {
		t.Errorf("GameCount = %d after remove, want 0", overlay.GameCount())
	}

	if overlay.GetGame(gameID) != nil {
		t.Error("GetGame should return nil after removal")
	}
}

func TestShadowPlayOverlay_UpdateGame(t *testing.T) {
	overlay := NewShadowPlayOverlay()

	gameID := [32]byte{1, 2, 3}
	info := &ShadowPlayInfo{
		GameID:      gameID,
		State:       ShadowPlayActive,
		RoundNumber: 1,
	}

	overlay.AddGame(info)

	// Update to voting state.
	updated := &ShadowPlayInfo{
		GameID:      gameID,
		State:       ShadowPlayVoting,
		RoundNumber: 2,
	}
	overlay.UpdateGame(updated)

	retrieved := overlay.GetGame(gameID)
	if retrieved.State != ShadowPlayVoting {
		t.Errorf("State = %v, want ShadowPlayVoting", retrieved.State)
	}
	if retrieved.RoundNumber != 2 {
		t.Errorf("RoundNumber = %d, want 2", retrieved.RoundNumber)
	}
}

func TestShadowPlayOverlay_UpdateGame_Nil(t *testing.T) {
	overlay := NewShadowPlayOverlay()
	overlay.UpdateGame(nil) // Should not panic.
}

func TestShadowPlayOverlay_Update(t *testing.T) {
	overlay := NewShadowPlayOverlay()

	gameID := [32]byte{1, 2, 3}
	info := &ShadowPlayInfo{
		GameID:    gameID,
		State:     ShadowPlayVoting,
		X:         100,
		Y:         200,
		StartTime: time.Now(),
	}

	overlay.AddGame(info)

	// Update should not panic.
	for i := 0; i < 10; i++ {
		overlay.Update()
	}
}

func TestShadowPlayOverlay_Clear(t *testing.T) {
	overlay := NewShadowPlayOverlay()

	// Add multiple games.
	for i := 0; i < 5; i++ {
		gameID := [32]byte{byte(i)}
		overlay.AddGame(&ShadowPlayInfo{
			GameID: gameID,
			State:  ShadowPlayActive,
		})
	}

	if overlay.GameCount() != 5 {
		t.Fatalf("GameCount = %d, want 5", overlay.GameCount())
	}

	overlay.Clear()

	if overlay.GameCount() != 0 {
		t.Errorf("GameCount = %d after Clear, want 0", overlay.GameCount())
	}
}

func TestShadowPlayOverlay_GetGame_NotFound(t *testing.T) {
	overlay := NewShadowPlayOverlay()

	result := overlay.GetGame([32]byte{99, 99, 99})
	if result != nil {
		t.Error("GetGame should return nil for non-existent game")
	}
}

func TestShadowPlayState_Values(t *testing.T) {
	// Verify state constants are defined correctly.
	states := []ShadowPlayState{
		ShadowPlayWaiting,
		ShadowPlayActive,
		ShadowPlayVoting,
		ShadowPlayEchoesWin,
		ShadowPlayShadesWin,
		ShadowPlayExpired,
	}

	for i, state := range states {
		if int(state) != i {
			t.Errorf("ShadowPlayState %d has value %d, want %d", i, state, i)
		}
	}
}

func TestShadowPlayerRole_Values(t *testing.T) {
	// Verify role constants are defined correctly.
	roles := []ShadowPlayerRole{
		ShadowRoleUnknown,
		ShadowRoleEcho,
		ShadowRoleShade,
	}

	for i, role := range roles {
		if int(role) != i {
			t.Errorf("ShadowPlayerRole %d has value %d, want %d", i, role, i)
		}
	}
}

func TestShadowPlayInfo_PlayerManagement(t *testing.T) {
	info := &ShadowPlayInfo{
		GameID: [32]byte{1},
		State:  ShadowPlayActive,
		Players: []ShadowPlayer{
			{SpecterKey: [32]byte{1}, Role: ShadowRoleEcho, X: 10, Y: 20},
			{SpecterKey: [32]byte{2}, Role: ShadowRoleShade, X: 30, Y: 40},
			{SpecterKey: [32]byte{3}, Role: ShadowRoleUnknown, X: 50, Y: 60, IsEliminated: true},
		},
	}

	if len(info.Players) != 3 {
		t.Errorf("Player count = %d, want 3", len(info.Players))
	}

	// Verify player properties.
	if info.Players[0].Role != ShadowRoleEcho {
		t.Error("First player should be Echo")
	}
	if info.Players[1].Role != ShadowRoleShade {
		t.Error("Second player should be Shade")
	}
	if !info.Players[2].IsEliminated {
		t.Error("Third player should be eliminated")
	}
}

func TestShadowPlayOverlay_MultipleGames(t *testing.T) {
	overlay := NewShadowPlayOverlay()

	// Add games with different states.
	games := []*ShadowPlayInfo{
		{GameID: [32]byte{1}, State: ShadowPlayWaiting, X: 0, Y: 0},
		{GameID: [32]byte{2}, State: ShadowPlayActive, X: 100, Y: 0},
		{GameID: [32]byte{3}, State: ShadowPlayVoting, X: 200, Y: 0},
		{GameID: [32]byte{4}, State: ShadowPlayEchoesWin, X: 300, Y: 0},
		{GameID: [32]byte{5}, State: ShadowPlayShadesWin, X: 400, Y: 0},
	}

	for _, game := range games {
		overlay.AddGame(game)
	}

	if overlay.GameCount() != 5 {
		t.Errorf("GameCount = %d, want 5", overlay.GameCount())
	}

	// Verify each game can be retrieved.
	for _, game := range games {
		retrieved := overlay.GetGame(game.GameID)
		if retrieved == nil {
			t.Errorf("Failed to retrieve game %v", game.GameID)
			continue
		}
		if retrieved.State != game.State {
			t.Errorf("Game %v state = %v, want %v", game.GameID, retrieved.State, game.State)
		}
	}
}

func TestShadowPlayOverlay_ConcurrentAccess(t *testing.T) {
	overlay := NewShadowPlayOverlay()

	done := make(chan bool)

	// Writer goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			gameID := [32]byte{byte(i % 10)}
			overlay.AddGame(&ShadowPlayInfo{
				GameID: gameID,
				State:  ShadowPlayState(i % 6),
			})
			overlay.Update()
		}
		done <- true
	}()

	// Reader goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			gameID := [32]byte{byte(i % 10)}
			_ = overlay.GetGame(gameID)
			_ = overlay.GameCount()
		}
		done <- true
	}()

	// Remover goroutine.
	go func() {
		for i := 0; i < 100; i++ {
			gameID := [32]byte{byte(i % 10)}
			overlay.RemoveGame(gameID)
		}
		done <- true
	}()

	// Wait for all goroutines.
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should not have panicked.
}

func TestShadowPlayer_Defaults(t *testing.T) {
	player := ShadowPlayer{}

	// Default values should be zero values.
	if player.Role != ShadowRoleUnknown {
		t.Errorf("Default Role = %v, want ShadowRoleUnknown", player.Role)
	}
	if player.IsEliminated {
		t.Error("Default IsEliminated should be false")
	}
	if player.X != 0 || player.Y != 0 {
		t.Errorf("Default position = (%f, %f), want (0, 0)", player.X, player.Y)
	}
}
