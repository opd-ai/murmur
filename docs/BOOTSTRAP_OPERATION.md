# Bootstrap Node Operation Manual

This guide covers operating a MURMUR bootstrap node to help new peers join the network.

## Overview

Bootstrap nodes are well-known entry points that help new MURMUR peers discover the network. They should be:
- Highly available (99.9%+ uptime)
- Well-connected (multiple IP addresses, good bandwidth)
- Geographically distributed (minimize latency for global users)

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

### Network

- Static public IP address
- Ports 9000/tcp and 9000/udp open
- Optional: IPv6 connectivity

## Installation

### Build MURMUR

```bash
git clone https://github.com/opd-ai/murmur.git
cd murmur
go build -o murmur ./cmd/murmur
sudo mv murmur /usr/local/bin/
```

### Create Service User

```bash
sudo useradd -r -s /bin/false murmur
sudo mkdir -p /var/lib/murmur
sudo chown murmur:murmur /var/lib/murmur
```

### Configure Systemd Service

Create `/etc/systemd/system/murmur-bootstrap.service`:

```ini
[Unit]
Description=MURMUR Bootstrap Node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=murmur
Group=murmur
ExecStart=/usr/local/bin/murmur \
    --data-dir=/var/lib/murmur \
    --listen=/ip4/0.0.0.0/tcp/9000 \
    --listen=/ip4/0.0.0.0/udp/9000/quic-v1 \
    --listen=/ip6/::/tcp/9000 \
    --listen=/ip6/::/udp/9000/quic-v1 \
    --dht-server \
    --verbose
Restart=always
RestartSec=10
LimitNOFILE=65536

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/murmur
PrivateTmp=true

[Install]
WantedBy=multi-user.target
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
sudo ufw allow 9000/tcp
sudo ufw allow 9000/udp

# iptables
sudo iptables -A INPUT -p tcp --dport 9000 -j ACCEPT
sudo iptables -A INPUT -p udp --dport 9000 -j ACCEPT
```

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

1. **Peer count**: Number of connected peers
2. **DHT queries**: Queries served per minute
3. **Memory usage**: Should stay under 256 MiB
4. **Network I/O**: Bandwidth utilization
5. **Connection errors**: Failed connection attempts

### Prometheus Metrics (Future)

MURMUR will expose Prometheus metrics at `/metrics`:

```
murmur_peers_total
murmur_dht_queries_total
murmur_memory_bytes
murmur_connections_active
murmur_connections_failed_total
```

## Publishing Your Bootstrap Node

Once your node is stable (running 48+ hours without issues):

1. Get your peer ID:
   ```bash
   sudo -u murmur cat /var/lib/murmur/peer_id
   ```

2. Construct your multiaddr:
   ```
   /ip4/<YOUR_IP>/tcp/9000/p2p/<PEER_ID>
   /ip4/<YOUR_IP>/udp/9000/quic-v1/p2p/<PEER_ID>
   ```

3. Submit a PR to add your node to `pkg/config/bootstrap.go`:
   ```go
   var DefaultBootstrapPeers = []string{
       "/ip4/203.0.113.1/tcp/9000/p2p/12D3KooW...",
       // Add your multiaddr here
   }
   ```

## Maintenance

### Updates

```bash
cd /path/to/murmur
git pull
go build -o murmur ./cmd/murmur
sudo mv murmur /usr/local/bin/
sudo systemctl restart murmur-bootstrap
```

### Database Maintenance

The Bbolt database grows over time. Periodic compaction:

```bash
sudo systemctl stop murmur-bootstrap
# Backup first!
sudo -u murmur cp /var/lib/murmur/murmur.db /var/lib/murmur/murmur.db.bak
sudo systemctl start murmur-bootstrap
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

- Check for excessive DHT queries (possible attack)
- Verify connection limits are enforced

### Memory Growth

- Peer table growing unbounded: check cleanup goroutine
- Wave cache not expiring: verify TTL enforcement

### Connection Failures

- Firewall rules blocking traffic
- NAT/CGN issues (use static IP)
- ISP blocking P2P traffic

### Disk Full

- Prune old database entries
- Increase disk space
- Enable more aggressive GC

## Security

### Best Practices

1. **Minimal exposure**: Only expose required ports
2. **Regular updates**: Keep OS and MURMUR updated
3. **Monitoring**: Set up alerts for anomalies
4. **Backups**: Daily backups of critical data
5. **Isolation**: Run in container or VM if possible

### DDoS Mitigation

- Rate limiting at firewall level
- Connection limits per IP
- Cloud-based DDoS protection if needed

---

For questions, see the MURMUR community channels or open a GitHub issue.
