package transport

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	gtransport "github.com/libp2p/go-libp2p/core/transport"
)

// AdapterConstructor constructs a custom libp2p transport for host registration.
type AdapterConstructor func(ctx context.Context, upgrader gtransport.Upgrader, rcmgr network.ResourceManager) (gtransport.Transport, error)

var (
	adapterMu       sync.RWMutex
	adapterRegistry = make(map[string]AdapterConstructor)
)

// RegisterAdapter registers a named custom transport adapter for future hosts.
func RegisterAdapter(name string, constructor AdapterConstructor) error {
	if name == "" {
		return fmt.Errorf("adapter name is required")
	}
	if constructor == nil {
		return fmt.Errorf("adapter constructor is nil")
	}

	adapterMu.Lock()
	defer adapterMu.Unlock()

	if _, exists := adapterRegistry[name]; exists {
		return fmt.Errorf("adapter %q already registered", name)
	}
	adapterRegistry[name] = constructor
	return nil
}

// RegisteredAdapters returns the registered adapter names in ascending order.
func RegisteredAdapters() []string {
	adapterMu.RLock()
	defer adapterMu.RUnlock()

	names := make([]string, 0, len(adapterRegistry))
	for name := range adapterRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func appendRegisteredAdapters(opts []libp2p.Option, ctx context.Context) []libp2p.Option {
	adapterMu.RLock()
	names := make([]string, 0, len(adapterRegistry))
	constructors := make(map[string]AdapterConstructor, len(adapterRegistry))
	for name, constructor := range adapterRegistry {
		names = append(names, name)
		constructors[name] = constructor
	}
	adapterMu.RUnlock()

	sort.Strings(names)
	for _, name := range names {
		constructor := constructors[name]
		wrapped := func(upgrader gtransport.Upgrader, rcmgr network.ResourceManager) (gtransport.Transport, error) {
			return constructor(ctx, upgrader, rcmgr)
		}
		opts = append(opts, libp2p.Transport(wrapped))
	}
	return opts
}

func resetAdapterRegistry() {
	adapterMu.Lock()
	defer adapterMu.Unlock()
	adapterRegistry = make(map[string]AdapterConstructor)
}
