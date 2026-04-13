# Contributing to MURMUR

Thank you for your interest in contributing to MURMUR!

## Getting Started

### Prerequisites

- Go 1.22+
- `gofumpt` for code formatting
- Protocol Buffers compiler (`protoc`) for schema changes

### Building

```bash
go build ./...
```

### Testing

```bash
go test -race ./...
```

### Formatting

```bash
gofumpt -w -extra .
go vet ./...
```

## Documentation Reading Order

To understand MURMUR's design, read the specification documents in this order:

1. **README.md** — Project overview and quick reference
2. **DESIGN_DOCUMENT.md** — Complete specification (7 design principles, 6 subsystems)
3. **TECHNICAL_IMPLEMENTATION.md** — Technology stack, wire protocols, concurrency model
4. **ROADMAP.md** — Implementation plan and milestones
5. **Subsystem documents** (as needed):
   - `NETWORK_ARCHITECTURE.md` — libp2p, GossipSub, DHT
   - `SHADOW_GRADIENT.md` — Privacy modes (Open/Hybrid/Guarded/Fortress)
   - `WAVES.md` — Wave types and propagation
   - `RESONANCE_SYSTEM.md` — Reputation scoring
   - `ANONYMOUS_GAME_MECHANICS.md` — Mini-games and mechanics
   - `PULSE_MAP.md` — Force-directed visualization
   - `ONBOARDING.md` — Six-phase introduction flow

## Code Style

- All code must be formatted with `gofumpt -w -extra .`
- All code must pass `go vet ./...`
- Follow the package structure in `pkg/` (not `internal/`)
- Use exact cryptographic primitives as specified in SECURITY_PRIVACY.md
- Target >80% test coverage for `pkg/identity/`, `pkg/content/`, `pkg/anonymous/`

## Package Boundaries

Per TECHNICAL_IMPLEMENTATION.md §2:
- Only `pkg/pulsemap/` may import Ebitengine
- All networking, identity, content, and anonymous packages must be testable without graphics
- Communication between goroutines uses typed channels, not shared mutable state

## Pull Request Process

1. Create a feature branch from `main`
2. Write tests for new functionality
3. Ensure all tests pass: `go test -race ./...`
4. Format code: `gofumpt -w -extra .`
5. Update documentation if needed
6. Submit PR with clear description

## Security

- Do not commit secrets or key material
- Report security vulnerabilities privately
- Follow cryptographic specifications exactly — no algorithm substitutions
