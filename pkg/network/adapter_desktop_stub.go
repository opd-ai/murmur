//go:build js && wasm

package network

func newDesktopAdapter(Config) (Adapter, error) {
	return nil, ErrNotImplemented
}
