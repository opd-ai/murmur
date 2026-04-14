package relay

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultICEConfig(t *testing.T) {
	cfg := DefaultICEConfig()

	assert.NotEmpty(t, cfg.STUNServers)
	assert.Empty(t, cfg.TURNServers) // No TURN by default (requires auth)
	assert.Equal(t, "all", cfg.ICETransportPolicy)
	assert.Equal(t, 10, cfg.ICECandidatePoolSize)
	assert.Equal(t, 10*time.Second, cfg.GatherTimeout)
}

func TestDefaultSTUNServers(t *testing.T) {
	servers := DefaultSTUNServers()

	assert.GreaterOrEqual(t, len(servers), 3, "should have multiple STUN servers")

	for _, server := range servers {
		require.NotEmpty(t, server.URLs)
		assert.True(t, IsSTUNURL(server.URLs[0]), "URL should be STUN: %s", server.URLs[0])
	}
}

func TestICEConfig_AddTURNServer(t *testing.T) {
	cfg := DefaultICEConfig()
	originalLen := len(cfg.TURNServers)

	turn := ICEServer{
		URLs:       []string{"turn:example.com:3478"},
		Username:   "user",
		Credential: "pass",
	}
	cfg.AddTURNServer(turn)

	assert.Len(t, cfg.TURNServers, originalLen+1)
	assert.Equal(t, turn, cfg.TURNServers[len(cfg.TURNServers)-1])
}

func TestICEConfig_AddSTUNServer(t *testing.T) {
	cfg := DefaultICEConfig()
	originalLen := len(cfg.STUNServers)

	stun := ICEServer{
		URLs: []string{"stun:custom.stun.server:3478"},
	}
	cfg.AddSTUNServer(stun)

	assert.Len(t, cfg.STUNServers, originalLen+1)
}

func TestICEConfig_AllServers(t *testing.T) {
	cfg := DefaultICEConfig()
	cfg.AddTURNServer(ICEServer{URLs: []string{"turn:turn.example.com:3478"}})

	all := cfg.AllServers()
	assert.Len(t, all, len(cfg.STUNServers)+len(cfg.TURNServers))
}

func TestICEConfig_SetRelayOnly(t *testing.T) {
	cfg := DefaultICEConfig()
	assert.Equal(t, "all", cfg.ICETransportPolicy)

	cfg.SetRelayOnly()
	assert.Equal(t, "relay", cfg.ICETransportPolicy)
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
	}{
		{"stun:stun.example.com:3478", false},
		{"stuns:stun.example.com:5349", false},
		{"turn:turn.example.com:3478", false},
		{"turns:turn.example.com:5349", false},
		{"http://invalid.com", true}, // Wrong scheme
		{"stun:", true},              // No host
		{"://invalid", true},         // Invalid URL
	}

	for _, tt := range tests {
		err := ValidateURL(tt.url)
		if tt.wantErr {
			assert.Error(t, err, "URL: %s", tt.url)
		} else {
			assert.NoError(t, err, "URL: %s", tt.url)
		}
	}
}

func TestIsTURNURL(t *testing.T) {
	assert.True(t, IsTURNURL("turn:turn.example.com:3478"))
	assert.True(t, IsTURNURL("turns:turn.example.com:5349"))
	assert.False(t, IsTURNURL("stun:stun.example.com:3478"))
	assert.False(t, IsTURNURL("stuns:stun.example.com:5349"))
	assert.False(t, IsTURNURL("invalid"))
}

func TestIsSTUNURL(t *testing.T) {
	assert.True(t, IsSTUNURL("stun:stun.example.com:3478"))
	assert.True(t, IsSTUNURL("stuns:stun.example.com:5349"))
	assert.False(t, IsSTUNURL("turn:turn.example.com:3478"))
	assert.False(t, IsSTUNURL("turns:turn.example.com:5349"))
	assert.False(t, IsSTUNURL("invalid"))
}

func TestTURNServerInfo_IsExpired(t *testing.T) {
	fresh := TURNServerInfo{
		TTL:          1 * time.Hour,
		DiscoveredAt: time.Now(),
	}
	assert.False(t, fresh.IsExpired())

	expired := TURNServerInfo{
		TTL:          1 * time.Hour,
		DiscoveredAt: time.Now().Add(-2 * time.Hour),
	}
	assert.True(t, expired.IsExpired())
}

func TestTURNServerInfo_ToICEServer(t *testing.T) {
	info := TURNServerInfo{
		URL:      "turn:turn.example.com:3478",
		Username: "user",
		Password: "pass",
		Realm:    "example.com",
	}

	server := info.ToICEServer()
	assert.Equal(t, []string{"turn:turn.example.com:3478"}, server.URLs)
	assert.Equal(t, "user", server.Username)
	assert.Equal(t, "pass", server.Credential)
	assert.Equal(t, "password", server.CredentialType)
}

func TestCommunityTURNServers(t *testing.T) {
	servers := CommunityTURNServers()

	assert.NotEmpty(t, servers)
	for _, s := range servers {
		assert.NotEmpty(t, s.URL)
		assert.NotEmpty(t, s.Username)
		assert.NotEmpty(t, s.Password)
		assert.False(t, s.IsExpired())
	}
}

func TestICEConfig_MergeWithTURN(t *testing.T) {
	cfg := DefaultICEConfig()

	turnServers := []TURNServerInfo{
		{
			URL:          "turn:turn1.example.com:3478",
			Username:     "user1",
			Password:     "pass1",
			TTL:          1 * time.Hour,
			DiscoveredAt: time.Now(),
		},
		{
			URL:          "turn:turn2.example.com:3478",
			Username:     "user2",
			Password:     "pass2",
			TTL:          1 * time.Hour,
			DiscoveredAt: time.Now().Add(-2 * time.Hour), // Expired
		},
	}

	merged := cfg.MergeWithTURN(turnServers)

	// Should have original STUN servers
	assert.Equal(t, len(cfg.STUNServers), len(merged.STUNServers))

	// Should have only the non-expired TURN server
	assert.Len(t, merged.TURNServers, 1)
	assert.Equal(t, "turn:turn1.example.com:3478", merged.TURNServers[0].URLs[0])
}

func TestTURNDiscoveryKey(t *testing.T) {
	assert.Equal(t, "/murmur/turn/v1", TURNDiscoveryKey)
}
