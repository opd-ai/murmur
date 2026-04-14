// Package mesh provides peer scoring, mesh health monitoring, and connection management.
// This file implements multi-region diversity constraints for eclipse resistance.
// Per SECURITY_PRIVACY.md, maintaining connections to peers in different regions
// helps resist eclipse attacks.
package mesh

import (
	"net"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// RegionDiversity constants for eclipse resistance.
const (
	// MinUniqueRegions is the minimum number of distinct regions to maintain.
	MinUniqueRegions = 2

	// TargetUniqueRegions is the target number of distinct regions.
	TargetUniqueRegions = 3

	// MaxPeersPerRegion limits peers from any single region.
	MaxPeersPerRegion = 6
)

// Region represents a geographic/network region.
type Region string

const (
	RegionUnknown Region = "unknown"
	RegionLocal   Region = "local"   // 127.0.0.0/8, ::1
	RegionPrivate Region = "private" // RFC1918, RFC4193
	RegionNAT64   Region = "nat64"   // 64:ff9b::/96
)

// RegionDiversityManager ensures peer connections span multiple regions.
type RegionDiversityManager struct {
	peerRegions map[peer.ID]Region
	regionPeers map[Region][]peer.ID
	mu          sync.RWMutex
}

// NewRegionDiversityManager creates a new region diversity manager.
func NewRegionDiversityManager() *RegionDiversityManager {
	return &RegionDiversityManager{
		peerRegions: make(map[peer.ID]Region),
		regionPeers: make(map[Region][]peer.ID),
	}
}

// AddPeer registers a peer's region based on their addresses.
func (rdm *RegionDiversityManager) AddPeer(p peer.ID, addrs []ma.Multiaddr) Region {
	region := determineRegionFromAddrs(addrs)

	rdm.mu.Lock()
	defer rdm.mu.Unlock()

	rdm.peerRegions[p] = region
	rdm.regionPeers[region] = append(rdm.regionPeers[region], p)

	return region
}

// RemovePeer removes a peer from region tracking.
func (rdm *RegionDiversityManager) RemovePeer(p peer.ID) {
	rdm.mu.Lock()
	defer rdm.mu.Unlock()

	region, ok := rdm.peerRegions[p]
	if !ok {
		return
	}

	delete(rdm.peerRegions, p)

	// Remove from region's peer list
	peers := rdm.regionPeers[region]
	for i, pid := range peers {
		if pid == p {
			rdm.regionPeers[region] = append(peers[:i], peers[i+1:]...)
			break
		}
	}

	// Clean up empty regions
	if len(rdm.regionPeers[region]) == 0 {
		delete(rdm.regionPeers, region)
	}
}

// GetPeerRegion returns the region for a peer.
func (rdm *RegionDiversityManager) GetPeerRegion(p peer.ID) (Region, bool) {
	rdm.mu.RLock()
	defer rdm.mu.RUnlock()

	region, ok := rdm.peerRegions[p]
	return region, ok
}

// UniqueRegionCount returns the number of distinct regions.
func (rdm *RegionDiversityManager) UniqueRegionCount() int {
	rdm.mu.RLock()
	defer rdm.mu.RUnlock()

	return len(rdm.regionPeers)
}

// RegionCounts returns the count of peers per region.
func (rdm *RegionDiversityManager) RegionCounts() map[Region]int {
	rdm.mu.RLock()
	defer rdm.mu.RUnlock()

	counts := make(map[Region]int)
	for region, peers := range rdm.regionPeers {
		counts[region] = len(peers)
	}
	return counts
}

// DiversityStatus returns the current diversity status.
type DiversityStatus struct {
	UniqueRegions       int
	MinRegions          int
	TargetRegions       int
	RegionCounts        map[Region]int
	LargestRegionCount  int
	IsWellDistributed   bool
	NeedsMoreRegions    bool
	HasOverloadedRegion bool
}

// Status returns the current diversity status.
func (rdm *RegionDiversityManager) Status() DiversityStatus {
	rdm.mu.RLock()
	defer rdm.mu.RUnlock()

	counts := make(map[Region]int)
	maxCount := 0

	for region, peers := range rdm.regionPeers {
		count := len(peers)
		counts[region] = count
		if count > maxCount {
			maxCount = count
		}
	}

	uniqueRegions := len(rdm.regionPeers)

	return DiversityStatus{
		UniqueRegions:       uniqueRegions,
		MinRegions:          MinUniqueRegions,
		TargetRegions:       TargetUniqueRegions,
		RegionCounts:        counts,
		LargestRegionCount:  maxCount,
		IsWellDistributed:   uniqueRegions >= MinUniqueRegions && maxCount <= MaxPeersPerRegion,
		NeedsMoreRegions:    uniqueRegions < MinUniqueRegions,
		HasOverloadedRegion: maxCount > MaxPeersPerRegion,
	}
}

// ShouldAcceptPeer returns whether we should accept a peer from this region.
func (rdm *RegionDiversityManager) ShouldAcceptPeer(addrs []ma.Multiaddr) bool {
	region := determineRegionFromAddrs(addrs)

	rdm.mu.RLock()
	defer rdm.mu.RUnlock()

	// Always accept if we need more regions
	if len(rdm.regionPeers) < MinUniqueRegions {
		return true
	}

	// Check if this region is overloaded
	if len(rdm.regionPeers[region]) >= MaxPeersPerRegion {
		return false
	}

	return true
}

// GetPeersToDropForDiversity returns peers that should be dropped
// to improve region diversity. Returns peers from the most overloaded region.
func (rdm *RegionDiversityManager) GetPeersToDropForDiversity() []peer.ID {
	rdm.mu.RLock()
	defer rdm.mu.RUnlock()

	var mostLoaded Region
	maxCount := 0

	for region, peers := range rdm.regionPeers {
		if len(peers) > maxCount {
			maxCount = len(peers)
			mostLoaded = region
		}
	}

	// If no region is overloaded, don't drop anyone
	if maxCount <= MaxPeersPerRegion {
		return nil
	}

	// Return excess peers from the overloaded region
	peers := rdm.regionPeers[mostLoaded]
	excessCount := maxCount - MaxPeersPerRegion
	if excessCount > len(peers) {
		excessCount = len(peers)
	}

	return peers[:excessCount]
}

// determineRegionFromAddrs determines the region from multiaddrs.
// Uses IP geolocation heuristics based on address characteristics.
func determineRegionFromAddrs(addrs []ma.Multiaddr) Region {
	for _, addr := range addrs {
		ip := extractIP(addr)
		if ip == nil {
			continue
		}

		// Check for local addresses
		if ip.IsLoopback() {
			return RegionLocal
		}

		// Check for private addresses
		if ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return RegionPrivate
		}

		// Use IP prefix as a simple region identifier
		// This is a simplified approach - production would use GeoIP
		return deriveRegionFromIP(ip)
	}

	return RegionUnknown
}

// extractIP extracts the IP address from a multiaddr.
func extractIP(addr ma.Multiaddr) net.IP {
	// Try to extract IP4 or IP6
	for _, proto := range []int{ma.P_IP4, ma.P_IP6} {
		if val, err := addr.ValueForProtocol(proto); err == nil {
			return net.ParseIP(val)
		}
	}
	return nil
}

// deriveRegionFromIP derives a region identifier from an IP address.
// Uses the first octet/prefix as a simple region approximation.
func deriveRegionFromIP(ip net.IP) Region {
	if ip == nil {
		return RegionUnknown
	}

	// Use first octet for IPv4, or first 16 bits for IPv6
	var prefix string
	if ip4 := ip.To4(); ip4 != nil {
		// Use first octet as region (very simplified)
		prefix = string(rune('A' + (ip4[0] / 64))) // Divides into 4 broad regions
	} else if ip6 := ip.To16(); ip6 != nil {
		// Use first byte as region
		prefix = string(rune('A' + (ip6[0] / 64)))
	}

	if prefix == "" {
		return RegionUnknown
	}

	return Region("region-" + prefix)
}

// IPClassification provides IP address classification utilities.
type IPClassification struct{}

// IsAnycasted returns true if the IP appears to be anycast.
// Note: This is a heuristic that checks for well-known anycast prefixes.
func (ic *IPClassification) IsAnycasted(ip net.IP) bool {
	// Check for well-known anycast prefixes
	anycasts := []string{
		"1.1.1.1/32",      // Cloudflare DNS
		"8.8.8.8/32",      // Google DNS
		"8.8.4.4/32",      // Google DNS
		"9.9.9.9/32",      // Quad9
		"208.67.222.0/24", // OpenDNS
	}

	for _, cidr := range anycasts {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

// IsDatacenter returns true if the IP appears to be from a datacenter.
// Note: Production would use a datacenter IP database.
func (ic *IPClassification) IsDatacenter(ip net.IP) bool {
	// Simplified heuristic - would use real datacenter IP lists in production
	return false
}
