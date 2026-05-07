# Bootstrap Server Operation Manual

This guide covers operating the standalone MURMUR bootstrap server at `cmd/bootstrap`.

The bootstrap server is both:
- an HTTP endpoint that serves `/peers.json` and `/health`, and
- a full libp2p bootstrap participant that joins DHT server mode and learns peers over time.

It does not require a static `peers.json` input file.

## Overview

Bootstrap servers should be:
- Highly available (99.9%+ uptime)
- Running with a persistent identity key
- Reachable via public libp2p addresses
- Able to expose HTTP discovery endpoints

The server exposes:
- `/peers.json` — dynamically generated and signed peer bundle (self + recently observed peers)
- `/health` — runtime status including peer ID and known peer count
- `/` — plaintext index

Key runtime flags:
- `-state-dir` (persistent identity + runtime state)
- `-listen` (HTTP endpoint)
- `-p2p-listen` (libp2p listen multiaddrs)
- `-announce-addrs` (optional public multiaddrs to advertise)
- `-ngrok`, `-ngrok-domain`
- `-tor`, `-tor-name`, `-tor-port`
- `-i2p`, `-i2p-name`, `-i2p-sam`

## Installation

### Build

```bash
git clone https://github.com/opd-ai/murmur.git
cd murmur
go build -o murmur-bootstrap ./cmd/bootstrap
sudo mv murmur-bootstrap /usr/local/bin/
```

### Prepare State Directory

```bash
sudo mkdir -p /var/lib/murmur-bootstrap
sudo chown murmur-bootstrap:murmur-bootstrap /var/lib/murmur-bootstrap
sudo chmod 700 /var/lib/murmur-bootstrap
```

On first start, the server creates and stores:
- `/var/lib/murmur-bootstrap/bootstrap_identity.key`

### Systemd Service

Create `/etc/systemd/system/murmur-bootstrap.service`:

```ini
[Unit]
Description=MURMUR Bootstrap Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=murmur-bootstrap
Group=murmur-bootstrap
ExecStart=/usr/local/bin/murmur-bootstrap \
    -state-dir=/var/lib/murmur-bootstrap \
    -listen=:8081 \
    -p2p-listen=/ip4/0.0.0.0/tcp/4001,/ip4/0.0.0.0/udp/4001/quic-v1
Restart=always
RestartSec=10
LimitNOFILE=65536

NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/murmur-bootstrap
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

### Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable murmur-bootstrap
sudo systemctl start murmur-bootstrap
```

## Transport Variants

### Direct HTTP + libp2p

```bash
/usr/local/bin/murmur-bootstrap \
  -state-dir=/var/lib/murmur-bootstrap \
  -listen=:8081 \
  -p2p-listen=/ip4/0.0.0.0/tcp/4001,/ip4/0.0.0.0/udp/4001/quic-v1
```

### Add ngrok HTTP ingress

```bash
NGROK_AUTHTOKEN=... /usr/local/bin/murmur-bootstrap \
  -state-dir=/var/lib/murmur-bootstrap \
  -listen=:8081 \
  -p2p-listen=/ip4/0.0.0.0/tcp/4001,/ip4/0.0.0.0/udp/4001/quic-v1 \
  -ngrok \
  -ngrok-domain=consuming-dangling-commodore.ngrok-free.dev
```

### Add Tor/I2P HTTP ingress

```bash
/usr/local/bin/murmur-bootstrap \
  -state-dir=/var/lib/murmur-bootstrap \
  -listen=:8081 \
  -p2p-listen=/ip4/0.0.0.0/tcp/4001,/ip4/0.0.0.0/udp/4001/quic-v1 \
  -tor \
  -i2p
```

### Explicit public advertisement

```bash
/usr/local/bin/murmur-bootstrap \
  -state-dir=/var/lib/murmur-bootstrap \
  -listen=:8081 \
  -p2p-listen=/ip4/0.0.0.0/tcp/4001,/ip4/0.0.0.0/udp/4001/quic-v1 \
  -announce-addrs=/dns4/bootstrap.example.org/tcp/4001,/dns4/bootstrap.example.org/udp/4001/quic-v1
```

## Docker

### Image Build

```bash
docker build -f Dockerfile.bootstrap -t murmur-bootstrap:local .
```

If your Docker builder cannot resolve `proxy.golang.org`, use host networking so
the build uses the host resolver instead of an isolated Docker DNS path:

```bash
docker build --network host -f Dockerfile.bootstrap -t murmur-bootstrap:local .
```

`Dockerfile.bootstrap` also accepts Go module environment overrides as build
arguments for restricted networks:

```bash
docker build \
  --network host \
  --build-arg GOPROXY=direct \
  --build-arg GOSUMDB=off \
  -f Dockerfile.bootstrap \
  -t murmur-bootstrap:local .
```

### Compose Example

```bash
mkdir -p bootstrap-data
docker compose -f docker-compose.bootstrap.example.yml up --build -d
```

Optional:

```bash
export NGROK_AUTHTOKEN=...
export NGROK_DOMAIN=consuming-dangling-commodore.ngrok-free.dev
export ANNOUNCE_ADDRS=/dns4/bootstrap.example.org/tcp/4001,/dns4/bootstrap.example.org/udp/4001/quic-v1
docker compose -f docker-compose.bootstrap.example.yml up --build -d
```

If module resolution is blocked in your environment, override the Go proxy for
the compose build:

```bash
export GOPROXY=direct
export GOSUMDB=off
docker compose -f docker-compose.bootstrap.example.yml build --no-cache bootstrap
docker compose -f docker-compose.bootstrap.example.yml up -d
```

## Firewall

```bash
sudo ufw allow 8081/tcp
sudo ufw allow 4001/tcp
sudo ufw allow 4001/udp
```

## Monitoring

```bash
curl http://127.0.0.1:8081/health
curl http://127.0.0.1:8081/peers.json
```

Track at minimum:
- `known_peers` from `/health`
- stability of `peer_id`
- accessibility of advertised libp2p addresses

## Maintenance

### Binary Update

```bash
cd /path/to/murmur
git pull
go build -o murmur-bootstrap ./cmd/bootstrap
sudo mv murmur-bootstrap /usr/local/bin/
sudo systemctl restart murmur-bootstrap
```

### Identity Rotation (optional)

Only do this intentionally, as it changes the bootstrap peer identity:

```bash
sudo systemctl stop murmur-bootstrap
sudo rm /var/lib/murmur-bootstrap/bootstrap_identity.key
sudo systemctl start murmur-bootstrap
```

## Security Notes

- Keep `-state-dir` backed up and permissioned tightly.
- Prefer explicit `-announce-addrs` in NAT/proxy environments.
- Treat `/peers.json` as generated output, not a hand-maintained artifact.
- For sensitive reseed workflows, place authorization in front of public endpoints.
