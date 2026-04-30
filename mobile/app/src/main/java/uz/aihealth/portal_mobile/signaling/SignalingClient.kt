package uz.aihealth.portal_mobile.signaling

import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import uz.aihealth.portal_mobile.protocol.ErrorMessage
import uz.aihealth.portal_mobile.protocol.MessageType
import uz.aihealth.portal_mobile.protocol.PortalClosed
import uz.aihealth.portal_mobile.protocol.PortalCreate
import uz.aihealth.portal_mobile.protocol.PortalCreated
import uz.aihealth.portal_mobile.protocol.PortalJoin
import uz.aihealth.portal_mobile.protocol.PortalJoined
import uz.aihealth.portal_mobile.protocol.PortalKick
import uz.aihealth.portal_mobile.protocol.PortalKicked
import uz.aihealth.portal_mobile.protocol.PortalLeave
import uz.aihealth.portal_mobile.protocol.PortalLock
import uz.aihealth.portal_mobile.protocol.PortalLocked
import uz.aihealth.portal_mobile.protocol.PortalPeerJoined
import uz.aihealth.portal_mobile.protocol.PortalPeerLeft
import uz.aihealth.portal_mobile.protocol.WebRTCAnswer
import uz.aihealth.portal_mobile.protocol.WebRTCICE
import uz.aihealth.portal_mobile.protocol.WebRTCOffer
import uz.aihealth.portal_mobile.protocol.portalJson
import uz.aihealth.portal_mobile.protocol.typeOf
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean

const val DEFAULT_SIGNALING_URL = "wss://signaling.1pro.uz/ws"

/**
 * Mirrors client/signaling/client.go: typed WebSocket client. Outbound
 * sends are typed setters; inbound frames are decoded into a sealed
 * [SignalingEvent] and delivered through [events].
 */
class SignalingClient(
    private val url: String = DEFAULT_SIGNALING_URL,
) {
    private val tag = "Signaling"

    private val http = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(0, TimeUnit.MILLISECONDS) // WebSocket: never time out reads
        .pingInterval(30, TimeUnit.SECONDS)
        .build()

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val _events = Channel<SignalingEvent>(64)
    val events: Flow<SignalingEvent> = _events.receiveAsFlow()

    private val closed = AtomicBoolean(false)
    private var ws: WebSocket? = null

    fun url(): String = url

    /** Open the WebSocket. Events start arriving on [events] once the connection opens. */
    fun connect() {
        if (closed.get()) error("SignalingClient already closed")
        val req = Request.Builder().url(url).build()
        ws = http.newWebSocket(req, listener)
    }

    /** Close the connection. Idempotent. */
    fun close() {
        if (!closed.compareAndSet(false, true)) return
        ws?.close(1000, null)
        ws = null
        _events.close()
        scope.cancel()
    }

    // ------------------------------------------------------------------------
    // Outbound — typed senders
    // ------------------------------------------------------------------------

    fun createPortal(nickname: String, publicNick: Boolean = false, capacity: Int = 0) =
        send(PortalCreate(nickname = nickname, publicNick = publicNick, capacity = capacity))

    fun joinPortal(portalId: String, code: String, nickname: String) =
        send(PortalJoin(portalId = portalId, code = code, nickname = nickname))

    fun leave() = send(PortalLeave())

    fun kick(peerId: String) = send(PortalKick(peerId = peerId))

    fun lock(locked: Boolean) = send(PortalLock(locked = locked))

    fun sendOffer(to: String, sdp: String) = send(WebRTCOffer(to = to, sdp = sdp))

    fun sendAnswer(to: String, sdp: String) = send(WebRTCAnswer(to = to, sdp = sdp))

    fun sendIce(to: String, candidate: String, sdpMid: String?, sdpMLineIndex: Int?) =
        send(
            WebRTCICE(
                to = to,
                candidate = candidate,
                sdpMid = sdpMid,
                sdpMLineIndex = sdpMLineIndex,
            ),
        )

    private inline fun <reified T> send(payload: T): Boolean {
        val socket = ws ?: return false
        if (closed.get()) return false
        val text = portalJson.encodeToString(payload)
        return socket.send(text)
    }

    // ------------------------------------------------------------------------
    // Inbound — listener decodes and pushes to events channel
    // ------------------------------------------------------------------------

    private val listener = object : WebSocketListener() {
        override fun onOpen(webSocket: WebSocket, response: Response) {
            Log.d(tag, "open")
            push(SignalingEvent.Connected)
        }

        override fun onMessage(webSocket: WebSocket, text: String) {
            val ev = decode(text) ?: return
            push(ev)
        }

        override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
            Log.d(tag, "closing $code $reason")
            webSocket.close(1000, null)
        }

        override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
            Log.d(tag, "closed $code $reason")
            push(SignalingEvent.ConnectionLost(reason))
            _events.close()
        }

        override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
            Log.w(tag, "failure: ${t.message}")
            push(SignalingEvent.ConnectionLost(t.message ?: "unknown failure"))
            _events.close()
        }
    }

    private fun push(ev: SignalingEvent) {
        // Use a coroutine so a slow consumer applies backpressure rather than
        // dropping ICE candidates on a non-blocking trySend.
        if (closed.get()) return
        scope.launch {
            runCatching { _events.send(ev) }
        }
    }

    private fun decode(raw: String): SignalingEvent? {
        val type = typeOf(raw) ?: run {
            Log.w(tag, "decode: missing type in $raw")
            return null
        }
        return try {
            when (type) {
                MessageType.PORTAL_CREATED -> SignalingEvent.Created(portalJson.decodeFromString<PortalCreated>(raw))
                MessageType.PORTAL_JOINED -> SignalingEvent.Joined(portalJson.decodeFromString<PortalJoined>(raw))
                MessageType.PORTAL_PEER_JOINED -> SignalingEvent.PeerJoined(portalJson.decodeFromString<PortalPeerJoined>(raw))
                MessageType.PORTAL_PEER_LEFT -> SignalingEvent.PeerLeft(portalJson.decodeFromString<PortalPeerLeft>(raw))
                MessageType.PORTAL_KICKED -> SignalingEvent.Kicked(portalJson.decodeFromString<PortalKicked>(raw))
                MessageType.PORTAL_LOCKED -> SignalingEvent.Locked(portalJson.decodeFromString<PortalLocked>(raw))
                MessageType.PORTAL_CLOSED -> SignalingEvent.Closed(portalJson.decodeFromString<PortalClosed>(raw))
                MessageType.WEBRTC_OFFER -> SignalingEvent.Offer(portalJson.decodeFromString<WebRTCOffer>(raw))
                MessageType.WEBRTC_ANSWER -> SignalingEvent.Answer(portalJson.decodeFromString<WebRTCAnswer>(raw))
                MessageType.WEBRTC_ICE -> SignalingEvent.Ice(portalJson.decodeFromString<WebRTCICE>(raw))
                MessageType.ERROR -> SignalingEvent.Error(portalJson.decodeFromString<ErrorMessage>(raw))
                else -> {
                    Log.w(tag, "decode: unknown type $type")
                    null
                }
            }
        } catch (e: Exception) {
            Log.w(tag, "decode: ${e.message} for type=$type", e)
            null
        }
    }
}

/** Discriminated union of every signaling event surfaced to consumers. */
sealed class SignalingEvent {
    data object Connected : SignalingEvent()
    data class ConnectionLost(val reason: String) : SignalingEvent()

    data class Created(val msg: PortalCreated) : SignalingEvent()
    data class Joined(val msg: PortalJoined) : SignalingEvent()
    data class PeerJoined(val msg: PortalPeerJoined) : SignalingEvent()
    data class PeerLeft(val msg: PortalPeerLeft) : SignalingEvent()
    data class Kicked(val msg: PortalKicked) : SignalingEvent()
    data class Locked(val msg: PortalLocked) : SignalingEvent()
    data class Closed(val msg: PortalClosed) : SignalingEvent()
    data class Offer(val msg: WebRTCOffer) : SignalingEvent()
    data class Answer(val msg: WebRTCAnswer) : SignalingEvent()
    data class Ice(val msg: WebRTCICE) : SignalingEvent()
    data class Error(val msg: ErrorMessage) : SignalingEvent()
}
