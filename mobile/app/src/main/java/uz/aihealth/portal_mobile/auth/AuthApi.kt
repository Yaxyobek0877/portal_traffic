// Talks to the signal server's /api/auth/* endpoints. The wire contract
// is documented in server/auth_handlers.go:
//
//   POST /api/auth/signup    {username, password}            → 201 + cookie + {id, username}
//   POST /api/auth/signin    {username, password}            → 200 + cookie + {id, username}
//   POST /api/auth/signout                                   → 204
//   GET  /api/me                                             → 200 {id, username} | 401 {error}
//
// On error the server returns 4xx with body
//   {"error": "<code>", "lockoutSeconds": <n optional>}
//
// Where the error codes are the ones in userstore.go (username_taken,
// username_invalid, password_too_short, password_weak,
// invalid_credentials, locked_out, no_session). We map them to a typed
// [AuthErrorCode] so the UI doesn't have to grep server strings.
//
// Cookie handling: the server sets `__Host-portal_session=...` on
// sign-in / sign-up. We don't use OkHttp's CookieJar (overkill for one
// cookie name) — we extract the value from Set-Cookie headers and
// re-attach it via `Cookie:` on subsequent requests. The token lives
// in PortalSettings (DataStore) so it survives process death.

package uz.aihealth.portal_mobile.auth

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import java.util.concurrent.TimeUnit

private const val SESSION_COOKIE = "__Host-portal_session"
private const val DEFAULT_API_BASE = "https://signaling.1pro.uz"

@Serializable
data class UserInfo(val id: String, val username: String)

@Serializable
private data class CredsRequest(val username: String, val password: String)

@Serializable
private data class ApiError(val error: String = "", val lockoutSeconds: Int = 0)

sealed class AuthResult {
    /** Server accepted and returned a fresh session cookie. */
    data class Ok(val user: UserInfo, val sessionToken: String) : AuthResult()

    /** Server returned a typed error (4xx). [lockoutSeconds] is non-zero on LOCKED_OUT. */
    data class Err(val code: AuthErrorCode, val lockoutSeconds: Int = 0) : AuthResult()

    /** Couldn't reach the server / parse its response. UI shows a generic message. */
    data class NetworkError(val message: String) : AuthResult()
}

/**
 * Tri-state result of fetchMe(). Distinguishing "session is invalid"
 * from "I couldn't ask" matters for mandatory-login mode: 401 means
 * boot the user back to the auth screen; a network blip means trust
 * the cached identity until they retry.
 */
sealed class MeResult {
    data class Ok(val user: UserInfo) : MeResult()
    /** 401/403 — cookie is gone, expired, or the server rejected it. */
    object Unauthorized : MeResult()
    /** Anything else — DNS, timeout, 5xx, malformed body. Keep the cache. */
    object NetworkError : MeResult()
}

enum class AuthErrorCode {
    USERNAME_TAKEN,
    USERNAME_INVALID,
    PASSWORD_TOO_SHORT,
    PASSWORD_WEAK,
    INVALID_CREDENTIALS,
    LOCKED_OUT,
    NO_SESSION,
    BAD_REQUEST,
    /**
     * 2xx with a body that isn't UserInfo — almost always means the
     * deployed server doesn't have /api/auth wired up yet (the root
     * handler returned its service-info JSON instead). Surfaced as a
     * specific code so the UI can show "server bu API ni
     * qo'llab-quvvatlamaydi" instead of "noma'lum xato".
     */
    SERVER_OUTDATED,
    UNKNOWN;

    companion object {
        fun fromServer(code: String): AuthErrorCode = when (code) {
            "username_taken" -> USERNAME_TAKEN
            "username_invalid" -> USERNAME_INVALID
            "password_too_short" -> PASSWORD_TOO_SHORT
            "password_weak" -> PASSWORD_WEAK
            "invalid_credentials" -> INVALID_CREDENTIALS
            "locked_out" -> LOCKED_OUT
            "no_session" -> NO_SESSION
            "bad_request" -> BAD_REQUEST
            else -> UNKNOWN
        }
    }
}

class AuthApi {
    private val http = OkHttpClient.Builder()
        .connectTimeout(8, TimeUnit.SECONDS)
        .readTimeout(8, TimeUnit.SECONDS)
        .build()

    private val json = Json { ignoreUnknownKeys = true }

    suspend fun signUp(apiBase: String, username: String, password: String): AuthResult =
        postCreds(apiBase, "/api/auth/signup", username, password)

    suspend fun signIn(apiBase: String, username: String, password: String): AuthResult =
        postCreds(apiBase, "/api/auth/signin", username, password)

    /**
     * POST /api/auth/signout. Returns true on 204, false otherwise.
     * Treat false as "best-effort — clear local state anyway"; we don't
     * want a network blip to leave the user stuck "signed in" forever.
     */
    suspend fun signOut(apiBase: String, token: String): Boolean = withContext(Dispatchers.IO) {
        if (token.isBlank()) return@withContext true
        runCatching {
            http.newCall(
                Request.Builder()
                    .url("$apiBase/api/auth/signout")
                    .header("Cookie", "$SESSION_COOKIE=$token")
                    .post("".toRequestBody())
                    .build(),
            ).execute().use { it.isSuccessful }
        }.getOrDefault(false)
    }

    /**
     * GET /api/me. Returns a tri-state — see [MeResult]. Mandatory-login
     * mode uses Unauthorized to evict the cached session, and treats
     * NetworkError as "keep current state, retry later."
     */
    suspend fun fetchMe(apiBase: String, token: String): MeResult = withContext(Dispatchers.IO) {
        if (token.isBlank()) return@withContext MeResult.Unauthorized
        runCatching {
            http.newCall(
                Request.Builder()
                    .url("$apiBase/api/me")
                    .header("Cookie", "$SESSION_COOKIE=$token")
                    .get()
                    .build(),
            ).execute().use { resp ->
                when {
                    resp.isSuccessful -> {
                        val user = runCatching {
                            json.decodeFromString<UserInfo>(resp.body?.string().orEmpty())
                        }.getOrNull()
                        if (user != null) MeResult.Ok(user) else MeResult.NetworkError
                    }
                    resp.code == 401 || resp.code == 403 -> MeResult.Unauthorized
                    else -> MeResult.NetworkError
                }
            }
        }.getOrElse { MeResult.NetworkError }
    }

    private suspend fun postCreds(
        apiBase: String,
        path: String,
        username: String,
        password: String,
    ): AuthResult = withContext(Dispatchers.IO) {
        val body = json.encodeToString(CredsRequest(username, password))
            .toRequestBody("application/json".toMediaType())
        runCatching {
            http.newCall(
                Request.Builder()
                    .url("$apiBase$path")
                    .post(body)
                    .build(),
            ).execute().use { resp ->
                val raw = resp.body?.string().orEmpty()
                if (resp.isSuccessful) {
                    val user = runCatching { json.decodeFromString<UserInfo>(raw) }.getOrNull()
                    val sessionToken = extractSessionCookie(resp)
                    if (user != null && sessionToken != null) {
                        AuthResult.Ok(user, sessionToken)
                    } else {
                        // 2xx but body isn't UserInfo — almost always means
                        // the deployed server is older than this client and
                        // routed our /api/auth/* request to the root
                        // handler (which returns service-info JSON). Tell
                        // the user that, not "unknown error".
                        AuthResult.Err(AuthErrorCode.SERVER_OUTDATED)
                    }
                } else {
                    val err = runCatching { json.decodeFromString<ApiError>(raw) }.getOrNull()
                    if (err != null && err.error.isNotBlank()) {
                        AuthResult.Err(AuthErrorCode.fromServer(err.error), err.lockoutSeconds)
                    } else {
                        AuthResult.Err(AuthErrorCode.UNKNOWN)
                    }
                }
            }
        }.getOrElse { e ->
            AuthResult.NetworkError(e.message ?: "network error")
        }
    }

    /**
     * Pull the `__Host-portal_session` value out of the response's
     * Set-Cookie headers. We deliberately don't use OkHttp's Cookie.parse
     * because it can be strict about __Host- prefix expectations
     * (Path=/ and Secure); we only care about the name=value pair.
     */
    private fun extractSessionCookie(resp: Response): String? {
        for (sc in resp.headers("Set-Cookie")) {
            val firstPair = sc.substringBefore(';')
            val eq = firstPair.indexOf('=')
            if (eq < 0) continue
            val name = firstPair.substring(0, eq).trim()
            val value = firstPair.substring(eq + 1).trim()
            if (name == SESSION_COOKIE) return value
        }
        return null
    }
}

/**
 * Convert the WebSocket signal URL into the matching HTTP API base.
 *
 *   wss://signaling.1pro.uz/ws  → https://signaling.1pro.uz
 *   ws://localhost:8080/ws       → http://localhost:8080
 *
 * The server hosts /ws and the /api endpoints on the same listener,
 * so this is always a path swap, not a host swap.
 */
fun apiBaseFromSignal(signalUrl: String): String {
    val isSecure = signalUrl.startsWith("wss://", ignoreCase = true)
    val rest = signalUrl
        .removePrefix("wss://")
        .removePrefix("WSS://")
        .removePrefix("ws://")
        .removePrefix("WS://")
    val authority = rest.substringBefore('/').takeIf { it.isNotBlank() } ?: return DEFAULT_API_BASE
    val scheme = if (isSecure) "https" else "http"
    return "$scheme://$authority"
}
