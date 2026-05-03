package pcp

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
)

// Identity is a long-term Ed25519 keypair tied to a single device.
//
// On first launch, generate one with NewIdentity, store its Seed
// in the OS keychain, and load it on subsequent launches with
// FromSeed. Public key and fingerprint are safe to display.
type Identity struct {
	// Public is the 32-byte Ed25519 public key.
	Public ed25519.PublicKey
	// private is the 64-byte Ed25519 expanded private key.
	private ed25519.PrivateKey
}

// SeedSize is the size of the seed bytes that uniquely determine an
// Identity. Persist these (and only these) — Ed25519 expands them
// deterministically. 32 bytes.
const SeedSize = ed25519.SeedSize

// PublicKeySize is 32.
const PublicKeySize = ed25519.PublicKeySize

// SignatureSize is 64.
const SignatureSize = ed25519.SignatureSize

// NewIdentity generates a fresh Identity from the OS RNG.
//
// Persist the result with id.Seed() — that's the only thing the OS
// keychain needs to store. Re-create with FromSeed on next launch.
func NewIdentity() (*Identity, error) {
	seed := make([]byte, SeedSize)
	if _, err := rand.Read(seed); err != nil {
		return nil, fmt.Errorf("pcp: rng failed: %w", err)
	}
	return FromSeed(seed)
}

// FromSeed reconstructs an Identity from a 32-byte seed. Inverse of
// Identity.Seed.
func FromSeed(seed []byte) (*Identity, error) {
	if len(seed) != SeedSize {
		return nil, fmt.Errorf("pcp: seed must be %d bytes, got %d", SeedSize, len(seed))
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub, ok := priv.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("pcp: ed25519 public key type assertion failed")
	}
	return &Identity{Public: pub, private: priv}, nil
}

// Seed returns the 32-byte seed that uniquely identifies this Identity.
// This is the secret to persist — never log, never transmit.
func (id *Identity) Seed() []byte {
	if id == nil || len(id.private) == 0 {
		return nil
	}
	return id.private.Seed()
}

// Sign produces an Ed25519 signature over msg using the long-term
// identity key.
func (id *Identity) Sign(msg []byte) []byte {
	return ed25519.Sign(id.private, msg)
}

// Verify checks an Ed25519 signature against a public key. Pure
// function — no Identity needed on the verify path.
func Verify(pub ed25519.PublicKey, msg, sig []byte) bool {
	if len(pub) != PublicKeySize || len(sig) != SignatureSize {
		return false
	}
	return ed25519.Verify(pub, msg, sig)
}

// fingerprintEncoding is base32 without padding, lowercase. We chose
// base32 because it's case-insensitive when read aloud and survives
// being typed back in by hand.
var fingerprintEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// Fingerprint returns a 10-character human-readable identifier derived
// from the public key. Format: "XXXX-XXXX-XX" (lowercase base32 of
// SHA-256(pubkey), first 50 bits, dashed every 4 chars).
//
// Two identities with the same fingerprint would imply finding a
// 50-bit SHA-256 collision (≈ 33M operations average) — we accept
// that risk for human readability. Anyone wanting full assurance
// can compare the full public key.
func Fingerprint(pub ed25519.PublicKey) string {
	if len(pub) != PublicKeySize {
		return ""
	}
	h := sha256.Sum256(pub)
	// 10 base32 chars = 50 bits of entropy.
	enc := fingerprintEncoding.EncodeToString(h[:])
	if len(enc) > 10 {
		enc = enc[:10]
	}
	return strings.ToLower(enc[:4] + "-" + enc[4:8] + "-" + enc[8:10])
}

// MyFingerprint is a convenience for id.Public.
func (id *Identity) MyFingerprint() string {
	if id == nil {
		return ""
	}
	return Fingerprint(id.Public)
}
