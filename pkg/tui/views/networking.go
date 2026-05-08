package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// NetworkingModel displays peer/mesh/DHT health surfaces.
type NetworkingModel struct {
	Peers        int
	DHTStatus    string
	DHTPeers     int
	MeshHealth   string
	RateLimit    string
	LastEvent    string
	ShroudStatus string
	Topics       map[string]int
	Transport    string
	Health       string
}

// NewNetworkingModel creates a networking status model.
func NewNetworkingModel() NetworkingModel {
	return NetworkingModel{
		DHTStatus:    "idle",
		MeshHealth:   "initializing",
		RateLimit:    "normal",
		ShroudStatus: "not-connected",
		Topics:       map[string]int{"/murmur/waves/1": 0, "/murmur/identity/1": 0, "/murmur/shroud/1": 0, "/murmur/pulse/1": 0},
		Transport:    "noise/quic/tcp",
		Health:       "ok",
	}
}

// ApplyEventType applies event-type updates from an external event bridge.
func (m *NetworkingModel) ApplyEventType(eventType string) {
	m.LastEvent = eventType
	switch eventType {
	case "PeerConnected":
		m.Peers++
		m.DHTPeers++
		m.MeshHealth = "improving"
	case "PeerDisconnected":
		if m.Peers > 0 {
			m.Peers--
		}
		m.MeshHealth = "degraded"
		if m.DHTPeers > 0 {
			m.DHTPeers--
		}
	case "ShroudCircuitBuilt":
		m.ShroudStatus = "circuit-built"
	case "ShroudCircuitFailed":
		m.ShroudStatus = "circuit-failed"
		m.RateLimit = "elevated"
	case "HeartbeatReceived":
		m.DHTStatus = "active"
		m.Topics["/murmur/pulse/1"]++
	case "WaveReceived":
		m.Topics["/murmur/waves/1"]++
	case "IdentityUpdated":
		m.Topics["/murmur/identity/1"]++
	case "ShroudRelayDiscovered":
		m.Topics["/murmur/shroud/1"]++
	}
}

// Update handles key interactions.
func (m NetworkingModel) Update(msg tea.Msg) (NetworkingModel, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "r":
			m.RateLimit = "normal"
			m.LastEvent = "rate-limit-reset"
		case "d":
			m.DHTStatus = "refreshing"
			m.LastEvent = "dht-refresh"
		case "g":
			m.Topics["/murmur/waves/1"]++
			m.LastEvent = "gossip-message-simulated"
		}
	}
	return m, nil
}

// View renders networking status.
func (m NetworkingModel) View(width int) string {
	return fmt.Sprintf(
		"Peers: %d\nDHT: %s peers=%d\nMesh health: %s\nRate-limit: %s\nShroud: %s\nTransport: %s\nHealth endpoint: %s\nTopics: waves=%d identity=%d shroud=%d pulse=%d\nLast event: %s",
		m.Peers,
		m.DHTStatus,
		m.DHTPeers,
		m.MeshHealth,
		m.RateLimit,
		m.ShroudStatus,
		m.Transport,
		m.Health,
		m.Topics["/murmur/waves/1"],
		m.Topics["/murmur/identity/1"],
		m.Topics["/murmur/shroud/1"],
		m.Topics["/murmur/pulse/1"],
		m.LastEvent,
	)
}
