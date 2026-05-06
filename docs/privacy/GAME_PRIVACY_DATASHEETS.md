# Game Privacy Datasheets

**Purpose:** Per-game privacy disclosures per PLAN.md §2.5

Each game has a privacy datasheet documenting:
1. Metadata collected during gameplay
2. Anonymity guarantees provided
3. Known limitations and leak surfaces
4. Recommended user precautions

---

## Cipher Puzzles

### Metadata Collected
- **Participation timing:** When you announce a puzzle, submit a solution, or view active puzzles (minute-level precision)
- **Solution attempts:** Number of solutions you submit per puzzle (visible to all observers)
- **Win/loss record:** Which puzzles you solved first (public for Fragment puzzles)

### Anonymity Guarantees
✅ **No real-time latency fingerprints** — All interactions on 15–60 minute timescales  
✅ **No behavioral fingerprints** — Solution submission is a single Wave; no distinctive interaction pattern  
✅ **No location metadata** — Puzzles are network-global; participation reveals no geographic proximity  

### Known Limitations
⚠️ **Participation is public** — Anyone observing the Anonymous Layer can see which Specters participate in which puzzles  
⚠️ **Skill level observable** — Consistent puzzle wins may reveal computational resources or cryptographic expertise  

### Recommended Precautions
- **For casual participants:** No additional precautions required
- **For Fortress-mode users:** Participation timing is coarse enough that Shroud routing alone provides strong anonymity

**Privacy Rating:** ✅ **ZERO-LEAK** — Best-in-class anonymity

---

## Specter Hunts

### Metadata Collected
- **Participation timing:** When you join a hunt, claim fragments, and view hunt announcements (minute-level)
- **Fragment claim locations:** Which fragments you claim (reveals approximate network position via Pulse Map topology)
- **Claim sequence:** Order in which you claim fragments (reveals exploration path)

### Anonymity Guarantees
✅ **No sub-minute timing** — All interactions on multi-minute timescales  
✅ **No direct IP correlation** — Fragment locations are network-topological, not geographic  
✅ **Coarse location only** — Fragment proximity reveals "near node X" but not specific connections  

### Known Limitations
⚠️ **Topology-based location** — Claiming fragments near specific nodes reveals you have network proximity to those nodes (within 3 hops)  
⚠️ **Exploration patterns** — Consistent hunt participation may reveal your typical network region  

### Recommended Precautions
- **For Hybrid/Guarded users:** Hunt participation is safe; topology-based location is inherent to the mechanic and acceptable
- **For Fortress users:** If your threat model includes adversaries who can correlate network topology with geographic locations, consider avoiding hunts or using Tor transport

**Privacy Rating:** ⚠️ **LOW-LEAK** — Topology metadata inherent to mechanic

---

## Territory Drift

### Metadata Collected
- **Influence activity:** Waves amplified, connections formed, and mechanics participated in within specific territories (30-day lookback)
- **Territory control:** Which territories you control or contest (public)

### Anonymity Guarantees
✅ **No new metadata** — Territory influence is computed from existing activity (Waves, connections, mechanics); no additional data collection  
✅ **Persistent but coarse** — Territory boundaries shift dynamically; controller status does not reveal precise network position  

### Known Limitations
⚠️ **Activity concentration visible** — Persistent territory control reveals sustained presence in a network region  
⚠️ **Long-term behavioral patterns** — Controlling the same territory over weeks may aid adversary profiling  

### Recommended Precautions
- **For all users:** Territory Drift is safe; it aggregates existing activity and introduces no new leak vectors
- **For paranoid users:** Rotate your network region periodically (change bootstrap peers, form new connections) to avoid persistent territorial association

**Privacy Rating:** ✅ **ZERO-LEAK** — Aggregates existing data only

---

## Oracle Pools

### Metadata Collected
- **Commitment timing:** When you submit a prediction commitment (minute-level)
- **Reveal timing:** When you reveal your prediction after the deadline (minute-level)
- **Prediction accuracy:** Your prediction value and rank among participants (public after resolution)

### Anonymity Guarantees
✅ **Commitment scheme prevents timing leaks** — Hash-then-reveal design eliminates distinctive submission patterns  
✅ **No interactive timing** — Predictions are non-interactive; no latency fingerprints  
✅ **Deterministic resolution** — Outcome verification is network-observable; no trusted authority  

### Known Limitations
⚠️ **Prediction patterns** — Consistently accurate predictions across many pools may reveal domain expertise  
⚠️ **Participation history** — Observers can see which Specters participate in which pools over time  

### Recommended Precautions
- **For all users:** Oracle Pools are safe; commitment scheme provides strong timing protection
- **For users concerned about behavioral profiling:** Avoid participating in pools on topics where your expertise is identifying

**Privacy Rating:** ✅ **ZERO-LEAK** — Commitment scheme eliminates timing metadata

---

## Sigil Forge

### Metadata Collected
- **Entry submission timing:** When you submit creative entries (minute-level within 30–60 minute window)
- **Amplification received:** How many amplifications your entries receive (public)
- **Entry content:** Your Sigil Art, Micro Fiction, or Remix Chain contributions (public)

### Anonymity Guarantees
✅ **No real-time interaction** — Submissions over 30–60 minute windows; evaluation period is async  
✅ **No distinctive timing** — Entry submission follows standard Wave timing; no behavioral fingerprint  
✅ **Content unlinkable** — Entries are Specter-signed; no link to Surface identity  

### Known Limitations
⚠️ **Creative style fingerprinting** — Consistent artistic or writing style across multiple Forge events may aid deanonymization  
⚠️ **Participation frequency** — Entering every Forge may reveal a pattern  

### Recommended Precautions
- **For casual participants:** No additional precautions required
- **For Fortress users concerned about style fingerprinting:** Vary your creative approach, avoid signature techniques, or participate selectively

**Privacy Rating:** ✅ **ZERO-LEAK** — Standard Wave timing with no new metadata

---

## Shadow Play

### Metadata Collected
- **Join timing:** When you join a Shadow Play (minute-level)
- **Discussion Wave timing:** When you post during discussion rounds (5-minute round precision)
- **Voting timing:** When you cast elimination votes (within 5-minute voting windows)
- **Interaction patterns:** Who you accuse, defend, or interact with (public within game)

### Anonymity Guarantees
✅ **Turn-based structure** — 5-minute rounds reduce real-time pressure; not synchronous chat  
✅ **Batched reveals** — Votes revealed at round end, not immediately  
✅ **High Resonance gate** — Resonance 200 requirement restricts to committed Specters with established presence  

### Known Limitations
⚠️ **Discussion timing patterns** — How quickly you respond within 5-minute rounds may create a behavioral fingerprint  
⚠️ **Interaction style** — Accusation patterns, defensive language, and social deduction strategy may be consistent across games  
⚠️ **Participation frequency** — Playing Shadow Play regularly increases correlation opportunities  

### Recommended Precautions
- **For Guarded/Fortress users:** Consider using Tor transport for Shadow Play if your threat model includes adversaries who can observe network timing at sub-minute precision
- **For all users:** Vary your discussion timing (respond early/late within rounds), avoid signature phrases, mix up your social deduction strategy
- **Advanced users:** Introduce random delays (10–60 seconds) before posting discussion Waves

**Privacy Rating:** ⚠️ **MEDIUM-LEAK** — Timing and behavioral patterns observable; mitigated by turn-based design and high Resonance gate

---

## Masked Events

### Metadata Collected
- **Join timing:** When you join an event (minute-level before event start)
- **Masked Wave timing:** When you post during the event (minute-level within 30–240 minute duration)
- **Amplification received:** How many amplifications your Masked Waves receive (used for Resonance Burst computation)

### Anonymity Guarantees
✅ **Ephemeral keypairs** — Masked keys destroyed after event; no persistent link to Specter identity  
✅ **Post-event unlinkability** — Observers cannot link Masked pseudonyms to Specter identities after event ends  
✅ **Join encryption** — Masked public keys encrypted to event creator; creator cannot link Masked identities to Specters during event  

### Known Limitations
⚠️ **Posting timing within event** — How frequently and when you post during the event creates a per-event behavioral pattern  
⚠️ **Content style** — Writing style, vocabulary, and topics may be consistent across events if you participate in multiple  
⚠️ **Resonance Burst reveals participation** — Your Specter Resonance increases after the event, revealing you participated (but not which Masked identity you used)  

### Recommended Precautions
- **For all users:** Masked Events provide strong anonymity; ephemeral keypairs are the deepest unlinkability layer in MURMUR
- **For users concerned about style fingerprinting:** Vary your writing style, avoid signature topics, limit participation frequency
- **Advanced users:** Use Tor transport to prevent network-level correlation of event participation

**Privacy Rating:** ⚠️ **LOW-LEAK** — Ephemeral keys provide post-event unlinkability; timing metadata within event is coarse

---

## Phantom Councils

### Metadata Collected
- **Membership:** Council membership is public (admission votes are visible on Anonymous Layer)
- **Activity timing:** When the council's encrypted topic receives traffic (coarse timing, content encrypted)
- **Voting records:** Council proposal votes are visible to all members (but encrypted from non-members)

### Anonymity Guarantees
✅ **End-to-end encrypted communication** — Council Waves are encrypted; non-members see only encrypted traffic  
✅ **Unanimous admission** — Every member approves new entrants; high trust threshold  
✅ **Rotated group keys** — Keys rotated on membership changes; expelled members lose future access  

### Known Limitations
⚠️ **Membership is public** — Observers know which Specters are in which councils  
⚠️ **Activity patterns visible** — Network traffic to council topics reveals when councils are active (but not what they discuss)  
⚠️ **Small group correlation** — Small councils (3–13 members) allow adversaries to narrow correlation sets  

### Recommended Precautions
- **For all council members:** Council membership is inherently semi-public; accept this tradeoff for exclusive coordination
- **For Fortress users:** Use Tor transport to prevent network-level observation of council activity timing
- **For high-sensitivity discussions:** Supplement council communication with Whisper Chains for critical messages

**Privacy Rating:** ⚠️ **LOW-LEAK** — Membership public but communication encrypted; acceptable for high-trust coordination

---

## Surface Sparks

### Metadata Collected
- **Initiation timing:** When you start a Spark (second-level precision for Echo Races)
- **Response timing:** When you respond to a Spark (second-level for Echo Races, minute-level for Wave Relays)
- **Echo Race latency:** For Echo Races, the exact time of your amplification reveals network latency to the initiator

### Anonymity Guarantees
⚠️ **NONE** — Surface Sparks are Surface Layer mechanics; no anonymity guarantees  

### Known Limitations
⚠️ **Echo Races leak latency fingerprints** — Sub-second timing reveals network position and propagation delays; **incompatible with anonymity**  
⚠️ **Wave Relays are coarse** — 5-minute response windows make timing less distinctive but still observable  

### Recommended Precautions
- **Surface Sparks are Surface Layer only** — These mechanics are NOT available on the Anonymous Layer
- **Do not participate if anonymity is required** — Echo Races inherently leak latency; avoid if your threat model includes network-level observers
- **Anonymous Layer alternative:** Use Cipher Puzzles or Sigil Forge for gamified interaction with anonymity

**Privacy Rating:** ❌ **HIGH-LEAK (Surface Layer Only)** — Real-time latency incompatible with anonymity; correctly isolated

---

## Whisper Chains

### Metadata Collected
- **Relay participation:** When you forward Whisper Chain messages (self-reported, not independently verifiable)
- **Message timing:** When you send or receive Whisper Chains (minute-level via multi-hop delays)
- **Chain length:** Number of hops in the chain (affects timing variance)

### Anonymity Guarantees
✅ **Onion routing** — Layered encryption; each hop sees only previous/next hop, not origin or destination  
✅ **Multi-hop timing obfuscation** — 2–4 hops introduce variable delays; prevents precise timing correlation  
✅ **No content leak** — Relay nodes see only encrypted blobs; message content never exposed  

### Known Limitations
⚠️ **Timing analysis possible** — Adversaries observing multiple hops may correlate send/receive timing across the chain  
⚠️ **Small anonymity set** — With 2–4 hops, the potential sender/receiver set is limited to network diameter  
⚠️ **Rate limit reveals activity** — 10 messages per 24 hours per Specter; adversaries can observe Whisper Chain frequency  

### Recommended Precautions
- **For Hybrid/Guarded users:** Whisper Chains provide good anonymity for private messaging; multi-hop delays prevent easy correlation
- **For Fortress users:** Use Tor transport for Whisper Chains if your threat model includes adversaries capable of multi-hop timing analysis
- **For maximum security:** Transition to Specter connections after initial Whisper Chain introduction; direct encrypted channels eliminate relay correlation risk

**Privacy Rating:** ⚠️ **LOW-LEAK** — Onion routing provides strong anonymity; timing analysis requires multi-hop observation

---

## Summary Table

| Game | Privacy Rating | Key Limitations | Recommended Precautions |
|------|----------------|-----------------|-------------------------|
| Cipher Puzzles | ✅ Zero-Leak | Participation public, skill level observable | None required |
| Specter Hunts | ⚠️ Low-Leak | Topology-based location, exploration patterns | Fortress users consider Tor |
| Territory Drift | ✅ Zero-Leak | Activity concentration visible | Rotate network region periodically |
| Oracle Pools | ✅ Zero-Leak | Prediction patterns, participation history | Avoid expertise-revealing topics |
| Sigil Forge | ✅ Zero-Leak | Creative style fingerprinting | Vary creative approach |
| Shadow Play | ⚠️ Medium-Leak | Discussion timing, interaction style | Use Tor, vary timing/strategy |
| Masked Events | ⚠️ Low-Leak | Posting timing, content style | Vary writing style, use Tor |
| Phantom Councils | ⚠️ Low-Leak | Membership public, activity timing | Use Tor, supplement with Whisper Chains |
| Surface Sparks | ❌ High-Leak | Real-time latency | Surface Layer only — never migrate |
| Whisper Chains | ⚠️ Low-Leak | Timing analysis, small anonymity set | Use Tor, transition to direct connections |

---

## In-App Disclosure

Before first participation in each game, display a modal:

**Shadow Play Privacy Notice**

This game observes:
- Discussion timing within 5-minute rounds
- Interaction patterns (accusations, defenses)
- Voting behavior

**Mitigation:** Shadow Play is turn-based (not real-time) and gated at Resonance 200. For maximum anonymity, enable Tor transport.

[Learn More] [Play Anyway] [Cancel]

---

**Implementation:** Integrate privacy modals into `pkg/anonymous/mechanics/` — each game's `CreateMatch()` checks a "privacy_notice_acknowledged" flag per Specter and blocks participation until acknowledged.
