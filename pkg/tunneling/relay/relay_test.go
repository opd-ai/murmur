package relay

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/tunneling"
	"github.com/opd-ai/murmur/pkg/tunneling/protocol"
)

func TestPlaintextUnregisterIsRejected(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	relayListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	relayAddr := relayListener.Addr().String()
	relayListener.Close()

	r := NewRelay(relayAddr)
	if err := r.Start(ctx); err != nil {
		t.Fatalf("start relay: %v", err)
	}
	defer r.Stop(context.Background())

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tunnelID := tunneling.GenerateTunnelID(pub, "alice")

	operatorConn, err := net.Dial("tcp", relayAddr)
	if err != nil {
		t.Fatalf("dial operator: %v", err)
	}
	defer operatorConn.Close()

	regPayload, err := protocol.EncodeRegisterCell(tunnelID, pub, priv, 0)
	if err != nil {
		t.Fatalf("encode register cell: %v", err)
	}
	if err := protocol.WriteFrame(operatorConn, protocol.FrameTypeRegister, regPayload); err != nil {
		t.Fatalf("write register frame: %v", err)
	}
	ack := make([]byte, 3)
	if _, err := operatorConn.Read(ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if string(ack) != "OK\n" {
		t.Fatalf("unexpected ack: %q", string(ack))
	}

	waitForTunnel(t, r, tunnelID, true)

	attackerConn, err := net.Dial("tcp", relayAddr)
	if err != nil {
		t.Fatalf("dial attacker: %v", err)
	}
	defer attackerConn.Close()

	if _, err := fmt.Fprintf(attackerConn, "UNREGISTER %s\n", tunnelID); err != nil {
		t.Fatalf("write unregister: %v", err)
	}
	_ = attackerConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 256)
	n, err := attackerConn.Read(buf)
	if err != nil {
		t.Fatalf("read unregister response: %v", err)
	}
	if !strings.Contains(string(buf[:n]), "401 Unauthorized") {
		t.Fatalf("expected 401 unauthorized response, got %q", string(buf[:n]))
	}

	waitForTunnel(t, r, tunnelID, true)
}

func waitForTunnel(t *testing.T, r *Relay, id tunneling.TunnelID, expected bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.RLock()
		_, ok := r.tunnels[id]
		r.mu.RUnlock()
		if ok == expected {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("tunnel %q presence expected %v", id, expected)
}
