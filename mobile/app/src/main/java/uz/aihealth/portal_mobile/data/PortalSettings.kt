package uz.aihealth.portal_mobile.data

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import uz.aihealth.portal_mobile.protocol.portalJson
import uz.aihealth.portal_mobile.signaling.DEFAULT_SIGNALING_URL
import uz.aihealth.portal_mobile.turn.CloudflareTurnConfig
import uz.aihealth.portal_mobile.turn.ManualTurnConfig

@Serializable
data class RecentPortal(
    val portalId: String,
    val code: String,
    val role: String,            // "owner" | "joiner"
    val nickname: String,
    val lastUsedMs: Long,
)

@Serializable
private data class PersistedRecents(val items: List<RecentPortal> = emptyList())

private val Context.portalDataStore by preferencesDataStore(name = "portal_settings")

private object Keys {
    val NICKNAME = stringPreferencesKey("nickname")
    val SIGNAL_URL = stringPreferencesKey("signal_url")
    val RECENT_PORTALS = stringPreferencesKey("recent_portals")

    // Added in v1.1: language + TURN settings. Defaults are applied at
    // read time (see corresponding Flow below) so existing installs
    // don't need a migration.
    val LANG = stringPreferencesKey("lang")
    val CF_TURN = stringPreferencesKey("cf_turn")          // JSON
    val MANUAL_TURN = stringPreferencesKey("manual_turn")  // JSON

    // Added in v1.2: server account session. Token comes from the
    // Set-Cookie returned by /api/auth/{signup,signin}; we re-attach it
    // verbatim on later /api/* calls. ACCOUNT_USERNAME and ACCOUNT_USER_ID
    // are cached so the UI can show "Salom, X" without an /api/me round-trip
    // on every cold start; the actual identity is re-validated on app
    // launch via fetchMe(token). Cleared on sign-out or on a 401 from /api/me.
    val SESSION_TOKEN = stringPreferencesKey("session_token")
    val ACCOUNT_USERNAME = stringPreferencesKey("account_username")
    val ACCOUNT_USER_ID = stringPreferencesKey("account_user_id")
}

private const val MAX_RECENTS = 10

/**
 * Persistent settings backed by Preferences DataStore. Survives process
 * death and reinstalls (respecting Android backup rules). Reads are
 * Flows; writes are suspend functions.
 */
class PortalSettings(private val context: Context) {

    val nickname: Flow<String> = context.portalDataStore.data
        .map { it[Keys.NICKNAME].orEmpty() }

    val signalUrl: Flow<String> = context.portalDataStore.data
        .map { it[Keys.SIGNAL_URL] ?: DEFAULT_SIGNALING_URL }

    /** "uz" or "en"; legacy installs without a stored value default to "uz". */
    val lang: Flow<String> = context.portalDataStore.data
        .map { it[Keys.LANG] ?: "uz" }

    val cloudflareTurn: Flow<CloudflareTurnConfig> = context.portalDataStore.data
        .map { prefs ->
            prefs[Keys.CF_TURN]
                ?.let { runCatching { portalJson.decodeFromString<CloudflareTurnConfig>(it) }.getOrNull() }
                ?: CloudflareTurnConfig()
        }

    val manualTurn: Flow<ManualTurnConfig> = context.portalDataStore.data
        .map { prefs ->
            prefs[Keys.MANUAL_TURN]
                ?.let { runCatching { portalJson.decodeFromString<ManualTurnConfig>(it) }.getOrNull() }
                ?: ManualTurnConfig()
        }

    val recentPortals: Flow<List<RecentPortal>> = context.portalDataStore.data
        .map {
            val raw = it[Keys.RECENT_PORTALS] ?: return@map emptyList()
            runCatching {
                portalJson.decodeFromString<PersistedRecents>(raw).items
            }.getOrElse { emptyList() }
        }

    /** Server-issued session cookie value, or "" if not signed in. */
    val sessionToken: Flow<String> = context.portalDataStore.data
        .map { it[Keys.SESSION_TOKEN].orEmpty() }

    /** Cached username + ID from the last /api/me. Empty when signed out. */
    val accountUsername: Flow<String> = context.portalDataStore.data
        .map { it[Keys.ACCOUNT_USERNAME].orEmpty() }

    val accountUserId: Flow<String> = context.portalDataStore.data
        .map { it[Keys.ACCOUNT_USER_ID].orEmpty() }

    suspend fun setNickname(value: String) {
        context.portalDataStore.edit { it[Keys.NICKNAME] = value }
    }

    suspend fun setSignalUrl(value: String) {
        context.portalDataStore.edit { it[Keys.SIGNAL_URL] = value }
    }

    suspend fun setLang(value: String) {
        context.portalDataStore.edit { it[Keys.LANG] = value }
    }

    suspend fun setCloudflareTurn(cfg: CloudflareTurnConfig) {
        context.portalDataStore.edit { it[Keys.CF_TURN] = portalJson.encodeToString(cfg) }
    }

    suspend fun setManualTurn(cfg: ManualTurnConfig) {
        context.portalDataStore.edit { it[Keys.MANUAL_TURN] = portalJson.encodeToString(cfg) }
    }

    /**
     * Persist the active session. Pass empty strings for token/username/userId
     * to clear (i.e. on sign-out). All three are stored together — partial
     * states would only confuse the UI.
     */
    suspend fun setSession(token: String, username: String, userId: String) {
        context.portalDataStore.edit {
            if (token.isBlank()) {
                it.remove(Keys.SESSION_TOKEN)
                it.remove(Keys.ACCOUNT_USERNAME)
                it.remove(Keys.ACCOUNT_USER_ID)
            } else {
                it[Keys.SESSION_TOKEN] = token
                it[Keys.ACCOUNT_USERNAME] = username
                it[Keys.ACCOUNT_USER_ID] = userId
            }
        }
    }

    suspend fun clearSession() = setSession("", "", "")

    /**
     * Insert/refresh a recent-portal entry. The most recent goes first;
     * the list is capped at [MAX_RECENTS] entries (oldest fall off).
     */
    suspend fun rememberPortal(entry: RecentPortal) {
        context.portalDataStore.edit { prefs ->
            val current = runCatching {
                prefs[Keys.RECENT_PORTALS]
                    ?.let { portalJson.decodeFromString<PersistedRecents>(it).items }
                    .orEmpty()
            }.getOrElse { emptyList() }

            val merged = (listOf(entry) + current.filterNot { it.portalId == entry.portalId })
                .take(MAX_RECENTS)
            prefs[Keys.RECENT_PORTALS] = portalJson.encodeToString(PersistedRecents(merged))
        }
    }

    suspend fun clearRecent(portalId: String) {
        context.portalDataStore.edit { prefs ->
            val current = runCatching {
                prefs[Keys.RECENT_PORTALS]
                    ?.let { portalJson.decodeFromString<PersistedRecents>(it).items }
                    .orEmpty()
            }.getOrElse { emptyList() }
            val pruned = current.filterNot { it.portalId == portalId }
            prefs[Keys.RECENT_PORTALS] = portalJson.encodeToString(PersistedRecents(pruned))
        }
    }
}
