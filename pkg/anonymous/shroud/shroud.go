// Package shroud provides three-hop onion circuit construction.
// Per SECURITY_PRIVACY.md, Shroud circuits use XChaCha20-Poly1305
// for layer encryption with Curve25519 key exchange.
package shroud

// CircuitLength is the number of hops in a Shroud circuit.
const CircuitLength = 3

// TODO: Implement Shroud circuit construction per ROADMAP.md Priority 6.
