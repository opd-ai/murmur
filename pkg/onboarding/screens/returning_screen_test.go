// Package screens provides tests for Returning User screen.

//go:build noebiten
// +build noebiten

package screens

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/stretchr/testify/assert"
)

func TestReturningScreen(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	assert.NoError(t, err)

	continueCalled := false
	screen := NewReturningScreen(
		"TestUser",
		kp,
		func() { continueCalled = true },
	)

	assert.NotNil(t, screen)

	// Update should trigger callback after 2 seconds
	for i := 0; i < 130; i++ {
		err := screen.Update()
		assert.NoError(t, err)
	}

	assert.True(t, continueCalled, "callback should be called after timeout")
}

func TestReturningScreenLayout(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	assert.NoError(t, err)

	screen := NewReturningScreen("TestUser", kp, nil)

	w, h := screen.Layout(800, 600)
	assert.Equal(t, 800, w)
	assert.Equal(t, 600, h)
}

func TestReturningScreenDraw(t *testing.T) {
	kp, err := keys.GenerateKeyPair()
	assert.NoError(t, err)

	screen := NewReturningScreen("TestUser", kp, nil)
	img := ebiten.NewImage(800, 600)

	// Should not panic
	assert.NotPanics(t, func() {
		screen.Draw(img)
	})
}
