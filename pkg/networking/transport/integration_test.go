// Package transport integration tests for Tor and I2P transport adapters.
// These tests exercise reachability checks and validate host creation scenarios
// per PLAN.md §5.8. Full dial/listen lifecycle tests require running Tor/I2P daemons.
//
//go:build integration

package transport

import (
"context"
"testing"
"time"

ma "github.com/multiformats/go-multiaddr"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"

"github.com/opd-ai/murmur/pkg/config"
"github.com/opd-ai/murmur/pkg/networking/transport/diagnostics"
)

// TestTorReachability tests Tor daemon reachability check.
func TestTorReachability(t *testing.T) {
if testing.Short() {
g integration test in short mode")
}

status := diagnostics.CheckTor("127.0.0.1:9051", 5*time.Second)
if !status.Reachable {
daemon not reachable: %v", status.Error)
}

t.Logf("Tor daemon reachable: latency=%dms", status.LatencyMs)
assert.True(t, status.Reachable)
assert.Empty(t, status.Error)
}

// TestI2PReachability tests I2P SAM bridge reachability check.
func TestI2PReachability(t *testing.T) {
if testing.Short() {
g integration test in short mode")
}

status := diagnostics.CheckI2P("127.0.0.1:7656", 5*time.Second)
if !status.Reachable {
SAM bridge not reachable: %v", status.Error)
}

t.Logf("I2P SAM bridge reachable: latency=%dms", status.LatencyMs)
assert.True(t, status.Reachable)
assert.Empty(t, status.Error)
}

// TestHostCreationWithTor tests libp2p host creation with Tor transport.
func TestHostCreationWithTor(t *testing.T) {
if testing.Short() {
g integration test in short mode")
}

ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

cfg := Config{
ableTor:      true,
trolAddr: "127.0.0.1:9051",
ableI2P:      false,
}

h, err := NewHost(ctx, cfg)
if err != nil {
creation failed (Tor unavailable?): %v", err)
}
defer h.Close()

addrs := h.Addrs()
require.NotEmpty(t, addrs, "host should have addresses")

hasOnion := false
for _, addr := range addrs {
_, p := range addr.Protocols() {
p.Name == "onion3" {
ion = true
listening on: %s", addr)
hasOnion, "host should have onion3 address when Tor enabled")
}

// TestHostCreationWithI2P tests libp2p host creation with I2P transport.
func TestHostCreationWithI2P(t *testing.T) {
if testing.Short() {
g integration test in short mode")
}

ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

cfg := Config{
ableTor:  false,
ableI2P:  true,
"127.0.0.1:7656",
}

h, err := NewHost(ctx, cfg)
if err != nil {
creation failed (I2P unavailable?): %v", err)
}
defer h.Close()

addrs := h.Addrs()
require.NotEmpty(t, addrs, "host should have addresses")

hasGarlic := false
for _, addr := range addrs {
_, p := range addr.Protocols() {
p.Name == "garlic64" {
= true
listening on: %s", addr)
hasGarlic, "host should have garlic64 address when I2P enabled")
}

// TestHostCreationWithBoth tests host creation with both Tor and I2P.
func TestHostCreationWithBoth(t *testing.T) {
if testing.Short() {
g integration test in short mode")
}

ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
defer cancel()

cfg := Config{
ableTor:      true,
trolAddr: "127.0.0.1:9051",
ableI2P:      true,
    "127.0.0.1:7656",
}

h, err := NewHost(ctx, cfg)
if err != nil {
creation failed (Tor/I2P unavailable?): %v", err)
}
defer h.Close()

addrs := h.Addrs()
require.NotEmpty(t, addrs, "host should have addresses")

hasOnion, hasGarlic, hasClearnet := false, false, false
for _, addr := range addrs {
_, p := range addr.Protocols() {
p.Name {
"onion3":
ion = true
"garlic64":
= true
"tcp", "quic-v1":
et = true
hasClearnet, "should have clearnet addresses")
t.Logf("Multi-transport: Clearnet=%v Tor=%v I2P=%v", hasClearnet, hasOnion, hasGarlic)
}

// TestFallbackToClearnet verifies fail-fast when transports unavailable.
func TestFallbackToClearnet(t *testing.T) {
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

cfg := Config{
ableTor:      true,
ableI2P:      true,
trolAddr: "127.0.0.1:9999", // Unreachable
    "127.0.0.1:9998", // Unreachable
}

_, err := NewHost(ctx, cfg)
require.Error(t, err, "should fail when required transports unavailable")
t.Logf("Expected failure: %v", err)
}

// TestMultiaddrProtocols validates onion3 and garlic64 multiaddr parsing.
func TestMultiaddrProtocols(t *testing.T) {
onionMaddr, err := ma.NewMultiaddr("/onion3/vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:9001")
require.NoError(t, err)

hasOnion := false
for _, p := range onionMaddr.Protocols() {
p.Name == "onion3" {
ion = true
hasOnion, "should parse onion3 protocol")

garlicDest := make([]byte, 400)
for i := range garlicDest {
= 'A'
}
garlicMaddr, err := ma.NewMultiaddr("/garlic64/" + string(garlicDest) + ":9001")
require.NoError(t, err)

hasGarlic := false
for _, p := range garlicMaddr.Protocols() {
p.Name == "garlic64" {
= true
hasGarlic, "should parse garlic64 protocol")
}

// TestAnonymityTransportDiagnosticsIntegration tests CheckAll function.
func TestAnonymityTransportDiagnosticsIntegration(t *testing.T) {
if testing.Short() {
g integration test in short mode")
}

cfg := config.Config{
ableTor:      true,
ableI2P:      true,
trolAddr: "127.0.0.1:9051",
    "127.0.0.1:7656",
}

statuses, err := diagnostics.CheckAll(cfg, 5*time.Second)
if err != nil {
onymity transports unavailable: %v", err)
}

require.NotEmpty(t, statuses, "should return status for each transport")

for _, status := range statuses {
sport %s: reachable=%v latency=%dms",
ame, status.Reachable, status.LatencyMs)
}
}
