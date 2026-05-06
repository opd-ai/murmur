// Package config provides configuration loading, defaults, and validation for MURMUR.
// Per TECHNICAL_IMPLEMENTATION.md §2, configuration is loaded from the user data
// directory and merged with command-line overrides.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/multiformats/go-multiaddr"
)

// Config holds application configuration options.
type Config struct {
	Version              string
	DataDir              string
	ListenAddrs          []string
	BootstrapPeers       []string
	SkipUI               bool
	CLIMode              bool
	EnableRelay          bool
	RelayBandwidth       uint64
	EnableHealthEndpoint bool          // Enable HTTP health check endpoint (default false for privacy)
	HealthEndpointPort   int           // Port for health check endpoint (default 8080)
	HeartbeatInterval    time.Duration // Interval for sending heartbeat pings (default 30s)
	EnableTor            bool          // Enable Tor transport adapter for /onion3 addresses
	EnableI2P            bool          // Enable I2P transport adapter for /garlic64 addresses
	TorControlAddr       string        // Tor control port address (default: 127.0.0.1:9051)
	I2PSAMAddr           string        // I2P SAMv3 address (default: 127.0.0.1:7656)
}

// LoadConfig loads configuration from defaults and applies overrides.
// Per TECHNICAL_IMPLEMENTATION.md §2, configuration is merged from:
// 1. Defaults (DefaultListenAddrs, DefaultBootstrapPeers)
// 2. User config file (future: ~/.murmur/config.toml)
// 3. Command-line flags (passed via Config struct)
func LoadConfig(overrides Config) (*Config, error) {
	cfg := &Config{
		Version:              overrides.Version,
		DataDir:              overrides.DataDir,
		ListenAddrs:          overrides.ListenAddrs,
		BootstrapPeers:       overrides.BootstrapPeers,
		SkipUI:               overrides.SkipUI,
		CLIMode:              overrides.CLIMode,
		EnableRelay:          overrides.EnableRelay,
		RelayBandwidth:       overrides.RelayBandwidth,
		EnableHealthEndpoint: overrides.EnableHealthEndpoint,
		HealthEndpointPort:   overrides.HealthEndpointPort,
		HeartbeatInterval:    overrides.HeartbeatInterval,
		EnableTor:            overrides.EnableTor,
		EnableI2P:            overrides.EnableI2P,
		TorControlAddr:       overrides.TorControlAddr,
		I2PSAMAddr:           overrides.I2PSAMAddr,
	}

	if err := cfg.applyDefaults(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// applyDefaults fills in default values for unspecified configuration fields.
func (cfg *Config) applyDefaults() error {
	if err := cfg.applyDataDirDefault(); err != nil {
		return err
	}
	cfg.applyListenAddrsDefault()
	cfg.applyBootstrapPeersDefault()
	cfg.applyRelayBandwidthDefault()
	cfg.applyHealthEndpointPortDefault()
	cfg.applyHeartbeatIntervalDefault()
	cfg.applyTorControlAddrDefault()
	cfg.applyI2PSAMAddrDefault()
	return nil
}

// applyDataDirDefault sets the data directory to ~/.murmur if unspecified.
func (cfg *Config) applyDataDirDefault() error {
	if cfg.DataDir != "" {
		return nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}
	cfg.DataDir = filepath.Join(homeDir, ".murmur")
	return nil
}

// applyListenAddrsDefault sets listen addresses to defaults if unspecified.
func (cfg *Config) applyListenAddrsDefault() {
	if len(cfg.ListenAddrs) == 0 {
		cfg.ListenAddrs = make([]string, len(DefaultListenAddrs))
		copy(cfg.ListenAddrs, DefaultListenAddrs)
	}
}

// applyBootstrapPeersDefault sets bootstrap peers to defaults if unspecified.
func (cfg *Config) applyBootstrapPeersDefault() {
	if len(cfg.BootstrapPeers) == 0 {
		cfg.BootstrapPeers = make([]string, len(DefaultBootstrapPeers))
		copy(cfg.BootstrapPeers, DefaultBootstrapPeers)
	}
}

// applyRelayBandwidthDefault sets relay bandwidth to 10 MiB/s if relay is enabled and unspecified.
func (cfg *Config) applyRelayBandwidthDefault() {
	if cfg.EnableRelay && cfg.RelayBandwidth == 0 {
		cfg.RelayBandwidth = 10 * 1024 * 1024
	}
}

// applyHealthEndpointPortDefault sets health endpoint port to 8080 if enabled and unspecified.
func (cfg *Config) applyHealthEndpointPortDefault() {
	if cfg.EnableHealthEndpoint && cfg.HealthEndpointPort == 0 {
		cfg.HealthEndpointPort = 8080
	}
}

// applyHeartbeatIntervalDefault sets heartbeat interval to 30 seconds if unspecified.
func (cfg *Config) applyHeartbeatIntervalDefault() {
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 30 * time.Second
	}
}

// applyTorControlAddrDefault sets Tor control address to 127.0.0.1:9051 if Tor is enabled and unspecified.
func (cfg *Config) applyTorControlAddrDefault() {
	if cfg.EnableTor && cfg.TorControlAddr == "" {
		cfg.TorControlAddr = "127.0.0.1:9051"
	}
}

// applyI2PSAMAddrDefault sets I2P SAM address to 127.0.0.1:7656 if I2P is enabled and unspecified.
func (cfg *Config) applyI2PSAMAddrDefault() {
	if cfg.EnableI2P && cfg.I2PSAMAddr == "" {
		cfg.I2PSAMAddr = "127.0.0.1:7656"
	}
}

// DefaultConfig returns a configuration with all default values.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".murmur")

	listenAddrs := make([]string, len(DefaultListenAddrs))
	copy(listenAddrs, DefaultListenAddrs)

	bootstrapPeers := make([]string, len(DefaultBootstrapPeers))
	copy(bootstrapPeers, DefaultBootstrapPeers)

	return &Config{
		Version:           "0.0.0-alpha",
		DataDir:           dataDir,
		ListenAddrs:       listenAddrs,
		BootstrapPeers:    bootstrapPeers,
		SkipUI:            false,
		CLIMode:           false,
		EnableRelay:       false,
		RelayBandwidth:    10 * 1024 * 1024,
		HeartbeatInterval: 30 * time.Second,
		EnableTor:         false,
		EnableI2P:         false,
		TorControlAddr:    "127.0.0.1:9051",
		I2PSAMAddr:        "127.0.0.1:7656",
	}
}

// ValidateConfig checks configuration values for correctness.
// Per TECHNICAL_IMPLEMENTATION.md §2, validation includes:
// - DataDir must be writable
// - ListenAddrs must be valid multiaddrs
// - BootstrapPeers must be valid multiaddrs (if provided)
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if err := validateDataDir(cfg.DataDir); err != nil {
		return err
	}
	if err := validateListenAddrs(cfg.ListenAddrs); err != nil {
		return err
	}
	if err := validateBootstrapPeers(cfg.BootstrapPeers); err != nil {
		return err
	}
	if err := validateRelaySettings(cfg.EnableRelay, cfg.RelayBandwidth); err != nil {
		return err
	}

	return nil
}

// validateDataDir ensures DataDir is non-empty.
func validateDataDir(dataDir string) error {
	if dataDir == "" {
		return fmt.Errorf("DataDir cannot be empty")
	}
	return nil
}

// validateListenAddrs validates all listen addresses.
func validateListenAddrs(addrs []string) error {
	if len(addrs) == 0 {
		return fmt.Errorf("ListenAddrs cannot be empty")
	}
	return validateMultiaddrs(addrs, "ListenAddr")
}

// validateBootstrapPeers validates bootstrap peer addresses.
func validateBootstrapPeers(addrs []string) error {
	return validateMultiaddrs(addrs, "BootstrapPeer")
}

// validateMultiaddrs validates a slice of multiaddr strings.
func validateMultiaddrs(addrs []string, prefix string) error {
	for i, addr := range addrs {
		if _, err := multiaddr.NewMultiaddr(addr); err != nil {
			return fmt.Errorf("invalid %s[%d]: %w", prefix, i, err)
		}
	}
	return nil
}

// validateRelaySettings checks relay bandwidth if relay is enabled.
func validateRelaySettings(enableRelay bool, bandwidth uint64) error {
	if enableRelay && bandwidth == 0 {
		return fmt.Errorf("RelayBandwidth must be > 0 when EnableRelay is true")
	}
	return nil
}
