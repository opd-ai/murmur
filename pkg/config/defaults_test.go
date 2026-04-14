// Package config tests verify configuration defaults and validation.
package config

import "testing"

// TestDefaultListenAddrs verifies default listen addresses are set.
func TestDefaultListenAddrs(t *testing.T) {
	if len(DefaultListenAddrs) == 0 {
		t.Error("DefaultListenAddrs should not be empty")
	}

	// Per TECHNICAL_IMPLEMENTATION.md, should support TCP and QUIC.
	foundTCP := false
	foundQUIC := false

	for _, addr := range DefaultListenAddrs {
		if containsString(addr, "tcp") {
			foundTCP = true
		}
		if containsString(addr, "quic") {
			foundQUIC = true
		}
	}

	if !foundTCP {
		t.Error("DefaultListenAddrs should include TCP multiaddr")
	}
	if !foundQUIC {
		t.Error("DefaultListenAddrs should include QUIC multiaddr")
	}
}

// TestDefaultBootstrapPeers verifies bootstrap peers configuration exists.
// Note: Actual bootstrap nodes require infrastructure (see AUDIT.md BLOCKED item).
func TestDefaultBootstrapPeers(t *testing.T) {
	// DefaultBootstrapPeers is currently empty pending infrastructure setup.
	// This test documents the expected structure.
	if DefaultBootstrapPeers == nil {
		t.Error("DefaultBootstrapPeers should not be nil")
	}
	// When infrastructure is available, this should be updated to:
	// if len(DefaultBootstrapPeers) == 0 {
	//     t.Error("DefaultBootstrapPeers should not be empty")
	// }
}

// containsString checks if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && containsAt(s, substr)
}

// containsAt checks if substr appears anywhere in s.
func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
