# Tunnel Host Configuration Profile

Date: 2026-05-06
Scope: PLAN.md 6.6 (Phase 6)
Audience: Tunnel operators

## Purpose

This document defines an operator-facing profile for running a MURMUR tunnel host safely.

It prioritizes:
- explicit opt-in,
- abuse-resistant defaults,
- bounded resource usage,
- privacy-preserving operation.

## Host Modes

1. Relay-only host
- For operators who forward encrypted tunnel traffic but do not exit to destination services.

2. Exit-enabled host
- For operators who permit tunnel egress to allowed destinations.
- Higher legal/abuse surface. Disabled by default.

## Recommended Default Profile

Use this as the baseline production profile.

```yaml
# tunnel-host.profile.yaml
version: 1
role: relay-only # relay-only | exit-enabled

identity:
  key_source: local_keystore
  rotate_days: 90

network:
  listen_addrs:
    - /ip4/0.0.0.0/tcp/4001
    - /ip4/0.0.0.0/udp/4001/quic-v1
  heartbeat_interval_seconds: 30

transport_modes:
  shroud_enabled: true
  tor_enabled: false
  i2p_enabled: false

tunnel:
  enabled: true
  explicit_opt_in: true
  max_concurrent_tunnels: 24
  max_tunnels_per_peer: 3
  max_bandwidth_per_tunnel_kibps: 256
  max_daily_egress_mib: 512
  idle_timeout_seconds: 120
  request_timeout_seconds: 30
  connect_timeout_seconds: 10

policy:
  default_deny_executables: true
  allowed_content_types:
    - text/plain
    - application/json
    - text/html
  blocked_content_types:
    - application/octet-stream
    - application/x-msdownload
  allowed_destinations:
    - localhost
    - 127.0.0.1
  deny_private_lan_egress: true

accounting:
  enabled: true
  settlement_window_seconds: 900
  persist_interval_seconds: 30

abuse_controls:
  rate_limit_requests_per_minute: 120
  burst_limit: 30
  strike_window_minutes: 60
  strike_threshold: 3
  cooldown_minutes: 180

observability:
  metrics_enabled: true
  metrics_bind: 127.0.0.1:9404
  log_level: info
  log_retention_days: 7
  redact_destinations: true

safety:
  panic_on_policy_misconfig: true
  refuse_on_missing_allowlist: true
```

## Exit-Enabled Profile (Advanced)

Use only when you intentionally provide egress capacity.

```yaml
# tunnel-host-exit.profile.yaml
version: 1
role: exit-enabled

transport_modes:
  shroud_enabled: true
  tor_enabled: true
  i2p_enabled: false

tunnel:
  enabled: true
  explicit_opt_in: true
  max_concurrent_tunnels: 12
  max_tunnels_per_peer: 2
  max_bandwidth_per_tunnel_kibps: 128
  max_daily_egress_mib: 256

policy:
  default_deny_executables: true
  allowed_content_types:
    - text/plain
    - application/json
  allowed_destinations:
    - localhost
    - example.internal
  deny_private_lan_egress: true

abuse_controls:
  rate_limit_requests_per_minute: 80
  burst_limit: 20
  strike_threshold: 2
  cooldown_minutes: 360
```

## Operational Runbook

1. Preflight
- Confirm explicit opt-in is enabled.
- Confirm allowlists are non-empty.
- Confirm executable payload deny rule is active.
- Confirm quota/rate limits are configured.

2. Startup checks
- Validate transport dependencies (Tor/I2P if enabled).
- Validate key presence and permissions.
- Validate profile schema and fail fast on unsafe defaults.

3. Runtime checks
- Monitor tunnel acceptance/denial rates.
- Monitor quota exhaustion and strike events.
- Alert on repeated policy violations.

4. Incident response
- Trigger temporary cooldown for violating peers.
- Publish refusal reasons according to host rights policy.
- Preserve only minimal aggregate telemetry (no deanonymizing logs).

5. Shutdown
- Graceful drain active tunnels.
- Persist accounting snapshots.
- Rotate/backup policy config and keys.

## Minimal CLI Contract (Proposed)

```bash
murmur tunnel host --profile ./tunnel-host.profile.yaml
murmur tunnel validate-profile --profile ./tunnel-host.profile.yaml
murmur tunnel status
```

## Security and Privacy Notes

- Tunnel hosting is optional and must remain disabled by default.
- Do not enable wildcard destination egress unless explicitly intended.
- Keep destination logging redacted to preserve metadata unlinkability.
- Use conservative caps first; increase only after stable operation.

## Mapping to Existing Specs

- TUNNEL_DESIGN.md: transport and addressing model.
- TUNNEL_ABUSE_POLICY.md: refusal rights, allowlists, takedown posture.
- SECURITY_PRIVACY.md: threat model and anonymity guarantees.
- TRANSPORT_ANONYMITY.md: Tor/I2P mode tradeoffs.

## Completion Criteria for PLAN 6.6

- Operator-facing tunnel hosting guide documented.
- Concrete configuration profile published with safe defaults.
- Distinct relay-only and exit-enabled profiles provided.
- Operational runbook and safety checks documented.
