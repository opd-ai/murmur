//go:build android || ios

// Package identity provides mobile-specific sharing integration.
// On Android and iOS, native share sheets are invoked through a registered
// callback. The host application (cmd/murmur-mobile) registers a callback
// via [RegisterMobileShareHandler] after gomobile initialisation. If no
// callback is registered the content is written to the OS temp directory so
// that the native layer can pick it up through a well-known path.
package identity

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// MobileShareHandler is a callback invoked by [openSystemShareImpl] on mobile
// platforms to trigger the platform's native share sheet.
// content is the shareable text (URI, mailto URL, or file path).
// method identifies the sharing method used.
type MobileShareHandler func(content string, method ShareMethod) error

var (
	mobileHandlerMu sync.RWMutex
	mobileHandler   MobileShareHandler
)

// RegisterMobileShareHandler registers a native share sheet callback.
// The gomobile host application MUST call this during its initialisation
// sequence so that [OpenSystemShare] can invoke the platform's native UI.
// Calling this function with nil clears any previously registered handler.
func RegisterMobileShareHandler(h MobileShareHandler) {
	mobileHandlerMu.Lock()
	defer mobileHandlerMu.Unlock()
	mobileHandler = h
}

// openSystemShareImpl implements mobile share-sheet invocation.
// If a [MobileShareHandler] has been registered (by the gomobile host), it is
// called with the prepared content string. Otherwise the content is written to
// a well-known file in the OS temp directory so that native code can retrieve it.
func openSystemShareImpl(content string, opts ShareOptions) error {
	mobileHandlerMu.RLock()
	h := mobileHandler
	mobileHandlerMu.RUnlock()

	if h != nil {
		return h(content, opts.Method)
	}

	// No callback registered — write to temp file as a best-effort fallback.
	// The native host layer can watch this path and open the share sheet itself.
	tmpPath := filepath.Join(os.TempDir(), "murmur-share-pending.txt")
	if err := os.WriteFile(tmpPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing share payload to %s: %w", tmpPath, err)
	}
	return nil
}
