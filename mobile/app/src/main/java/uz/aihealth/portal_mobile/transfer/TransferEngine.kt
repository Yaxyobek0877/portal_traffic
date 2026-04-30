package uz.aihealth.portal_mobile.transfer

import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import uz.aihealth.portal_mobile.protocol.portalJson
import java.io.File
import java.io.FileOutputStream
import java.io.InputStream
import java.io.RandomAccessFile
import java.nio.ByteBuffer
import java.nio.ByteOrder
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicLong

// Wire format mirrors ../../../../client/transfer/transfer.go:
//
//   | type (1) | xfer_id (8 BE) | header_len (2 BE) | header_json | body... |
//
// Frame types:
//   0x10 START — header = Manifest{name, size, mime}
//   0x11 CHUNK — body = file bytes
//   0x12 END
//   0x13 ABORT — body = reason string
//
// Each frame is sealed with the portal-derived secretbox key by the
// MeshManager (see [MeshManager.sendTransferFrame] / [MeshManager.handleTransfer]).
// The bytes that hit the data channel are nonce(24) || ciphertext(...).

internal const val FRAME_START: Byte = 0x10
internal const val FRAME_CHUNK: Byte = 0x11
internal const val FRAME_END: Byte = 0x12
internal const val FRAME_ABORT: Byte = 0x13

/** WebRTC data channels cap at ~64 KiB per message; stay well under to
 * leave room for the secretbox envelope and SCTP framing. */
private const val CHUNK_SIZE = 16 * 1024

/** Wire-format frame as it appears on the transfer data channel (after
 * the secretbox envelope is opened). Made internal so unit tests in the
 * same module can verify round-tripping. */
internal data class TransferFrame(
    val type: Byte,
    val xferId: Long,
    val header: ByteArray,
    val body: ByteArray,
) {
    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (other !is TransferFrame) return false
        return type == other.type &&
            xferId == other.xferId &&
            header.contentEquals(other.header) &&
            body.contentEquals(other.body)
    }
    override fun hashCode(): Int {
        var result = type.toInt()
        result = 31 * result + xferId.hashCode()
        result = 31 * result + header.contentHashCode()
        result = 31 * result + body.contentHashCode()
        return result
    }
}

internal fun encodeTransferFrame(type: Byte, xferId: Long, header: ByteArray, body: ByteArray): ByteArray {
    require(header.size <= 0xFFFF) { "transfer: header too large" }
    val out = ByteArray(11 + header.size + body.size)
    val bb = ByteBuffer.wrap(out).order(ByteOrder.BIG_ENDIAN)
    bb.put(0, type)
    bb.putLong(1, xferId)
    bb.putShort(9, header.size.toShort())
    System.arraycopy(header, 0, out, 11, header.size)
    System.arraycopy(body, 0, out, 11 + header.size, body.size)
    return out
}

internal fun decodeTransferFrame(payload: ByteArray): TransferFrame? {
    if (payload.size < 11) return null
    val bb = ByteBuffer.wrap(payload).order(ByteOrder.BIG_ENDIAN)
    val type = bb.get(0)
    val xferId = bb.getLong(1)
    val hdrLen = bb.getShort(9).toInt() and 0xFFFF
    if (11 + hdrLen > payload.size) return null
    val header = payload.copyOfRange(11, 11 + hdrLen)
    val body = payload.copyOfRange(11 + hdrLen, payload.size)
    return TransferFrame(type, xferId, header, body)
}

private const val TAG = "Transfer"

@Serializable
data class TransferManifest(
    val name: String,
    val size: Long,
    val mime: String? = null,
)

enum class Direction { SEND, RECV }

data class FileTransfer(
    val xferId: Long,
    val peerId: String,
    val peerNickname: String,
    val direction: Direction,
    val manifest: TransferManifest,
    val bytes: Long,
    val done: Boolean,
    val error: String? = null,
    val savePath: String? = null,
    val startedMs: Long,
    val updatedMs: Long,
)

/** What the engine needs from the MeshManager — kept narrow to avoid a
 * circular dependency. */
interface TransferSink {
    fun sendTransferFrame(peerId: String, payload: ByteArray): Boolean

    /** Used for nicer UI labels — falls back to peerId.take(8) if unknown. */
    fun nicknameOf(peerId: String): String
}

class TransferEngine(
    private val saveDir: File,
    private val mesh: TransferSink,
    private val scope: CoroutineScope,
) {
    private val nextId = AtomicLong(0)
    private val sends = ConcurrentHashMap<Pair<String, Long>, Job>()
    private val recvs = ConcurrentHashMap<Pair<String, Long>, RecvState>()

    private val _transfers = MutableStateFlow<List<FileTransfer>>(emptyList())
    val transfers: StateFlow<List<FileTransfer>> = _transfers.asStateFlow()

    init {
        saveDir.mkdirs()
    }

    // ------------------------------------------------------------------------
    // Send side
    // ------------------------------------------------------------------------

    /**
     * Stream `input` to `peerId`. `input` is closed when the transfer ends
     * or aborts. Returns the assigned xfer-id so a later UI button can
     * cancel it; null means we couldn't even kick off (no peer / channel
     * not open).
     */
    fun sendFile(
        peerId: String,
        manifest: TransferManifest,
        input: InputStream,
    ): Long? {
        val xferId = nextId.incrementAndGet()
        val key = peerId to xferId

        val header = portalJson.encodeToString(manifest).toByteArray(Charsets.UTF_8)
        if (!sendFrame(peerId, FRAME_START, xferId, header, ByteArray(0))) {
            runCatching { input.close() }
            return null
        }

        publish(
            FileTransfer(
                xferId = xferId,
                peerId = peerId,
                peerNickname = mesh.nicknameOf(peerId),
                direction = Direction.SEND,
                manifest = manifest,
                bytes = 0,
                done = false,
                startedMs = System.currentTimeMillis(),
                updatedMs = System.currentTimeMillis(),
            ),
        )

        val job = scope.launch(Dispatchers.IO) {
            pumpSend(peerId, xferId, manifest, input)
        }
        sends[key] = job
        return xferId
    }

    private suspend fun pumpSend(
        peerId: String,
        xferId: Long,
        manifest: TransferManifest,
        input: InputStream,
    ) = withContext(Dispatchers.IO) {
        val key = peerId to xferId
        var sent = 0L
        val buf = ByteArray(CHUNK_SIZE)
        try {
            while (true) {
                val n = input.read(buf)
                if (n < 0) break
                if (n == 0) continue
                val ok = sendFrame(peerId, FRAME_CHUNK, xferId, ByteArray(0), buf.copyOf(n))
                if (!ok) {
                    failSend(xferId, peerId, manifest, sent, "data channel closed")
                    return@withContext
                }
                sent += n
                publishUpdate(xferId, peerId) { it.copy(bytes = sent, updatedMs = System.currentTimeMillis()) }
            }
            sendFrame(peerId, FRAME_END, xferId, ByteArray(0), ByteArray(0))
            publishUpdate(xferId, peerId) {
                it.copy(bytes = sent, done = true, updatedMs = System.currentTimeMillis())
            }
        } catch (e: Exception) {
            sendFrame(
                peerId,
                FRAME_ABORT,
                xferId,
                ByteArray(0),
                (e.message ?: "io error").toByteArray(Charsets.UTF_8),
            )
            failSend(xferId, peerId, manifest, sent, e.message ?: "io error")
        } finally {
            runCatching { input.close() }
            sends.remove(key)
        }
    }

    private fun failSend(
        xferId: Long,
        peerId: String,
        manifest: TransferManifest,
        bytes: Long,
        reason: String,
    ) {
        publishUpdate(xferId, peerId) {
            it.copy(bytes = bytes, done = true, error = reason, updatedMs = System.currentTimeMillis())
        }
        Log.w(TAG, "send xfer=$xferId to $peerId failed: $reason")
    }

    fun cancelSend(peerId: String, xferId: Long) {
        val key = peerId to xferId
        sends[key]?.cancel()
        sendFrame(peerId, FRAME_ABORT, xferId, ByteArray(0), "cancelled".toByteArray(Charsets.UTF_8))
        sends.remove(key)
    }

    // ------------------------------------------------------------------------
    // Receive side
    // ------------------------------------------------------------------------

    private inner class RecvState(
        val peerId: String,
        val xferId: Long,
        val manifest: TransferManifest,
        val tempFile: File,
        val raf: RandomAccessFile,
        val startedMs: Long,
    ) {
        @Volatile var bytes: Long = 0L
    }

    /** Called by [uz.aihealth.portal_mobile.mesh.MeshManager] for every
     * inbound frame on the transfer channel — payload is already decrypted. */
    fun handleFrame(peerId: String, payload: ByteArray) {
        val frame = decodeTransferFrame(payload) ?: return
        when (frame.type) {
            FRAME_START -> onStart(peerId, frame.xferId, frame.header)
            FRAME_CHUNK -> onChunk(peerId, frame.xferId, frame.body)
            FRAME_END -> onEnd(peerId, frame.xferId)
            FRAME_ABORT -> onAbort(peerId, frame.xferId, frame.body)
        }
    }

    private fun onStart(peerId: String, xferId: Long, headerBytes: ByteArray) {
        val manifest = runCatching {
            portalJson.decodeFromString<TransferManifest>(String(headerBytes, Charsets.UTF_8))
        }.getOrNull() ?: return
        val safe = safeName(manifest.name)
        val temp = uniqueFile(File(saveDir, "$safe.part"))
        val raf = try {
            RandomAccessFile(temp, "rw")
        } catch (e: Exception) {
            Log.w(TAG, "can't create $temp: ${e.message}")
            return
        }
        val st = RecvState(peerId, xferId, manifest, temp, raf, System.currentTimeMillis())
        recvs[peerId to xferId] = st
        publish(
            FileTransfer(
                xferId = xferId,
                peerId = peerId,
                peerNickname = mesh.nicknameOf(peerId),
                direction = Direction.RECV,
                manifest = manifest,
                bytes = 0,
                done = false,
                startedMs = st.startedMs,
                updatedMs = st.startedMs,
            ),
        )
    }

    private fun onChunk(peerId: String, xferId: Long, body: ByteArray) {
        val st = recvs[peerId to xferId] ?: return
        try {
            st.raf.write(body)
            st.bytes += body.size
            publishUpdate(xferId, peerId) {
                it.copy(bytes = st.bytes, updatedMs = System.currentTimeMillis())
            }
        } catch (e: Exception) {
            Log.w(TAG, "write chunk $xferId from $peerId: ${e.message}")
            cleanupRecv(peerId, xferId, error = e.message ?: "write failed")
        }
    }

    private fun onEnd(peerId: String, xferId: Long) {
        val st = recvs.remove(peerId to xferId) ?: return
        runCatching { st.raf.close() }
        val finalName = uniqueFile(File(saveDir, safeName(st.manifest.name)))
        val moved = st.tempFile.renameTo(finalName)
        if (!moved) {
            // Fallback: copy bytes manually.
            try {
                st.tempFile.inputStream().use { ins ->
                    FileOutputStream(finalName).use { out -> ins.copyTo(out) }
                }
                st.tempFile.delete()
            } catch (e: Exception) {
                Log.w(TAG, "rename/copy failed: ${e.message}")
                publishUpdate(xferId, peerId) {
                    it.copy(
                        bytes = st.bytes,
                        done = true,
                        error = e.message ?: "save failed",
                        updatedMs = System.currentTimeMillis(),
                    )
                }
                return
            }
        }
        publishUpdate(xferId, peerId) {
            it.copy(
                bytes = st.bytes,
                done = true,
                savePath = finalName.absolutePath,
                updatedMs = System.currentTimeMillis(),
            )
        }
    }

    private fun onAbort(peerId: String, xferId: Long, reasonBytes: ByteArray) {
        cleanupRecv(peerId, xferId, error = String(reasonBytes, Charsets.UTF_8))
    }

    private fun cleanupRecv(peerId: String, xferId: Long, error: String) {
        val st = recvs.remove(peerId to xferId) ?: return
        runCatching { st.raf.close() }
        runCatching { st.tempFile.delete() }
        publishUpdate(xferId, peerId) {
            it.copy(
                bytes = st.bytes,
                done = true,
                error = error,
                updatedMs = System.currentTimeMillis(),
            )
        }
    }

    // Send a single wire-format frame. Encoding lives at the top level
    // of this file so the unit tests can verify round-tripping without
    // a full Engine instance.
    private fun sendFrame(peerId: String, type: Byte, xferId: Long, header: ByteArray, body: ByteArray): Boolean {
        return mesh.sendTransferFrame(peerId, encodeTransferFrame(type, xferId, header, body))
    }

    // ------------------------------------------------------------------------
    // UI state plumbing
    // ------------------------------------------------------------------------

    private fun publish(t: FileTransfer) {
        _transfers.update { current ->
            val without = current.filterNot { it.xferId == t.xferId && it.peerId == t.peerId }
            without + t
        }
    }

    private inline fun publishUpdate(xferId: Long, peerId: String, transform: (FileTransfer) -> FileTransfer) {
        _transfers.update { current ->
            current.map { if (it.xferId == xferId && it.peerId == peerId) transform(it) else it }
        }
    }
}

private fun uniqueFile(target: File): File {
    if (!target.exists()) return target
    val dir = target.parentFile ?: target
    val name = target.name
    val ext = name.substringAfterLast('.', "")
    val stem = if (ext.isEmpty()) name else name.removeSuffix(".$ext")
    for (i in 1..999) {
        val candidate = File(dir, if (ext.isEmpty()) "$stem ($i)" else "$stem ($i).$ext")
        if (!candidate.exists()) return candidate
    }
    return target
}

private fun safeName(name: String): String {
    if (name.isBlank()) return "untitled"
    return name.map { c ->
        if (c == '/' || c == '\\' || c == ':' || c == ' ') '_' else c
    }.joinToString("")
}

