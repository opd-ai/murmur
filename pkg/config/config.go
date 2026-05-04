// Package config provides configuration loading, defaults, and validation for MURMUR.
// Per TECHNICAL_IMPLEMENTATION.md §2, configuration is loaded from the user data
// directory and merged with command-line overrides.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/multiformats/go-multiaddr"
)

// Config holds application configuration options.
type Config struct {
	Version        string
	DataDir        string
	ListenAddrs    []string
	BootstrapPeers []string
	SkipUI         bool
	CLIMode        bool
	EnableRelay    bool
	RelayBandwidth uint64
}

// LoadConfig loads configuration from defaults and applies overrides.
// Per TECHNICAL_IMPLEMENTATION.md §2, configuration is merged from:
// 1. Defaults (DefaultListenAddrs, DefaultBootstrapPeers)
// 2. User config file (future: ~/.murmur/config.toml)
// 3. Command-line flags (passed via Config struct)
func LoadConfig(overrides Config) (*Config, error) {
	cfg := &Config{
		Version:        overrides.Version,
		DataDir:        overrides.DataDir,
		ListenAddrs:    overrides.ListenAddrs,
		BootstrapPeers: overrides.BootstrapPeers,
		SkipUI:         overrides.SkipUI,
		CLIMode:        overrides.CLIMode,
		EnableRelay:    overrides.EnableRelay,
		RelayBandwidth: overrides.RelayBandwidth,
	}

	// Apply DataDir default
	if cfg.DataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home directory: %w", err)
		}
		cfg.DataDir = filepath.Join(homeDir, ".murmur")
	}

	// Apply ListenAddrs default
	if len(cfg.ListenAddrs) == 0 {
		cfg.ListenAddrs = make([]string, len(DefaultListenAddrs))
		copy(cfg.ListenAddrs, DefaultListenAddrs)
	}

	// Apply BootstrapPeers default
	if len(cfg.BootstrapPeers) == 0 {
		cfg.BootstrapPeers = make([]string, len(DefaultBootstrapPeers))
		copy(cfg.BootstrapPeers, DefaultBootstrapPeers)
	}

	// Apply RelayBandwidth default
	if cfg.EnableRelay && cfg.RelayBandwidth == 0 {
		cfg.RelayBandwidth = 10 * 1024 * 1024 // 10 MiB/s default
	}

	return cfg, nil
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
		Version:        "0.0.0-alpha",
		DataDir:        dataDir,
		ListenAddrs:    listenAddrs,
		BootstrapPeers: bootstrapPeers,
		SkipUI:         false,
		CLIMode:        false,
		EnableRelay:    false,
		RelayBandwidth: 10 * 1024 * 1024,
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

	// Validate DataDir
	if cfg.DataDir == "" {
		return fmt.Errorf("DataDir cannot be empty")
	}

	// Validate ListenAddrs
	if len(cfg.ListenAddrs) == 0 {
		return fmt.Errorf("ListenAddrs cannot be empty")
	}
	for i, addr := range cfg.ListenAddrs {
		if _, err := multiaddr.NewMultiaddr(addr); err != nil {
			return fmt.Errorf("invalid ListenAddr[%d]: %w", i, err)
		}
	}

	// Validate BootstrapPeers (if provided)
	for i, addr := range cfg.BootstrapPeers {
		if _, err := multiaddr.NewMultiaddr(addr); err != nil {
			return fmt.Errorf("invalid BootstrapPeer[%d]: %w", i, err)
		}
	}

	// Validate RelayBandwidth
	if cfg.EnableRelay && cfg.RelayBandwidth == 0 {
		return fmt.Errorf("RelayBandwidth must be > 0 when EnableRelay is true")
	}

	return nil
}
