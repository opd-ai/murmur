package waves

import (
	"fmt"
	"sort"
	"sync"

	pb "github.com/opd-ai/murmur/proto"
)

const (
	// TypeExtensionStart is the first custom Wave type ID reserved for extensions.
	TypeExtensionStart WaveType = 0x40
	// TypeExtensionEnd is the last custom Wave type ID reserved for extensions.
	TypeExtensionEnd WaveType = 0xFF
)

// ExtensionHandler validates and renders extension Wave payloads.
type ExtensionHandler interface {
	Validate(wave *pb.Wave) error
	Render(wave *pb.Wave) ([]byte, error)
}

var (
	errReservedWaveType   = fmt.Errorf("wave type is reserved for core MURMUR")
	errInvalidWaveType    = fmt.Errorf("wave type is outside extension range")
	errNilExtensionHandle = fmt.Errorf("extension handler is nil")

	waveExtensionMu       sync.RWMutex
	waveExtensionRegistry = make(map[WaveType]ExtensionHandler)
)

// RegisterWaveType registers a custom Wave type handler in the extension range.
func RegisterWaveType(typ WaveType, handler ExtensionHandler) error {
	if !isExtensionType(typ) {
		if typ < TypeExtensionStart {
			return errReservedWaveType
		}
		return errInvalidWaveType
	}
	if handler == nil {
		return errNilExtensionHandle
	}

	waveExtensionMu.Lock()
	defer waveExtensionMu.Unlock()

	if _, exists := waveExtensionRegistry[typ]; exists {
		return fmt.Errorf("wave type 0x%02x already registered", typ)
	}
	waveExtensionRegistry[typ] = handler
	return nil
}

// RegisteredWaveTypes returns the registered extension Wave type IDs in ascending order.
func RegisteredWaveTypes() []WaveType {
	waveExtensionMu.RLock()
	defer waveExtensionMu.RUnlock()

	types := make([]WaveType, 0, len(waveExtensionRegistry))
	for typ := range waveExtensionRegistry {
		types = append(types, typ)
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i] < types[j]
	})
	return types
}

func validateExtensionWave(wave *pb.Wave) error {
	typ := WaveType(wave.GetWaveType())
	if !isExtensionType(typ) {
		return nil
	}

	handler, ok := lookupWaveExtensionHandler(typ)
	if !ok {
		return fmt.Errorf("wave type 0x%02x has no registered extension handler", typ)
	}
	return handler.Validate(wave)
}

func isExtensionType(typ WaveType) bool {
	return typ >= TypeExtensionStart && typ <= TypeExtensionEnd
}

func lookupWaveExtensionHandler(typ WaveType) (ExtensionHandler, bool) {
	waveExtensionMu.RLock()
	defer waveExtensionMu.RUnlock()
	handler, ok := waveExtensionRegistry[typ]
	return handler, ok
}

func resetWaveTypeRegistry() {
	waveExtensionMu.Lock()
	defer waveExtensionMu.Unlock()
	waveExtensionRegistry = make(map[WaveType]ExtensionHandler)
}
