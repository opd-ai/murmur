package network

// buildBrowserDiscoveryPeers builds the browser discovery set with relay peers
// first and bootstrap peers second. Only configured peers are eligible so
// browser discovery does not depend on mDNS.
func buildBrowserDiscoveryPeers(relayPeers, bootstrapPeers []string) []string {
	seen := make(map[string]struct{}, len(relayPeers)+len(bootstrapPeers))
	peers := make([]string, 0, len(relayPeers)+len(bootstrapPeers))

	appendUnique := func(values []string) {
		for _, value := range values {
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			peers = append(peers, value)
		}
	}

	appendUnique(relayPeers)
	appendUnique(bootstrapPeers)
	return peers
}
