// Domain types for TURN configuration. Mirrors the desktop's
// `client/storage` layout (CloudflareTurnConfig + TurnConfig) plus a
// minimal test-result shape used by the Settings UI to surface "is it
// actually working?" without creating a portal.

package uz.aihealth.portal_mobile.turn

import kotlinx.serialization.Serializable

/**
 * Long-lived Cloudflare Calls TURN credentials. Short-lived ICE-server
 * creds are minted from these on demand by [CloudflareTurn.fetch].
 *
 * Both fields blank → Cloudflare TURN disabled.
 */
@Serializable
data class CloudflareTurnConfig(
    val tokenId: String = "",
    val apiToken: String = "",
) {
    fun isConfigured(): Boolean = tokenId.isNotBlank() && apiToken.isNotBlank()
}

/**
 * User-supplied TURN config (self-hosted coturn / Twilio / Metered.ca).
 * [url] may contain multiple newline-separated URLs; the resolver
 * splits and trims them before handing to WebRTC.
 */
@Serializable
data class ManualTurnConfig(
    val url: String = "",
    val username: String = "",
    val credential: String = "",
) {
    fun isConfigured(): Boolean = url.isNotBlank()
}

/**
 * Result of [CloudflareTurn.test]. Mirrors the desktop's TurnTestResult
 * but pruned to fields the mobile UI shows.
 */
data class TurnTestResult(
    val ok: Boolean,
    val message: String,
    val urls: List<String> = emptyList(),
    val gatherMs: Long = 0L,
)

/**
 * Convenience preset: the Open Relay Project public TURN. Free, slow,
 * occasionally down — but a one-tap "see whether TURN unlocks the
 * connection" button is more useful than a wall of empty fields.
 *
 * Lists every transport variant Open Relay exposes (UDP/80, TCP/80,
 * TLS/443) so WebRTC picks whichever the carrier permits — UDP-blocking
 * carriers are common in CIS mobile networks.
 */
val FREE_TURN: ManualTurnConfig = ManualTurnConfig(
    url = listOf(
        "turn:openrelay.metered.ca:80",
        "turn:openrelay.metered.ca:80?transport=tcp",
        "turns:openrelay.metered.ca:443?transport=tcp",
    ).joinToString("\n"),
    username = "openrelayproject",
    credential = "openrelayproject",
)
