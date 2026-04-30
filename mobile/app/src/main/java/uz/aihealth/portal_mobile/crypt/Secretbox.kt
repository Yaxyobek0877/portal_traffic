package uz.aihealth.portal_mobile.crypt

import com.goterl.lazysodium.LazySodiumAndroid
import com.goterl.lazysodium.SodiumAndroid
import com.goterl.lazysodium.interfaces.SecretBox
import java.security.SecureRandom
import javax.crypto.SecretKeyFactory
import javax.crypto.spec.PBEKeySpec

// Mirrors ../../../../client/crypt/crypt.go. The constants below MUST stay
// in lockstep with the Go side or peers can't decrypt each other:
//
//   - PBKDF2-SHA256, 200_000 iterations
//   - salt "portal-app-v1:secretbox"
//   - 32-byte key
//   - NaCl secretbox (XSalsa20-Poly1305)
//   - 24-byte random nonce per frame
//   - on-the-wire layout: nonce (24) || ciphertext (plaintext_len + 16)

private const val KEY_LEN = 32
private const val NONCE_LEN = 24
private const val MAC_LEN = SecretBox.MACBYTES // 16
private const val PBKDF_ITER = 200_000
private const val PBKDF_SALT = "portal-app-v1:secretbox"

class PortalKey(internal val bytes: ByteArray) {
    init {
        require(bytes.size == KEY_LEN) { "PortalKey: wrong key length ${bytes.size}" }
    }
}

object Crypt {

    private val sodium: SecretBox.Native = LazySodiumAndroid(SodiumAndroid())
    private val rng = SecureRandom()

    /**
     * Stretch the 6-digit (or any low-entropy) portal code into a 32-byte
     * symmetric key via PBKDF2-SHA256. Deterministic — both sides of a
     * portal derive the same value from the same code.
     */
    fun derive(code: String): PortalKey {
        val spec = PBEKeySpec(
            code.toCharArray(),
            PBKDF_SALT.toByteArray(Charsets.UTF_8),
            PBKDF_ITER,
            KEY_LEN * 8, // bits
        )
        val factory = SecretKeyFactory.getInstance("PBKDF2WithHmacSHA256")
        val keyBytes = factory.generateSecret(spec).encoded
        spec.clearPassword()
        return PortalKey(keyBytes)
    }

    /**
     * Encrypt + authenticate `plaintext` with the portal key.
     * Returns the on-wire bytes: nonce(24) || ciphertext(len(plaintext)+16).
     */
    fun seal(key: PortalKey, plaintext: ByteArray): ByteArray {
        val nonce = ByteArray(NONCE_LEN).also { rng.nextBytes(it) }
        val ciphertext = ByteArray(plaintext.size + MAC_LEN)
        val ok = sodium.cryptoSecretBoxEasy(
            ciphertext, plaintext, plaintext.size.toLong(), nonce, key.bytes,
        )
        check(ok) { "crypt: secretbox_easy failed" }
        val out = ByteArray(NONCE_LEN + ciphertext.size)
        System.arraycopy(nonce, 0, out, 0, NONCE_LEN)
        System.arraycopy(ciphertext, 0, out, NONCE_LEN, ciphertext.size)
        return out
    }

    /**
     * Decrypt + verify. Returns null on tag mismatch (wrong key or
     * tampered) or short input — same shape as the Go reference.
     */
    fun open(key: PortalKey, sealed: ByteArray): ByteArray? {
        if (sealed.size < NONCE_LEN + MAC_LEN) return null
        val nonce = sealed.copyOfRange(0, NONCE_LEN)
        val ciphertext = sealed.copyOfRange(NONCE_LEN, sealed.size)
        val plaintext = ByteArray(ciphertext.size - MAC_LEN)
        val ok = sodium.cryptoSecretBoxOpenEasy(
            plaintext, ciphertext, ciphertext.size.toLong(), nonce, key.bytes,
        )
        return if (ok) plaintext else null
    }
}
