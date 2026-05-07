package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadSignedPeers_Valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "peers.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"timestamp":1700000000,"peers":[{"id":"peer-1","addrs":["/ip4/127.0.0.1/tcp/4001"],"seen":1700000000}],"signature":"sig","signed_by":"pub"}`), 0o600); err != nil {
		t.Fatalf("write peers file: %v", err)
	}

	raw, modTime, err := loadSignedPeers(path)
	if err != nil {
		t.Fatalf("loadSignedPeers returned error: %v", err)
	}
	if len(raw) == 0 {
		t.Fatalf("expected non-empty payload")
	}
	if modTime.IsZero() {
		t.Fatalf("expected non-zero modTime")
	}
}

func TestLoadSignedPeers_InvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "peers.json")
	if err := os.WriteFile(path, []byte(`{`), 0o600); err != nil {
		t.Fatalf("write peers file: %v", err)
	}

	if _, _, err := loadSignedPeers(path); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestLoadSignedPeers_EmptyPeers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "peers.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"timestamp":1700000000,"peers":[],"signature":"sig","signed_by":"pub"}`), 0o600); err != nil {
		t.Fatalf("write peers file: %v", err)
	}

	if _, _, err := loadSignedPeers(path); err == nil {
		t.Fatalf("expected empty peers validation error")
	}
}

func TestPeerFileProvider_LoadCachesAndRefreshes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "peers.json")

	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write peers file: %v", err)
		}
	}

	write(`{"version":1,"timestamp":1700000000,"peers":[{"id":"peer-1","addrs":["/ip4/127.0.0.1/tcp/4001"],"seen":1700000000}],"signature":"sig","signed_by":"pub"}`)

	p := &peerFileProvider{path: path}
	first, err := p.load()
	if err != nil {
		t.Fatalf("first load: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	write(`{"version":1,"timestamp":1700000001,"peers":[{"id":"peer-2","addrs":["/ip4/127.0.0.1/tcp/4002"],"seen":1700000001}],"signature":"sig","signed_by":"pub"}`)

	second, err := p.load()
	if err != nil {
		t.Fatalf("second load: %v", err)
	}

	if string(first) == string(second) {
		t.Fatalf("expected payload to refresh after file update")
	}
}
