// Package cli provides an interactive command-line interface for MURMUR.
// Per AUDIT.md remediation, this allows testing networking/content features
// without requiring the GUI to be complete.
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/opd-ai/murmur/pkg/content/storage"
	"github.com/opd-ai/murmur/pkg/content/waves"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
	"github.com/opd-ai/murmur/pkg/networking/transport"
	pb "github.com/opd-ai/murmur/proto"
	"google.golang.org/protobuf/proto"
)

// REPL provides an interactive command-line interface for MURMUR.
type REPL struct {
	host      *transport.Host
	pubsub    *gossip.PubSub
	keypair   *keys.KeyPair
	waveCache *storage.Cache
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	// Input/output streams for testing.
	in  io.Reader
	out io.Writer
	err io.Writer
}

// Config configures the REPL.
type Config struct {
	Host      *transport.Host
	PubSub    *gossip.PubSub
	KeyPair   *keys.KeyPair
	WaveCache *storage.Cache

	// Optional: Custom I/O streams (defaults to os.Stdin/Stdout/Stderr).
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

// NewREPL creates a new interactive REPL with the given subsystems.
func NewREPL(cfg Config) (*REPL, error) {
	if cfg.Host == nil {
		return nil, errors.New("host is required")
	}
	if cfg.PubSub == nil {
		return nil, errors.New("pubsub is required")
	}
	if cfg.KeyPair == nil {
		return nil, errors.New("keypair is required")
	}
	if cfg.WaveCache == nil {
		return nil, errors.New("wave cache is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Default to standard streams if not specified.
	in := cfg.In
	if in == nil {
		in = os.Stdin
	}
	out := cfg.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := cfg.Err
	if errOut == nil {
		errOut = os.Stderr
	}

	return &REPL{
		host:      cfg.Host,
		pubsub:    cfg.PubSub,
		keypair:   cfg.KeyPair,
		waveCache: cfg.WaveCache,
		ctx:       ctx,
		cancel:    cancel,
		in:        in,
		out:       out,
		err:       errOut,
	}, nil
}

// Run starts the REPL and blocks until quit command or context cancellation.
func (r *REPL) Run() error {
	// Start background goroutine to print incoming Waves.
	r.wg.Add(1)
	go r.printIncomingWaves()

	fmt.Fprintln(r.out, "MURMUR CLI — Interactive Mode")
	fmt.Fprintln(r.out, "Type 'help' for available commands, 'quit' to exit.")
	fmt.Fprintln(r.out)

	scanner := bufio.NewScanner(r.in)

	for {
		fmt.Fprint(r.out, "murmur> ")

		if !scanner.Scan() {
			// EOF or error.
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if err := r.handleCommand(line); err != nil {
			if errors.Is(err, errQuit) {
				break
			}
			fmt.Fprintf(r.err, "Error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(r.err, "Scanner error: %v\n", err)
	}

	// Clean shutdown.
	r.cancel()
	r.wg.Wait()

	fmt.Fprintln(r.out, "Goodbye!")
	return nil
}

var errQuit = errors.New("quit requested")

// handleCommand parses and executes a single command.
func (r *REPL) handleCommand(line string) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "wave":
		return r.cmdWave(args)
	case "peers":
		return r.cmdPeers(args)
	case "waves":
		return r.cmdWaves(args)
	case "connect":
		return r.cmdConnect(args)
	case "help":
		return r.cmdHelp(args)
	case "quit", "exit":
		return errQuit
	default:
		return fmt.Errorf("unknown command: %s (type 'help' for available commands)", cmd)
	}
}

// cmdWave creates and publishes a Wave.
// Usage: wave <text>
func (r *REPL) cmdWave(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: wave <text>")
	}

	content := strings.Join(args, " ")
	if len(content) > waves.MaxContentSize {
		return fmt.Errorf("content exceeds maximum size (%d bytes)", waves.MaxContentSize)
	}

	fmt.Fprintln(r.out, "Creating Wave...")
	fmt.Fprintln(r.out, "Computing Proof of Work (this may take 2-5 seconds)...")

	start := time.Now()

	// Create Wave with PoW.
	wave, err := waves.Create(
		waves.TypeSurface,
		[]byte(content),
		r.keypair,
		waves.DefaultCreateOptions(),
	)
	if err != nil {
		return fmt.Errorf("creating wave: %w", err)
	}

	duration := time.Since(start)
	fmt.Fprintf(r.out, "PoW completed in %.2f seconds\n", duration.Seconds())

	// Store in local cache.
	if err := r.waveCache.Put(wave); err != nil {
		return fmt.Errorf("storing wave: %w", err)
	}

	// Wrap in MurmurEnvelope.
	envelope := &pb.MurmurEnvelope{
		Version:       1,
		Type:          pb.MessageType_MESSAGE_TYPE_WAVE,
		Payload:       mustMarshal(wave),
		SenderPubkey:  r.keypair.PublicKey,
		Signature:     wave.Signature,
		TimestampUnix: time.Now().Unix(),
		MessageId:     wave.WaveId,
	}

	// Publish to /murmur/waves/1 topic.
	envelopeBytes := mustMarshal(envelope)
	if err := r.pubsub.Publish(r.ctx, "/murmur/waves/1", envelopeBytes); err != nil {
		return fmt.Errorf("publishing wave: %w", err)
	}

	fmt.Fprintf(r.out, "Published Wave [%x]\n", wave.WaveId[:8])
	return nil
}

// cmdPeers lists connected peers.
// Usage: peers
func (r *REPL) cmdPeers(args []string) error {
	peerIDs := r.host.Network().Peers()

	if len(peerIDs) == 0 {
		fmt.Fprintln(r.out, "No connected peers.")
		return nil
	}

	fmt.Fprintf(r.out, "Connected peers (%d):\n", len(peerIDs))
	for _, pid := range peerIDs {
		conns := r.host.Network().ConnsToPeer(pid)
		if len(conns) > 0 {
			fmt.Fprintf(r.out, "  %s (%s)\n", pid.String(), conns[0].RemoteMultiaddr().String())
		} else {
			fmt.Fprintf(r.out, "  %s\n", pid.String())
		}
	}

	return nil
}

// cmdWaves lists cached Waves.
// Usage: waves [limit]
func (r *REPL) cmdWaves(args []string) error {
	limit := 10
	if len(args) > 0 {
		if _, err := fmt.Sscanf(args[0], "%d", &limit); err != nil {
			return fmt.Errorf("invalid limit: %v", err)
		}
	}

	waveList, err := r.waveCache.List(limit)
	if err != nil {
		return fmt.Errorf("listing waves: %w", err)
	}

	if len(waveList) == 0 {
		fmt.Fprintln(r.out, "No cached Waves.")
		return nil
	}

	fmt.Fprintf(r.out, "Cached Waves (showing %d):\n", len(waveList))
	for i, wave := range waveList {
		timestamp := time.Unix(wave.CreatedAt, 0).Format("2006-01-02 15:04:05")
		content := string(wave.Content)
		if len(content) > 50 {
			content = content[:47] + "..."
		}
		fmt.Fprintf(r.out, "  %d. [%s] %s\n", i+1, timestamp, content)
	}

	return nil
}

// cmdConnect connects to a peer by multiaddr.
// Usage: connect <multiaddr>
func (r *REPL) cmdConnect(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: connect <multiaddr>")
	}

	addrStr := args[0]
	maddr, err := multiaddr.NewMultiaddr(addrStr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr: %w", err)
	}

	// Extract peer ID from multiaddr.
	addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("extracting peer info: %w", err)
	}

	fmt.Fprintf(r.out, "Connecting to %s...\n", addrInfo.ID.String())

	if err := r.host.Connect(r.ctx, *addrInfo); err != nil {
		return fmt.Errorf("connecting to peer: %w", err)
	}

	fmt.Fprintln(r.out, "Connected successfully!")
	return nil
}

// cmdHelp displays available commands.
// Usage: help
func (r *REPL) cmdHelp(args []string) error {
	help := `Available commands:

  wave <text>         Create and publish a Wave with the given text
  peers               List all connected peers
  waves [limit]       List cached Waves (default limit: 10)
  connect <multiaddr> Connect to a peer by multiaddr
  help                Show this help message
  quit, exit          Exit the REPL

Examples:
  wave Hello, MURMUR!
  peers
  waves 20
  connect /ip4/127.0.0.1/tcp/4001/p2p/12D3K...
`
	fmt.Fprint(r.out, help)
	return nil
}

// printIncomingWaves prints Waves as they are received.
func (r *REPL) printIncomingWaves() {
	defer r.wg.Done()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastCheck := time.Now()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			lastCheck = r.processWaveUpdates(lastCheck)
		}
	}
}

// processWaveUpdates fetches and prints new Waves since lastCheck.
func (r *REPL) processWaveUpdates(lastCheck time.Time) time.Time {
	waves, err := r.waveCache.List(100)
	if err != nil {
		return lastCheck
	}

	for _, wave := range waves {
		waveTime := time.Unix(wave.CreatedAt, 0)
		if waveTime.After(lastCheck) {
			r.printWave(wave, waveTime)
		}
	}

	return time.Now()
}

// printWave formats and prints a single Wave to the output.
func (r *REPL) printWave(wave *pb.Wave, waveTime time.Time) {
	content := string(wave.Content)
	if len(content) > 80 {
		content = content[:77] + "..."
	}
	fmt.Fprintf(r.out, "\n[%s] Received: %s\nmurmur> ",
		waveTime.Format("15:04:05"), content)
}

// mustMarshal marshals a protobuf message, panicking on error.
func mustMarshal(msg proto.Message) []byte {
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal proto: %v", err))
	}
	return data
}
