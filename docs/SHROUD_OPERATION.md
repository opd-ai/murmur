# Shroud Node Operation Guide

This guide covers operating a MURMUR Shroud relay node to support anonymous traffic routing.

## Overview

Shroud nodes form the anonymous overlay network that routes traffic for users in Guarded and Fortress modes. They are critical infrastructure for privacy.

### What Shroud Nodes Do

- Relay encrypted traffic between circuit hops
- Advertise availability via beacon protocol
- Earn Resonance bonuses for service
- Never see plaintext content (onion encryption)

## Requirements

### Hardware

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 2 cores | 4+ cores |
| RAM | 4 GB | 8+ GB |
| Storage | 20 GB SSD | 50+ GB SSD |
| Network | 100 Mbps | 500+ Mbps |

Higher bandwidth enables more circuit capacity and earns more Resonance.

### Network

- Stable, low-latency connection
- Static IP or reliable dynamic DNS
- Minimal packet loss (<0.1%)

## Installation

### Build with Shroud Support

```bash
git clone https://github.com/opd-ai/murmur.git
cd murmur
go build -tags shroud -o murmur ./cmd/murmur
sudo mv murmur /usr/local/bin/
```

### Create Service Configuration

Create `/etc/systemd/system/murmur-shroud.service`:

```ini
[Unit]
Description=MURMUR Shroud Relay Node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=murmur
Group=murmur
ExecStart=/usr/local/bin/murmur \
    --data-dir=/var/lib/murmur-shroud \
    --listen=/ip4/0.0.0.0/tcp/9001 \
    --listen=/ip4/0.0.0.0/udp/9001/quic-v1 \
    --shroud-relay \
    --shroud-bandwidth=50000000 \
    --verbose
Restart=always
RestartSec=10
LimitNOFILE=65536

# Higher memory limit for Shroud relay
MemoryMax=1G

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/murmur-shroud
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

### Enable and Start

```bash
sudo mkdir -p /var/lib/murmur-shroud
sudo chown murmur:murmur /var/lib/murmur-shroud
sudo systemctl daemon-reload
sudo systemctl enable murmur-shroud
sudo systemctl start murmur-shroud
```

## Configuration Options

### Bandwidth Allocation

Specify maximum bandwidth to allocate to Shroud relay traffic:

```bash
--shroud-bandwidth=50000000  # 50 MB/s (400 Mbps)
```

Choose based on your connection capacity. Reserve bandwidth for other services.

### Circuit Limits

Limit concurrent circuits to manage resource usage:

```bash
--shroud-max-circuits=1000
```

### Beacon Interval

Control how often your node advertises availability:

```bash
--shroud-beacon-interval=5m  # Default: 5 minutes
```

## Security Considerations

### What Shroud Nodes See

- Encrypted circuit cells (no plaintext)
- IP addresses of adjacent hops only
- Timing of traffic (potential correlation attack vector)

### What Shroud Nodes DON'T See

- Message content (end-to-end encrypted)
- Message origin (onion routing)
- Message destination (known only to exit hop)

### Operational Security

1. **Separate identity**: Run Shroud on dedicated hardware/VM
2. **Legal awareness**: Understand your jurisdiction's relay laws
3. **Logging**: Disable verbose logging in production
4. **Monitoring**: Watch for abuse patterns

## Monitoring

### Key Metrics

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| Active circuits | Current circuit count | >80% of max |
| Bandwidth used | Current relay bandwidth | >90% of allocated |
| Memory usage | Process memory | >80% of limit |
| CPU usage | Processing overhead | >70% sustained |
| Failed circuits | Circuit build failures | >10% |

### Health Checks

```bash
# Check service status
sudo systemctl status murmur-shroud

# View recent logs
sudo journalctl -u murmur-shroud -n 100

# Check active circuits (future feature)
curl http://localhost:9001/api/shroud/status
```

## Resonance Rewards

Operating a Shroud relay earns Resonance bonuses:

| Service Level | Uptime | Bandwidth | Daily Resonance |
|---------------|--------|-----------|-----------------|
| Basic | 95%+ | 10 Mbps | +1 |
| Standard | 99%+ | 100 Mbps | +5 |
| Premium | 99.9%+ | 500 Mbps+ | +15 |

Resonance accrues to your Specter identity and enables access to higher-tier anonymous features.

## Troubleshooting

### High Memory Usage

Shroud nodes cache circuit state. If memory grows:
1. Reduce `--shroud-max-circuits`
2. Restart service to clear state
3. Increase `MemoryMax` in systemd unit

### Circuit Build Failures

Common causes:
- NAT/firewall blocking return traffic
- Insufficient bandwidth
- Clock drift (ensure NTP is configured)

### Poor Performance

- Check network path quality (mtr/traceroute)
- Verify no bandwidth throttling
- Monitor disk I/O (Bbolt writes)

### Connection Drops

- ISP issues or rate limiting
- Firewall state table overflow
- Increase `nf_conntrack_max`

## Legal Considerations

**Note**: This is not legal advice. Consult a lawyer familiar with your jurisdiction.

Shroud relay operators may face similar legal considerations as Tor relay operators:
- Exit relay vs. middle relay liability differences
- Data retention requirements
- Law enforcement cooperation obligations

MURMUR Shroud nodes are middle relays by design—they never decrypt content or know final destinations.

## Abuse Response

If you receive abuse reports:

1. **Document everything**: Keep copies of reports
2. **Review logs**: Check for patterns (but don't log content)
3. **Rate limiting**: Consider temporary throttling
4. **Contact MURMUR community**: Report patterns for network-wide mitigation

## Best Practices Summary

1. ✅ Use dedicated hardware/VM
2. ✅ Monitor resource usage
3. ✅ Keep software updated
4. ✅ Configure proper firewall rules
5. ✅ Set appropriate bandwidth limits
6. ✅ Run NTP for clock synchronization
7. ❌ Don't log more than necessary
8. ❌ Don't run exit services
9. ❌ Don't try to decrypt traffic

---

Thank you for supporting MURMUR's anonymous layer. Your relay strengthens privacy for everyone.
