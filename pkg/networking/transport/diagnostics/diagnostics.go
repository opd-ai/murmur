// Package diagnostics provides transport reachability checks for Tor and I2P.
// Per PLAN.md §5.7, startup checks probe Tor control port (9051) and I2P SAMv3
// (7656) before host construction and surface actionable errors.
package diagnostics

import (
	"context"
	"fmt"
	"net"
	"time"
)

// TransportStatus represents the reachability status of a transport.
type TransportStatus struct {
	// Name is the transport name ("Tor" or "I2P").
	Name string

	// Reachable indicates whether the transport daemon is reachable.
	Reachable bool

	// Error is the error message if unreachable (empty if reachable).
	Error string

	// Latency is the probe latency in milliseconds (0 if unreachable).
	LatencyMs int64

	// Address is the probed address (e.g., "127.0.0.1:9051").
	Address string
}

// CheckTor probes the Tor control port and returns reachability status.
// Per PLAN.md §5.7, surfaces actionable error with installation link if unreachable.
func CheckTor(ctx context.Context, controlAddr string) TransportStatus {
	start := time.Now()

	// Probe Tor control port with 3-second timeout.
	conn, err := net.DialTimeout("tcp", controlAddr, 3*time.Second)
	if err != nil {
		return TransportStatus{
			Name:      "Tor",
			Reachable: false,
			Error: fmt.Sprintf("Tor daemon unreachable at %s. "+
				"Install: apt install tor (Linux) or download from torproject.org. "+
				"Ensure Tor daemon is running with control port on %s.",
				controlAddr, controlAddr),
			Address: controlAddr,
		}
	}
	defer conn.Close()

	// Verify Tor control protocol by sending PROTOCOLINFO command.
	// This distinguishes a Tor control port from a random TCP listener.
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Write([]byte("PROTOCOLINFO 1\r\n"))
	if err != nil {
		return TransportStatus{
			Name:      "Tor",
			Reachable: false,
			Error: fmt.Sprintf("Connected to %s but Tor control protocol not responding. "+
				"Ensure Tor daemon is running and control port is enabled.",
				controlAddr),
			Address: controlAddr,
		}
	}

	// Read response (expect "250-" prefix for Tor control protocol).
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil || n < 4 || string(buf[:3]) != "250" {
		return TransportStatus{
			Name:      "Tor",
			Reachable: false,
			Error: fmt.Sprintf("Connected to %s but received invalid Tor response. "+
				"Verify Tor daemon is running correctly.",
				controlAddr),
			Address: controlAddr,
		}
	}

	latency := time.Since(start)
	return TransportStatus{
		Name:      "Tor",
		Reachable: true,
		LatencyMs: latency.Milliseconds(),
		Address:   controlAddr,
	}
}

// CheckI2P probes the I2P SAMv3 bridge and returns reachability status.
// Per PLAN.md §5.7, surfaces actionable error with installation link if unreachable.
func CheckI2P(ctx context.Context, samAddr string) TransportStatus {
	start := time.Now()

	// Probe I2P SAM bridge with 3-second timeout.
	conn, err := net.DialTimeout("tcp", samAddr, 3*time.Second)
	if err != nil {
		return TransportStatus{
			Name:      "I2P",
			Reachable: false,
			Error: fmt.Sprintf("I2P router unreachable at %s. "+
				"Download i2pd from i2pd.website or java-i2p from geti2p.net. "+
				"Enable SAM bridge on port 7656 (i2pd: sam.enabled=true in config). "+
				"Documentation: i2pd.readthedocs.io/en/latest/user-guide/SAM/",
				samAddr),
			Address: samAddr,
		}
	}
	defer conn.Close()

	// Verify SAM protocol by sending HELLO VERSION command.
	// SAM v3 responds with "HELLO REPLY RESULT=OK VERSION=3.1"
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	_, err = conn.Write([]byte("HELLO VERSION MIN=3.0 MAX=3.3\n"))
	if err != nil {
		return TransportStatus{
			Name:      "I2P",
			Reachable: false,
			Error: fmt.Sprintf("Connected to %s but SAM protocol not responding. "+
				"Ensure I2P router is running and SAM bridge is enabled.",
				samAddr),
			Address: samAddr,
		}
	}

	// Read response (expect "HELLO REPLY RESULT=OK").
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil || n < 17 {
		return TransportStatus{
			Name:      "I2P",
			Reachable: false,
			Error: fmt.Sprintf("Connected to %s but received invalid SAM response. "+
				"Verify I2P router is running with SAM v3 bridge enabled.",
				samAddr),
			Address: samAddr,
		}
	}

	response := string(buf[:n])
	if len(response) < 17 || response[:17] != "HELLO REPLY RESUL" {
		return TransportStatus{
			Name:      "I2P",
			Reachable: false,
			Error: fmt.Sprintf("I2P SAM bridge responded with: %q. "+
				"Expected HELLO REPLY. Verify SAM v3 configuration.",
				response[:min(40, len(response))]),
			Address: samAddr,
		}
	}

	latency := time.Since(start)
	return TransportStatus{
		Name:      "I2P",
		Reachable: true,
		LatencyMs: latency.Milliseconds(),
		Address:   samAddr,
	}
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CheckAll probes all configured transports and returns a summary.
// Returns an error if any required transport is unreachable.
// Per PLAN.md §5.7, checks are performed before host construction.
func CheckAll(ctx context.Context, enableTor bool, torAddr string, enableI2P bool, i2pAddr string) ([]TransportStatus, error) {
	var statuses []TransportStatus
	var errors []string

	if enableTor {
		status := CheckTor(ctx, torAddr)
		statuses = append(statuses, status)
		if !status.Reachable {
			errors = append(errors, status.Error)
		}
	}

	if enableI2P {
		status := CheckI2P(ctx, i2pAddr)
		statuses = append(statuses, status)
		if !status.Reachable {
			errors = append(errors, status.Error)
		}
	}

	if len(errors) > 0 {
		return statuses, fmt.Errorf("transport reachability check failed:\n%s",
			joinErrors(errors))
	}

	return statuses, nil
}

// joinErrors concatenates error messages with newlines and bullet points.
func joinErrors(errors []string) string {
	result := ""
	for i, err := range errors {
		result += fmt.Sprintf("  %d. %s", i+1, err)
		if i < len(errors)-1 {
			result += "\n"
		}
	}
	return result
}
