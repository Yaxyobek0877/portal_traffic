package uz.aihealth.portal_mobile.data

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map

/**
 * DataStore-backed persistence for the web-account login.
 *
 * What lives here:
 *  - sessionToken    — value of the __Host-portal_session cookie the
 *                      server set on /api/auth/{signin,signup}. We keep
 *                      it raw and replay it on every API call. Cleared
 *                      on sign-out and on a 401 response.
 *  - username        — last known signed-in username, cached so the
 *                      UI can render the user chip immediately on
 *                      cold start before /api/me has a chance to round-
 *                      trip. Treated as advisory; the source of truth
 *                      is /api/me with the cookie.
 *  - apiBaseUrl      — root of the auth API (defaults to deriving from
 *                      the signaling URL — the same host serves both
 *                      /ws and the /api endpoints in our deployment).
 *                      User-overridable from Settings for local testing.
 *
 * Lives in its own DataStore file so the existing PortalSettings
 * (nickname / signal URL / recent portals) doesn't have to migrate
 * its schema.
 */
private val Context.authStore by preferencesDataStore(name = "portal_auth")

private object AuthKeys {
    val SESSION_TOKEN = stringPreferencesKey("session_token")
    val USERNAME = stringPreferencesKey("username")
    val API_BASE_URL = stringPreferencesKey("api_base_url")
}

class AuthSession(private val context: Context) {

    val sessionToken: Flow<String?> = context.authStore.data
        .map { it[AuthKeys.SESSION_TOKEN]?.takeIf { v -> v.isNotEmpty() } }

    val username: Flow<String?> = context.authStore.data
        .map { it[AuthKeys.USERNAME]?.takeIf { v -> v.isNotEmpty() } }

    val apiBaseUrl: Flow<String?> = context.authStore.data
        .map { it[AuthKeys.API_BASE_URL]?.takeIf { v -> v.isNotEmpty() } }

    /** One-shot read of the current session token. Used at request time. */
    suspend fun currentToken(): String? = sessionToken.first()

    /** One-shot read of the API base URL, falling back to the default. */
    suspend fun resolveApiBaseUrl(defaultUrl: String): String =
        apiBaseUrl.first() ?: defaultUrl

    suspend fun saveSession(token: String, username: String) {
        context.authStore.edit {
            it[AuthKeys.SESSION_TOKEN] = token
            it[AuthKeys.USERNAME] = username
        }
    }

    /** Clear cookie + cached username on sign-out or 401. */
    suspend fun clear() {
        context.authStore.edit {
            it.remove(AuthKeys.SESSION_TOKEN)
            it.remove(AuthKeys.USERNAME)
        }
    }

    suspend fun setApiBaseUrl(value: String) {
        context.authStore.edit {
            if (value.isBlank()) it.remove(AuthKeys.API_BASE_URL)
            else it[AuthKeys.API_BASE_URL] = value.trimEnd('/')
        }
    }
}

/**
 * Translate the wss/ws signaling URL into the HTTP origin that serves
 * the auth endpoints alongside /ws. Fast path: same host, scheme bumps
 * from wss→https or ws→http, and the trailing /ws (if any) is dropped.
 *
 * Used as the default for AuthSession.apiBaseUrl when the user hasn't
 * configured an explicit override in Settings.
 */
fun deriveApiBaseUrl(signalingUrl: String): String {
    var s = signalingUrl.trim()
    if (s.startsWith("wss://", ignoreCase = true)) s = "https://" + s.substring(6)
    else if (s.startsWith("ws://", ignoreCase = true)) s = "http://" + s.substring(5)
    if (s.endsWith("/ws")) s = s.dropLast(3)
    return s.trimEnd('/')
}
