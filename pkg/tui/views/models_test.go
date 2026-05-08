package views

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/opd-ai/murmur/pkg/content/waves"
	pb "github.com/opd-ai/murmur/proto"
)

func TestPulseMapModelUpdate_Table(t *testing.T) {
	m := NewPulseMapModel(NewSessionState())
	tests := []struct {
		name  string
		msg   tea.Msg
		check func(t *testing.T, got PulseMapModel)
	}{
		{
			name: "pan right with l",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}},
			check: func(t *testing.T, got PulseMapModel) {
				if got.CameraX <= 0 {
					t.Fatalf("expected CameraX > 0, got %v", got.CameraX)
				}
			},
		},
		{
			name: "zoom in",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}},
			check: func(t *testing.T, got PulseMapModel) {
				if got.Zoom <= 1 {
					t.Fatalf("expected Zoom > 1, got %v", got.Zoom)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := m.Update(tc.msg)
			tc.check(t, got)
		})
	}
}

func TestIdentityModelUpdate_Table(t *testing.T) {
	session := NewSessionState()
	m := NewIdentityModel(session)
	tests := []struct {
		name  string
		msg   tea.Msg
		check func(t *testing.T, got IdentityModel)
	}{
		{
			name: "generate keypair",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}},
			check: func(t *testing.T, got IdentityModel) {
				if got.Session.KeyPair == nil {
					t.Fatal("expected keypair generated")
				}
			},
		},
		{
			name: "switch mode",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}},
			check: func(t *testing.T, got IdentityModel) {
				if got.Session.ModeManager.Current().String() != "Open" {
					t.Fatalf("expected open mode, got %s", got.Session.ModeManager.Current().String())
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := m.Update(tc.msg)
			tc.check(t, got)
		})
	}
}

func TestWavesModelUpdate_Table(t *testing.T) {
	session := NewSessionState()
	id := NewIdentityModel(session)
	id, _ = id.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m := NewWavesModel(session)

	tests := []struct {
		name  string
		msg   tea.Msg
		check func(t *testing.T, got WavesModel)
	}{
		{
			name: "toggle compose",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}},
			check: func(t *testing.T, got WavesModel) {
				if !got.Compose {
					t.Fatal("expected compose enabled")
				}
			},
		},
		{
			name: "set wave type 3",
			msg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}},
			check: func(t *testing.T, got WavesModel) {
				if got.TypeIndex != 2 {
					t.Fatalf("expected type index 2, got %d", got.TypeIndex)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := m.Update(tc.msg)
			tc.check(t, got)
		})
	}
}

func TestWavesModelSubmitRoutesSpecializedTypes(t *testing.T) {
	session := NewSessionState()
	id := NewIdentityModel(session)
	id, _ = id.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if session.KeyPair == nil {
		t.Fatal("expected generated keypair")
	}

	base := NewWavesModel(session)
	base.Compose = true
	base.Content = "submit path coverage"
	base.Difficulty = 8
	base.TTL = waves.DefaultTTL

	tests := []struct {
		name      string
		typeIndex int
		check     func(t *testing.T, wave *pb.Wave)
	}{
		{
			name:      "veiled",
			typeIndex: 2,
			check: func(t *testing.T, wave *pb.Wave) {
				if !bytes.Equal(wave.Metadata[waves.VeiledMetadataKey], []byte(waves.VeiledMetadataValue)) {
					t.Fatal("veiled submit should include veil metadata")
				}
			},
		},
		{
			name:      "specter",
			typeIndex: 3,
			check: func(t *testing.T, wave *pb.Wave) {
				if wave.WaveType != pb.WaveType(waves.TypeSpecter) {
					t.Fatalf("expected specter wave type, got %d", wave.WaveType)
				}
			},
		},
		{
			name:      "masked",
			typeIndex: 6,
			check: func(t *testing.T, wave *pb.Wave) {
				if len(wave.Metadata[waves.MetaMaskedEventID]) == 0 {
					t.Fatal("masked submit should include event_id metadata")
				}
			},
		},
		{
			name:      "beacon",
			typeIndex: 7,
			check: func(t *testing.T, wave *pb.Wave) {
				if len(wave.Metadata[waves.BeaconTypeKey]) == 0 {
					t.Fatal("beacon submit should include beacon_type metadata")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := base
			m.TypeIndex = tc.typeIndex
			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if cmd == nil {
				t.Fatal("expected submit command")
			}
			msg := cmd()
			created, ok := msg.(waveCreatedMsg)
			if !ok {
				t.Fatalf("expected waveCreatedMsg, got %T", msg)
			}
			if created.err != nil {
				t.Fatalf("submit command returned error: %v", created.err)
			}
			if created.wave == nil {
				t.Fatal("expected created wave")
			}
			tc.check(t, created.wave)
			if updated.Status == "" {
				t.Fatal("expected status update")
			}
		})
	}
}

func TestAnonymousModelUpdate_Table(t *testing.T) {
	m := NewAnonymousModel(NewSessionState())
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if len(got.Session.Specters) != 1 {
		t.Fatalf("expected one specter, got %d", len(got.Session.Specters))
	}
}

func TestOnboardingModelUpdate_Table(t *testing.T) {
	m := NewOnboardingModel(NewSessionState())
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got.Controller.OverallProgress() <= 0 {
		t.Fatal("expected onboarding progress to advance")
	}
}

func TestNetworkingModelUpdate_Table(t *testing.T) {
	m := NewNetworkingModel()
	m.ApplyEventType("PeerConnected")
	if m.Peers != 1 {
		t.Fatalf("expected peers 1, got %d", m.Peers)
	}
	m.ApplyEventType("ShroudCircuitFailed")
	if m.ShroudStatus != "circuit-failed" {
		t.Fatalf("expected shroud failed status, got %s", m.ShroudStatus)
	}
}
