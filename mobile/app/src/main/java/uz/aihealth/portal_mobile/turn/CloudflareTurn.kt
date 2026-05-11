// Cloudflare Calls TURN client. Mirrors client/cloudflareturn.go on the
// desktop: one POST to credentials/generate using a long-lived TOKEN_ID
// + API_TOKEN, returning short-lived ICE-server creds.
//
// We cache the result for 30 minutes (well under the 1h default TTL)
// because hitting the API on every portal create / join is wasteful
// and adds 100-300ms of latency to the welcome→ready transition.
//
// Docs: https://developers.cloudflare.com/calls/turn/
//
// Threading: all entry points are suspend; HTTP runs on Dispatchers.IO
// inside [withContext]. The cache is a synchronized object — call
// volume is too low for atomic primitives to matter.

package uz.aihealth.portal_mobile.turn

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.coroutines.withTimeoutOrNull
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.webrtc.PeerConnection
import java.util.concurrent.TimeUnit

private const val TAG = "CloudflareTurn"
private const val ENDPOINT = "https://rtc.live.cloudflare.com/v1/turn/keys/%s/credentials/generate"

object CloudflareTurn {

    private val http = OkHttpClient.Builder()
        .connectTimeout(8, TimeUnit.SECONDS)
        .readTimeout(8, TimeUnit.SECONDS)
        .build()

    private val lenientJson = Json { ignoreUnknownKeys = true; isLenient = true }

    /**
     * Single cache slot. Cloudflare creds are scoped to (tokenId), so
     * we store the tokenId we minted them with — if the user changes
     * their config we'll naturally invalidate without an explicit
     * `.invalidate()` call.
     */
    private data class CacheEntry(
        val tokenId: String,
        val server: PeerConnection.IceServer,
        val expiresAtMs: Long,
    )

    @Volatile private var cache: CacheEntry? = null

    /**
     * Fetch fresh ICE-server creds. Returns null on any failure — the
     * caller should fall back to STUN-only / manual TURN.
     *
     * [ttlSeconds] becomes the credential lifetime Cloudflare hands us;
     * 3600 (1h) matches the desktop default and the API's own default.
     */
    suspend fun fetch(cfg: CloudflareTurnConfig, ttlSeconds: Int = 3600): PeerConnection.IceServer? {
        if (!cfg.isConfigured()) return null

        cache?.let { c ->
            if (c.tokenId == cfg.tokenId && System.currentTimeMillis() < c.expiresAtMs) {
                return c.server
            }
        }

        val server = doFetch(cfg, ttlSeconds) ?: return null
        // Cache for 30 minutes regardless of the TTL Cloudflare claims;
        // if we cached for the full hour and the user toggled the config
        // we'd still serve stale creds for 60min.
        cache = CacheEntry(
            tokenId = cfg.tokenId,
            server = server,
            expiresAtMs = System.currentTimeMillis() + 30 * 60 * 1_000L,
        )
        return server
    }

    /**
     * Settings → "Sinash" button. Bypasses cache and reports whether
     * the API actually answered, what URLs came back, and how long it
     * took — enough info for the user to debug a typo without leaving
     * the screen.
     */
    suspend fun test(cfg: CloudflareTurnConfig): TurnTestResult {
        if (!cfg.isConfigured()) {
            return TurnTestResult(ok = false, message = "Cloudflare TURN: Token ID + API Token to'liq emas")
        }
        val start = System.currentTimeMillis()
        val server = withTimeoutOrNull(8_000L) { doFetch(cfg, ttlSeconds = 600) }
        val elapsed = System.currentTimeMillis() - start
        return if (server == null) {
            TurnTestResult(ok = false, message = "Cloudflare TURN xato — credentials olinmadi", gatherMs = elapsed)
        } else {
            TurnTestResult(
                ok = true,
                message = "Cloudflare TURN ishlamoqda — short-lived credentials qaytarildi ✓",
                urls = server.urls.toList(),
                gatherMs = elapsed,
            )
        }
    }

    fun invalidateCache() {
        cache = null
    }

    private suspend fun doFetch(cfg: CloudflareTurnConfig, ttlSeconds: Int): PeerConnection.IceServer? =
        withContext(Dispatchers.IO) {
            val body = """{"ttl": $ttlSeconds}""".toRequestBody("application/json".toMediaType())
            val req = Request.Builder()
                .url(ENDPOINT.format(cfg.tokenId))
                .post(body)
                .header("Authorization", "Bearer ${cfg.apiToken}")
                .header("Content-Type", "application/json")
                .build()

            runCatching {
                http.newCall(req).execute().use { resp ->
                    val raw = resp.body?.string().orEmpty()
                    if (!resp.isSuccessful) {
                        android.util.Log.w(TAG, "HTTP ${resp.code}: ${raw.take(200)}")
                        return@use null
                    }
                    parse(raw)
                }
            }.getOrElse { e ->
                android.util.Log.w(TAG, "fetch failed: ${e.message}")
                null
            }
        }

    private fun parse(raw: String): PeerConnection.IceServer? {
        val obj = runCatching { lenientJson.parseToJsonElement(raw).jsonObject }.getOrNull() ?: return null
        val ice = obj["iceServers"] as? JsonObject ?: return null
        val urls = normaliseUrls(ice["urls"]) ?: return null
        if (urls.isEmpty()) return null
        val username = (ice["username"] as? JsonPrimitive)?.contentOrNull.orEmpty()
        val credential = (ice["credential"] as? JsonPrimitive)?.contentOrNull.orEmpty()
        return PeerConnection.IceServer.builder(urls)
            .setUsername(username)
            .setPassword(credential)
            .createIceServer()
    }

    /** Cloudflare may return urls as a single string or an array of strings. */
    private fun normaliseUrls(v: JsonElement?): List<String>? = when (v) {
        null -> null
        is JsonPrimitive -> v.contentOrNull?.let { listOf(it) }
        is JsonArray -> v.mapNotNull { (it as? JsonPrimitive)?.contentOrNull }
        else -> null
    }
}
