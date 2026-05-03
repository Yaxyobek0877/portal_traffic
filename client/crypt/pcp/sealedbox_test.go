package pcp

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestGenerateX25519_Distinct(t *testing.T) {
	a, err := GenerateX25519()
	if err != nil {
		t.Fatal(err)
	}
	b, err := GenerateX25519()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a.Public, b.Public) {
		t.Fatal("two GenerateX25519 calls produced identical public keys")
	}
	if bytes.Equal(a.Private(), b.Private()) {
		t.Fatal("two GenerateX25519 calls produced identical private keys")
	}
	if len(a.Public) != X25519PublicKeySize {
		t.Fatalf("public key size = %d, want %d", len(a.Public), X25519PublicKeySize)
	}
}

func TestSharedSecret_Symmetric(t *testing.T) {
	alice, _ := GenerateX25519()
	bob, _ := GenerateX25519()

	// Alice computes secret from her private + Bob's public.
	secretA, err := SharedSecret(alice.Private(), bob.Public)
	if err != nil {
		t.Fatal(err)
	}
	// Bob computes secret from his private + Alice's public.
	secretB, err := SharedSecret(bob.Private(), alice.Public)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(secretA, secretB) {
		t.Fatal("X25519 shared secrets differ between Alice and Bob")
	}
}

func TestSealedBox_RoundTrip(t *testing.T) {
	recipient, _ := GenerateX25519()

	plaintexts := [][]byte{
		[]byte("hello"),
		[]byte(""),
		bytes.Repeat([]byte{0xab}, 1024),
		make([]byte, 65536),
	}
	if _, err := rand.Read(plaintexts[3]); err != nil {
		t.Fatal(err)
	}

	for i, pt := range plaintexts {
		sealed, err := SealedBox(recipient.Public, pt)
		if err != nil {
			t.Fatalf("case %d: SealedBox failed: %v", i, err)
		}
		// Sealed output must be longer than plaintext (header + tag).
		if len(sealed) < len(pt) {
			t.Fatalf("case %d: sealed shorter than plaintext", i)
		}
		opened, err := SealedOpen(recipient.Private(), sealed)
		if err != nil {
			t.Fatalf("case %d: SealedOpen failed: %v", i, err)
		}
		if !bytes.Equal(pt, opened) {
			t.Fatalf("case %d: round-trip mismatch", i)
		}
	}
}

func TestSealedBox_DifferentEphemeralEachCall(t *testing.T) {
	recipient, _ := GenerateX25519()
	plaintext := []byte("same message")

	sealed1, err := SealedBox(recipient.Public, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	sealed2, err := SealedBox(recipient.Public, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	// The first 32 bytes are the ephemeral pubkey — must differ.
	if bytes.Equal(sealed1[:X25519PublicKeySize], sealed2[:X25519PublicKeySize]) {
		t.Fatal("two sealed boxes used the same ephemeral pubkey")
	}
	// Whole ciphertext must differ.
	if bytes.Equal(sealed1, sealed2) {
		t.Fatal("two sealed boxes of the same plaintext produced identical output")
	}
}

func TestSealedOpen_WrongRecipientFails(t *testing.T) {
	intended, _ := GenerateX25519()
	wrong, _ := GenerateX25519()

	sealed, err := SealedBox(intended.Public, []byte("secret message"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SealedOpen(wrong.Private(), sealed); err == nil {
		t.Fatal("SealedOpen accepted wrong recipient's private key")
	}
}

func TestSealedOpen_TamperDetect(t *testing.T) {
	recipient, _ := GenerateX25519()
	sealed, _ := SealedBox(recipient.Public, []byte("important"))

	// Flip a byte in the ciphertext (after the header).
	tampered := append([]byte{}, sealed...)
	const headerLen = X25519PublicKeySize + 24 // 24 = NonceSizeX
	tampered[headerLen+1] ^= 0x01
	if _, err := SealedOpen(recipient.Private(), tampered); err == nil {
		t.Fatal("SealedOpen accepted tampered ciphertext")
	}

	// Flip a byte in the ephemeral pubkey portion (used as AD).
	tamperedPub := append([]byte{}, sealed...)
	tamperedPub[3] ^= 0x01
	if _, err := SealedOpen(recipient.Private(), tamperedPub); err == nil {
		t.Fatal("SealedOpen accepted tampered ephemeral pubkey")
	}
}

func TestSealedOpen_RejectShort(t *testing.T) {
	recipient, _ := GenerateX25519()
	for _, n := range []int{0, 1, 16, 32, 55} {
		short := make([]byte, n)
		if _, err := SealedOpen(recipient.Private(), short); err == nil {
			t.Errorf("SealedOpen accepted %d-byte input", n)
		}
	}
}

func TestSharedSecret_RejectBadSizes(t *testing.T) {
	priv := make([]byte, 16)
	pub := make([]byte, 32)
	if _, err := SharedSecret(priv, pub); err == nil {
		t.Fatal("SharedSecret accepted 16-byte private key")
	}
	priv = make([]byte, 32)
	pub = make([]byte, 16)
	if _, err := SharedSecret(priv, pub); err == nil {
		t.Fatal("SharedSecret accepted 16-byte public key")
	}
}
