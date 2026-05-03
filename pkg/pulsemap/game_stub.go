// Package pulsemap provides the force-directed graph visualization (Pulse Map).
// This file provides a stub Game implementation for test builds.
//
//go:build noebiten
// +build noebiten

package pulsemap

import (
	"context"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
)

// Game is a stub for test builds without Ebitengine.
type Game struct{}

// NewGame creates a stub game instance for test builds.
func NewGame(ctx context.Context, keypair *keys.KeyPair, pubsub *gossip.PubSub) (*Game, error) {
	return &Game{}, nil
}

// Update is a no-op stub.
func (g *Game) Update() error {
	return nil
}

// Draw is a no-op stub.
func (g *Game) Draw(screen interface{}) {
}

// Layout returns fixed dimensions for test builds.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}

// Shutdown is a no-op stub.
func (g *Game) Shutdown() {
}
