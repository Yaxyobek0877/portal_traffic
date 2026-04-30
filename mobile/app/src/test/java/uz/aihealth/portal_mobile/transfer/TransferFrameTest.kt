package uz.aihealth.portal_mobile.transfer

import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

/** The transfer wire format must stay byte-compatible with
 * ../../../../../../../client/transfer/transfer.go. These tests pin the
 * byte layout. */
class TransferFrameTest {

    @Test fun `encode produces 11-byte header layout`() {
        val out = encodeTransferFrame(
            type = FRAME_START,
            xferId = 0x0102030405060708L,
            header = byteArrayOf(0xAA.toByte(), 0xBB.toByte()),
            body = byteArrayOf(),
        )
        // type
        assertEquals(0x10.toByte(), out[0])
        // xfer_id (8 BE)
        assertEquals(0x01.toByte(), out[1])
        assertEquals(0x02.toByte(), out[2])
        assertEquals(0x03.toByte(), out[3])
        assertEquals(0x04.toByte(), out[4])
        assertEquals(0x05.toByte(), out[5])
        assertEquals(0x06.toByte(), out[6])
        assertEquals(0x07.toByte(), out[7])
        assertEquals(0x08.toByte(), out[8])
        // header_len (2 BE) = 2
        assertEquals(0x00.toByte(), out[9])
        assertEquals(0x02.toByte(), out[10])
        // header
        assertEquals(0xAA.toByte(), out[11])
        assertEquals(0xBB.toByte(), out[12])
        assertEquals(13, out.size)
    }

    @Test fun `START frame round trip`() {
        val header = """{"name":"hello.txt","size":5}""".toByteArray()
        val encoded = encodeTransferFrame(FRAME_START, 1L, header, byteArrayOf())
        val decoded = decodeTransferFrame(encoded)!!
        assertEquals(FRAME_START, decoded.type)
        assertEquals(1L, decoded.xferId)
        assertArrayEquals(header, decoded.header)
        assertEquals(0, decoded.body.size)
    }

    @Test fun `CHUNK frame round trip`() {
        val body = ByteArray(16384) { (it and 0xFF).toByte() }
        val encoded = encodeTransferFrame(FRAME_CHUNK, 42L, byteArrayOf(), body)
        val decoded = decodeTransferFrame(encoded)!!
        assertEquals(FRAME_CHUNK, decoded.type)
        assertEquals(42L, decoded.xferId)
        assertEquals(0, decoded.header.size)
        assertArrayEquals(body, decoded.body)
    }

    @Test fun `END frame is just the 11-byte header`() {
        val encoded = encodeTransferFrame(FRAME_END, 7L, byteArrayOf(), byteArrayOf())
        assertEquals(11, encoded.size)
        val decoded = decodeTransferFrame(encoded)!!
        assertEquals(FRAME_END, decoded.type)
        assertEquals(7L, decoded.xferId)
    }

    @Test fun `ABORT frame body carries reason text`() {
        val reason = "cancelled by sender".toByteArray()
        val encoded = encodeTransferFrame(FRAME_ABORT, 99L, byteArrayOf(), reason)
        val decoded = decodeTransferFrame(encoded)!!
        assertEquals(FRAME_ABORT, decoded.type)
        assertArrayEquals(reason, decoded.body)
    }

    @Test fun `decode returns null on truncated input`() {
        assertNull(decodeTransferFrame(byteArrayOf()))
        assertNull(decodeTransferFrame(byteArrayOf(0x10, 0x00, 0x00)))      // <11 bytes
        // Header_len says 5 but only 2 follow → body underrun guarded
        val bad = byteArrayOf(
            0x10, 0, 0, 0, 0, 0, 0, 0, 1, // type + xfer_id = 1
            0, 0x05,                       // header_len = 5
            0xAA.toByte(), 0xBB.toByte(),  // only 2 bytes
        )
        assertNull(decodeTransferFrame(bad))
    }

    @Test fun `large xferId stays big-endian`() {
        val xfer = Long.MAX_VALUE
        val encoded = encodeTransferFrame(FRAME_END, xfer, byteArrayOf(), byteArrayOf())
        val decoded = decodeTransferFrame(encoded)!!
        assertEquals(xfer, decoded.xferId)
    }
}
