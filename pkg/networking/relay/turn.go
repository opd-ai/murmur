// Package relay provides NAT traversal, DCUtR hole punching, and relay fallback.
// This file implements TURN server configuration for WebRTC ICE.
// Per NETWORK_ARCHITECTURE.md §TURN Fallback for WebRTC:
// "MURMUR configures ICE with public STUN servers for address discovery
// and falls back to community-operated TURN servers when direct connectivity fails."
package relay

import (
	"fmt"
	"net/url"
	"time"
)

// ICEServer represents a STUN or TURN server for WebRTC ICE.
type ICEServer struct {
	// URLs is the list of server URLs (e.g., "stun:stun.example.com:3478").
	URLs []string

	// Username is the username for TURN authentication (optional for STUN).
	Username string

	// Credential is the credential for TURN authentication (optional for STUN).
	Credential string

	// CredentialType is the type of credential ("password" or "oauth").
	CredentialType string
}

// ICEConfig holds ICE configuration for WebRTC NAT traversal.
type ICEConfig struct {
	// STUNServers is the list of STUN servers for address discovery.
	STUNServers []ICEServer

	// TURNServers is the list of TURN servers for relay fallback.
	TURNServers []ICEServer

	// ICETransportPolicy specifies which ICE candidates to use.
	// "all" allows all candidates, "relay" forces relay-only.
	ICETransportPolicy string

	// ICECandidatePoolSize is the number of ICE candidates to pre-gather.
	ICECandidatePoolSize int

	// GatherTimeout is the maximum time to gather ICE candidates.
	GatherTimeout time.Duration
}

// DefaultICEConfig returns the default ICE configuration.
// Uses public STUN servers for address discovery.
func DefaultICEConfig() ICEConfig {
	return ICEConfig{
		STUNServers:          DefaultSTUNServers(),
		TURNServers:          []ICEServer{}, // TURN requires auth, none by default
		ICETransportPolicy:   "all",
		ICECandidatePoolSize: 10,
		GatherTimeout:        10 * time.Second,
	}
}

// DefaultSTUNServers returns a list of public STUN servers.
// These are well-known servers for ICE address discovery.
func DefaultSTUNServers() []ICEServer {
	return []ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
		{URLs: []string{"stun:stun1.l.google.com:19302"}},
		{URLs: []string{"stun:stun2.l.google.com:19302"}},
		{URLs: []string{"stun:stun.cloudflare.com:3478"}},
		{URLs: []string{"stun:stun.services.mozilla.com:3478"}},
	}
}

// TURNServerInfo represents TURN server information discovered from DHT.
type TURNServerInfo struct {
	// URL is the TURN server URL (e.g., "turn:turn.example.com:3478").
	URL string

	// Username is the username for authentication.
	Username string

	// Password is the password for authentication.
	Password string

	// Realm is the TURN realm.
	Realm string

	// TTL is how long this server info is valid.
	TTL time.Duration

	// DiscoveredAt is when this server was discovered.
	DiscoveredAt time.Time
}

// IsExpired returns true if the TURN server info has expired.
func (t TURNServerInfo) IsExpired() bool {
	return time.Since(t.DiscoveredAt) > t.TTL
}

// ToICEServer converts TURNServerInfo to ICEServer.
func (t TURNServerInfo) ToICEServer() ICEServer {
	return ICEServer{
		URLs:           []string{t.URL},
		Username:       t.Username,
		Credential:     t.Password,
		CredentialType: "password",
	}
}

// TURNDiscoveryKey is the DHT service key for TURN server discovery.
// Per NETWORK_ARCHITECTURE.md: "TURN server addresses are distributed
// via the DHT as provider records with a well-known service key."
const TURNDiscoveryKey = "/murmur/turn/v1"

// AddTURNServer adds a TURN server to the ICE configuration.
func (c *ICEConfig) AddTURNServer(server ICEServer) {
	c.TURNServers = append(c.TURNServers, server)
}

// AddSTUNServer adds a STUN server to the ICE configuration.
func (c *ICEConfig) AddSTUNServer(server ICEServer) {
	c.STUNServers = append(c.STUNServers, server)
}

// AllServers returns all ICE servers (STUN and TURN combined).
func (c ICEConfig) AllServers() []ICEServer {
	all := make([]ICEServer, 0, len(c.STUNServers)+len(c.TURNServers))
	all = append(all, c.STUNServers...)
	all = append(all, c.TURNServers...)
	return all
}

// SetRelayOnly forces relay-only mode (TURN servers only).
// This is useful for maximum privacy but increases latency.
func (c *ICEConfig) SetRelayOnly() {
	c.ICETransportPolicy = "relay"
}

// ValidateURL checks if a URL is a valid STUN or TURN URL.
func ValidateURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	switch u.Scheme {
	case "stun", "stuns", "turn", "turns":
		// Valid schemes
	default:
		return fmt.Errorf("invalid scheme: %s (must be stun, stuns, turn, or turns)", u.Scheme)
	}

	// STUN/TURN URLs use the Opaque field (e.g., "stun:host:port")
	// or Host field (e.g., "stun://host:port")
	if u.Host == "" && u.Opaque == "" {
		return fmt.Errorf("missing host in URL")
	}

	return nil
}

// IsTURNURL returns true if the URL is a TURN URL.
func IsTURNURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return u.Scheme == "turn" || u.Scheme == "turns"
}

// IsSTUNURL returns true if the URL is a STUN URL.
func IsSTUNURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return u.Scheme == "stun" || u.Scheme == "stuns"
}

// CommunityTURNServers returns community-operated TURN servers discovered via DHT.
// Per NETWORK_ARCHITECTURE.md: "falls back to community-operated TURN servers."
//
// Production TURN servers are advertised as DHT provider records under TURNDiscoveryKey
// and fetched at runtime by the relay discovery subsystem.  This function returns an
// empty list as the authoritative static fallback; callers that need TURN connectivity
// must use DHT-based discovery (see TURNDiscoveryKey) and health-filter results through
// FilterHealthyTURNServers before adding them to the ICEConfig.
func CommunityTURNServers() []TURNServerInfo {
	// Production TURN servers are distributed via DHT provider records.
	// There are no globally-hardcoded TURN credentials; static entries here would
	// require embedded credentials that cannot be rotated securely.
	return nil
}

// FilterHealthyTURNServers returns only non-expired TURN server entries.
// Use this to health-filter a slice obtained from DHT discovery before
// merging into an ICEConfig with MergeWithTURN.
func FilterHealthyTURNServers(servers []TURNServerInfo) []TURNServerInfo {
	healthy := make([]TURNServerInfo, 0, len(servers))
	for _, s := range servers {
		if !s.IsExpired() {
			healthy = append(healthy, s)
		}
	}
	return healthy
}

// MergeWithTURN creates a new ICEConfig with additional TURN servers.
func (c ICEConfig) MergeWithTURN(turnServers []TURNServerInfo) ICEConfig {
	result := ICEConfig{
		STUNServers:          make([]ICEServer, len(c.STUNServers)),
		TURNServers:          make([]ICEServer, len(c.TURNServers)),
		ICETransportPolicy:   c.ICETransportPolicy,
		ICECandidatePoolSize: c.ICECandidatePoolSize,
		GatherTimeout:        c.GatherTimeout,
	}
	copy(result.STUNServers, c.STUNServers)
	copy(result.TURNServers, c.TURNServers)

	for _, ts := range turnServers {
		if !ts.IsExpired() {
			result.TURNServers = append(result.TURNServers, ts.ToICEServer())
		}
	}

	return result
}
