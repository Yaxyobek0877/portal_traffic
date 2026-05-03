package pcp

import (
	"crypto/sha256"
	"hash"
	"io"

	"golang.org/x/crypto/hkdf"
)

// sha256New is the constructor reference HKDF / pbkdf2 want.
func sha256New() hash.Hash { return sha256.New() }

// hkdfNewWithSalt returns an HKDF reader configured with the given
// salt. Empty salt means the implementation defaults (zeroes), which
// is fine when ikm is high-entropy (e.g., an X25519 shared secret).
func hkdfNewWithSalt(ikm, salt, info []byte) io.Reader {
	return hkdf.New(sha256New, ikm, salt, info)
}
