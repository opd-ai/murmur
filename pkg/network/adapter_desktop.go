//go:build !js

package network

import "context"

type desktopAdapter struct {
	cfg Config
}

func newDesktopAdapter(cfg Config) (Adapter, error) {
	return &desktopAdapter{cfg: cfg}, nil
}

func (a *desktopAdapter) Start(context.Context) error {
	return ErrNotImplemented
}

func (a *desktopAdapter) Stop(context.Context) error {
	return nil
}

func (a *desktopAdapter) Publish(context.Context, string, []byte) error {
	return ErrNotImplemented
}

func (a *desktopAdapter) Subscribe(string) (<-chan Message, error) {
	return nil, ErrNotImplemented
}

func (a *desktopAdapter) DialPeer(context.Context, string) error {
	return ErrNotImplemented
}

func (a *desktopAdapter) Name() string {
	return "libp2p-desktop"
}
