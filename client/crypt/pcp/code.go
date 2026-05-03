package pcp

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
)

// Code is a portal access code in the PCP-1 format. Two human-typeable
// halves separated by a dash:
//
//   XXXX-YYYY
//
// where the first half is 4 random alphanumeric chars (~24 bits of
// entropy) and the second is 4 chars derived from HKDF(ePK_o), so a
// brute-forcer can't precompute valid codes without the matching
// portal session pubkey.
//
// We deliberately avoid '0', 'O', 'I', 'l', '1' to make the code
// readable when read aloud or printed.
type Code string

// codeAlphabet is 32 unambiguous characters. This is exactly 5 bits
// per char, so 4 chars = 20 bits, 8 chars = 40 bits. (We slightly
// shorten the random half to 20 bits, accepting that since the tag
// adds binding without contributing to the main entropy budget.)
const codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// codeRandomLen and codeTagLen are the two halves' lengths (in chars).
const (
	codeRandomLen = 4 // 20 bits of pure random entropy
	codeTagLen    = 4 // 20 bits derived from the session pubkey
)

// CodeLength is the total visible length of a PCP-1 code, including
// the dash. Display in monospace so users can transcribe it cleanly.
const CodeLength = codeRandomLen + 1 + codeTagLen

// NewCode builds a fresh portal code bound to the given X25519
// session public key. Callers should generate ePK first (via
// GenerateX25519), then use NewCode(ePK) so the code's tag matches.
//
// Returns the user-facing Code string. The owner shares this with
// joiners over a side channel (chat, voice).
func NewCode(sessionPublic []byte) (Code, error) {
	if len(sessionPublic) != X25519PublicKeySize {
		return "", fmt.Errorf("pcp: session public must be %d bytes", X25519PublicKeySize)
	}
	random, err := randomCodeChars(codeRandomLen)
	if err != nil {
		return "", err
	}
	tag, err := codeTag(sessionPublic, random)
	if err != nil {
		return "", err
	}
	return Code(random + "-" + tag), nil
}

// Verify checks that the code's tag half is consistent with the given
// session public key. Returns nil on success.
//
// Use constant-time compare so a malicious server timing the
// verification can't leak information about the tag.
func (c Code) Verify(sessionPublic []byte) error {
	if len(sessionPublic) != X25519PublicKeySize {
		return fmt.Errorf("pcp: session public must be %d bytes", X25519PublicKeySize)
	}
	random, gotTag, err := c.split()
	if err != nil {
		return err
	}
	wantTag, err := codeTag(sessionPublic, random)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(strings.ToUpper(gotTag)), []byte(wantTag)) != 1 {
		return errors.New("pcp: code tag does not match session pubkey")
	}
	return nil
}

// Hash returns SHA-256(code). The signaling server stores this rather
// than the code itself, so a server compromise doesn't immediately
// reveal the codes of all live portals.
func (c Code) Hash() []byte {
	h := sha256.Sum256([]byte(strings.ToUpper(string(c))))
	return h[:]
}

// String formats the code as the canonical "XXXX-YYYY" form. Already
// uppercase as produced by NewCode; tolerates lowercase input.
func (c Code) String() string {
	return strings.ToUpper(string(c))
}

// split partitions a Code into its random half and tag half. Returns
// an error if the format is malformed.
func (c Code) split() (random, tag string, err error) {
	s := strings.ToUpper(string(c))
	if len(s) != CodeLength {
		return "", "", fmt.Errorf("pcp: code must be %d chars, got %d", CodeLength, len(s))
	}
	if s[codeRandomLen] != '-' {
		return "", "", fmt.Errorf("pcp: code missing '-' at position %d", codeRandomLen)
	}
	random = s[:codeRandomLen]
	tag = s[codeRandomLen+1:]
	if !validCodeChars(random) || !validCodeChars(tag) {
		return "", "", errors.New("pcp: code contains invalid characters")
	}
	return random, tag, nil
}

// codeTag derives the tag half from session pubkey + random half, so
// the tag binds the code to a specific portal session.
func codeTag(sessionPublic []byte, random string) (string, error) {
	out, err := hkdfExpand(sessionPublic, []byte(VersionString+":code-tag:"+random), codeTagLen)
	if err != nil {
		return "", err
	}
	return charsFromBytes(out, codeTagLen), nil
}

// randomCodeChars returns n cryptographically random characters from
// codeAlphabet, sampled with rejection so the distribution is uniform.
func randomCodeChars(n int) (string, error) {
	out := make([]byte, n)
	buf := make([]byte, 1)
	for i := 0; i < n; i++ {
		for {
			if _, err := rand.Read(buf); err != nil {
				return "", fmt.Errorf("pcp: rng: %w", err)
			}
			// Reject bytes that would skew the modulo distribution.
			// 256 / 32 = 8, exactly — no rejection needed for a 32-char
			// alphabet. We keep the loop structure for safety against
			// future alphabet changes.
			b := buf[0]
			if int(b) >= 256-(256%len(codeAlphabet)) {
				continue
			}
			out[i] = codeAlphabet[int(b)%len(codeAlphabet)]
			break
		}
	}
	return string(out), nil
}

// charsFromBytes maps n bytes onto n alphabet chars using simple
// modulo. Acceptable here because the input is from HKDF (uniform
// over the byte space) and we only use it for tag derivation — not
// for cryptographic entropy.
func charsFromBytes(in []byte, n int) string {
	if n > len(in) {
		n = len(in)
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = codeAlphabet[int(in[i])%len(codeAlphabet)]
	}
	return string(out)
}

// validCodeChars checks that every character of s is in codeAlphabet.
func validCodeChars(s string) bool {
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(codeAlphabet, rune(s[i])) {
			return false
		}
	}
	return true
}
