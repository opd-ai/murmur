// Package discovery provides the bootstrap verification key.
// Per PLAN.md: "public half embedded in binary as BootstrapVerifyKey".
package discovery

import "crypto/ed25519"

// BootstrapVerifyKey is the Ed25519 public key used to verify signed peer lists.
// Per PLAN.md provisioning: "Run `go run ./cmd/murmur --gen-bootstrap-key` locally;
// set the private key as `BOOTSTRAP_SIGN_KEY` secret; embed the public key in
// `pkg/networking/discovery/verify_key.go`."
//
// This is a placeholder that will be replaced during provisioning.
// The actual key will be generated and committed during CI setup.
var BootstrapVerifyKey = ed25519.PublicKey(nil)

// TODO: Replace with actual public key during provisioning:
// 1. Run: go run ./cmd/murmur --gen-bootstrap-key
// 2. Set BOOTSTRAP_SIGN_KEY GitHub Actions secret with private key
// 3. Update this variable with the generated public key (32 bytes)
