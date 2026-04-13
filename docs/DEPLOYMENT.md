# MURMUR Deployment Guide

This guide covers deploying MURMUR on desktop platforms.

## Prerequisites

- Go 1.22 or later
- Git
- System dependencies for graphics (see Platform-Specific Notes)

## Building from Source

### Clone and Build

```bash
git clone https://github.com/opd-ai/murmur.git
cd murmur
go build -o murmur ./cmd/murmur
```

### Build for Specific Platform

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o murmur-linux-amd64 ./cmd/murmur

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o murmur-linux-arm64 ./cmd/murmur

# macOS AMD64 (Intel)
GOOS=darwin GOARCH=amd64 go build -o murmur-darwin-amd64 ./cmd/murmur

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o murmur-darwin-arm64 ./cmd/murmur

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o murmur-windows-amd64.exe ./cmd/murmur
```

## Platform-Specific Notes

### Linux

Install graphics dependencies:

```bash
# Debian/Ubuntu
sudo apt-get install libgl1-mesa-dev xorg-dev

# Fedora
sudo dnf install mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel

# Arch
sudo pacman -S mesa libxcursor libxrandr libxinerama libxi
```

### macOS

No additional dependencies required. Xcode Command Line Tools should be sufficient:

```bash
xcode-select --install
```

### Windows

No additional dependencies required. The Go toolchain handles everything.

## Running MURMUR

### First Launch

```bash
./murmur
```

On first launch, MURMUR will:
1. Display the Welcome screen
2. Generate your Ed25519 keypair
3. Show your unique Sigil
4. Guide you through privacy mode selection
5. Attempt to connect to bootstrap peers

### Command-Line Options

```bash
# Specify data directory
./murmur --data-dir=/path/to/data

# Specify listen addresses
./murmur --listen=/ip4/0.0.0.0/tcp/9000 --listen=/ip4/0.0.0.0/udp/9000/quic-v1

# Enable verbose logging
./murmur --verbose

# Run in Shroud relay mode
./murmur --shroud-relay
```

## Data Directory

MURMUR stores data in platform-specific locations:

| Platform | Default Location |
|----------|------------------|
| Linux    | `~/.local/share/murmur/` |
| macOS    | `~/Library/Application Support/murmur/` |
| Windows  | `%APPDATA%\murmur\` |

### Data Directory Contents

```
murmur/
├── murmur.db          # Bbolt database (identity, waves, peers)
├── keystore.enc       # Encrypted keystore
└── config.json        # User configuration
```

## Firewall Configuration

MURMUR uses the following ports by default:

| Protocol | Default Port | Purpose |
|----------|--------------|---------|
| TCP      | Dynamic (0)  | libp2p connections |
| UDP/QUIC | Dynamic (0)  | Fast transport |

For better connectivity, configure your firewall to allow outbound connections on all ports, and optionally forward a specific port for inbound connections.

### UFW (Linux)

```bash
# Allow outbound
sudo ufw allow out to any

# Allow specific inbound port (optional)
sudo ufw allow 9000/tcp
sudo ufw allow 9000/udp
```

### macOS Firewall

System Preferences → Security & Privacy → Firewall → Firewall Options → Allow incoming connections for MURMUR.

## Backup and Recovery

### Backup Your Identity

Your identity is stored in the encrypted keystore. Back up the entire data directory:

```bash
# Linux/macOS
tar -czf murmur-backup.tar.gz ~/.local/share/murmur/

# Windows (PowerShell)
Compress-Archive -Path "$env:APPDATA\murmur" -DestinationPath murmur-backup.zip
```

### Recovery

To restore:

```bash
# Linux/macOS
tar -xzf murmur-backup.tar.gz -C ~

# Windows (PowerShell)
Expand-Archive -Path murmur-backup.zip -DestinationPath "$env:APPDATA"
```

## Troubleshooting

### Connection Issues

1. **No peers found**: Wait a few minutes for DHT discovery. Ensure you have internet connectivity.

2. **NAT traversal failing**: Try connecting from a different network, or configure port forwarding.

3. **Firewall blocking**: Check firewall settings and ensure MURMUR is allowed.

### Graphics Issues

1. **Black screen**: Ensure graphics drivers are up to date.

2. **Poor performance**: Try disabling effects in settings.

3. **HiDPI issues**: Set `MURMUR_SCALE=2` environment variable.

### Database Issues

1. **Corrupted database**: Delete `murmur.db` to start fresh (loses local data).

2. **Locked database**: Ensure only one MURMUR instance is running.

## Resource Limits

MURMUR enforces the following default limits:

| Resource | Limit |
|----------|-------|
| Memory | 256 MiB |
| Connections | 128 |
| Bandwidth | ~50 KB/s sustained |

These can be adjusted via configuration.

## Security Considerations

1. **Protect your keystore**: The encrypted keystore contains your identity. Keep backups secure.

2. **Network exposure**: Running as a Shroud relay exposes your IP to more peers.

3. **Privacy mode**: Choose your privacy mode carefully. Fortress mode provides maximum anonymity but limited connectivity.

---

For more information, see:
- `DESIGN_DOCUMENT.md` — Complete specification
- `SECURITY_PRIVACY.md` — Security model details
- `SHADOW_GRADIENT.md` — Privacy mode details
