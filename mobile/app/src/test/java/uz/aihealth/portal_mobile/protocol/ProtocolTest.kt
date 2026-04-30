package uz.aihealth.portal_mobile.protocol

import kotlinx.serialization.encodeToString
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/** Pin the wire format. These tests catch accidental field-name drift —
 * if any of them fails, mobile peers will silently fail to interop with
 * desktop peers (the signal server speaks Go's snake_case JSON tags). */
class ProtocolTest {

    @Test fun `decode portal_created from server`() {
        val raw = """{"type":"portal.created","portal_id":"428591","code":"739204","peer_id":"abc-123","virtual_ip":"10.42.0.1","capacity":16}"""
        val msg = portalJson.decodeFromString<PortalCreated>(raw)
        assertEquals("portal.created", msg.type)
        assertEquals("428591", msg.portalId)
        assertEquals("739204", msg.code)
        assertEquals("abc-123", msg.peerId)
        assertEquals("10.42.0.1", msg.virtualIp)
        assertEquals(16, msg.capacity)
    }

    @Test fun `decode portal_joined with peers list`() {
        val raw = """{"type":"portal.joined","portal_id":"x","peer_id":"me","virtual_ip":"10.42.0.2","peers":[{"peer_id":"alice","nickname":"Alice","virtual_ip":"10.42.0.1","is_owner":true}]}"""
        val msg = portalJson.decodeFromString<PortalJoined>(raw)
        assertEquals(1, msg.peers.size)
        assertEquals("alice", msg.peers[0].peerId)
        assertTrue(msg.peers[0].isOwner)
    }

    @Test fun `decode portal_joined with absent peers list works`() {
        val raw = """{"type":"portal.joined","portal_id":"x","peer_id":"me","virtual_ip":"10.42.0.2"}"""
        val msg = portalJson.decodeFromString<PortalJoined>(raw)
        assertTrue(msg.peers.isEmpty())
    }

    @Test fun `encode portal_create uses snake_case nickname and public_nick`() {
        val msg = PortalCreate(nickname = "alice", publicNick = false)
        val json = portalJson.encodeToString(msg)
        assertTrue("type field present", json.contains("\"type\":\"portal.create\""))
        assertTrue("nickname field present", json.contains("\"nickname\":\"alice\""))
        assertTrue("public_nick field present", json.contains("\"public_nick\":false"))
    }

    @Test fun `encode webrtc_ice with optional fields uses snake_case`() {
        val msg = WebRTCICE(to = "peer-x", candidate = "candidate:foo", sdpMid = "0", sdpMLineIndex = 0)
        val json = portalJson.encodeToString(msg)
        assertTrue(json.contains("\"sdp_mid\":\"0\""))
        assertTrue(json.contains("\"sdp_mline_index\":0"))
    }

    @Test fun `decode webrtc_ice with optional fields omitted`() {
        val raw = """{"type":"webrtc.ice","from":"a","to":"b","candidate":"candidate:bar"}"""
        val msg = portalJson.decodeFromString<WebRTCICE>(raw)
        assertEquals("a", msg.from)
        assertEquals("b", msg.to)
        assertEquals("candidate:bar", msg.candidate)
        assertNull(msg.sdpMid)
        assertNull(msg.sdpMLineIndex)
    }

    @Test fun `decode error message`() {
        val raw = """{"type":"error","code":"PORTAL_CODE_WRONG","message":"kod noto'g'ri"}"""
        val msg = portalJson.decodeFromString<ErrorMessage>(raw)
        assertEquals(ErrorCode.PORTAL_CODE_WRONG, msg.code)
        assertEquals("kod noto'g'ri", msg.message)
    }

    @Test fun `typeOf extracts type from valid JSON`() {
        assertEquals("portal.joined", typeOf("""{"type":"portal.joined","x":1}"""))
        assertEquals("webrtc.offer", typeOf("""{"sdp":"v=0","type":"webrtc.offer"}"""))
    }

    @Test fun `typeOf returns null for malformed input`() {
        assertNull(typeOf(""))
        assertNull(typeOf("not json"))
        assertNull(typeOf("[]"))           // array, not object
        assertNull(typeOf("""{"x":1}"""))  // missing type
    }

    @Test fun `unknown server fields are tolerated`() {
        // The signal server may add fields in a future revision; we must
        // not refuse to decode known messages just because they have
        // additional keys.
        val raw = """{"type":"portal.created","portal_id":"x","code":"y","peer_id":"z","virtual_ip":"10.42.0.1","capacity":16,"some_future_field":"hi"}"""
        val msg = portalJson.decodeFromString<PortalCreated>(raw)
        assertEquals("x", msg.portalId)
    }

    @Test fun `peer_info round trip`() {
        val original = PeerInfo(peerId = "p1", nickname = "Alice", virtualIp = "10.42.0.5", isOwner = true)
        val json = portalJson.encodeToString(original)
        val decoded = portalJson.decodeFromString<PeerInfo>(json)
        assertEquals(original, decoded)
    }
}
