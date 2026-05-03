package pcp

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

// X25519PublicKeySize is 32.
const X25519PublicKeySize = 32

// X25519PrivateKeySize is 32.
const X25519PrivateKeySize = 32

// X25519Keypair is an X25519 (Curve25519 ECDH) keypair. Used both as
// long-term portal-session keys (held by the owner across the lifetime
// of one portal) and as ephemeral per-message keys for sealed boxes.
//
// Unlike Ed25519 identity keys, X25519 keypairs are not generally
// stored across sessions — generate fresh ones for each portal or
// each message.
type X25519Keypair struct {
	Public  []byte // 32 bytes
	private []byte // 32 bytes
}

// GenerateX25519 generates a fresh X25519 keypair from the OS RNG.
func GenerateX25519() (*X25519Keypair, error) {
	priv := make([]byte, X25519PrivateKeySize)
	if _, err := rand.Read(priv); err != nil {
		return nil, fmt.Errorf("pcp: rng failed: %w", err)
	}
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("pcp: derive x25519 pubkey: %w", err)
	}
	return &X25519Keypair{Public: pub, private: priv}, nil
}

// Private returns a copy of the 32-byte private scalar. Callers
// should treat the returned bytes as secret.
func (kp *X25519Keypair) Private() []byte {
	if kp == nil || len(kp.private) == 0 {
		return nil
	}
	out := make([]byte, len(kp.private))
	copy(out, kp.private)
	return out
}

// SharedSecret performs X25519(private, peerPublic) and returns the
// 32-byte shared secret. Callers must run this through HKDF before
// using it as a key — the raw secret has structure that AEAD doesn't
// expect.
func SharedSecret(privateScalar, peerPublic []byte) ([]byte, error) {
	if len(privateScalar) != X25519PrivateKeySize {
		return nil, fmt.Errorf("pcp: private key must be %d bytes, got %d",
			X25519PrivateKeySize, len(privateScalar))
	}
	if len(peerPublic) != X25519PublicKeySize {
		return nil, fmt.Errorf("pcp: peer public key must be %d bytes, got %d",
			X25519PublicKeySize, len(peerPublic))
	}
	return curve25519.X25519(privateScalar, peerPublic)
}

// SealedBox encrypts plaintext anonymously to recipientPublic. No
// long-term keys involved — each message uses a fresh ephemeral
// keypair, so the recipient can't link two messages from the same
// sender unless the plaintext itself reveals the sender.
//
// Output format: [ephPubkey (32) || nonce (24) || ciphertext+tag].
//
// To open the result, the recipient calls SealedOpen with the
// matching X25519 private key.
func SealedBox(recipientPublic, plaintext []byte) ([]byte, error) {
	if len(recipientPublic) != X25519PublicKeySize {
		return nil, fmt.Errorf("pcp: recipient public must be %d bytes, got %d",
			X25519PublicKeySize, len(recipientPublic))
	}

	eph, err := GenerateX25519()
	if err != nil {
		return nil, err
	}
	shared, err := curve25519.X25519(eph.private, recipientPublic)
	if err != nil {
		return nil, fmt.Errorf("pcp: ephemeral DH: %w", err)
	}

	key, err := hkdfExpand(shared, []byte(VersionString+":sealedbox"), chacha20poly1305.KeySize)
	if err != nil {
		return nil, err
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("pcp: xchacha init: %w", err)
	}

	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("pcp: nonce rng: %w", err)
	}

	// AD = ephemeral pubkey, so a flipped ephPubkey breaks the seal.
	ct := aead.Seal(nil, nonce, plaintext, eph.Public)

	out := make([]byte, 0, X25519PublicKeySize+chacha20poly1305.NonceSizeX+len(ct))
	out = append(out, eph.Public...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

// SealedOpen reverses SealedBox. recipientPrivate is the 32-byte
// scalar matching the public key the sender used.
func SealedOpen(recipientPrivate, sealed []byte) ([]byte, error) {
	if len(recipientPrivate) != X25519PrivateKeySize {
		return nil, fmt.Errorf("pcp: recipient private must be %d bytes", X25519PrivateKeySize)
	}
	const headerLen = X25519PublicKeySize + chacha20poly1305.NonceSizeX
	if len(sealed) < headerLen+chacha20poly1305.Overhead {
		return nil, errors.New("pcp: sealed box too short")
	}
	ephPub := sealed[:X25519PublicKeySize]
	nonce := sealed[X25519PublicKeySize:headerLen]
	ct := sealed[headerLen:]

	shared, err := curve25519.X25519(recipientPrivate, ephPub)
	if err != nil {
		return nil, fmt.Errorf("pcp: dh: %w", err)
	}
	key, err := hkdfExpand(shared, []byte(VersionString+":sealedbox"), chacha20poly1305.KeySize)
	if err != nil {
		return nil, err
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("pcp: xchacha init: %w", err)
	}
	pt, err := aead.Open(nil, nonce, ct, ephPub)
	if err != nil {
		return nil, fmt.Errorf("pcp: sealed open: %w", err)
	}
	return pt, nil
}

// hkdfExpand expands ikm with HKDF-SHA256 to outLen bytes using info
// as the optional context string. Salt is empty (we control the input
// secret quality and use distinct info strings per purpose).
func hkdfExpand(ikm, info []byte, outLen int) ([]byte, error) {
	r := hkdf.New(sha256New, ikm, nil, info)
	out := make([]byte, outLen)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, fmt.Errorf("pcp: hkdf: %w", err)
	}
	return out, nil
}
