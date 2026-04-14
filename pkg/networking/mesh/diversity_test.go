package mesh

import (
	"net"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

func TestRegionDiversityConstants(t *testing.T) {
	// Verify constants are reasonable
	if MinUniqueRegions < 1 {
		t.Error("MinUniqueRegions should be at least 1")
	}
	if TargetUniqueRegions < MinUniqueRegions {
		t.Error("TargetUniqueRegions should be >= MinUniqueRegions")
	}
	if MaxPeersPerRegion < 1 {
		t.Error("MaxPeersPerRegion should be at least 1")
	}
}

func TestNewRegionDiversityManager(t *testing.T) {
	rdm := NewRegionDiversityManager()
	if rdm == nil {
		t.Fatal("expected non-nil manager")
	}
	if rdm.peerRegions == nil {
		t.Error("peerRegions map not initialized")
	}
	if rdm.regionPeers == nil {
		t.Error("regionPeers map not initialized")
	}
}

func TestRegionDiversityManager_AddRemovePeer(t *testing.T) {
	rdm := NewRegionDiversityManager()

	// Create test peer and addresses
	testPeer := peer.ID("test-peer-1")
	addr, _ := ma.NewMultiaddr("/ip4/192.168.1.100/tcp/4001")

	// Add peer
	region := rdm.AddPeer(testPeer, []ma.Multiaddr{addr})
	if region == RegionUnknown {
		t.Error("expected a valid region")
	}

	// Verify peer is tracked
	gotRegion, ok := rdm.GetPeerRegion(testPeer)
	if !ok {
		t.Error("peer should be tracked")
	}
	if gotRegion != region {
		t.Errorf("expected region %s, got %s", region, gotRegion)
	}

	// Remove peer
	rdm.RemovePeer(testPeer)

	_, ok = rdm.GetPeerRegion(testPeer)
	if ok {
		t.Error("peer should be removed")
	}
}

func TestRegionDiversityManager_UniqueRegionCount(t *testing.T) {
	rdm := NewRegionDiversityManager()

	// Initially zero
	if rdm.UniqueRegionCount() != 0 {
		t.Error("expected 0 regions initially")
	}

	// Add peers from different regions
	peer1 := peer.ID("peer-1")
	addr1, _ := ma.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
	rdm.AddPeer(peer1, []ma.Multiaddr{addr1})

	peer2 := peer.ID("peer-2")
	addr2, _ := ma.NewMultiaddr("/ip4/10.0.0.1/tcp/4001")
	rdm.AddPeer(peer2, []ma.Multiaddr{addr2})

	// Both are private, so same region
	if rdm.UniqueRegionCount() < 1 {
		t.Error("expected at least 1 region")
	}
}

func TestRegionDiversityManager_Status(t *testing.T) {
	rdm := NewRegionDiversityManager()

	status := rdm.Status()
	if status.UniqueRegions != 0 {
		t.Errorf("expected 0 unique regions, got %d", status.UniqueRegions)
	}
	if status.MinRegions != MinUniqueRegions {
		t.Errorf("expected min regions %d, got %d", MinUniqueRegions, status.MinRegions)
	}
	if status.NeedsMoreRegions != true {
		t.Error("should need more regions when at 0")
	}
}

func TestRegionDiversityManager_ShouldAcceptPeer(t *testing.T) {
	rdm := NewRegionDiversityManager()

	// Should accept first peer
	addr, _ := ma.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
	if !rdm.ShouldAcceptPeer([]ma.Multiaddr{addr}) {
		t.Error("should accept first peer")
	}

	// Add many peers to same region to exceed limit
	for i := 0; i < MaxPeersPerRegion; i++ {
		peerID := peer.ID(string(rune('A' + i)))
		rdm.AddPeer(peerID, []ma.Multiaddr{addr})
	}

	// After filling up the region, should reject additional peers from same region
	// when we have at least MinUniqueRegions
	if rdm.UniqueRegionCount() >= MinUniqueRegions {
		// Region is now full
		if rdm.ShouldAcceptPeer([]ma.Multiaddr{addr}) {
			t.Error("should reject peer from full region")
		}
	}
}

func TestRegionDiversityManager_GetPeersToDropForDiversity(t *testing.T) {
	rdm := NewRegionDiversityManager()

	// With no peers, should return nil
	toDrop := rdm.GetPeersToDropForDiversity()
	if toDrop != nil {
		t.Error("should return nil with no peers")
	}

	// Add many peers to one region
	addr, _ := ma.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
	for i := 0; i < MaxPeersPerRegion+5; i++ {
		peerID := peer.ID(string(rune('A' + i)))
		rdm.AddPeer(peerID, []ma.Multiaddr{addr})
	}

	// Should recommend dropping excess peers
	toDrop = rdm.GetPeersToDropForDiversity()
	if len(toDrop) == 0 {
		t.Error("should recommend dropping peers from overloaded region")
	}
}

func TestDetermineRegionFromAddrs_Local(t *testing.T) {
	addr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/4001")
	region := determineRegionFromAddrs([]ma.Multiaddr{addr})
	if region != RegionLocal {
		t.Errorf("expected RegionLocal, got %s", region)
	}

	// IPv6 loopback
	addr6, _ := ma.NewMultiaddr("/ip6/::1/tcp/4001")
	region6 := determineRegionFromAddrs([]ma.Multiaddr{addr6})
	if region6 != RegionLocal {
		t.Errorf("expected RegionLocal for IPv6 loopback, got %s", region6)
	}
}

func TestDetermineRegionFromAddrs_Private(t *testing.T) {
	testCases := []struct {
		addr   string
		expect Region
	}{
		{"/ip4/10.0.0.1/tcp/4001", RegionPrivate},
		{"/ip4/192.168.1.1/tcp/4001", RegionPrivate},
		{"/ip4/172.16.0.1/tcp/4001", RegionPrivate},
	}

	for _, tc := range testCases {
		addr, _ := ma.NewMultiaddr(tc.addr)
		region := determineRegionFromAddrs([]ma.Multiaddr{addr})
		if region != tc.expect {
			t.Errorf("addr %s: expected %s, got %s", tc.addr, tc.expect, region)
		}
	}
}

func TestDetermineRegionFromAddrs_Public(t *testing.T) {
	// Public address should get a region derived from IP
	addr, _ := ma.NewMultiaddr("/ip4/8.8.8.8/tcp/4001")
	region := determineRegionFromAddrs([]ma.Multiaddr{addr})
	if region == RegionUnknown || region == RegionLocal || region == RegionPrivate {
		t.Error("public address should get a specific region")
	}
}

func TestDetermineRegionFromAddrs_Empty(t *testing.T) {
	region := determineRegionFromAddrs(nil)
	if region != RegionUnknown {
		t.Errorf("expected RegionUnknown for nil addrs, got %s", region)
	}

	region = determineRegionFromAddrs([]ma.Multiaddr{})
	if region != RegionUnknown {
		t.Errorf("expected RegionUnknown for empty addrs, got %s", region)
	}
}

func TestExtractIP(t *testing.T) {
	// IPv4
	addr4, _ := ma.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
	ip4 := extractIP(addr4)
	if ip4 == nil {
		t.Error("expected to extract IPv4")
	}
	if ip4.String() != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", ip4.String())
	}

	// IPv6
	addr6, _ := ma.NewMultiaddr("/ip6/2001:db8::1/tcp/4001")
	ip6 := extractIP(addr6)
	if ip6 == nil {
		t.Error("expected to extract IPv6")
	}
}

func TestDeriveRegionFromIP(t *testing.T) {
	// Test that different IP ranges get different regions
	ip1 := net.ParseIP("8.8.8.8")
	ip2 := net.ParseIP("200.1.1.1")

	region1 := deriveRegionFromIP(ip1)
	region2 := deriveRegionFromIP(ip2)

	if region1 == region2 {
		t.Error("different IP ranges should get different regions")
	}

	// Nil IP
	region := deriveRegionFromIP(nil)
	if region != RegionUnknown {
		t.Errorf("expected RegionUnknown for nil IP, got %s", region)
	}
}

func TestIPClassification_IsAnycasted(t *testing.T) {
	ic := &IPClassification{}

	// Known anycast addresses
	if !ic.IsAnycasted(net.ParseIP("1.1.1.1")) {
		t.Error("1.1.1.1 should be detected as anycast")
	}
	if !ic.IsAnycasted(net.ParseIP("8.8.8.8")) {
		t.Error("8.8.8.8 should be detected as anycast")
	}

	// Regular address
	if ic.IsAnycasted(net.ParseIP("192.168.1.1")) {
		t.Error("private address should not be anycast")
	}
}

func TestIPClassification_IsDatacenter(t *testing.T) {
	ic := &IPClassification{}

	// Currently returns false (placeholder)
	if ic.IsDatacenter(net.ParseIP("8.8.8.8")) {
		t.Error("datacenter detection is not yet implemented")
	}
}

func TestRegionDiversityManager_RemoveNonexistentPeer(t *testing.T) {
	rdm := NewRegionDiversityManager()

	// Should not panic
	rdm.RemovePeer(peer.ID("nonexistent"))
}

func TestRegionDiversityManager_RegionCounts(t *testing.T) {
	rdm := NewRegionDiversityManager()

	// Initially empty
	counts := rdm.RegionCounts()
	if len(counts) != 0 {
		t.Error("expected empty counts initially")
	}

	// Add peers
	addr, _ := ma.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
	rdm.AddPeer(peer.ID("peer-1"), []ma.Multiaddr{addr})
	rdm.AddPeer(peer.ID("peer-2"), []ma.Multiaddr{addr})

	counts = rdm.RegionCounts()
	totalPeers := 0
	for _, count := range counts {
		totalPeers += count
	}
	if totalPeers != 2 {
		t.Errorf("expected 2 total peers in counts, got %d", totalPeers)
	}
}

func TestDiversityStatus_Fields(t *testing.T) {
	rdm := NewRegionDiversityManager()

	// Add several peers to create overloading
	addr, _ := ma.NewMultiaddr("/ip4/192.168.1.1/tcp/4001")
	for i := 0; i < MaxPeersPerRegion+1; i++ {
		peerID := peer.ID(string(rune('A' + i)))
		rdm.AddPeer(peerID, []ma.Multiaddr{addr})
	}

	status := rdm.Status()

	if status.LargestRegionCount <= MaxPeersPerRegion {
		t.Error("largest region count should exceed max")
	}
	if !status.HasOverloadedRegion {
		t.Error("should have overloaded region")
	}
	if status.IsWellDistributed {
		t.Error("should not be well distributed with overloaded region")
	}
}
