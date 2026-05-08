package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// NetworkingModel displays peer/mesh/DHT health surfaces.
type NetworkingModel struct {
	Peers        int
	DHTStatus    string
	MeshHealth   string
	RateLimit    string
	LastEvent    string
	ShroudStatus string
}

// NewNetworkingModel creates a networking status model.
func NewNetworkingModel() NetworkingModel {
	return NetworkingModel{
		DHTStatus:    "idle",
		MeshHealth:   "initializing",
		RateLimit:    "normal",
		ShroudStatus: "not-connected",
	}
}

// ApplyEventType applies event-type updates from an external event bridge.
func (m *NetworkingModel) ApplyEventType(eventType string) {
	m.LastEvent = eventType
	switch eventType {
	case "PeerConnected":
		m.Peers++
		m.MeshHealth = "improving"
	case "PeerDisconnected":
		if m.Peers > 0 {
			m.Peers--
		}
		m.MeshHealth = "degraded"
	case "ShroudCircuitBuilt":
		m.ShroudStatus = "circuit-built"
	case "ShroudCircuitFailed":
		m.ShroudStatus = "circuit-failed"
		m.RateLimit = "elevated"
	case "HeartbeatReceived":
		m.DHTStatus = "active"
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
		}
	}
	return m, nil
}

// View renders networking status.
func (m NetworkingModel) View(width int) string {
	return fmt.Sprintf("Peers: %d\nDHT: %s\nMesh health: %s\nRate-limit: %s\nShroud: %s\nLast event: %s", m.Peers, m.DHTStatus, m.MeshHealth, m.RateLimit, m.ShroudStatus, m.LastEvent)
}
