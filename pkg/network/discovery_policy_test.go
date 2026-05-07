package network

import (
	"reflect"
	"testing"
)

func TestBuildBrowserDiscoveryPeersRelayFirst(t *testing.T) {
	t.Parallel()

	relayPeers := []string{"relay-a", "relay-b"}
	bootstrapPeers := []string{"boot-a", "boot-b"}

	got := buildBrowserDiscoveryPeers(relayPeers, bootstrapPeers)
	want := []string{"relay-a", "relay-b", "boot-a", "boot-b"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildBrowserDiscoveryPeers() = %v, want %v", got, want)
	}
}

func TestBuildBrowserDiscoveryPeersDeduplicatesAndSkipsEmpty(t *testing.T) {
	t.Parallel()

	relayPeers := []string{"relay-a", "", "relay-b", "relay-a"}
	bootstrapPeers := []string{"", "boot-a", "relay-b", "boot-a", "boot-b"}

	got := buildBrowserDiscoveryPeers(relayPeers, bootstrapPeers)
	want := []string{"relay-a", "relay-b", "boot-a", "boot-b"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildBrowserDiscoveryPeers() = %v, want %v", got, want)
	}
}
