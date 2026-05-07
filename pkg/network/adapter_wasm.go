//go:build js && wasm

package network

import "context"

type wasmAdapter struct {
	cfg Config
}

func newWASMAdapter(cfg Config) (Adapter, error) {
	return &wasmAdapter{cfg: cfg}, nil
}

func (a *wasmAdapter) Start(context.Context) error {
	return ErrNotImplemented
}

func (a *wasmAdapter) Stop(context.Context) error {
	return nil
}

func (a *wasmAdapter) Publish(context.Context, string, []byte) error {
	return ErrNotImplemented
}

func (a *wasmAdapter) Subscribe(string) (<-chan Message, error) {
	return nil, ErrNotImplemented
}

func (a *wasmAdapter) DialPeer(context.Context, string) error {
	return ErrNotImplemented
}

func (a *wasmAdapter) Name() string {
	return "pion-webrtc-wasm"
}
