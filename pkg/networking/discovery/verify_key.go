// Package discovery provides the bootstrap verification key.
// Per PLAN.md: "public half embedded in binary as BootstrapVerifyKey".
package discovery

import "crypto/ed25519"

// BootstrapVerifyKey is the Ed25519 public key used to verify signed peer lists.
// Per PLAN.md provisioning: "Run `go run ./cmd/murmur --gen-bootstrap-key` locally;
// set the private key as `BOOTSTRAP_SIGN_KEY` secret; embed the public key in
// `pkg/networking/discovery/verify_key.go`."
var BootstrapVerifyKey = ed25519.PublicKey{
	0x24, 0x5f, 0x89, 0x1c, 0x49, 0xa7, 0x31, 0x62,
	0x4d, 0xb8, 0x10, 0x37, 0xa2, 0x53, 0x7e, 0x9b,
	0x91, 0x08, 0xcc, 0x54, 0x3a, 0x6f, 0x10, 0x27,
	0x8d, 0xee, 0x11, 0x65, 0x9f, 0x7a, 0x30, 0xc4,
}
