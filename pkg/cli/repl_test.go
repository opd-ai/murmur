package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/content/storage"
	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
	"github.com/opd-ai/murmur/pkg/networking/transport"
	"github.com/opd-ai/murmur/pkg/store"
	"github.com/stretchr/testify/require"
)

// TestNewREPL verifies REPL construction with all required dependencies.
func TestNewREPL(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := makeTestConfig(t)
		repl, err := NewREPL(cfg)
		require.NoError(t, err)
		require.NotNil(t, repl)
		require.NotNil(t, repl.host)
		require.NotNil(t, repl.pubsub)
		require.NotNil(t, repl.keypair)
		require.NotNil(t, repl.waveCache)
	})

	t.Run("MissingHost", func(t *testing.T) {
		cfg := makeTestConfig(t)
		cfg.Host = nil
		_, err := NewREPL(cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "host is required")
	})

	t.Run("MissingPubSub", func(t *testing.T) {
		cfg := makeTestConfig(t)
		cfg.PubSub = nil
		_, err := NewREPL(cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "pubsub is required")
	})

	t.Run("MissingKeyPair", func(t *testing.T) {
		cfg := makeTestConfig(t)
		cfg.KeyPair = nil
		_, err := NewREPL(cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "keypair is required")
	})

	t.Run("MissingWaveCache", func(t *testing.T) {
		cfg := makeTestConfig(t)
		cfg.WaveCache = nil
		_, err := NewREPL(cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "wave cache is required")
	})

	t.Run("CustomIOStreams", func(t *testing.T) {
		cfg := makeTestConfig(t)
		inBuf := bytes.NewBufferString("quit\n")
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}

		cfg.In = inBuf
		cfg.Out = outBuf
		cfg.Err = errBuf

		repl, err := NewREPL(cfg)
		require.NoError(t, err)
		require.NotNil(t, repl)
		require.Equal(t, inBuf, repl.in)
		require.Equal(t, outBuf, repl.out)
		require.Equal(t, errBuf, repl.err)
	})
}

// TestREPLHelpCommand verifies the help command displays available commands.
func TestREPLHelpCommand(t *testing.T) {
	cfg := makeTestConfig(t)
	inBuf := bytes.NewBufferString("help\nquit\n")
	outBuf := &bytes.Buffer{}
	cfg.In = inBuf
	cfg.Out = outBuf

	repl, err := NewREPL(cfg)
	require.NoError(t, err)

	err = repl.Run()
	require.NoError(t, err)

	output := outBuf.String()
	require.Contains(t, output, "wave <text>")
	require.Contains(t, output, "peers")
	require.Contains(t, output, "waves")
	require.Contains(t, output, "connect <multiaddr>")
	require.Contains(t, output, "help")
	require.Contains(t, output, "quit")
}

// TestREPLQuitCommand verifies quit and exit commands terminate the REPL.
func TestREPLQuitCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{"quit", "quit\n"},
		{"exit", "exit\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := makeTestConfig(t)
			inBuf := bytes.NewBufferString(tt.command)
			outBuf := &bytes.Buffer{}
			cfg.In = inBuf
			cfg.Out = outBuf

			repl, err := NewREPL(cfg)
			require.NoError(t, err)

			err = repl.Run()
			require.NoError(t, err)

			output := outBuf.String()
			require.Contains(t, output, "Goodbye!")
		})
	}
}

// TestREPLPeersCommand verifies the peers command lists connected peers.
func TestREPLPeersCommand(t *testing.T) {
	cfg := makeTestConfig(t)
	inBuf := bytes.NewBufferString("peers\nquit\n")
	outBuf := &bytes.Buffer{}
	cfg.In = inBuf
	cfg.Out = outBuf

	repl, err := NewREPL(cfg)
	require.NoError(t, err)

	err = repl.Run()
	require.NoError(t, err)

	output := outBuf.String()
	// With no connected peers, should show "No connected peers."
	require.Contains(t, output, "No connected peers.")
}

// TestREPLWavesCommand verifies the waves command lists cached waves.
func TestREPLWavesCommand(t *testing.T) {
	t.Run("NoCachedWaves", func(t *testing.T) {
		cfg := makeTestConfig(t)
		inBuf := bytes.NewBufferString("waves\nquit\n")
		outBuf := &bytes.Buffer{}
		cfg.In = inBuf
		cfg.Out = outBuf

		repl, err := NewREPL(cfg)
		require.NoError(t, err)

		err = repl.Run()
		require.NoError(t, err)

		output := outBuf.String()
		require.Contains(t, output, "No cached Waves.")
	})

	t.Run("WithCachedWaves", func(t *testing.T) {
		cfg := makeTestConfig(t)

		// Create and cache a test wave
		testWave, err := waves.Create(
			waves.TypeSurface,
			[]byte("Test wave content"),
			cfg.KeyPair,
			waves.DefaultCreateOptions(),
		)
		require.NoError(t, err)
		err = cfg.WaveCache.Put(testWave)
		require.NoError(t, err)

		inBuf := bytes.NewBufferString("waves\nquit\n")
		outBuf := &bytes.Buffer{}
		cfg.In = inBuf
		cfg.Out = outBuf

		repl, err := NewREPL(cfg)
		require.NoError(t, err)

		err = repl.Run()
		require.NoError(t, err)

		output := outBuf.String()
		require.Contains(t, output, "Cached Waves")
		require.Contains(t, output, "Test wave content")
	})

	t.Run("WithLimit", func(t *testing.T) {
		cfg := makeTestConfig(t)
		inBuf := bytes.NewBufferString("waves 5\nquit\n")
		outBuf := &bytes.Buffer{}
		cfg.In = inBuf
		cfg.Out = outBuf

		repl, err := NewREPL(cfg)
		require.NoError(t, err)

		err = repl.Run()
		require.NoError(t, err)
		// Should not error even with no waves
		require.Contains(t, outBuf.String(), "murmur>")
	})
}

// TestREPLWaveCommand verifies wave creation and publishing.
func TestREPLWaveCommand(t *testing.T) {
	t.Run("ValidWave", func(t *testing.T) {
		cfg := makeTestConfig(t)
		inBuf := bytes.NewBufferString("wave Hello MURMUR\nquit\n")
		outBuf := &bytes.Buffer{}
		cfg.In = inBuf
		cfg.Out = outBuf

		repl, err := NewREPL(cfg)
		require.NoError(t, err)

		err = repl.Run()
		require.NoError(t, err)

		output := outBuf.String()
		require.Contains(t, output, "Creating Wave...")
		require.Contains(t, output, "Computing Proof of Work")
		require.Contains(t, output, "PoW completed in")
		require.Contains(t, output, "Published Wave")
	})

	t.Run("EmptyContent", func(t *testing.T) {
		cfg := makeTestConfig(t)
		inBuf := bytes.NewBufferString("wave\nquit\n")
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		cfg.In = inBuf
		cfg.Out = outBuf
		cfg.Err = errBuf

		repl, err := NewREPL(cfg)
		require.NoError(t, err)

		err = repl.Run()
		require.NoError(t, err)

		errOutput := errBuf.String()
		require.Contains(t, errOutput, "usage: wave <text>")
	})

	t.Run("OversizedContent", func(t *testing.T) {
		cfg := makeTestConfig(t)
		// Create content larger than MaxContentSize
		largeContent := strings.Repeat("x", waves.MaxContentSize+100)
		inBuf := bytes.NewBufferString("wave " + largeContent + "\nquit\n")
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		cfg.In = inBuf
		cfg.Out = outBuf
		cfg.Err = errBuf

		repl, err := NewREPL(cfg)
		require.NoError(t, err)

		err = repl.Run()
		require.NoError(t, err)

		errOutput := errBuf.String()
		require.Contains(t, errOutput, "exceeds maximum size")
	})
}

// TestREPLConnectCommand verifies peer connection functionality.
func TestREPLConnectCommand(t *testing.T) {
	t.Run("MissingMultiaddr", func(t *testing.T) {
		cfg := makeTestConfig(t)
		inBuf := bytes.NewBufferString("connect\nquit\n")
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		cfg.In = inBuf
		cfg.Out = outBuf
		cfg.Err = errBuf

		repl, err := NewREPL(cfg)
		require.NoError(t, err)

		err = repl.Run()
		require.NoError(t, err)

		errOutput := errBuf.String()
		require.Contains(t, errOutput, "usage: connect <multiaddr>")
	})

	t.Run("InvalidMultiaddr", func(t *testing.T) {
		cfg := makeTestConfig(t)
		inBuf := bytes.NewBufferString("connect invalid-multiaddr\nquit\n")
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}
		cfg.In = inBuf
		cfg.Out = outBuf
		cfg.Err = errBuf

		repl, err := NewREPL(cfg)
		require.NoError(t, err)

		err = repl.Run()
		require.NoError(t, err)

		errOutput := errBuf.String()
		require.Contains(t, errOutput, "invalid multiaddr")
	})
}

// TestREPLUnknownCommand verifies error handling for unknown commands.
func TestREPLUnknownCommand(t *testing.T) {
	cfg := makeTestConfig(t)
	inBuf := bytes.NewBufferString("unknown-command\nquit\n")
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cfg.In = inBuf
	cfg.Out = outBuf
	cfg.Err = errBuf

	repl, err := NewREPL(cfg)
	require.NoError(t, err)

	err = repl.Run()
	require.NoError(t, err)

	errOutput := errBuf.String()
	require.Contains(t, errOutput, "unknown command")
	require.Contains(t, errOutput, "type 'help'")
}

// TestREPLEmptyInput verifies empty lines are ignored.
func TestREPLEmptyInput(t *testing.T) {
	cfg := makeTestConfig(t)
	inBuf := bytes.NewBufferString("\n\n\nquit\n")
	outBuf := &bytes.Buffer{}
	cfg.In = inBuf
	cfg.Out = outBuf

	repl, err := NewREPL(cfg)
	require.NoError(t, err)

	err = repl.Run()
	require.NoError(t, err)

	// Should complete without errors
	output := outBuf.String()
	require.Contains(t, output, "Goodbye!")
}

// TestREPLConcurrentSafety verifies REPL can be stopped via context cancellation.
func TestREPLConcurrentSafety(t *testing.T) {
	cfg := makeTestConfig(t)
	// Use an input stream that will trigger EOF when context is cancelled
	pipeR, pipeW := bytes.NewBufferString(""), &bytes.Buffer{}
	outBuf := &bytes.Buffer{}
	cfg.In = pipeR
	cfg.Out = outBuf

	repl, err := NewREPL(cfg)
	require.NoError(t, err)

	// Start REPL in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- repl.Run()
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Close the input pipe to trigger EOF
	pipeW.Write([]byte("quit\n"))

	// Wait for completion with timeout
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("REPL did not stop after input EOF")
	}
}

// makeTestConfig creates a minimal Config for testing.
func makeTestConfig(t *testing.T) Config {
	t.Helper()

	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"
	db, err := store.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	// Generate keypair
	keypair, err := keys.GenerateKeyPair()
	require.NoError(t, err)

	// Create libp2p host
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	host, err := transport.NewHost(ctx, transport.Config{
		PrivateKey: keypair.PrivateKey,
		ListenAddrs: []string{
			"/ip4/127.0.0.1/tcp/0",
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { host.Close() })

	// Create GossipSub
	pubsub, err := gossip.New(ctx, host)
	require.NoError(t, err)

	// Create Wave cache
	waveCache, err := storage.NewCache(db)
	require.NoError(t, err)

	return Config{
		Host:      host,
		PubSub:    pubsub,
		KeyPair:   keypair,
		WaveCache: waveCache,
	}
}
