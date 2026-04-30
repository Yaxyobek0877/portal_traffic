package uz.aihealth.portal_mobile.protocol

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonPrimitive

// Wire format mirrored from ../../shared/protocol/messages.go.
// All JSON field names match the Go tags exactly. Adding optional fields
// is safe; renaming or removing fields breaks every other client.

object MessageType {
    // Client → Server
    const val PORTAL_CREATE = "portal.create"
    const val PORTAL_JOIN = "portal.join"
    const val PORTAL_JOIN_BY_NICK = "portal.join_by_nick"
    const val PORTAL_JOIN_RESPONSE = "portal.join_response"
    const val PORTAL_LEAVE = "portal.leave"
    const val PORTAL_KICK = "portal.kick"
    const val PORTAL_LOCK = "portal.lock"
    const val NICK_SET_VISIBILITY = "nick.set_visibility"

    // Server → Client
    const val PORTAL_CREATED = "portal.created"
    const val PORTAL_JOINED = "portal.joined"
    const val PORTAL_PEER_JOINED = "portal.peer_joined"
    const val PORTAL_PEER_LEFT = "portal.peer_left"
    const val PORTAL_JOIN_REQUEST = "portal.join_request"
    const val PORTAL_KICKED = "portal.kicked"
    const val PORTAL_LOCKED = "portal.locked"
    const val PORTAL_CLOSED = "portal.closed"
    const val ERROR = "error"

    // Bidirectional (server relays between two peers)
    const val WEBRTC_OFFER = "webrtc.offer"
    const val WEBRTC_ANSWER = "webrtc.answer"
    const val WEBRTC_ICE = "webrtc.ice"

    // P2P over the control data channel
    const val PING = "ping"
    const val PONG = "pong"
    const val PRESENCE = "presence"
    const val SERVICE_EXPOSE = "service.expose"
    const val SERVICE_UNEXPOSE = "service.unexpose"
    const val SERVICE_LIST_REQUEST = "service.list_request"
    const val SERVICE_LIST_RESPONSE = "service.list_response"
}

object ErrorCode {
    const val INVALID_MESSAGE = "INVALID_MESSAGE"
    const val PORTAL_NOT_FOUND = "PORTAL_NOT_FOUND"
    const val PORTAL_CODE_WRONG = "PORTAL_CODE_WRONG"
    const val PORTAL_FULL = "PORTAL_FULL"
    const val PORTAL_LOCKED = "PORTAL_LOCKED"
    const val NICKNAME_INVALID = "NICKNAME_INVALID"
    const val NICKNAME_TAKEN = "NICKNAME_TAKEN"
    const val NICKNAME_NOT_FOUND = "NICKNAME_NOT_FOUND"
    const val NICKNAME_NOT_PUBLIC = "NICKNAME_NOT_PUBLIC"
    const val NOT_IN_PORTAL = "NOT_IN_PORTAL"
    const val ALREADY_IN_PORTAL = "ALREADY_IN_PORTAL"
    const val PEER_NOT_FOUND = "PEER_NOT_FOUND"
    const val NOT_OWNER = "NOT_OWNER"
    const val RATE_LIMITED = "RATE_LIMITED"
    const val INTERNAL = "INTERNAL"
}

/** Single JSON instance shared by encode + decode. */
val portalJson: Json = Json {
    ignoreUnknownKeys = true
    encodeDefaults = true
    explicitNulls = false
}

@Serializable
data class PeerInfo(
    @SerialName("peer_id") val peerId: String,
    val nickname: String,
    @SerialName("virtual_ip") val virtualIp: String,
    @SerialName("is_owner") val isOwner: Boolean = false,
)

// ----------------------------------------------------------------------------
// Client → Server
// ----------------------------------------------------------------------------

@Serializable
data class PortalCreate(
    val type: String = MessageType.PORTAL_CREATE,
    val nickname: String,
    @SerialName("public_nick") val publicNick: Boolean = false,
    val capacity: Int = 0,
)

@Serializable
data class PortalJoin(
    val type: String = MessageType.PORTAL_JOIN,
    @SerialName("portal_id") val portalId: String,
    val code: String,
    val nickname: String,
)

@Serializable
data class PortalLeave(
    val type: String = MessageType.PORTAL_LEAVE,
)

@Serializable
data class PortalKick(
    val type: String = MessageType.PORTAL_KICK,
    @SerialName("peer_id") val peerId: String,
)

@Serializable
data class PortalLock(
    val type: String = MessageType.PORTAL_LOCK,
    val locked: Boolean,
)

// ----------------------------------------------------------------------------
// Server → Client
// ----------------------------------------------------------------------------

@Serializable
data class PortalCreated(
    val type: String,
    @SerialName("portal_id") val portalId: String,
    val code: String,
    @SerialName("peer_id") val peerId: String,
    @SerialName("virtual_ip") val virtualIp: String,
    val capacity: Int = 0,
)

@Serializable
data class PortalJoined(
    val type: String,
    @SerialName("portal_id") val portalId: String,
    @SerialName("peer_id") val peerId: String,
    @SerialName("virtual_ip") val virtualIp: String,
    val peers: List<PeerInfo> = emptyList(),
)

@Serializable
data class PortalPeerJoined(
    val type: String,
    @SerialName("peer_id") val peerId: String,
    val nickname: String,
    @SerialName("virtual_ip") val virtualIp: String,
)

@Serializable
data class PortalPeerLeft(
    val type: String,
    @SerialName("peer_id") val peerId: String,
    val reason: String? = null,
)

@Serializable
data class PortalKicked(
    val type: String,
    @SerialName("portal_id") val portalId: String,
)

@Serializable
data class PortalLocked(
    val type: String,
    val locked: Boolean,
)

@Serializable
data class PortalClosed(
    val type: String,
    @SerialName("portal_id") val portalId: String,
    val reason: String? = null,
)

@Serializable
data class ErrorMessage(
    val type: String,
    val code: String,
    val message: String,
    @SerialName("request_id") val requestId: String? = null,
)

// ----------------------------------------------------------------------------
// WebRTC handshake — bidirectional, server relays
// ----------------------------------------------------------------------------

@Serializable
data class WebRTCOffer(
    val type: String = MessageType.WEBRTC_OFFER,
    val from: String? = null,
    val to: String,
    val sdp: String,
)

@Serializable
data class WebRTCAnswer(
    val type: String = MessageType.WEBRTC_ANSWER,
    val from: String? = null,
    val to: String,
    val sdp: String,
)

@Serializable
data class WebRTCICE(
    val type: String = MessageType.WEBRTC_ICE,
    val from: String? = null,
    val to: String,
    val candidate: String,
    @SerialName("sdp_mid") val sdpMid: String? = null,
    @SerialName("sdp_mline_index") val sdpMLineIndex: Int? = null,
)

// ----------------------------------------------------------------------------
// P2P data-channel messages (control channel)
// ----------------------------------------------------------------------------

@Serializable
data class Ping(
    val type: String = MessageType.PING,
    val ts: Long,
)

@Serializable
data class Pong(
    val type: String = MessageType.PONG,
    val ts: Long,
    @SerialName("echo_ts") val echoTs: Long,
)

@Serializable
data class ServiceExpose(
    val type: String = MessageType.SERVICE_EXPOSE,
    val name: String,
    val protocol: String,
    val port: Int,
)

@Serializable
data class ServiceUnexpose(
    val type: String = MessageType.SERVICE_UNEXPOSE,
    val port: Int,
)

// ----------------------------------------------------------------------------
// Type extraction helper
// ----------------------------------------------------------------------------

/** Reads only the "type" field from a frame — same role as protocol.TypeOf in Go. */
fun typeOf(raw: String): String? {
    return try {
        val obj = portalJson.parseToJsonElement(raw) as? JsonObject ?: return null
        obj["type"]?.jsonPrimitive?.content
    } catch (_: Exception) {
        null
    }
}
