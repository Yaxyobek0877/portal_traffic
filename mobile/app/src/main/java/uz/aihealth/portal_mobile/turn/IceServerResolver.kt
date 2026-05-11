// Builds the final ICE-server list for a peer connection. Precedence
// matches the desktop client (client/cloudflareturn.go resolveICEServers):
//
//   1. Default STUN servers (always included — give cheap srflx
//      candidates and don't conflict with TURN).
//   2. Cloudflare TURN, if configured AND the API call succeeds.
//   3. Manual TURN, if configured.
//
// We don't gate (2) on (3) — both can be present, and pion / WebRTC
// will pick whichever wins ICE. Falling back to (1) only happens when
// neither TURN source is configured or both failed.

package uz.aihealth.portal_mobile.turn

import org.webrtc.PeerConnection
import uz.aihealth.portal_mobile.peer.defaultIceServers

object IceServerResolver {

    /**
     * Resolve the live ICE-server list. May make a network call (to
     * Cloudflare) — call from a coroutine. [cfg] holds whatever the
     * user persisted; pass empties if unconfigured.
     */
    suspend fun resolve(cf: CloudflareTurnConfig, manual: ManualTurnConfig): List<PeerConnection.IceServer> {
        val servers = mutableListOf<PeerConnection.IceServer>()
        servers.addAll(defaultIceServers)

        if (cf.isConfigured()) {
            CloudflareTurn.fetch(cf)?.let { servers.add(it) }
        }
        if (manual.isConfigured()) {
            servers.add(buildManual(manual))
        }
        return servers
    }

    /**
     * Synchronous fallback — used when we can't await a coroutine
     * (very rare path; mostly a defensive default).
     */
    fun resolveOffline(manual: ManualTurnConfig): List<PeerConnection.IceServer> {
        val servers = mutableListOf<PeerConnection.IceServer>()
        servers.addAll(defaultIceServers)
        if (manual.isConfigured()) servers.add(buildManual(manual))
        return servers
    }

    private fun buildManual(m: ManualTurnConfig): PeerConnection.IceServer {
        val urls = m.url.split('\n', ',', ';')
            .map { it.trim() }
            .filter { it.startsWith("turn:") || it.startsWith("turns:") }
        return PeerConnection.IceServer.builder(urls)
            .setUsername(m.username)
            .setPassword(m.credential)
            .createIceServer()
    }
}
