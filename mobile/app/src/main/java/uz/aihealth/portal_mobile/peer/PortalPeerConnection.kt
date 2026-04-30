package uz.aihealth.portal_mobile.peer

import android.content.Context
import android.util.Log
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.channels.BufferOverflow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import org.webrtc.DataChannel
import org.webrtc.IceCandidate
import org.webrtc.MediaConstraints
import org.webrtc.MediaStream
import org.webrtc.PeerConnection
import org.webrtc.PeerConnectionFactory
import org.webrtc.RtpReceiver
import org.webrtc.SdpObserver
import org.webrtc.SessionDescription
import java.nio.ByteBuffer
import java.util.concurrent.atomic.AtomicBoolean
import kotlin.coroutines.resume
import kotlinx.coroutines.suspendCancellableCoroutine

// Mirror of client/peer/connection.go: one webrtc.PeerConnection plus the
// four multiplexed data channels Portal uses (control, chat, transfer, proxy).
//
// Channel IDs and reliability profiles MUST match the Go reference or
// peers from different clients won't be able to talk:
//   control  id=1  ordered  reliable
//   chat     id=2  ordered  reliable
//   transfer id=3  ordered  reliable
//   proxy    id=4  unordered  maxRetransmits=0  (datagram-ish)

const val CHAN_CONTROL = "control"
const val CHAN_CHAT = "chat"
const val CHAN_TRANSFER = "transfer"
const val CHAN_PROXY = "proxy"

private data class ChannelDef(
    val label: String,
    val id: Int,
    val ordered: Boolean,
    val maxRetransmits: Int, // -1 = reliable (no cap)
)

private val channelDefs = listOf(
    ChannelDef(CHAN_CONTROL, 1, ordered = true, maxRetransmits = -1),
    ChannelDef(CHAN_CHAT, 2, ordered = true, maxRetransmits = -1),
    ChannelDef(CHAN_TRANSFER, 3, ordered = true, maxRetransmits = -1),
    ChannelDef(CHAN_PROXY, 4, ordered = false, maxRetransmits = 0),
)

enum class Role { OFFERER, ANSWERER }

enum class PeerState { CONNECTING, CONNECTED, FAILED, CLOSED }

/** A decoded inbound payload. [text] is true if the frame was a JS string. */
data class PeerMessage(
    val channel: String,
    val text: Boolean,
    val data: ByteArray,
)

/**
 * One peer-to-peer relationship. Owns a single PeerConnection plus the
 * four negotiated data channels.
 *
 * Lifecycle: construct, then call [createOffer] (offerer) or [handleOffer]
 * (answerer). Forward [localIce] candidates to the remote via signaling;
 * feed received remote candidates with [addRemoteIce]. Listen on [messages]
 * for application data, on [state] for status. Call [close] when done.
 */
class PortalPeerConnection(
    factory: PeerConnectionFactory,
    val localPeerId: String,
    val remotePeerId: String,
    val role: Role,
    iceServers: List<PeerConnection.IceServer>,
) {
    private val tag = "Peer/$remotePeerId"

    private val _localIce = MutableSharedFlow<IceCandidate>(
        replay = 0,
        extraBufferCapacity = 64,
        onBufferOverflow = BufferOverflow.SUSPEND,
    )
    val localIce: SharedFlow<IceCandidate> = _localIce.asSharedFlow()

    private val _messages = MutableSharedFlow<PeerMessage>(
        replay = 0,
        extraBufferCapacity = 128,
        onBufferOverflow = BufferOverflow.SUSPEND,
    )
    val messages: SharedFlow<PeerMessage> = _messages.asSharedFlow()

    private val _state = MutableStateFlow(PeerState.CONNECTING)
    val state: StateFlow<PeerState> = _state.asStateFlow()

    private val channels = mutableMapOf<String, DataChannel>()
    private val pendingRemoteIce = mutableListOf<IceCandidate>()
    private val closed = AtomicBoolean(false)

    private val pc: PeerConnection

    init {
        val rtcConfig = PeerConnection.RTCConfiguration(iceServers).apply {
            sdpSemantics = PeerConnection.SdpSemantics.UNIFIED_PLAN
            bundlePolicy = PeerConnection.BundlePolicy.MAXBUNDLE
            rtcpMuxPolicy = PeerConnection.RtcpMuxPolicy.REQUIRE
        }
        pc = factory.createPeerConnection(rtcConfig, observer)
            ?: error("Peer: createPeerConnection returned null")

        // Pre-create all four channels with negotiated IDs; both sides do
        // this and pion/Android dedupes via the ID. Without negotiation,
        // the answerer would have to wait for ondatachannel.
        for (def in channelDefs) {
            val init = DataChannel.Init().apply {
                ordered = def.ordered
                maxRetransmits = def.maxRetransmits
                negotiated = true
                id = def.id
            }
            val dc = pc.createDataChannel(def.label, init)
                ?: error("Peer: createDataChannel(${def.label}) failed")
            attachChannel(dc)
        }
    }

    // ------------------------------------------------------------------------
    // PeerConnection.Observer
    // ------------------------------------------------------------------------

    private val observer = object : PeerConnection.Observer {
        override fun onIceCandidate(candidate: IceCandidate) {
            if (closed.get()) return
            _localIce.tryEmit(candidate)
        }

        override fun onConnectionChange(newState: PeerConnection.PeerConnectionState) {
            Log.d(tag, "pc state $newState")
            val translated = when (newState) {
                PeerConnection.PeerConnectionState.CONNECTED -> PeerState.CONNECTED
                PeerConnection.PeerConnectionState.FAILED -> PeerState.FAILED
                PeerConnection.PeerConnectionState.CLOSED,
                PeerConnection.PeerConnectionState.DISCONNECTED -> PeerState.CLOSED
                else -> PeerState.CONNECTING
            }
            _state.value = translated
            if (translated == PeerState.CLOSED || translated == PeerState.FAILED) {
                close()
            }
        }

        override fun onDataChannel(dc: DataChannel) {
            // We pre-create all channels with negotiated IDs, so this
            // shouldn't fire for our channels. If it does, attach anyway.
            Log.d(tag, "remote-initiated data channel: ${dc.label()}")
            attachChannel(dc)
        }

        // Unused but required by the interface.
        override fun onSignalingChange(p0: PeerConnection.SignalingState?) {}
        override fun onIceConnectionChange(p0: PeerConnection.IceConnectionState?) {}
        override fun onIceConnectionReceivingChange(p0: Boolean) {}
        override fun onIceGatheringChange(p0: PeerConnection.IceGatheringState?) {}
        override fun onIceCandidatesRemoved(p0: Array<out IceCandidate>?) {}
        override fun onAddStream(p0: MediaStream?) {}
        override fun onRemoveStream(p0: MediaStream?) {}
        override fun onRenegotiationNeeded() {}
        override fun onAddTrack(p0: RtpReceiver?, p1: Array<out MediaStream>?) {}
    }

    private fun attachChannel(dc: DataChannel) {
        val label = dc.label()
        synchronized(channels) {
            channels[label] = dc
        }
        dc.registerObserver(object : DataChannel.Observer {
            override fun onMessage(buffer: DataChannel.Buffer) {
                val data = ByteArray(buffer.data.remaining()).also { buffer.data.get(it) }
                if (closed.get()) return
                _messages.tryEmit(PeerMessage(label, !buffer.binary, data))
            }
            override fun onStateChange() {
                Log.d(tag, "channel $label state ${dc.state()}")
            }
            override fun onBufferedAmountChange(prev: Long) {}
        })
    }

    // ------------------------------------------------------------------------
    // Handshake API
    // ------------------------------------------------------------------------

    suspend fun createOffer(): String {
        check(role == Role.OFFERER) { "createOffer: not offerer" }
        val sdp = suspendCancellableCoroutine<SessionDescription> { cont ->
            pc.createOffer(object : SdpObserver {
                override fun onCreateSuccess(s: SessionDescription) { cont.resume(s) }
                override fun onCreateFailure(err: String) {
                    cont.cancel(IllegalStateException("createOffer: $err"))
                }
                override fun onSetSuccess() {}
                override fun onSetFailure(err: String) {}
            }, MediaConstraints())
        }
        setLocalDescription(sdp)
        return sdp.description
    }

    suspend fun handleOffer(remoteSdp: String): String {
        check(role == Role.ANSWERER) { "handleOffer: not answerer" }
        setRemoteDescription(SessionDescription(SessionDescription.Type.OFFER, remoteSdp))
        flushPendingRemoteIce()
        val answer = suspendCancellableCoroutine<SessionDescription> { cont ->
            pc.createAnswer(object : SdpObserver {
                override fun onCreateSuccess(s: SessionDescription) { cont.resume(s) }
                override fun onCreateFailure(err: String) {
                    cont.cancel(IllegalStateException("createAnswer: $err"))
                }
                override fun onSetSuccess() {}
                override fun onSetFailure(err: String) {}
            }, MediaConstraints())
        }
        setLocalDescription(answer)
        return answer.description
    }

    suspend fun handleAnswer(remoteSdp: String) {
        check(role == Role.OFFERER) { "handleAnswer: not offerer" }
        setRemoteDescription(SessionDescription(SessionDescription.Type.ANSWER, remoteSdp))
        flushPendingRemoteIce()
    }

    /**
     * Add an ICE candidate received from the remote. If the remote
     * description hasn't been set yet, the candidate is buffered.
     */
    fun addRemoteIce(candidate: String, sdpMid: String?, sdpMLineIndex: Int?) {
        val ice = IceCandidate(sdpMid ?: "", sdpMLineIndex ?: 0, candidate)
        synchronized(pendingRemoteIce) {
            if (pc.remoteDescription == null) {
                pendingRemoteIce.add(ice)
                return
            }
        }
        pc.addIceCandidate(ice)
    }

    private fun flushPendingRemoteIce() {
        val pending = synchronized(pendingRemoteIce) {
            val copy = pendingRemoteIce.toList()
            pendingRemoteIce.clear()
            copy
        }
        for (ice in pending) {
            pc.addIceCandidate(ice)
        }
    }

    private suspend fun setLocalDescription(sdp: SessionDescription) {
        val deferred = CompletableDeferred<Unit>()
        pc.setLocalDescription(object : SdpObserver {
            override fun onSetSuccess() { deferred.complete(Unit) }
            override fun onSetFailure(err: String) {
                deferred.completeExceptionally(IllegalStateException("setLocal: $err"))
            }
            override fun onCreateSuccess(p0: SessionDescription) {}
            override fun onCreateFailure(p0: String) {}
        }, sdp)
        deferred.await()
    }

    private suspend fun setRemoteDescription(sdp: SessionDescription) {
        val deferred = CompletableDeferred<Unit>()
        pc.setRemoteDescription(object : SdpObserver {
            override fun onSetSuccess() { deferred.complete(Unit) }
            override fun onSetFailure(err: String) {
                deferred.completeExceptionally(IllegalStateException("setRemote: $err"))
            }
            override fun onCreateSuccess(p0: SessionDescription) {}
            override fun onCreateFailure(p0: String) {}
        }, sdp)
        deferred.await()
    }

    // ------------------------------------------------------------------------
    // Send + status
    // ------------------------------------------------------------------------

    fun sendText(channel: String, text: String): Boolean {
        val dc = synchronized(channels) { channels[channel] } ?: return false
        if (dc.state() != DataChannel.State.OPEN) return false
        val buf = ByteBuffer.wrap(text.toByteArray(Charsets.UTF_8))
        return dc.send(DataChannel.Buffer(buf, false))
    }

    fun sendBinary(channel: String, data: ByteArray): Boolean {
        val dc = synchronized(channels) { channels[channel] } ?: return false
        if (dc.state() != DataChannel.State.OPEN) return false
        return dc.send(DataChannel.Buffer(ByteBuffer.wrap(data), true))
    }

    fun channelOpen(name: String): Boolean {
        val dc = synchronized(channels) { channels[name] } ?: return false
        return dc.state() == DataChannel.State.OPEN
    }

    fun allChannelsOpen(): Boolean = channelDefs.all { channelOpen(it.label) }

    // ------------------------------------------------------------------------
    // Lifecycle
    // ------------------------------------------------------------------------

    fun close() {
        if (!closed.compareAndSet(false, true)) return
        synchronized(channels) {
            channels.values.forEach { runCatching { it.close() } }
            channels.clear()
        }
        runCatching { pc.close() }
        _state.value = PeerState.CLOSED
    }
}

// ----------------------------------------------------------------------------
// Factory bootstrap — call once per process.
// ----------------------------------------------------------------------------

object WebRtcFactory {
    @Volatile private var factory: PeerConnectionFactory? = null

    fun get(context: Context): PeerConnectionFactory {
        factory?.let { return it }
        synchronized(this) {
            factory?.let { return it }
            PeerConnectionFactory.initialize(
                PeerConnectionFactory.InitializationOptions
                    .builder(context.applicationContext)
                    .createInitializationOptions()
            )
            val f = PeerConnectionFactory.builder().createPeerConnectionFactory()
            factory = f
            return f
        }
    }
}

/** Default ICE servers — public STUN. TURN is left to deploy-time config. */
val defaultIceServers: List<PeerConnection.IceServer> = listOf(
    PeerConnection.IceServer.builder("stun:stun.l.google.com:19302").createIceServer(),
    PeerConnection.IceServer.builder("stun:stun.cloudflare.com:3478").createIceServer(),
)
