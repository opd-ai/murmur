# Murmur Extension Proposal (MEP) Process

## Overview

A **Murmur Extension Proposal (MEP)** is a lightweight process for proposing new extension points, API improvements, or protocol extensions to MURMUR. MEPs allow the community to discuss and refine proposals before they enter the core codebase.

This process is **intentionally minimal**:

- No steering committee required
- No formal voting
- Merit-based discussion in pull requests
- Stable extensions may be implemented without MEP if they match EXTENSION_CONTRACT.md

## Motivation

The MEP process serves three key goals:

1. **Transparency**: Community members can see what's being considered
2. **Quality**: Proposals are documented, reviewed, and refined before implementation
3. **Autonomy**: Core maintainers can implement extensions without bureaucracy if the proposal aligns with the Extension Contract

## Process

### 1. Create a MEP

Create a markdown file in the `MEPs/` directory with filename `MEP-NNN-title.md`, where NNN is the next sequential number.

**Minimum sections**:

- **Title**: Brief one-line summary
- **Motivation**: Why is this extension needed?
- **Proposed Interface**: What API/protocol/message type does this add?
- **Stability**: Stable, Experimental, or Private (per EXTENSION_CONTRACT.md)
- **Backward Compatibility**: Can old clients ignore this extension gracefully?
- **Security Considerations**: Any new threat surfaces?

See `MEP-0-TEMPLATE.md` for a complete template.

### 2. Open a Pull Request

Open a GitHub PR adding your MEP to the `MEPs/` folder. Describe:

- The extension's purpose
- Why it aligns with design principles
- Any implementation concerns

### 3. Community Discussion

Discuss in the PR. Questions answered might include:

- "Does this method name match our conventions?"
- "Should this be part of the core or a third-party extension?"
- "How does this interact with existing extensions?"

**No explicit approval needed** — if consensus emerges, the PR is merged.

If discussion stalls, the MEP can be merged as "Proposed" with the understanding that implementation may not happen.

### 4. Implementation (Optional)

After a MEP is merged, anyone (maintainer or community) may implement it. Implementation does not require a second approval.

Implementation MUST include:

- Reference to the MEP in code comments
- Tests validating the extension
- Documentation updates (e.g., EXTENSION_CONTRACT.md, PROTOCOL.md)

### 5. Status Tracking

Each MEP tracks its status:

- **Proposed**: Merged, ready for feedback
- **In Progress**: Someone is actively implementing
- **Stable**: Implemented, tested, documented, and backward compatible
- **Deprecated**: No longer recommended; old clients may safely ignore

Update the status field in your MEP as work progresses.

## MEP Scope

**In Scope**:

- New custom message types
- New GossipSub topics
- New stream protocols
- New game mechanics or events
- New Resonance hooks or scoring signals
- New transport adapters
- New UI overlays

**Out of Scope**:

- Protocol version changes (require RFC-level discussion)
- Threat model changes (discuss in THREAT_MODEL.md, SECURITY_PRIVACY.md)
- Major architectural refactors (use core issues/discussions)

## Lightweight Governance

The MEP process is intentionally **not heavy**:

- No MEP numbers are reserved in advance
- Extensions are merged when discussion naturally concludes
- Duplicates are consolidated in discussion
- Anyone can write a MEP — no approval required to start discussion

The goal is to capture community ideas without bureaucracy.

## Examples

- `MEP-0-TEMPLATE.md` — Template for new MEPs
- `MEP-1-CUSTOM_WAVE_TYPES.md` — Hypothetical example: community game adds new wave type

---

**Process Version**: 1.0  
**Created**: 2026-05-07  
**Status**: Stable
