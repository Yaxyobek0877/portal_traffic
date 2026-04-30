// Package crypt is the application-layer encryption used by Portal on
// top of WebRTC's DTLS. The portal access code is the only secret the
// peers share over a side channel (chat, paper, voice). We stretch it
// with PBKDF2-SHA256 into a 32-byte key and seal each message with
// NaCl secretbox.
//
// What this gets us beyond DTLS:
//   - Defense in depth: a hypothetical break in pion's DTLS or in a
//     transport-layer hop (Cloudflare TLS, ISP) doesn't reveal app data.
//   - Future-proofing: if we ever change transports (relay over a
//     friend's server, third-party signaling), payloads stay sealed.
//   - Cleaner threat model in docs.
//
// What this does NOT defend against:
//   - A malicious peer who joined the portal. They have the code.
//   - Loss of the portal code (no perfect forward secrecy in v1).
package crypt

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"

	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// Key is the secretbox key length: 32 bytes.
	keyLen = 32

	// nonceLen is the secretbox nonce length: 24 bytes. Random per frame.
	nonceLen = 24

	// pbkdfIter is the PBKDF2-SHA256 iteration count. 200_000 is a
	// reasonable balance: ~80ms on a modern laptop, prohibitive for an
	// attacker iterating through all million 6-digit codes (≈22h CPU
	// for one machine to half-exhaust the space).
	pbkdfIter = 200_000

	// pbkdfSalt is the static salt mixed into the PBKDF2 input. Salt is
	// not secret — it's a domain separator. We embed a literal so a
	// malicious peer who derives a key for "Portal" can't reuse it for
	// another product on the same code, and vice versa.
	pbkdfSalt = "portal-app-v1:secretbox"
)

// Key is a derived 32-byte symmetric key. Use Derive to produce one.
type Key [keyLen]byte

// Derive turns a portal code (or any low-entropy secret) into a Key
// via PBKDF2-SHA256. Calling this on the same code always produces
// the same Key, so a peer that joins after handshake derives the
// same value as the host.
func Derive(code string) Key {
	out := pbkdf2.Key([]byte(code), []byte(pbkdfSalt), pbkdfIter, keyLen, sha256New)
	var k Key
	copy(k[:], out)
	return k
}

// sha256New is a function value with the right type for pbkdf2.Key.
// (pbkdf2 wants `func() hash.Hash`.)
func sha256New() hash.Hash { return sha256.New() }

// Seal encrypts plaintext with the key and returns "nonce || ciphertext".
// Output length is len(plaintext) + secretbox.Overhead + 24.
func Seal(k *Key, plaintext []byte) []byte {
	var nonce [nonceLen]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		// crypto/rand failure is genuinely catastrophic; panic is
		// the right behaviour because every caller would otherwise
		// have to handle it identically.
		panic(fmt.Errorf("crypt: rand failed: %w", err))
	}
	out := make([]byte, 0, nonceLen+len(plaintext)+secretbox.Overhead)
	out = append(out, nonce[:]...)
	out = secretbox.Seal(out, plaintext, &nonce, (*[keyLen]byte)(k))
	return out
}

// Open decrypts and authenticates ciphertext produced by Seal. Returns
// the plaintext or an error if the key/MAC don't match.
func Open(k *Key, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < nonceLen+secretbox.Overhead {
		return nil, errors.New("crypt: ciphertext too short")
	}
	var nonce [nonceLen]byte
	copy(nonce[:], ciphertext[:nonceLen])
	plaintext, ok := secretbox.Open(nil, ciphertext[nonceLen:], &nonce, (*[keyLen]byte)(k))
	if !ok {
		return nil, errors.New("crypt: open failed (wrong key or tampered)")
	}
	return plaintext, nil
}
