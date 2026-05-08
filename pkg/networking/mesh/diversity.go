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
	RegionUnknown    Region = "unknown"
	RegionLocal      Region = "local"      // 127.0.0.0/8, ::1
	RegionPrivate    Region = "private"    // RFC1918, RFC4193
	RegionNAT64      Region = "nat64"      // 64:ff9b::/96
	RegionDatacenter Region = "datacenter" // Cloud / hosting provider IP range.
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

// deriveRegionFromIP derives a region identifier from a public IP address.
// Datacenter IPs (cloud / hosting) are grouped into RegionDatacenter so that
// diversity scoring can limit their share per SECURITY_PRIVACY.md.
// For residential IPs, a broad prefix-based region approximation is used.
func deriveRegionFromIP(ip net.IP) Region {
	if ip == nil {
		return RegionUnknown
	}

	// Classify datacenter / cloud provider IPs.
	var ic IPClassification
	if ic.IsDatacenter(ip) {
		return RegionDatacenter
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

// IsDatacenter returns true if the IP appears to originate from a datacenter
// or cloud provider network.  The classification uses a curated list of CIDR
// ranges for major cloud providers (AWS, GCP, Azure, Cloudflare, Hetzner,
// DigitalOcean, OVH, Linode/Akamai) and large hosting networks.
//
// This heuristic is conservative: it only classifies IPs with known mappings
// and returns false when uncertain.  Per SECURITY_PRIVACY.md, datacenter peers
// are deprioritised in diversity scoring to reduce reliance on centralised
// infrastructure.
func (ic *IPClassification) IsDatacenter(ip net.IP) bool {
	if ip == nil {
		return false
	}

	for _, cidr := range datacenterCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

// datacenterCIDRs is a curated set of CIDR ranges used by major cloud
// providers and hosting companies.  Sourced from public provider IP lists.
// Intentionally conservative — prefer false negatives over false positives.
var datacenterCIDRs = []string{
	// Cloudflare (CDN / Workers / WARP)
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"108.162.192.0/18",
	"131.0.72.0/22",
	"141.101.64.0/18",
	"162.158.0.0/15",
	"172.64.0.0/13",
	"173.245.48.0/20",
	"188.114.96.0/20",
	"190.93.240.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",

	// Amazon AWS (representative / commonly seen blocks)
	"3.0.0.0/9",
	"13.32.0.0/15",
	"13.224.0.0/14",
	"18.64.0.0/10",
	"34.192.0.0/10",
	"35.160.0.0/11",
	"52.0.0.0/11",
	"54.64.0.0/11",

	// Google Cloud Platform
	"34.64.0.0/10",
	"34.128.0.0/10",
	"35.184.0.0/13",
	"35.192.0.0/14",
	"35.228.0.0/14",

	// Microsoft Azure
	"13.64.0.0/11",
	"13.96.0.0/13",
	"20.0.0.0/11",
	"20.32.0.0/11",
	"40.64.0.0/10",
	"52.128.0.0/9",

	// DigitalOcean
	"45.55.0.0/16",
	"67.205.0.0/16",
	"104.131.0.0/16",
	"104.236.0.0/16",
	"107.170.0.0/16",
	"138.68.0.0/15",
	"159.203.0.0/16",
	"165.227.0.0/16",

	// Hetzner
	"5.9.0.0/16",
	"78.46.0.0/15",
	"88.99.0.0/16",
	"95.216.0.0/16",
	"116.202.0.0/15",
	"135.181.0.0/16",
	"136.243.0.0/16",
	"157.90.0.0/16",

	// Linode / Akamai
	"45.33.0.0/17",
	"45.56.0.0/21",
	"45.79.0.0/16",
	"50.116.0.0/16",
	"69.164.192.0/18",
	"72.14.176.0/20",

	// OVH / OVHcloud
	"5.135.0.0/16",
	"51.68.0.0/16",
	"51.77.0.0/16",
	"54.38.0.0/16",
	"91.134.0.0/16",
	"178.32.0.0/15",
}
