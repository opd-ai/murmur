# Security Gaps — 2026-05-08

## Bounded Input Handling for Remote Bootstrap
- **Stated Goal**: Privacy/security are structural and resilient under adversarial network conditions.
- **Current State**: Remote bootstrap HTTP fetchers read response bodies without hard size limits.
- **Risk**: Oversized responses can trigger memory pressure or crashes during startup/bootstrap.
- **Closing the Gap**: Enforce strict body-size ceilings for all bootstrap HTTP reads, reject over-limit payloads before decode, and add regression tests for oversized responses.

## Monitoring Endpoint Exposure Controls
- **Stated Goal**: Privacy-first defaults and minimization of unnecessary disclosure.
- **Current State**: Bootstrap health endpoint is unauthenticated; default bootstrap CLI CORS is `*`; health response includes local operational details.
- **Risk**: External reconnaissance of node metadata and environment details for publicly deployed bootstrap nodes.
- **Closing the Gap**: Ship privacy-preserving defaults (no wildcard CORS, redacted health payload), add optional authenticated detailed health mode, and document secure deployment profiles.

## Dependency Vulnerability Triage Depth
- **Stated Goal**: Strong security posture with cryptographic correctness and safe dependency use.
- **Current State**: Advisory tooling flags Vault module vulnerabilities while code imports only the Shamir subpackage; `govulncheck` could not complete in this environment.
- **Risk**: Unknown residual dependency risk and delayed remediation decisions.
- **Closing the Gap**: Run `govulncheck` in CI images with required native deps, track advisories per imported subpackages, and pin/upgrade vulnerable modules when patches become available or replace with narrower dependency.

## Secret-Hygiene Guardrails in Repository Defaults
- **Stated Goal**: Protect identity materials and avoid secret leakage.
- **Current State**: `.gitignore` includes `.env` but does not explicitly ignore common key formats like `*.pem`/`*.key` globally.
- **Risk**: Accidental commit of key material from local/dev workflows.
- **Closing the Gap**: Add explicit ignore patterns for private-key artifacts and provide a security checklist for contributors handling bootstrap/identity keys.
