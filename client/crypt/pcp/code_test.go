package pcp

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewCode_FormatAndLength(t *testing.T) {
	kp, _ := GenerateX25519()
	code, err := NewCode(kp.Public)
	if err != nil {
		t.Fatal(err)
	}
	s := code.String()
	if len(s) != CodeLength {
		t.Fatalf("code length = %d (%q), want %d", len(s), s, CodeLength)
	}
	if s[codeRandomLen] != '-' {
		t.Fatalf("expected '-' at position %d in %q", codeRandomLen, s)
	}
	if strings.ToUpper(s) != s {
		t.Fatalf("code not uppercase: %q", s)
	}
	for _, ch := range s {
		if ch == '-' {
			continue
		}
		if !strings.ContainsRune(codeAlphabet, ch) {
			t.Fatalf("char %q not in codeAlphabet", ch)
		}
	}
}

func TestNewCode_DistinctEachCall(t *testing.T) {
	kp, _ := GenerateX25519()
	a, _ := NewCode(kp.Public)
	b, _ := NewCode(kp.Public)
	if a == b {
		t.Fatal("two NewCode calls returned identical codes")
	}
}

func TestCodeVerify_AcceptsCorrect(t *testing.T) {
	kp, _ := GenerateX25519()
	code, _ := NewCode(kp.Public)
	if err := code.Verify(kp.Public); err != nil {
		t.Fatalf("Verify failed for fresh code: %v", err)
	}
}

func TestCodeVerify_LowercaseAccepted(t *testing.T) {
	kp, _ := GenerateX25519()
	code, _ := NewCode(kp.Public)
	lower := Code(strings.ToLower(code.String()))
	if err := lower.Verify(kp.Public); err != nil {
		t.Fatalf("Verify rejected lowercase form of valid code: %v", err)
	}
}

func TestCodeVerify_RejectsWrongPubkey(t *testing.T) {
	kpA, _ := GenerateX25519()
	kpB, _ := GenerateX25519()
	code, _ := NewCode(kpA.Public)
	if err := code.Verify(kpB.Public); err == nil {
		t.Fatal("Verify accepted code with wrong session pubkey")
	}
}

func TestCodeVerify_RejectsTamperedRandom(t *testing.T) {
	kp, _ := GenerateX25519()
	code, _ := NewCode(kp.Public)
	s := code.String()
	// Flip the first char to a different alphabet char.
	first := s[0]
	var replacement byte
	for i := 0; i < len(codeAlphabet); i++ {
		if codeAlphabet[i] != first {
			replacement = codeAlphabet[i]
			break
		}
	}
	tampered := Code(string(replacement) + s[1:])
	if err := tampered.Verify(kp.Public); err == nil {
		t.Fatal("Verify accepted code with tampered random half")
	}
}

func TestCodeVerify_RejectsTamperedTag(t *testing.T) {
	kp, _ := GenerateX25519()
	code, _ := NewCode(kp.Public)
	s := code.String()
	// Flip the last char.
	last := s[len(s)-1]
	var replacement byte
	for i := 0; i < len(codeAlphabet); i++ {
		if codeAlphabet[i] != last {
			replacement = codeAlphabet[i]
			break
		}
	}
	tampered := Code(s[:len(s)-1] + string(replacement))
	if err := tampered.Verify(kp.Public); err == nil {
		t.Fatal("Verify accepted code with tampered tag")
	}
}

func TestCodeVerify_RejectsMalformed(t *testing.T) {
	kp, _ := GenerateX25519()
	cases := []Code{
		"",
		"AAAA",
		"AAAA-AAA",     // tag too short
		"AAAA-AAAAA",   // tag too long
		"AAAAAAAA",     // missing dash
		"AAAA AAAA",    // wrong separator
		"OOOO-1111",    // banned chars (O, 1)
		"ABCD-EFGI",    // I banned
	}
	for _, c := range cases {
		if err := c.Verify(kp.Public); err == nil {
			t.Errorf("Verify accepted malformed code %q", c)
		}
	}
}

func TestCodeHash_StableAcrossCase(t *testing.T) {
	kp, _ := GenerateX25519()
	code, _ := NewCode(kp.Public)
	upper := Code(strings.ToUpper(code.String()))
	lower := Code(strings.ToLower(code.String()))
	if !bytes.Equal(upper.Hash(), lower.Hash()) {
		t.Fatal("Hash differs between upper and lower-case form")
	}
}

func TestCodeHash_DifferentForDifferentCodes(t *testing.T) {
	kp, _ := GenerateX25519()
	a, _ := NewCode(kp.Public)
	b, _ := NewCode(kp.Public)
	if bytes.Equal(a.Hash(), b.Hash()) {
		t.Fatal("Hash collision on distinct codes — would imply SHA-256 collision")
	}
}

func TestNewCode_RejectsBadPubkey(t *testing.T) {
	if _, err := NewCode(make([]byte, 16)); err == nil {
		t.Fatal("NewCode accepted 16-byte session pubkey")
	}
	if _, err := NewCode(nil); err == nil {
		t.Fatal("NewCode accepted nil session pubkey")
	}
}
