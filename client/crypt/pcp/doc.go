// Package pcp implements the Portal Cipher Protocol v1 — the
// application-layer cipher suite that runs beneath WebRTC DTLS for
// Portal's chat, file transfer, and proxy traffic.
//
// PCP-1 is a hybrid protocol combining standard primitives:
//
//   - Ed25519 long-term identity per device (stored in OS keychain)
//   - X25519 ephemeral session keys per portal
//   - HKDF-SHA256 for key derivation
//   - XChaCha20-Poly1305 for AEAD frames
//   - Argon2id for legacy password-based KDF (during migration)
//
// We never roll our own crypto — primitives come from crypto/ed25519
// and golang.org/x/crypto. The novelty is in the combination and
// protocol-level guarantees: per-pair forward secrecy, identity-bound
// portal codes, monotonic counter nonces, and per-direction keys.
//
// The full protocol is specified in SPEC.md alongside this package.
// Test vectors live in *_test.go files; run `go test -v ./...` to
// verify the implementation against them.
//
// Layered API:
//
//   - Identity: long-term keypair, fingerprint, sign/verify
//   - SealedBox: anonymous-sender single-recipient encryption
//   - Code: portal code generation and verification (binds to ePK)
//   - Session: full pair-session AEAD (used for chat/transfer/proxy)
//
// PCP-1 ships in v0.4.0 alongside (not replacing) the legacy crypt
// package; mesh.Manager negotiates which one to use based on the
// server's reported protocol_version.
package pcp

// Version is the PCP wire-protocol version. Increment for breaking
// changes to the wire format.
const Version = 1

// VersionString is what we put into HKDF info strings and similar
// domain separators so a future PCP-2 implementation never collides
// with PCP-1 derivations.
const VersionString = "pcp1"
