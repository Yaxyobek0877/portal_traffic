package uz.aihealth.portal_mobile.data

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import uz.aihealth.portal_mobile.protocol.portalJson
import java.util.concurrent.TimeUnit

/**
 * Auth API client. Hits the same /api/auth/{signup,signin,signout},
 * /api/me, /api/portals endpoints the web admin uses (server-side code
 * lives in the gitignored server/ folder; the wire shape is the
 * Go server's apiError + userResponse + portalsResponse).
 *
 * Cookie handling is manual rather than via OkHttp's CookieJar: we
 * extract the __Host-portal_session value from the Set-Cookie header
 * on signin/signup responses and replay it on subsequent requests via
 * the Cookie header. This keeps the persistence story trivial — one
 * string in DataStore — and avoids wrestling with HttpOnly + cross-
 * process cookie handoff.
 *
 * All public methods suspend and switch to Dispatchers.IO; callers
 * launch them from viewModelScope as usual.
 */
class AuthApi(
    private val session: AuthSession,
    private val baseUrlProvider: suspend () -> String,
) {

    private val http = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .callTimeout(15, TimeUnit.SECONDS)
        .build()

    private val jsonMedia = "application/json; charset=utf-8".toMediaType()

    // ---- Wire types — mirror server/auth_handlers.go responses. ----

    @Serializable
    private data class CredsBody(val username: String, val password: String)

    @Serializable
    private data class UserResponse(val id: String, val username: String)

    @Serializable
    private data class ApiError(
        val error: String,
        val lockoutSeconds: Int = 0,
    )

    @Serializable
    data class PortalSummary(
        val portalId: String,
        val ownerNickname: String = "",
        val peerCount: Int = 0,
        val createdAt: String = "",
    )

    @Serializable
    private data class PortalsResponse(
        val portals: List<PortalSummary> = emptyList(),
        val note: String = "",
    )

    // ---- Result type — keeps the UI free of OkHttp imports. ----

    sealed class AuthResult<out T> {
        data class Success<T>(val data: T) : AuthResult<T>()
        data class Failure(val code: String, val lockoutSeconds: Int = 0) : AuthResult<Nothing>()
    }

    data class User(val id: String, val username: String)
    data class PortalsView(val portals: List<PortalSummary>, val note: String)

    // ---- Public API ----

    suspend fun signUp(username: String, password: String): AuthResult<User> =
        post("/api/auth/signup", CredsBody(username, password))

    suspend fun signIn(username: String, password: String): AuthResult<User> =
        post("/api/auth/signin", CredsBody(username, password))

    suspend fun signOut(): AuthResult<Unit> = withContext(Dispatchers.IO) {
        val baseUrl = baseUrlProvider()
        val req = Request.Builder()
            .url("$baseUrl/api/auth/signout")
            .post("".toRequestBody(jsonMedia))
            .applyCookieIfAny()
            .build()
        runCatching {
            http.newCall(req).execute().use { res ->
                // Always clear local state — on a 4xx the cookie is
                // already invalid; on a 5xx we'd rather force a fresh
                // sign-in than leave a half-broken session lying around.
                session.clear()
                if (res.isSuccessful) AuthResult.Success(Unit)
                else AuthResult.Failure("server_error", 0)
            }
        }.getOrElse { AuthResult.Failure("network", 0) }
    }

    suspend fun me(): AuthResult<User> = withContext(Dispatchers.IO) {
        val baseUrl = baseUrlProvider()
        val req = Request.Builder()
            .url("$baseUrl/api/me")
            .get()
            .applyCookieIfAny()
            .build()
        runCatching {
            http.newCall(req).execute().use { res ->
                if (!res.isSuccessful) {
                    if (res.code == 401) session.clear()
                    return@withContext AuthResult.Failure(decodeErrorCode(res), 0)
                }
                val body = res.body?.string().orEmpty()
                val parsed = portalJson.decodeFromString<UserResponse>(body)
                AuthResult.Success(User(parsed.id, parsed.username))
            }
        }.getOrElse { AuthResult.Failure("network", 0) }
    }

    suspend fun portals(): AuthResult<PortalsView> = withContext(Dispatchers.IO) {
        val baseUrl = baseUrlProvider()
        val req = Request.Builder()
            .url("$baseUrl/api/portals")
            .get()
            .applyCookieIfAny()
            .build()
        runCatching {
            http.newCall(req).execute().use { res ->
                if (!res.isSuccessful) {
                    if (res.code == 401) session.clear()
                    return@withContext AuthResult.Failure(decodeErrorCode(res), 0)
                }
                val body = res.body?.string().orEmpty()
                val parsed = portalJson.decodeFromString<PortalsResponse>(body)
                AuthResult.Success(PortalsView(parsed.portals, parsed.note))
            }
        }.getOrElse { AuthResult.Failure("network", 0) }
    }

    // ---- Internals ----

    /**
     * Generic JSON POST that captures the session cookie from Set-Cookie
     * if present. signup/signin both go through here; signout doesn't
     * (it has its own minimal handler above so we can clear() even on
     * an error response).
     */
    private suspend inline fun <reified Body : Any> post(
        path: String,
        body: Body,
    ): AuthResult<User> = withContext(Dispatchers.IO) {
        val baseUrl = baseUrlProvider()
        val req = Request.Builder()
            .url("$baseUrl$path")
            .post(portalJson.encodeToString(body).toRequestBody(jsonMedia))
            .applyCookieIfAny()
            .build()
        runCatching {
            http.newCall(req).execute().use { res ->
                if (!res.isSuccessful) {
                    return@withContext decodeError(res)
                }
                // Extract the session cookie BEFORE reading the body —
                // OkHttp lazily lazy reads, but headers are always there.
                val cookieToken = extractSessionToken(res.headers("Set-Cookie"))
                val raw = res.body?.string().orEmpty()
                val user = portalJson.decodeFromString<UserResponse>(raw)
                if (cookieToken != null) {
                    session.saveSession(cookieToken, user.username)
                } else {
                    // Server didn't set a cookie (unexpected) — still
                    // record the username so the UI has something to
                    // display, but the next /api/me will 401 and bounce
                    // the user back to login.
                    session.saveSession(session.currentToken().orEmpty(), user.username)
                }
                AuthResult.Success(User(user.id, user.username))
            }
        }.getOrElse { AuthResult.Failure("network", 0) }
    }

    /**
     * Pull the value of __Host-portal_session out of one of the
     * response's Set-Cookie headers, if any. Uses a simple substring
     * walk rather than a full RFC-6265 parser — the server only emits
     * one cookie and only this name.
     */
    private fun extractSessionToken(setCookies: List<String>): String? {
        val key = "__Host-portal_session="
        for (c in setCookies) {
            val idx = c.indexOf(key)
            if (idx == -1) continue
            val rest = c.substring(idx + key.length)
            // value ends at the first `;` or end of string
            val end = rest.indexOf(';').takeIf { it >= 0 } ?: rest.length
            return rest.substring(0, end)
        }
        return null
    }

    /**
     * Apply the saved session cookie if we have one. Builder extension
     * keeps the call sites readable. Note: we read the token at request-
     * build time, not at construction, so a sign-out that just cleared
     * it doesn't get sent on a queued retry.
     */
    private suspend fun Request.Builder.applyCookieIfAny(): Request.Builder {
        val tok = session.currentToken()
        if (!tok.isNullOrEmpty()) {
            header("Cookie", "__Host-portal_session=$tok")
        }
        return this
    }

    /**
     * Decode an error response body's `error` + `lockoutSeconds`. If the
     * body isn't valid JSON (e.g. the proxy returned an HTML error
     * page), fall back to a synthetic code derived from the HTTP status.
     */
    private fun decodeError(res: okhttp3.Response): AuthResult.Failure {
        val raw = res.body?.string().orEmpty()
        val parsed = runCatching {
            portalJson.decodeFromString<ApiError>(raw)
        }.getOrNull()
        if (parsed != null) {
            return AuthResult.Failure(parsed.error, parsed.lockoutSeconds)
        }
        return AuthResult.Failure("http_${res.code}", 0)
    }

    private fun decodeErrorCode(res: okhttp3.Response): String {
        val raw = res.body?.string().orEmpty()
        val parsed = runCatching {
            portalJson.decodeFromString<ApiError>(raw)
        }.getOrNull()
        return parsed?.error ?: "http_${res.code}"
    }
}
