package crypt

import (
	"bytes"
	"testing"
)

func TestSealOpenRoundtrip(t *testing.T) {
	k := Derive("123456")
	pt := []byte("hello mesh")
	ct := Seal(&k, pt)
	got, err := Open(&k, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, pt) {
		t.Errorf("roundtrip mismatch: got %q, want %q", got, pt)
	}
}

func TestDeriveDeterministic(t *testing.T) {
	a := Derive("000111")
	b := Derive("000111")
	if a != b {
		t.Error("Derive should be deterministic for the same code")
	}
}

func TestDifferentCodesDifferentKeys(t *testing.T) {
	a := Derive("000000")
	b := Derive("000001")
	if a == b {
		t.Error("different codes must produce different keys")
	}
}

func TestOpenWithWrongKeyFails(t *testing.T) {
	a := Derive("123456")
	b := Derive("999999")
	ct := Seal(&a, []byte("secret"))
	if _, err := Open(&b, ct); err == nil {
		t.Error("Open with wrong key should fail")
	}
}

func TestOpenTamperedFails(t *testing.T) {
	k := Derive("424242")
	ct := Seal(&k, []byte("secret"))
	ct[len(ct)-1] ^= 0x01
	if _, err := Open(&k, ct); err == nil {
		t.Error("Open of tampered ciphertext should fail")
	}
}

func TestNonceUniqueness(t *testing.T) {
	k := Derive("111111")
	pt := []byte("same")
	a := Seal(&k, pt)
	b := Seal(&k, pt)
	// Two seal calls of the same plaintext should produce different
	// ciphertexts thanks to the random nonce.
	if bytes.Equal(a, b) {
		t.Error("identical Seal outputs — nonce reuse?")
	}
}

func TestShortCiphertextRejected(t *testing.T) {
	k := Derive("111111")
	if _, err := Open(&k, []byte("too-short")); err == nil {
		t.Error("Open should reject short input")
	}
}
