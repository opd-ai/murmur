//go:build !js

package network

func newWASMAdapter(Config) (Adapter, error) {
	return nil, ErrNotImplemented
}
