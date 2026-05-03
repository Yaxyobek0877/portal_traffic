package pcp

import (
	"bytes"
	"crypto/ed25519"
	"strings"
	"testing"
)

func TestNewIdentity_DistinctSeeds(t *testing.T) {
	a, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a.Seed(), b.Seed()) {
		t.Fatal("two NewIdentity calls produced identical seeds")
	}
	if bytes.Equal(a.Public, b.Public) {
		t.Fatal("two NewIdentity calls produced identical public keys")
	}
}

func TestFromSeed_RoundTrip(t *testing.T) {
	original, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	seed := original.Seed()
	if len(seed) != SeedSize {
		t.Fatalf("seed size = %d, want %d", len(seed), SeedSize)
	}

	restored, err := FromSeed(seed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original.Public, restored.Public) {
		t.Fatal("restored public key does not match original")
	}

	// Signatures should also match (deterministic per Ed25519).
	msg := []byte("hello")
	if !bytes.Equal(original.Sign(msg), restored.Sign(msg)) {
		t.Fatal("signatures differ after seed round-trip")
	}
}

func TestFromSeed_RejectWrongSize(t *testing.T) {
	for _, n := range []int{0, 1, 16, 31, 33, 64} {
		seed := make([]byte, n)
		if _, err := FromSeed(seed); err == nil {
			t.Errorf("FromSeed(%d bytes) should fail", n)
		}
	}
}

func TestSignVerify(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("portal handshake message")
	sig := id.Sign(msg)
	if len(sig) != SignatureSize {
		t.Fatalf("signature size = %d, want %d", len(sig), SignatureSize)
	}
	if !Verify(id.Public, msg, sig) {
		t.Fatal("verify failed for fresh signature")
	}
}

func TestVerify_RejectTampered(t *testing.T) {
	id, _ := NewIdentity()
	msg := []byte("original message")
	sig := id.Sign(msg)

	tamperedMsg := []byte("tampered message")
	if Verify(id.Public, tamperedMsg, sig) {
		t.Fatal("verify accepted message with wrong content")
	}

	tamperedSig := append([]byte{}, sig...)
	tamperedSig[0] ^= 0xff
	if Verify(id.Public, msg, tamperedSig) {
		t.Fatal("verify accepted tampered signature")
	}
}

func TestVerify_RejectWrongKey(t *testing.T) {
	a, _ := NewIdentity()
	b, _ := NewIdentity()
	msg := []byte("signed by A")
	sig := a.Sign(msg)
	if Verify(b.Public, msg, sig) {
		t.Fatal("verify accepted A's signature with B's public key")
	}
}

func TestVerify_RejectMalformedInput(t *testing.T) {
	id, _ := NewIdentity()
	msg := []byte("test")
	sig := id.Sign(msg)

	// Wrong-size pubkey
	if Verify(make(ed25519.PublicKey, 16), msg, sig) {
		t.Fatal("verify accepted 16-byte public key")
	}
	// Wrong-size sig
	if Verify(id.Public, msg, sig[:32]) {
		t.Fatal("verify accepted 32-byte signature")
	}
}

func TestFingerprint_Format(t *testing.T) {
	id, _ := NewIdentity()
	fp := id.MyFingerprint()
	// Expect exactly "xxxx-xxxx-xx" — 12 chars including 2 dashes.
	if len(fp) != 12 {
		t.Fatalf("fingerprint length = %d (%q), want 12", len(fp), fp)
	}
	if fp[4] != '-' || fp[9] != '-' {
		t.Fatalf("fingerprint dashes wrong: %q", fp)
	}
	// Lowercase.
	if fp != strings.ToLower(fp) {
		t.Fatalf("fingerprint not lowercase: %q", fp)
	}
}

func TestFingerprint_Stable(t *testing.T) {
	id, _ := NewIdentity()
	fp1 := id.MyFingerprint()
	fp2 := Fingerprint(id.Public)
	if fp1 != fp2 {
		t.Fatalf("fingerprint differs: id.MyFingerprint=%q vs Fingerprint(pub)=%q", fp1, fp2)
	}
}

func TestFingerprint_DifferentForDifferentKeys(t *testing.T) {
	a, _ := NewIdentity()
	b, _ := NewIdentity()
	if Fingerprint(a.Public) == Fingerprint(b.Public) {
		t.Fatal("fingerprint collision on fresh identities — would imply 50-bit SHA-256 collision")
	}
}

func TestFingerprint_EmptyForBadInput(t *testing.T) {
	if Fingerprint(nil) != "" {
		t.Fatal("expected empty fingerprint for nil pubkey")
	}
	if Fingerprint(make(ed25519.PublicKey, 8)) != "" {
		t.Fatal("expected empty fingerprint for short pubkey")
	}
}
