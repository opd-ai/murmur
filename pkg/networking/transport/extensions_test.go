package transport

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	gtransport "github.com/libp2p/go-libp2p/core/transport"
)

func TestRegisterAdapter(t *testing.T) {
	resetAdapterRegistry()
	t.Cleanup(resetAdapterRegistry)

	constructor := func(context.Context, gtransport.Upgrader, network.ResourceManager) (gtransport.Transport, error) {
		return nil, nil
	}

	if err := RegisterAdapter("mock", constructor); err != nil {
		t.Fatalf("RegisterAdapter failed: %v", err)
	}

	names := RegisteredAdapters()
	if len(names) != 1 || names[0] != "mock" {
		t.Fatalf("unexpected adapter names: %#v", names)
	}

	if err := RegisterAdapter("mock", constructor); err == nil {
		t.Fatal("expected duplicate adapter error")
	}
	if err := RegisterAdapter("", constructor); err == nil {
		t.Fatal("expected empty adapter name error")
	}
	if err := RegisterAdapter("nil", nil); err == nil {
		t.Fatal("expected nil constructor error")
	}
}

func TestAppendRegisteredAdapters(t *testing.T) {
	resetAdapterRegistry()
	t.Cleanup(resetAdapterRegistry)

	constructor := func(context.Context, gtransport.Upgrader, network.ResourceManager) (gtransport.Transport, error) {
		return nil, nil
	}
	if err := RegisterAdapter("mock", constructor); err != nil {
		t.Fatalf("RegisterAdapter failed: %v", err)
	}

	base := []libp2p.Option{}
	options := appendRegisteredAdapters(base, context.Background())
	if len(options) != 1 {
		t.Fatalf("expected one adapter option, got %d", len(options))
	}
}
