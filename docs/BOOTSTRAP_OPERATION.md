# Bootstrap Server Operation Manual

This guide covers operating the standalone MURMUR bootstrap server at `cmd/bootstrap`.

The bootstrap server is a freestanding HTTP service that publishes a signed bootstrap bundle at `/peers.json`. Clients use that bundle to discover initial peers when normal bootstrap routes are unavailable or when operators want to expose alternate ingress paths such as ngrok, Tor, or I2P.

## Overview

Bootstrap servers should be:
- Highly available (99.9%+ uptime)
- Able to refresh signed peer bundles quickly
- Reachable over one or more ingress transports
- Operated with tight access and monitoring controls

The server exposes:
- `/peers.json` — signed bootstrap bundle served verbatim from `-peers-file`
- `/health` — lightweight status endpoint
- `/` — plaintext operator-facing index

The server can listen on:
- TCP via `-listen`
- ngrok via `-ngrok` and optional `-ngrok-domain`
- Tor via `-tor`, `-tor-name`, and `-tor-port`
- I2P via `-i2p`, `-i2p-name`, and `-i2p-sam`

## Requirements

### Hardware

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 2 cores | 4+ cores |
| RAM | 2 GB | 4+ GB |
| Storage | 10 GB SSD | 50+ GB SSD |
| Network | 100 Mbps | 1 Gbps |

### Software

- Linux (Ubuntu 22.04+ recommended)
- Go 1.22+
- systemd (for service management)
- Optional: ngrok account + auth token for `-ngrok`
- Optional: local Tor environment for `-tor`
- Optional: local I2P SAM bridge for `-i2p`

### Network

- Static public IP address if exposing direct TCP
- Port open for the HTTP listener you choose (for example 8081/tcp)
- Optional: Tor and I2P local services running on the host

## Installation

### Build MURMUR

```bash
git clone https://github.com/opd-ai/murmur.git
cd murmur
go build -o murmur-bootstrap ./cmd/bootstrap
sudo mv murmur-bootstrap /usr/local/bin/
```

### Prepare Signed Peer Bundle

`cmd/bootstrap` requires a signed bootstrap bundle passed through `-peers-file`.

At minimum, the file must:
- be valid JSON,
- decode as a signed peer list,
- contain at least one peer entry.

Typical operator flow:

```bash
mkdir -p /var/lib/murmur-bootstrap
cp peers.json /var/lib/murmur-bootstrap/peers.json
chmod 600 /var/lib/murmur-bootstrap/peers.json
```

The bootstrap server does not generate or sign `peers.json`; it serves an already prepared signed file.

### Create Service User

```bash
sudo useradd -r -s /bin/false murmur-bootstrap
sudo mkdir -p /var/lib/murmur-bootstrap
sudo chown murmur-bootstrap:murmur-bootstrap /var/lib/murmur-bootstrap
```

### Configure Systemd Service

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
        -peers-file=/var/lib/murmur-bootstrap/peers.json \
        -listen=:8081
Restart=always
RestartSec=10
LimitNOFILE=65536

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/murmur-bootstrap
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

### Optional Transport Variants

Direct TCP only:

```bash
/usr/local/bin/murmur-bootstrap \
    -peers-file=/var/lib/murmur-bootstrap/peers.json \
    -listen=:8081
```

TCP + ngrok:

```bash
NGROK_AUTHTOKEN=... /usr/local/bin/murmur-bootstrap \
    -peers-file=/var/lib/murmur-bootstrap/peers.json \
    -listen=:8081 \
    -ngrok
```

TCP + ngrok custom domain:

```bash
NGROK_AUTHTOKEN=... /usr/local/bin/murmur-bootstrap \
    -peers-file=/var/lib/murmur-bootstrap/peers.json \
    -listen=:8081 \
    -ngrok \
    -ngrok-domain=consuming-dangling-commodore.ngrok-free.dev
```

TCP + Tor hidden service:

```bash
/usr/local/bin/murmur-bootstrap \
    -peers-file=/var/lib/murmur-bootstrap/peers.json \
    -listen=:8081 \
    -tor \
    -tor-name=murmur-bootstrap \
    -tor-port=8081
```

TCP + I2P:

```bash
/usr/local/bin/murmur-bootstrap \
    -peers-file=/var/lib/murmur-bootstrap/peers.json \
    -listen=:8081 \
    -i2p \
    -i2p-name=murmur-bootstrap \
    -i2p-sam=127.0.0.1:7656
```

All configured together:

```bash
NGROK_AUTHTOKEN=... /usr/local/bin/murmur-bootstrap \
    -peers-file=/var/lib/murmur-bootstrap/peers.json \
    -listen=:8081 \
    -ngrok \
    -tor \
    -i2p
```

### Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable murmur-bootstrap
sudo systemctl start murmur-bootstrap
```

## Configuration

### Firewall

```bash
# UFW
sudo ufw allow 8081/tcp

# iptables
sudo iptables -A INPUT -p tcp --dport 8081 -j ACCEPT
```

If you expose only ngrok, Tor, or I2P, you may not need to open a public firewall port beyond local loopback access.

### Resource Limits

Edit `/etc/security/limits.d/murmur.conf`:

```
murmur soft nofile 65536
murmur hard nofile 65536
```

### Sysctl Tuning

Edit `/etc/sysctl.d/99-murmur.conf`:

```
# Increase socket buffers
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216

# Increase connection tracking
net.netfilter.nf_conntrack_max = 262144
```

Apply: `sudo sysctl --system`

## Monitoring

### Check Service Status

```bash
sudo systemctl status murmur-bootstrap
```

### View Logs

```bash
sudo journalctl -u murmur-bootstrap -f
```

### Key Metrics to Monitor

1. **Bundle freshness**: `peers.json` timestamp and rotation cadence
2. **HTTP reachability**: `GET /health` and `GET /peers.json`
3. **Memory usage**: Should stay comfortably below normal service limits
4. **Transport reachability**: ngrok URL, onion address, or I2P destination availability
5. **Connection errors**: Listener startup and ingress failures

### Health Check

Example:

```bash
curl http://127.0.0.1:8081/health
curl http://127.0.0.1:8081/peers.json
```

### Prometheus Metrics

`cmd/bootstrap` currently exposes `/health`, but does not expose a Prometheus `/metrics` endpoint.

If you need metrics today, use an external HTTP probe and log-based monitoring around these endpoints:

```
GET /health
GET /peers.json
```

## Publishing Your Bootstrap Server

Once your node is stable (running 48+ hours without issues):

1. Verify the served bundle:
   ```bash
    curl http://127.0.0.1:8081/peers.json
   ```

2. Publish the bootstrap URL you want clients to use:
    ```
    https://<ngrok-domain>/peers.json
    http://<public-ip>:8081/peers.json
    http://<onion-address>/peers.json
    http://<i2p-destination>/peers.json
    ```

3. Distribute the URL through the appropriate trust path:
- public bootstrap distribution,
- friend-to-friend reseed,
- transport-specific recovery instructions.

## Maintenance

### Updates

```bash
cd /path/to/murmur
git pull
go build -o murmur-bootstrap ./cmd/bootstrap
sudo mv murmur-bootstrap /usr/local/bin/
sudo systemctl restart murmur-bootstrap
```

### Bundle Rotation

Replace the signed peer bundle atomically when your source peer set changes:

```bash
install -m 600 peers.json /var/lib/murmur-bootstrap/peers.json
sudo systemctl restart murmur-bootstrap
```

### Log Rotation

Add to `/etc/logrotate.d/murmur`:

```
/var/log/murmur/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
}
```

## Troubleshooting

### High CPU Usage

- Check for repeated HTTP probing or crash loops
- Verify no transport listener is repeatedly failing and restarting

### Memory Growth

- Confirm `peers.json` is not being regenerated to excessively large payloads
- Check for runaway logs or repeated listener initialization failures

### Connection Failures

- Firewall rules blocking direct TCP access
- Missing `NGROK_AUTHTOKEN` when `-ngrok` is enabled
- Tor or I2P local services not running when `-tor` or `-i2p` is enabled

### Disk Full

- Remove old bundle backups and log files
- Increase disk space for retained signed bundle history

## Security

### Best Practices

1. **Minimal exposure**: Only expose required ports
2. **Regular updates**: Keep OS and MURMUR updated
3. **Monitoring**: Set up alerts for anomalies
4. **Backups**: Keep signed bundle generation inputs and release process backed up
5. **Isolation**: Run in container or VM if possible
6. **Bundle integrity**: Never hand-edit `peers.json`; regenerate and sign it from your trusted pipeline
7. **Restricted reseed**: For sensitive recovery use-cases, prefer Tor or I2P distribution paths and add an authorization layer in front of public URLs

### DDoS Mitigation

- Rate limiting at firewall level
- Reverse-proxy or CDN limits for public HTTP endpoints where appropriate
- Connection limits per IP
- Cloud-based DDoS protection if needed

---

For questions, see the MURMUR community channels or open a GitHub issue.
