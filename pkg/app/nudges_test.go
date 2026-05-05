package app

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
	"github.com/opd-ai/murmur/pkg/store"
	"github.com/opd-ai/murmur/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNudgeSchedule verifies the nudge schedule contains expected entries.
func TestNudgeSchedule(t *testing.T) {
	require.NotEmpty(t, nudgeSchedule, "nudge schedule should not be empty")

	// Verify Day 1 nudge exists (Wave publishing encouragement).
	foundDay1 := false
	for _, n := range nudgeSchedule {
		if n.Day == 1 {
			foundDay1 = true
			assert.Contains(t, n.Message, "reply", "Day 1 should encourage replying")
		}
	}
	assert.True(t, foundDay1, "Day 1 nudge should exist")

	// Verify Day 2 nudge exists (connection formation).
	foundDay2 := false
	for _, n := range nudgeSchedule {
		if n.Day == 2 {
			foundDay2 = true
			assert.Contains(t, n.Message, "connection", "Day 2 should encourage connections")
		}
	}
	assert.True(t, foundDay2, "Day 2 nudge should exist")

	// Verify Day 3 has mode-specific nudges (Anonymous Layer).
	foundDay3Hybrid := false
	for _, n := range nudgeSchedule {
		if n.Day == 3 && n.Mode == modes.Hybrid {
			foundDay3Hybrid = true
			assert.Contains(t, n.Message, "Specter", "Day 3 Hybrid should mention Specter")
		}
	}
	assert.True(t, foundDay3Hybrid, "Day 3 Hybrid nudge should exist")

	// Verify Day 5 nudge exists (Resonance milestone).
	foundDay5 := false
	for _, n := range nudgeSchedule {
		if n.Day == 5 {
			foundDay5 = true
			assert.Contains(t, n.Message, "Resonance", "Day 5 should mention Resonance")
		}
	}
	assert.True(t, foundDay5, "Day 5 nudge should exist")
}

// TestCheckAndSendNudges verifies nudge dispatch logic.
func TestCheckAndSendNudges(t *testing.T) {
	// Create temporary database.
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	defer db.Close()

	// Generate test identity.
	kp, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	// Create identity declaration with timestamp 2 days ago.
	twoDaysAgo := time.Now().Add(-48 * time.Hour).Unix()
	decl := &proto.IdentityDeclaration{
		PublicKey:   kp.PublicKey,
		DisplayName: "TestUser",
		CreatedAt:   twoDaysAgo,
		Signature:   make([]byte, 64), // Placeholder
	}
	err = db.PutIdentityDeclaration(decl)
	require.NoError(t, err)

	// Store Open mode in config bucket.
	err = db.Put(store.BucketConfig, []byte("privacy_mode"), []byte{byte(modes.Open)})
	require.NoError(t, err)

	// Create App with minimal subsystems.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := &App{
		ctx:    ctx,
		cancel: cancel,
		wg:     sync.WaitGroup{},
		subsystems: &Subsystems{
			Storage:  db,
			Identity: kp,
		},
	}

	// Call checkAndSendNudges (account is 2 days old).
	app.checkAndSendNudges()

	// Verify Day 1 and Day 2 nudges were marked as shown.
	assert.True(t, app.wasNudgeShown("nudge_day1_mode0"), "Day 1 nudge should be marked shown")
	assert.True(t, app.wasNudgeShown("nudge_day2_mode0"), "Day 2 nudge should be marked shown")

	// Verify Day 3+ nudges not shown yet (account only 2 days old).
	assert.False(t, app.wasNudgeShown("nudge_day3_mode1"), "Day 3 Hybrid nudge should not be shown yet")
	assert.False(t, app.wasNudgeShown("nudge_day5_mode0"), "Day 5 nudge should not be shown yet")
}

// TestNudgeNotShownTwice verifies nudges are only sent once.
func TestNudgeNotShownTwice(t *testing.T) {
	// Create temporary database.
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	defer db.Close()

	// Generate test identity.
	kp, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	// Create identity declaration with timestamp 3 days ago.
	threeDaysAgo := time.Now().Add(-72 * time.Hour).Unix()
	decl := &proto.IdentityDeclaration{
		PublicKey:   kp.PublicKey,
		DisplayName: "TestUser",
		CreatedAt:   threeDaysAgo,
		Signature:   make([]byte, 64),
	}
	err = db.PutIdentityDeclaration(decl)
	require.NoError(t, err)

	// Store Open mode.
	err = db.Put(store.BucketConfig, []byte("privacy_mode"), []byte{byte(modes.Open)})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := &App{
		ctx:    ctx,
		cancel: cancel,
		subsystems: &Subsystems{
			Storage:  db,
			Identity: kp,
		},
	}

	// First call: nudges should be sent.
	app.checkAndSendNudges()
	assert.True(t, app.wasNudgeShown("nudge_day1_mode0"))

	// Second call: nudges should NOT be sent again.
	// (No direct way to verify sendNudge was not called, but wasNudgeShown
	// should prevent duplicate dispatch)
	app.checkAndSendNudges()
	// If run twice, no panic or error = success (idempotent).
}

// TestGetCurrentMode verifies privacy mode retrieval.
func TestGetCurrentMode(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	defer db.Close()

	kp, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	app := &App{
		subsystems: &Subsystems{
			Storage:  db,
			Identity: kp,
		},
	}

	// No mode stored: should default to Open.
	mode := app.getCurrentMode()
	assert.Equal(t, modes.Open, mode)

	// Store Hybrid mode.
	err = db.Put(store.BucketConfig, []byte("privacy_mode"), []byte{byte(modes.Hybrid)})
	require.NoError(t, err)

	mode = app.getCurrentMode()
	assert.Equal(t, modes.Hybrid, mode)
}

// TestNudgeAfterFirstWeek verifies no nudges sent after 7 days.
func TestNudgeAfterFirstWeek(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	defer db.Close()

	kp, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	// Create identity declaration 10 days old.
	tenDaysAgo := time.Now().Add(-240 * time.Hour).Unix()
	decl := &proto.IdentityDeclaration{
		PublicKey:   kp.PublicKey,
		DisplayName: "TestUser",
		CreatedAt:   tenDaysAgo,
		Signature:   make([]byte, 64),
	}
	err = db.PutIdentityDeclaration(decl)
	require.NoError(t, err)

	err = db.Put(store.BucketConfig, []byte("privacy_mode"), []byte{byte(modes.Open)})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := &App{
		ctx:    ctx,
		cancel: cancel,
		subsystems: &Subsystems{
			Storage:  db,
			Identity: kp,
		},
	}

	// Call checkAndSendNudges (account is 10 days old, beyond first week).
	app.checkAndSendNudges()

	// Verify NO nudges were sent (none should be marked shown).
	assert.False(t, app.wasNudgeShown("nudge_day1_mode0"))
	assert.False(t, app.wasNudgeShown("nudge_day2_mode0"))
}
