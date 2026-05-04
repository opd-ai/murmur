package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("with empty overrides", func(t *testing.T) {
		cfg, err := LoadConfig(Config{})
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.DataDir == "" {
			t.Error("DataDir should have default value")
		}
		if len(cfg.ListenAddrs) == 0 {
			t.Error("ListenAddrs should have default values")
		}
		if cfg.ListenAddrs == nil {
			t.Error("ListenAddrs should not be nil")
		}
		if cfg.BootstrapPeers == nil {
			t.Error("BootstrapPeers should not be nil")
		}
	})

	t.Run("with overrides", func(t *testing.T) {
		overrides := Config{
			Version:        "1.0.0",
			DataDir:        "/tmp/murmur-test",
			ListenAddrs:    []string{"/ip4/127.0.0.1/tcp/4001"},
			BootstrapPeers: []string{},
			CLIMode:        true,
			EnableRelay:    true,
		}

		cfg, err := LoadConfig(overrides)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.Version != "1.0.0" {
			t.Errorf("Version = %q, want %q", cfg.Version, "1.0.0")
		}
		if cfg.DataDir != "/tmp/murmur-test" {
			t.Errorf("DataDir = %q, want %q", cfg.DataDir, "/tmp/murmur-test")
		}
		if len(cfg.ListenAddrs) != 1 || cfg.ListenAddrs[0] != "/ip4/127.0.0.1/tcp/4001" {
			t.Errorf("ListenAddrs not preserved: %v", cfg.ListenAddrs)
		}
		if !cfg.CLIMode {
			t.Error("CLIMode should be true")
		}
		if !cfg.EnableRelay {
			t.Error("EnableRelay should be true")
		}
		if cfg.RelayBandwidth == 0 {
			t.Error("RelayBandwidth should have default value when EnableRelay is true")
		}
	})

	t.Run("applies relay bandwidth default", func(t *testing.T) {
		cfg, err := LoadConfig(Config{EnableRelay: true})
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		expected := uint64(10 * 1024 * 1024)
		if cfg.RelayBandwidth != expected {
			t.Errorf("RelayBandwidth = %d, want %d", cfg.RelayBandwidth, expected)
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if cfg.DataDir == "" {
		t.Error("DataDir should not be empty")
	}

	homeDir, _ := os.UserHomeDir()
	expectedDataDir := filepath.Join(homeDir, ".murmur")
	if cfg.DataDir != expectedDataDir {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, expectedDataDir)
	}

	if len(cfg.ListenAddrs) == 0 {
		t.Error("ListenAddrs should not be empty")
	}

	if cfg.Version == "" {
		t.Error("Version should not be empty")
	}

	if cfg.RelayBandwidth != 10*1024*1024 {
		t.Errorf("RelayBandwidth = %d, want %d", cfg.RelayBandwidth, 10*1024*1024)
	}
}

func TestValidateConfig(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		err := ValidateConfig(nil)
		if err == nil {
			t.Error("ValidateConfig should reject nil config")
		}
	})

	t.Run("empty DataDir", func(t *testing.T) {
		cfg := &Config{
			ListenAddrs: []string{"/ip4/0.0.0.0/tcp/0"},
		}
		err := ValidateConfig(cfg)
		if err == nil {
			t.Error("ValidateConfig should reject empty DataDir")
		}
	})

	t.Run("empty ListenAddrs", func(t *testing.T) {
		cfg := &Config{
			DataDir:     "/tmp/murmur",
			ListenAddrs: []string{},
		}
		err := ValidateConfig(cfg)
		if err == nil {
			t.Error("ValidateConfig should reject empty ListenAddrs")
		}
	})

	t.Run("invalid ListenAddr", func(t *testing.T) {
		cfg := &Config{
			DataDir:     "/tmp/murmur",
			ListenAddrs: []string{"not-a-multiaddr"},
		}
		err := ValidateConfig(cfg)
		if err == nil {
			t.Error("ValidateConfig should reject invalid ListenAddr")
		}
	})

	t.Run("invalid BootstrapPeer", func(t *testing.T) {
		cfg := &Config{
			DataDir:        "/tmp/murmur",
			ListenAddrs:    []string{"/ip4/0.0.0.0/tcp/0"},
			BootstrapPeers: []string{"not-a-multiaddr"},
		}
		err := ValidateConfig(cfg)
		if err == nil {
			t.Error("ValidateConfig should reject invalid BootstrapPeer")
		}
	})

	t.Run("EnableRelay with zero bandwidth", func(t *testing.T) {
		cfg := &Config{
			DataDir:        "/tmp/murmur",
			ListenAddrs:    []string{"/ip4/0.0.0.0/tcp/0"},
			EnableRelay:    true,
			RelayBandwidth: 0,
		}
		err := ValidateConfig(cfg)
		if err == nil {
			t.Error("ValidateConfig should reject EnableRelay with zero bandwidth")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			Version:        "1.0.0",
			DataDir:        "/tmp/murmur",
			ListenAddrs:    []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic-v1"},
			BootstrapPeers: []string{},
			EnableRelay:    true,
			RelayBandwidth: 10 * 1024 * 1024,
		}
		err := ValidateConfig(cfg)
		if err != nil {
			t.Errorf("ValidateConfig rejected valid config: %v", err)
		}
	})
}
