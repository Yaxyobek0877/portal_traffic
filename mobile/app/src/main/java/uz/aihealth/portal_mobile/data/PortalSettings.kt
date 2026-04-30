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

    val recentPortals: Flow<List<RecentPortal>> = context.portalDataStore.data
        .map {
            val raw = it[Keys.RECENT_PORTALS] ?: return@map emptyList()
            runCatching {
                portalJson.decodeFromString<PersistedRecents>(raw).items
            }.getOrElse { emptyList() }
        }

    suspend fun setNickname(value: String) {
        context.portalDataStore.edit { it[Keys.NICKNAME] = value }
    }

    suspend fun setSignalUrl(value: String) {
        context.portalDataStore.edit { it[Keys.SIGNAL_URL] = value }
    }

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
