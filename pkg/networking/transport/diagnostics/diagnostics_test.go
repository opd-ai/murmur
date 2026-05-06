package diagnostics

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// TestCheckTor_Unreachable tests Tor check with no daemon running.
func TestCheckTor_Unreachable(t *testing.T) {
	ctx := context.Background()

	// Use a port unlikely to be in use.
	status := CheckTor(ctx, "127.0.0.1:19999")

	if status.Reachable {
		t.Error("Expected Tor to be unreachable, got reachable")
	}

	if status.Name != "Tor" {
		t.Errorf("Expected name 'Tor', got %q", status.Name)
	}

	if status.Error == "" {
		t.Error("Expected error message, got empty string")
	}

	if !strings.Contains(status.Error, "Tor daemon unreachable") {
		t.Errorf("Expected 'Tor daemon unreachable' in error, got: %s", status.Error)
	}

	if !strings.Contains(status.Error, "torproject.org") {
		t.Errorf("Expected installation link in error, got: %s", status.Error)
	}

	if status.LatencyMs != 0 {
		t.Errorf("Expected 0 latency for unreachable, got %d", status.LatencyMs)
	}
}

// TestCheckI2P_Unreachable tests I2P check with no router running.
func TestCheckI2P_Unreachable(t *testing.T) {
	ctx := context.Background()

	// Use a port unlikely to be in use.
	status := CheckI2P(ctx, "127.0.0.1:17999")

	if status.Reachable {
		t.Error("Expected I2P to be unreachable, got reachable")
	}

	if status.Name != "I2P" {
		t.Errorf("Expected name 'I2P', got %q", status.Name)
	}

	if status.Error == "" {
		t.Error("Expected error message, got empty string")
	}

	if !strings.Contains(status.Error, "I2P router unreachable") {
		t.Errorf("Expected 'I2P router unreachable' in error, got: %s", status.Error)
	}

	if !strings.Contains(status.Error, "i2pd.website") {
		t.Errorf("Expected installation link in error, got: %s", status.Error)
	}

	if status.LatencyMs != 0 {
		t.Errorf("Expected 0 latency for unreachable, got %d", status.LatencyMs)
	}
}

// TestCheckTor_InvalidProtocol tests Tor check with wrong protocol response.
func TestCheckTor_InvalidProtocol(t *testing.T) {
	ctx := context.Background()

	// Start a mock TCP server that responds with invalid data.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// Accept connections and send invalid response.
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Read request, send invalid response.
			buf := make([]byte, 256)
			conn.Read(buf)
			conn.Write([]byte("INVALID RESPONSE\r\n"))
			conn.Close()
		}
	}()

	status := CheckTor(ctx, addr)

	if status.Reachable {
		t.Error("Expected Tor to be unreachable with invalid protocol, got reachable")
	}

	if !strings.Contains(status.Error, "invalid Tor response") {
		t.Errorf("Expected 'invalid Tor response' in error, got: %s", status.Error)
	}
}

// TestCheckI2P_InvalidProtocol tests I2P check with wrong protocol response.
func TestCheckI2P_InvalidProtocol(t *testing.T) {
	ctx := context.Background()

	// Start a mock TCP server that responds with invalid data.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// Accept connections and send invalid response.
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Read request, send invalid response.
			buf := make([]byte, 256)
			conn.Read(buf)
			conn.Write([]byte("INVALID SAM RESPONSE\n"))
			conn.Close()
		}
	}()

	status := CheckI2P(ctx, addr)

	if status.Reachable {
		t.Error("Expected I2P to be unreachable with invalid protocol, got reachable")
	}

	if !strings.Contains(status.Error, "SAM bridge responded with") {
		t.Errorf("Expected SAM response error, got: %s", status.Error)
	}
}

// TestCheckAll_NoneEnabled tests CheckAll with no transports enabled.
func TestCheckAll_NoneEnabled(t *testing.T) {
	ctx := context.Background()

	statuses, err := CheckAll(ctx, false, "", false, "")
	if err != nil {
		t.Errorf("Expected no error with no transports, got: %v", err)
	}

	if len(statuses) != 0 {
		t.Errorf("Expected 0 statuses, got %d", len(statuses))
	}
}

// TestCheckAll_BothUnreachable tests CheckAll with both transports enabled but unreachable.
func TestCheckAll_BothUnreachable(t *testing.T) {
	ctx := context.Background()

	statuses, err := CheckAll(ctx, true, "127.0.0.1:19999", true, "127.0.0.1:17999")

	if err == nil {
		t.Error("Expected error with unreachable transports, got nil")
	}

	if len(statuses) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(statuses))
	}

	if statuses[0].Reachable {
		t.Error("Expected first status (Tor) to be unreachable")
	}

	if statuses[1].Reachable {
		t.Error("Expected second status (I2P) to be unreachable")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Tor daemon unreachable") {
		t.Errorf("Expected Tor error in combined message, got: %s", errMsg)
	}

	if !strings.Contains(errMsg, "I2P router unreachable") {
		t.Errorf("Expected I2P error in combined message, got: %s", errMsg)
	}
}

// TestCheckAll_OnlyTorEnabled tests CheckAll with only Tor enabled.
func TestCheckAll_OnlyTorEnabled(t *testing.T) {
	ctx := context.Background()

	statuses, err := CheckAll(ctx, true, "127.0.0.1:19999", false, "")

	if err == nil {
		t.Error("Expected error with unreachable Tor, got nil")
	}

	if len(statuses) != 1 {
		t.Errorf("Expected 1 status, got %d", len(statuses))
	}

	if statuses[0].Name != "Tor" {
		t.Errorf("Expected Tor status, got %q", statuses[0].Name)
	}
}

// TestCheckAll_OnlyI2PEnabled tests CheckAll with only I2P enabled.
func TestCheckAll_OnlyI2PEnabled(t *testing.T) {
	ctx := context.Background()

	statuses, err := CheckAll(ctx, false, "", true, "127.0.0.1:17999")

	if err == nil {
		t.Error("Expected error with unreachable I2P, got nil")
	}

	if len(statuses) != 1 {
		t.Errorf("Expected 1 status, got %d", len(statuses))
	}

	if statuses[0].Name != "I2P" {
		t.Errorf("Expected I2P status, got %q", statuses[0].Name)
	}
}

// TestMin tests the min helper function.
func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{-1, 0, -1},
		{100, 50, 50},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d,%d", tt.a, tt.b), func(t *testing.T) {
			got := min(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestTransportStatusFields tests that TransportStatus has expected fields.
func TestTransportStatusFields(t *testing.T) {
	status := TransportStatus{
		Name:      "Test",
		Reachable: true,
		Error:     "test error",
		LatencyMs: 123,
		Address:   "127.0.0.1:9999",
	}

	if status.Name != "Test" {
		t.Errorf("Expected Name 'Test', got %q", status.Name)
	}
	if !status.Reachable {
		t.Error("Expected Reachable true")
	}
	if status.Error != "test error" {
		t.Errorf("Expected Error 'test error', got %q", status.Error)
	}
	if status.LatencyMs != 123 {
		t.Errorf("Expected LatencyMs 123, got %d", status.LatencyMs)
	}
	if status.Address != "127.0.0.1:9999" {
		t.Errorf("Expected Address '127.0.0.1:9999', got %q", status.Address)
	}
}

// TestCheckTor_Timeout tests that Tor check respects timeout.
func TestCheckTor_Timeout(t *testing.T) {
	// Start a server that accepts but never responds.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	// Accept connections but never send data.
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Keep connection open but don't respond.
			time.Sleep(10 * time.Second)
			conn.Close()
		}
	}()

	ctx := context.Background()
	start := time.Now()
	status := CheckTor(ctx, addr)
	elapsed := time.Since(start)

	if status.Reachable {
		t.Error("Expected Tor to be unreachable with timeout")
	}

	// Should timeout within ~5 seconds (3s dial + 2s protocol).
	if elapsed > 6*time.Second {
		t.Errorf("Expected timeout within 6s, took %v", elapsed)
	}
}
