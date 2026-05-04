# Bootstrap Strategy: Zero-Infrastructure Peer Discovery

**Date:** 2026-05-04
**Status:** Proposed

## The Core Insight

GitHub's own free-tier CI/CD and hosting infrastructure, taken together, forms a surprisingly complete distributed signaling layer — not unlike how the GPL turned copyright law into a freedom engine. Every GitHub Actions workflow run is an ephemeral Linux server with a public-routable IP, internet access, and write permission to the repo's own Gist/Pages via `GITHUB_TOKEN`. The animating idea is to chain these ephemerals into a **self-reinforcing feedback loop**: each scheduled CI run spins up multiple parallel libp2p nodes (via matrix builds), they join the public IPFS DHT under a MURMUR-specific content-addressed namespace and cross-discover each other, then write a cryptographically signed merged peer list to a GitHub Gist and push a new `peers.json` to GitHub Pages. Every subsequent run reads that accumulating list before connecting, arriving pre-seeded with all peers ever seen by any prior run. Real users running MURMUR also publish their multiaddrs into the same IPFS DHT namespace, which CI harvests and re-broadcasts — so organic user growth directly improves bootstrap capacity without any user action beyond simply running the app. The system exhibits three emergent tiers: **live** (Gist, refreshed every 6 hours), **durable** (CID-indexed Pages build, survives Gist deletion), and **last-resort** (public IPFS bootstrap nodes as rendezvous entrypoint), each maintained automatically, none requiring a credit card.

## Recommended Architecture

The system is three mutually-reinforcing layers, not a list of fallbacks:

**Layer 1 — The Living Gist (fast, mutable, CI-maintained).**  A GitHub Gist (`peers.json`) holds the current signed peer list. It is writable by any CI job via `GITHUB_TOKEN` and readable by anyone at a stable raw URL — no auth required, no DNS to own; the raw URL is served by GitHub's CDN so reads are globally fast with no API rate limit. CI's matrix strategy runs four parallel ephemeral libp2p nodes per scheduled invocation, each joining from a distinct runner; they cross-announce multiaddrs via a shared IPFS DHT rendezvous key (`/murmur/bootstrap/v1`), collect all discovered multiaddrs (prior CI peers + any real users online), merge them, sign the result with a stable Ed25519 key held in a GitHub Actions secret (`BOOTSTRAP_SIGN_KEY`), and atomically update the Gist. The Gist raw URL is embedded in the binary as the primary bootstrap source.

**Layer 2 — GitHub Pages (durable, versioned, CID-indexed).** The same CI job also commits the signed `peers.json` to the `gh-pages` branch, making it available at a stable `https://opd-ai.github.io/murmur/peers.json`. Pages is served by GitHub's CDN, globally cached, and survives runner failures. Simultaneously, the CI job pushes the content to web3.storage (free tier, no credit card required, GitHub OAuth login) and records the resulting IPFS CID in a plain `cid.txt` file also committed to Pages. This gives a content-addressed `ipfs://` fallback that any IPFS-connected node (including real MURMUR users) can resolve without touching GitHub at all. The CID is immutable once pinned; only the `cid.txt` pointer is updated on each CI run.

**Layer 3 — Public IPFS DHT Namespace (passive, organic, zero-maintenance).** Every running MURMUR node — CI ephemeral or real user — announces itself under the DHT key `/murmur/bootstrap/v1` using go-libp2p's `routingDiscovery.Advertise()`. This requires zero infrastructure: the public IPFS bootstrap nodes (`/dnsaddr/bootstrap.libp2p.io/...`) are the entry points, and IPFS explicitly permits use of their DHT for rendezvous. A cold-start client calls `routingDiscovery.FindPeers()` on that key and gets back any node that has announced itself in the last TTL window — including prior CI runs and any live users. This layer is entirely passive from CI's perspective: CI contributes to it as a side effect of running, and it grows organically with user adoption.

**Reinforcing loops:**
- *CI → Gist → CI*: Each run reads the Gist before connecting, arriving with all previously seen peers. It then discovers more peers and writes them back. The list monotonically grows (with TTL pruning to evict stale entries).
- *Users → IPFS DHT → CI → Gist → Users*: Real users running MURMUR advertise on the DHT. The next CI run harvests them and publishes them to the Gist. The next user's cold start fetches the Gist and finds those real peers. Users now bootstrap off each other without CI's mediation.
- *CI matrix cross-discovery*: The four parallel runners in each CI invocation initially only know the prior Gist entries, but within their 3-minute run window they discover each other via the DHT and publish each runner's fresh multiaddr. Even if the Gist is empty (first ever run), the matrix jobs form a transient mesh and the aggregator job captures their addresses, seeding the Gist.

**Network state over time:**
- *After 1 run*: Gist contains 4 ephemeral addresses (expired) and any IPFS-DHT peers seen during the run. Pages has a valid `peers.json`. `cid.txt` holds the first content-addressed snapshot.
- *After 10 runs*: Gist has a rolling window of ~40 ephemeral addresses and any real users who ran during that period. Cold-start latency approaches 2–5s.
- *After 100 users*: The IPFS DHT namespace has enough live real-user peers that CI runs are supplementary rather than primary. The Gist still provides the fastest path (single HTTPS GET, sub-second), but IPFS DHT alone would be sufficient for bootstrap. CI overhead becomes negligible.

## Bootstrap Flow (Cold Start, Step-by-Step)

1. **App starts.** The `ResolverChain` in `pkg/networking/discovery/` is initialized with resolvers in priority order. User-supplied peers from `~/.murmur/config.toml` or the `--bootstrap` flag are **always merged in first**, before any resolver runs, because they represent explicit operator intent. The remaining resolvers then run in short-circuit order: the first one that yields ≥1 successful connection halts the chain.
2. **Resolver 1 — Gist fetch (target: <1s).** HTTP GET to `https://gist.githubusercontent.com/opd-ai/{BOOTSTRAP_GIST_ID}/raw/peers.json` (URL constructed at compile time via `ldflags -X`). Response is a JSON array of signed multiaddrs. Signature is verified against the embedded `BOOTSTRAP_VERIFY_KEY` (public key, not secret). On success, attempt connection to up to 3 random entries. If ≥1 connects → **bootstrap complete**, resolvers 2–4 skipped.
3. **Resolver 2 — GitHub Pages fetch (target: <2s, CDN-cached).** HTTP GET to `https://opd-ai.github.io/murmur/peers.json`. Same signed format. On success, attempt connection. If ≥1 connects → **bootstrap complete**, resolvers 3–4 skipped.
4. **Resolver 3 — IPFS DHT namespace query (target: <10s).** Dials the public IPFS bootstrap nodes (`/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN` et al., which are stable and permissioned for this use). Calls `routingDiscovery.FindPeers(ctx, "/murmur/bootstrap/v1")`. On success, attempt connection. If ≥1 connects → **bootstrap complete**, resolver 4 skipped.
5. **Resolver 4 — IPFS CID fallback (target: <30s).** Reads `cid.txt` from the Pages URL to get the latest IPFS CID, then fetches `https://dweb.link/ipfs/<CID>/peers.json` (Cloudflare IPFS gateway, no auth). Same signed format. If ≥1 connects → **bootstrap complete**.
6. **Failure path.** If all resolvers yield zero connections after their timeouts, the app starts in isolated mode with a UI warning and prompts the user to enter a peer address manually or retry after 60 seconds.

## CI Pipeline Design

**`bootstrap-refresh.yml`** — the sole new workflow file:

```
Trigger: schedule (every 6 hours), workflow_dispatch (manual), push to main
```

```
Job 1 — matrix probe (strategy: matrix[0..3], fail-fast: false):
  - Checkout repo, build murmur with -tags ci_bootstrap
  - Read current peers.json from Gist (via curl, GITHUB_TOKEN not needed for read)
  - Run ./murmur --ci-probe --duration=180s --peers-in=peers.json --peers-out=discovered-${{ matrix.index }}.json
    (ephemeral libp2p node: connects to prior peers, advertises on IPFS DHT, collects discovered peers, exits)
  - Upload discovered-${{ matrix.index }}.json as workflow artifact

Job 2 — aggregate (needs: matrix probe, runs-on: ubuntu-latest):
  - Download all discovered-*.json artifacts
  - Run ./murmur --ci-aggregate --sign-key=${{ secrets.BOOTSTRAP_SIGN_KEY }} \
      --inputs=discovered-*.json --output=peers.json --max-age=24h
    (merge, deduplicate, prune stale entries, sign)
  - Update Gist: curl -X PATCH https://api.github.com/gists/${{ vars.BOOTSTRAP_GIST_ID }} \
      -H "Authorization: Bearer ${{ secrets.GITHUB_TOKEN }}" --data @payload.json
  - Commit peers.json + cid.txt to gh-pages branch
  - (Optional, no new secret): Upload to web3.storage via its GitHub Action (free tier, OIDC)
```

**Secrets (GitHub Actions `secrets.*`):**
- `BOOTSTRAP_SIGN_KEY` — Ed25519 private key (base64) used to sign `peers.json`. Generated once via `murmur --gen-bootstrap-key`, public half embedded in binary as `BootstrapVerifyKey`.
- `GITHUB_TOKEN` — Auto-injected by Actions; used only for Gist PATCH and gh-pages push. No extra setup required.

**Repository variables (GitHub Actions `vars.*`, not secrets):**
- `BOOTSTRAP_GIST_ID` — The Gist ID (e.g. `a1b2c3d4e5f6...`). Set once during initial provisioning via the repository's *Variables* settings page. Referenced in workflow as `${{ vars.BOOTSTRAP_GIST_ID }}` and injected into the binary at build time via `ldflags`.

**Provisioning (one-time, manual):**
1. Create a public Gist with filename `peers.json`, content `[]`. Record the Gist ID.
2. Set `BOOTSTRAP_GIST_ID` as a repository variable.
3. Run `go run ./cmd/murmur --gen-bootstrap-key` locally; set the private key as `BOOTSTRAP_SIGN_KEY` secret; embed the public key in `pkg/networking/discovery/verify_key.go`.
4. Enable GitHub Pages from the `gh-pages` branch.
5. Trigger `bootstrap-refresh.yml` manually once to populate the Gist.

**No credit card, no external login, no DNS ownership required.**

## Components Required

| Component | Technology | Host | Updated By | Consumed By |
|-----------|------------|------|------------|-------------|
| Live peer list | GitHub Gist (`peers.json`) | GitHub (free) | CI aggregate job via `GITHUB_TOKEN` | Go `GistResolver` (HTTP GET, no auth) |
| Durable peer list | GitHub Pages (`peers.json`) | GitHub Pages (free) | CI aggregate job, gh-pages commit | Go `PagesResolver` (HTTP GET) |
| IPFS-addressed copy | web3.storage / IPFS | Cloudflare/IPFS (free) | CI aggregate job via OIDC action | Go `IPFSGatewayResolver` (reads `cid.txt` from Pages, then HTTP GET to dweb.link) |
| DHT rendezvous | Public IPFS DHT, key `/murmur/bootstrap/v1` | Distributed (IPFS bootstrap nodes) | Every running MURMUR node (CI + users) via `routingDiscovery.Advertise()` | Go `DHTNamespaceResolver` via `routingDiscovery.FindPeers()` |
| Signing key (public) | Ed25519 verify key | Embedded in binary (`verify_key.go`) | One-time provisioning | All resolver-chain peers verification |
| Signing key (private) | Ed25519 sign key | GitHub Actions secret | One-time provisioning | CI aggregate job only |
| Ephemeral CI probes | Go binary with `-tags ci_bootstrap` | GitHub Actions runners | `bootstrap-refresh.yml` matrix | IPFS DHT namespace (advertise side) |

## Changes to opd-ai/murmur

- **`pkg/config/defaults.go`** — Replace the commented-out placeholder `DefaultBootstrapPeers` slice with a `BootstrapSources` struct containing: `GistURL` (raw Gist URL, set at compile time via `ldflags`), `PagesURL` (`https://opd-ai.github.io/murmur/peers.json`), `IPFSGatewayURL` (derived from the IPFS CID stored in Pages-hosted `cid.txt`), `DHTNamespace` (`/murmur/bootstrap/v1`). The old `DefaultBootstrapPeers []string` is retained for the `--bootstrap` flag path.

- **`pkg/networking/discovery/`** — Add the following new files (existing files unchanged):
  - `resolver.go` — `BootstrapResolver` interface (`Resolve(ctx) ([]peer.AddrInfo, error)`) and `ResolverChain` struct that tries each resolver in order, returning on first success.
  - `gist_resolver.go` — `GistResolver`: HTTP GET → JSON decode → Ed25519 signature verify → return `[]peer.AddrInfo`. 2s timeout.
  - `pages_resolver.go` — `PagesResolver`: same logic against Pages URL. 3s timeout.
  - `dht_namespace_resolver.go` — `DHTNamespaceResolver`: wraps go-libp2p `routingDiscovery`, calls `Advertise()` + `FindPeers()` on `/murmur/bootstrap/v1`. 12s timeout.
  - `ipfs_gateway_resolver.go` — `IPFSGatewayResolver`: fetches `cid.txt` from Pages to get the latest IPFS CID, then fetches `https://dweb.link/ipfs/<CID>/peers.json`. 20s timeout.
  - `verify_key.go` — `BootstrapVerifyKey []byte` (32-byte Ed25519 public key, populated at provisioning time).
  - `signed_peers.go` — `SignedPeerList` protobuf-compatible struct, `Sign()` and `Verify()` helpers.

- **`.github/workflows/bootstrap-refresh.yml`** — New workflow file (described above). Matrix probe uses a `ci_bootstrap` build tag to avoid importing Ebitengine in the CI binary. No other workflows modified.

- **`cmd/murmur/`** — Add `--ci-probe` and `--ci-aggregate` subcommands (behind `//go:build ci_bootstrap` tag) that implement the ephemeral probe node and the merge/sign/publish logic. These compile only when the `ci_bootstrap` tag is set, so the production binary is unaffected.

## Fallback Chain

1. **GitHub Gist** — fastest (single HTTPS GET, ~100ms), updated every 6h by CI. *CI-maintained.*
2. **GitHub Pages** — fast (CDN-cached, ~200ms), updated every 6h alongside Gist. *CI-maintained.*
3. **IPFS DHT namespace `/murmur/bootstrap/v1`** — medium (10–15s DHT walk), populated passively by every running node. *Organic + CI-maintained.*
4. **IPFS CID fallback (`dweb.link`)** — slow (20–30s: Pages `cid.txt` fetch + IPFS gateway fetch), survives GitHub outage. *CI-maintained (content addressed, immutable once pinned).*
5. **`--bootstrap` / config file** — manual last resort, always available. *User-maintained.*
6. **Isolated mode** — app starts without network, prompts user to retry or enter peer address manually.

*Layers 1–2 are actively refreshed by CI. Layer 3 is passively maintained by the network itself. Layer 4 is immutable once pinned. Layer 5 is always available regardless of all infrastructure state.*

## Emergent Properties

- **CI becomes optional once ~50 users exist.** When the IPFS DHT namespace has 50+ regularly-online peers, the DHT resolver alone delivers bootstrap within the 30s target. CI runs degrade gracefully from *necessary* to *accelerating*.
- **The Gist becomes a community health dashboard.** The signed `peers.json` is publicly readable. Any user can inspect it to see how many peers are live, their geographic spread (from multiaddr prefixes), and when CI last ran — without any dedicated monitoring infrastructure.
- **Matrix cross-discovery primes the pump for real users.** Even before any real users exist, the four CI matrix jobs form a transient mesh every 6 hours. Any user who happens to start MURMUR during a CI window will discover those ephemeral CI nodes, which are live for 3 minutes. This provides a non-zero bootstrap probability even at day 1.
- **Users contribute to bootstrap capacity automatically.** Every MURMUR node calls `routingDiscovery.Advertise()` at startup as a side effect of DHT initialization. There is no opt-in required; users improve the network just by running the app. This mirrors BitTorrent's DHT design, where the act of downloading contributes to discoverability.
- **The signing key creates a trust anchor without a CA.** The `BootstrapVerifyKey` embedded in the binary means only CI (holder of `BOOTSTRAP_SIGN_KEY`) can publish authoritative peer lists to Layers 1 and 2. The DHT layer (Layer 3) is trustless by design — callers connect to whatever peers are advertised and rely on libp2p's Noise handshake for authentication. Layers 1/2 provide curated-but-signed lists; Layer 3 provides organic-but-unfiltered lists. The combination is more robust than either alone.
- **Infrastructure re-provisioning takes under 10 minutes.** If GitHub deletes the Gist or Pages site, a single `workflow_dispatch` on `bootstrap-refresh.yml` recreates everything. No DNS, no VPS, no ticket to file with a hosting provider.

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| GitHub Actions minutes exhausted (free tier: 2,000 min/month) | Each invocation uses ~4 × 3 min + 1 × 2 min = 14 min. At 6h intervals: 4 × 14 = ~56 min/day, ~1,680 min/month — within the 2,000 limit with margin. Reduce matrix size to 2 if needed. |
| Gist API rate-limited during cold-start surge | Gist raw URL is served by GitHub's CDN, not the API. Read path has no rate limit. Write path (CI only) is once per 6h, well under API limits. |
| `BOOTSTRAP_SIGN_KEY` secret compromised | Rotate: generate a new keypair, update the secret, release a new binary with the new `BootstrapVerifyKey`, old signed lists become unverifiable and are ignored. DHT layer is unaffected. |
| Public IPFS DHT eclipsed / Sybil-attacked | MURMUR is not relying on the DHT for routing correctness, only for peer *discovery*. All connections still use Noise XX for authentication. A Sybil attack delivers fake multiaddrs that simply fail to connect — the resolver chain moves on. |
| web3.storage free tier discontinued | The IPFS layer is a fallback, not primary. Remove `IPFSGatewayResolver` from chain; Layers 1–3 remain intact. The `cid.txt` file in Pages can be left as an empty string, which the resolver detects and skips. |
| GitHub Pages or Gist unavailable during cold start | ResolverChain moves to next layer within its timeout. Cold-start time degrades from <1s to <15s. Isolated mode is a safe final state. |
| Ephemeral CI node IPs added to peer list expire | `ci-aggregate` job prunes entries older than `--max-age=24h`. CI addresses are intentionally short-lived; their value is seeding the DHT namespace, not appearing in the long-term peer list. |
| First run has no peers in Gist (chicken-and-egg) | Matrix strategy: 4 parallel jobs all advertise on the DHT namespace independently. The aggregator collects all 4, writing 4 valid (if ephemeral) addresses. Second run has 4 prior entries and can connect to public IPFS bootstrap nodes for additional DHT peers. |

## Success Criteria

- Cold-start install discovers ≥1 peer within 30s with empty `BootstrapPeers` config
- Entire bootstrap infrastructure re-deployable by triggering one GitHub Actions workflow
- No component requires a credit card or non-GitHub login to provision
- Each CI run leaves the network *more* connected than before it ran
- Graceful degradation to manual `--bootstrap` flag if all layers fail
- Changes confined to `pkg/config/`, `pkg/networking/discovery/`, `.github/workflows/`

---

