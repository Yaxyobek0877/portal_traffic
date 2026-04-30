package uz.aihealth.portal_mobile.mesh

import android.content.Context
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import uz.aihealth.portal_mobile.crypt.Crypt
import uz.aihealth.portal_mobile.crypt.PortalKey
import uz.aihealth.portal_mobile.peer.CHAN_CHAT
import uz.aihealth.portal_mobile.peer.CHAN_CONTROL
import uz.aihealth.portal_mobile.peer.PeerMessage
import uz.aihealth.portal_mobile.peer.PeerState
import uz.aihealth.portal_mobile.peer.PortalPeerConnection
import uz.aihealth.portal_mobile.peer.Role
import uz.aihealth.portal_mobile.peer.WebRtcFactory
import uz.aihealth.portal_mobile.peer.defaultIceServers
import uz.aihealth.portal_mobile.protocol.MessageType
import uz.aihealth.portal_mobile.protocol.Ping
import uz.aihealth.portal_mobile.protocol.Pong
import uz.aihealth.portal_mobile.protocol.portalJson
import uz.aihealth.portal_mobile.protocol.typeOf
import uz.aihealth.portal_mobile.signaling.DEFAULT_SIGNALING_URL
import uz.aihealth.portal_mobile.signaling.SignalingClient
import uz.aihealth.portal_mobile.signaling.SignalingEvent
import java.util.concurrent.ConcurrentHashMap

private const val TAG = "Mesh"
private const val HEARTBEAT_INTERVAL_MS = 5_000L
private const val PING_TIMEOUT_MS = 15_000L

data class PortalInfo(
    val portalId: String,
    val code: String,
    val ownerId: String,
    val ownPeerId: String,
    val ownVip: String,
    val isOwner: Boolean,
)

data class PeerSnapshot(
    val id: String,
    val nickname: String,
    val virtualIp: String,
    val isOwner: Boolean,
    val state: PeerState,
    val rttMs: Long,
)

data class ChatMessage(
    val fromPeerId: String,
    val fromNickname: String,
    val text: String,
    val timestampMs: Long,
)

sealed class MeshState {
    data object Idle : MeshState()
    data object Connecting : MeshState()
    data class Ready(val portal: PortalInfo) : MeshState()
    data class Failed(val reason: String) : MeshState()
    data object Closed : MeshState()
}

private class MeshPeer(
    val id: String,
    val nickname: String,
    val virtualIp: String,
    val isOwner: Boolean,
    val conn: PortalPeerConnection,
) {
    @Volatile var state: PeerState = PeerState.CONNECTING
    @Volatile var rttMs: Long = 0L
    val outstandingPings = ConcurrentHashMap<Long, Long>()
}

/**
 * High-level orchestrator. Owns one [SignalingClient] and a fan-out of
 * [PortalPeerConnection]s — drives create/join, the WebRTC handshake
 * (per the same glare-free rule as client/mesh/manager.go: the joiner
 * is offerer for every existing peer, existing peers are answerers
 * for newcomers), heartbeats over the control channel, and chat
 * over the chat channel sealed with the portal-derived secretbox key.
 *
 * Currently implemented: chat + RTT. Transfer, proxy, services, and
 * auto-reconnect are deferred.
 */
class MeshManager(
    appContext: Context,
    private val scope: CoroutineScope,
    private val nickname: String,
    private val signalingUrl: String = DEFAULT_SIGNALING_URL,
) {
    private val factory = WebRtcFactory.get(appContext)
    private val sig = SignalingClient(signalingUrl)

    private val peers = ConcurrentHashMap<String, MeshPeer>()

    @Volatile private var portalKey: PortalKey? = null
    @Volatile private var myPeerId: String = ""
    @Volatile private var myNickname: String = nickname
    @Volatile private var queuedOperation: (() -> Unit)? = null

    private val _state = MutableStateFlow<MeshState>(MeshState.Idle)
    val state: StateFlow<MeshState> = _state.asStateFlow()

    private val _peers = MutableStateFlow<List<PeerSnapshot>>(emptyList())
    val peerList: StateFlow<List<PeerSnapshot>> = _peers.asStateFlow()

    private val _chats = MutableSharedFlow<ChatMessage>(
        replay = 0,
        extraBufferCapacity = 64,
    )
    val chats: SharedFlow<ChatMessage> = _chats.asSharedFlow()

    private var consumerJob: Job? = null
    private var heartbeatJob: Job? = null

    // ------------------------------------------------------------------------
    // Public API
    // ------------------------------------------------------------------------

    fun createPortal(publicNick: Boolean = false) {
        check(consumerJob == null) { "mesh already running" }
        _state.value = MeshState.Connecting
        queuedOperation = { sig.createPortal(nickname, publicNick) }
        startConsumer()
        startHeartbeat()
        sig.connect()
    }

    fun joinPortal(portalId: String, code: String) {
        check(consumerJob == null) { "mesh already running" }
        _state.value = MeshState.Connecting
        portalKey = Crypt.derive(code)
        queuedOperation = { sig.joinPortal(portalId, code, nickname) }
        startConsumer()
        startHeartbeat()
        sig.connect()
    }

    /** Encrypt and broadcast `text` to every peer's chat channel. */
    fun sendChat(text: String): Int {
        val k = portalKey ?: return 0
        val sealed = Crypt.seal(k, text.toByteArray(Charsets.UTF_8))
        var n = 0
        for (p in peers.values) {
            if (p.conn.channelOpen(CHAN_CHAT) && p.conn.sendBinary(CHAN_CHAT, sealed)) n++
        }
        if (n > 0) {
            _chats.tryEmit(
                ChatMessage(
                    fromPeerId = myPeerId,
                    fromNickname = myNickname,
                    text = text,
                    timestampMs = System.currentTimeMillis(),
                ),
            )
        }
        return n
    }

    fun leave() {
        sig.leave()
        close()
    }

    fun close() {
        consumerJob?.cancel()
        heartbeatJob?.cancel()
        for (p in peers.values) p.conn.close()
        peers.clear()
        sig.close()
        _state.value = MeshState.Closed
        _peers.value = emptyList()
    }

    // ------------------------------------------------------------------------
    // Internals
    // ------------------------------------------------------------------------

    private fun startConsumer() {
        consumerJob = scope.launch {
            sig.events.collect { handleSignalingEvent(it) }
        }
    }

    private fun startHeartbeat() {
        heartbeatJob = scope.launch {
            while (true) {
                delay(HEARTBEAT_INTERVAL_MS)
                tickHeartbeat()
            }
        }
    }

    private fun tickHeartbeat() {
        val now = System.currentTimeMillis()
        val ping = Ping(ts = now)
        val pingText = portalJson.encodeToString(ping)
        for (p in peers.values) {
            if (!p.conn.channelOpen(CHAN_CONTROL)) continue
            if (p.conn.sendText(CHAN_CONTROL, pingText)) {
                p.outstandingPings[now] = now
            }
        }
        // Prune stale outstanding pings (peer never replied → packet loss).
        val cutoff = now - PING_TIMEOUT_MS
        for (p in peers.values) {
            val it = p.outstandingPings.entries.iterator()
            while (it.hasNext()) if (it.next().value < cutoff) it.remove()
        }
    }

    private suspend fun handleSignalingEvent(ev: SignalingEvent) {
        when (ev) {
            is SignalingEvent.Connected -> {
                queuedOperation?.invoke()
                queuedOperation = null
            }

            is SignalingEvent.Created -> {
                myPeerId = ev.msg.peerId
                portalKey = Crypt.derive(ev.msg.code)
                _state.value = MeshState.Ready(
                    PortalInfo(
                        portalId = ev.msg.portalId,
                        code = ev.msg.code,
                        ownerId = ev.msg.peerId,
                        ownPeerId = ev.msg.peerId,
                        ownVip = ev.msg.virtualIp,
                        isOwner = true,
                    ),
                )
            }

            is SignalingEvent.Joined -> {
                myPeerId = ev.msg.peerId
                val ownerId = ev.msg.peers.firstOrNull { it.isOwner }?.peerId ?: ""
                _state.value = MeshState.Ready(
                    PortalInfo(
                        portalId = ev.msg.portalId,
                        code = "",
                        ownerId = ownerId,
                        ownPeerId = ev.msg.peerId,
                        ownVip = ev.msg.virtualIp,
                        isOwner = false,
                    ),
                )
                // Joiner: initiate handshake with each existing peer.
                for (p in ev.msg.peers) {
                    addPeer(p.peerId, p.nickname, p.virtualIp, p.isOwner, weAreJoiner = true)
                }
            }

            is SignalingEvent.PeerJoined -> {
                addPeer(ev.msg.peerId, ev.msg.nickname, ev.msg.virtualIp, isOwner = false, weAreJoiner = false)
            }

            is SignalingEvent.PeerLeft -> handlePeerLeft(ev.msg.peerId)

            is SignalingEvent.Offer -> {
                val from = ev.msg.from ?: return
                handleRemoteOffer(from, ev.msg.sdp)
            }

            is SignalingEvent.Answer -> {
                val from = ev.msg.from ?: return
                handleRemoteAnswer(from, ev.msg.sdp)
            }

            is SignalingEvent.Ice -> {
                val from = ev.msg.from ?: return
                peers[from]?.conn?.addRemoteIce(
                    candidate = ev.msg.candidate,
                    sdpMid = ev.msg.sdpMid,
                    sdpMLineIndex = ev.msg.sdpMLineIndex,
                )
            }

            is SignalingEvent.Closed -> {
                _state.value = MeshState.Failed("portal closed: ${ev.msg.reason ?: ""}")
                close()
            }

            is SignalingEvent.Kicked -> {
                _state.value = MeshState.Failed("kicked from portal")
                close()
            }

            is SignalingEvent.Locked -> {
                // Just an info update — would surface to UI in a fuller impl.
                Log.d(TAG, "portal locked=${ev.msg.locked}")
            }

            is SignalingEvent.Error -> {
                Log.w(TAG, "signaling error ${ev.msg.code}: ${ev.msg.message}")
                if (_state.value !is MeshState.Ready) {
                    _state.value = MeshState.Failed(ev.msg.message)
                    close()
                }
            }

            is SignalingEvent.ConnectionLost -> {
                if (_state.value !is MeshState.Closed) {
                    _state.value = MeshState.Failed("signaling: ${ev.reason}")
                }
                close()
            }
        }
    }

    private fun addPeer(
        peerId: String,
        nickname: String,
        virtualIp: String,
        isOwner: Boolean,
        weAreJoiner: Boolean,
    ) {
        if (peers.containsKey(peerId)) return
        // Glare-free rule: joiner is offerer for every existing peer;
        // existing peers receiving peer_joined are answerers.
        val role = if (weAreJoiner) Role.OFFERER else Role.ANSWERER
        val pc = PortalPeerConnection(
            factory = factory,
            localPeerId = myPeerId,
            remotePeerId = peerId,
            role = role,
            iceServers = defaultIceServers,
        )
        val mp = MeshPeer(peerId, nickname, virtualIp, isOwner, pc)
        peers[peerId] = mp
        publishPeers()

        // Forward local ICE → signaling.
        scope.launch {
            pc.localIce.collect { ice ->
                sig.sendIce(peerId, ice.sdp, ice.sdpMid, ice.sdpMLineIndex)
            }
        }
        // Inbound data-channel messages.
        scope.launch {
            pc.messages.collect { msg -> routeMessage(mp, msg) }
        }
        // State updates → re-publish snapshot.
        scope.launch {
            pc.state.collect { st ->
                mp.state = st
                publishPeers()
                if (st == PeerState.CLOSED || st == PeerState.FAILED) {
                    peers.remove(peerId)
                    publishPeers()
                }
            }
        }

        if (role == Role.OFFERER) {
            scope.launch {
                runCatching {
                    val sdp = pc.createOffer()
                    sig.sendOffer(peerId, sdp)
                }.onFailure { Log.w(TAG, "create offer to $peerId: ${it.message}") }
            }
        }
    }

    private fun handleRemoteOffer(from: String, sdp: String) {
        val mp = peers[from]
        if (mp == null) {
            // Could happen if peer_joined was lost; treat the offer as a join trigger.
            addPeer(from, nickname = "", virtualIp = "", isOwner = false, weAreJoiner = false)
        }
        val target = peers[from] ?: return
        scope.launch {
            runCatching {
                val answer = target.conn.handleOffer(sdp)
                sig.sendAnswer(from, answer)
            }.onFailure { Log.w(TAG, "handle offer from $from: ${it.message}") }
        }
    }

    private fun handleRemoteAnswer(from: String, sdp: String) {
        val mp = peers[from] ?: return
        scope.launch {
            runCatching { mp.conn.handleAnswer(sdp) }
                .onFailure { Log.w(TAG, "handle answer from $from: ${it.message}") }
        }
    }

    private fun handlePeerLeft(peerId: String) {
        val mp = peers.remove(peerId) ?: return
        mp.conn.close()
        publishPeers()
    }

    private fun routeMessage(p: MeshPeer, msg: PeerMessage) {
        when (msg.channel) {
            CHAN_CONTROL -> handleControl(p, msg)
            CHAN_CHAT -> handleChat(p, msg)
            // transfer / proxy: deferred
        }
    }

    private fun handleControl(p: MeshPeer, msg: PeerMessage) {
        if (!msg.text) return
        val raw = String(msg.data, Charsets.UTF_8)
        val type = typeOf(raw) ?: return
        when (type) {
            MessageType.PING -> {
                runCatching {
                    val ping = portalJson.decodeFromString<Ping>(raw)
                    val pong = Pong(ts = System.currentTimeMillis(), echoTs = ping.ts)
                    p.conn.sendText(CHAN_CONTROL, portalJson.encodeToString(pong))
                }
            }
            MessageType.PONG -> {
                runCatching {
                    val pong = portalJson.decodeFromString<Pong>(raw)
                    val sentAt = p.outstandingPings.remove(pong.echoTs) ?: return
                    p.rttMs = System.currentTimeMillis() - sentAt
                    publishPeers()
                }
            }
        }
    }

    private fun handleChat(p: MeshPeer, msg: PeerMessage) {
        val k = portalKey ?: return
        // Older builds may have sent text frames — fall through if so.
        val plain: ByteArray = if (msg.text) msg.data else (Crypt.open(k, msg.data) ?: return)
        val text = String(plain, Charsets.UTF_8)
        _chats.tryEmit(
            ChatMessage(
                fromPeerId = p.id,
                fromNickname = p.nickname,
                text = text,
                timestampMs = System.currentTimeMillis(),
            ),
        )
    }

    private fun publishPeers() {
        _peers.value = peers.values.map {
            PeerSnapshot(
                id = it.id,
                nickname = it.nickname,
                virtualIp = it.virtualIp,
                isOwner = it.isOwner,
                state = it.state,
                rttMs = it.rttMs,
            )
        }
    }
}
