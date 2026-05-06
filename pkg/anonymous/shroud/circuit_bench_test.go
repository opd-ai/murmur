package shroud

import (
	"fmt"
	"testing"
)

// BenchmarkCircuitConstruction measures full circuit build time.
// Per TECHNICAL_IMPLEMENTATION.md, target is <3 seconds.
func BenchmarkCircuitConstruction(b *testing.B) {
	beacon, err := NewBeacon()
	if err != nil {
		b.Fatalf("NewBeacon failed: %v", err)
	}

	// Add 20 relays to ensure sufficient diversity.
	for i := 0; i < 20; i++ {
		relay := &RelayInfo{
			PeerID:    string(rune('a' + i)),
			Bandwidth: 1000000,
		}
		beacon.AddRelay(relay)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		relays, err := beacon.SelectRelays(nil)
		if err != nil {
			b.Fatalf("SelectRelays failed: %v", err)
		}

		circuit, err := beacon.BuildCircuit(relays)
		if err != nil {
			b.Fatalf("BuildCircuit failed: %v", err)
		}

		// Clean up circuit to avoid resource leaks.
		circuit.Close()
	}
}

// BenchmarkSelectRelays measures relay selection time.
func BenchmarkSelectRelays(b *testing.B) {
	beacon, err := NewBeacon()
	if err != nil {
		b.Fatalf("NewBeacon failed: %v", err)
	}

	for i := 0; i < 100; i++ {
		relay := &RelayInfo{
			PeerID:    fmt.Sprintf("%c%d", rune('a'+(i%26)), i),
			Bandwidth: 1000000,
		}
		beacon.AddRelay(relay)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := beacon.SelectRelays(nil)
		if err != nil {
			b.Fatalf("SelectRelays failed: %v", err)
		}
	}
}

// BenchmarkSelectRelaysWithExclusion measures relay selection with large exclusion lists.
func BenchmarkSelectRelaysWithExclusion(b *testing.B) {
	beacon, err := NewBeacon()
	if err != nil {
		b.Fatalf("NewBeacon failed: %v", err)
	}

	// Add 100 relays.
	for i := 0; i < 100; i++ {
		relay := &RelayInfo{
			PeerID:    fmt.Sprintf("%c%d", rune('a'+(i%26)), i),
			Bandwidth: 1000000,
		}
		beacon.AddRelay(relay)
	}

	// Exclude 50 relays.
	exclude := make([]string, 50)
	for i := 0; i < 50; i++ {
		exclude[i] = string(rune('a'+(i%26))) + fmt.Sprintf("%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := beacon.SelectRelays(exclude)
		if err != nil {
			b.Fatalf("SelectRelays failed: %v", err)
		}
	}
}

// BenchmarkBuildCircuit measures circuit construction time.
func BenchmarkBuildCircuit(b *testing.B) {
	beacon, err := NewBeacon()
	if err != nil {
		b.Fatalf("NewBeacon failed: %v", err)
	}

	for i := 0; i < 20; i++ {
		relay := &RelayInfo{
			PeerID:    string(rune('a' + i)),
			Bandwidth: 1000000,
		}
		beacon.AddRelay(relay)
	}

	relays, err := beacon.SelectRelays(nil)
	if err != nil {
		b.Fatalf("SelectRelays failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		circuit, err := beacon.BuildCircuit(relays)
		if err != nil {
			b.Fatalf("BuildCircuit failed: %v", err)
		}
		circuit.Close()
	}
}

// BenchmarkCircuitSend measures packet sending through a circuit.
func BenchmarkCircuitSend(b *testing.B) {
	beacon, err := NewBeacon()
	if err != nil {
		b.Fatalf("NewBeacon failed: %v", err)
	}

	for i := 0; i < 20; i++ {
		relay := &RelayInfo{
			PeerID:    string(rune('a' + i)),
			Bandwidth: 1000000,
		}
		beacon.AddRelay(relay)
	}

	relays, err := beacon.SelectRelays(nil)
	if err != nil {
		b.Fatalf("SelectRelays failed: %v", err)
	}

	circuit, err := beacon.BuildCircuit(relays)
	if err != nil {
		b.Fatalf("BuildCircuit failed: %v", err)
	}
	defer circuit.Close()

	payload := []byte("benchmark test payload")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := circuit.Encrypt(payload)
		if err != nil {
			b.Fatalf("Encrypt failed: %v", err)
		}
	}
}

// BenchmarkCircuitDecryptLayer measures layer decryption through a circuit.
func BenchmarkCircuitDecryptLayer(b *testing.B) {
	beacon, err := NewBeacon()
	if err != nil {
		b.Fatalf("NewBeacon failed: %v", err)
	}

	for i := 0; i < 20; i++ {
		relay := &RelayInfo{
			PeerID:    string(rune('a' + i)),
			Bandwidth: 1000000,
		}
		beacon.AddRelay(relay)
	}

	relays, err := beacon.SelectRelays(nil)
	if err != nil {
		b.Fatalf("SelectRelays failed: %v", err)
	}

	circuit, err := beacon.BuildCircuit(relays)
	if err != nil {
		b.Fatalf("BuildCircuit failed: %v", err)
	}
	defer circuit.Close()

	payload := []byte("benchmark test payload")
	encrypted, err := circuit.Encrypt(payload)
	if err != nil {
		b.Fatalf("Encrypt failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := circuit.DecryptLayerWithReplayCheck(encrypted, 0)
		if err != nil {
			b.Fatalf("DecryptLayerWithReplayCheck failed: %v", err)
		}
	}
}
