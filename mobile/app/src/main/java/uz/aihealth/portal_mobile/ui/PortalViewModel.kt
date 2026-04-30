package uz.aihealth.portal_mobile.ui

import android.app.Application
import android.net.Uri
import android.os.Environment
import android.provider.OpenableColumns
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.flatMapLatest
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import uz.aihealth.portal_mobile.data.PortalSettings
import uz.aihealth.portal_mobile.data.RecentPortal
import uz.aihealth.portal_mobile.mesh.ChatMessage
import uz.aihealth.portal_mobile.mesh.MeshManager
import uz.aihealth.portal_mobile.mesh.MeshState
import uz.aihealth.portal_mobile.mesh.PeerSnapshot
import uz.aihealth.portal_mobile.service.MeshService
import uz.aihealth.portal_mobile.signaling.DEFAULT_SIGNALING_URL
import uz.aihealth.portal_mobile.transfer.FileTransfer
import uz.aihealth.portal_mobile.transfer.TransferEngine
import uz.aihealth.portal_mobile.transfer.TransferManifest
import uz.aihealth.portal_mobile.transfer.TransferSink
import java.io.File

@OptIn(ExperimentalCoroutinesApi::class)
class PortalViewModel(app: Application) : AndroidViewModel(app) {

    private val settings = PortalSettings(app)

    var nickname by mutableStateOf("")
    var signalUrl by mutableStateOf(DEFAULT_SIGNALING_URL)

    val recentPortals: StateFlow<List<RecentPortal>> = settings.recentPortals
        .stateIn(viewModelScope, SharingStarted.Eagerly, emptyList())

    private val _mesh = MutableStateFlow<MeshManager?>(null)
    private val _engine = MutableStateFlow<TransferEngine?>(null)

    val meshState: StateFlow<MeshState> = _mesh
        .flatMapLatest { it?.state ?: flowOf(MeshState.Idle) }
        .stateIn(viewModelScope, SharingStarted.Eagerly, MeshState.Idle)

    val peers: StateFlow<List<PeerSnapshot>> = _mesh
        .flatMapLatest { it?.peerList ?: flowOf(emptyList()) }
        .stateIn(viewModelScope, SharingStarted.Eagerly, emptyList())

    val transfers: StateFlow<List<FileTransfer>> = _engine
        .flatMapLatest { it?.transfers ?: flowOf(emptyList()) }
        .stateIn(viewModelScope, SharingStarted.Eagerly, emptyList())

    private val _chatLog = mutableStateListOf<ChatMessage>()
    val chatLog: List<ChatMessage> get() = _chatLog

    init {
        // Pre-fill nickname / signal URL from saved settings on startup.
        viewModelScope.launch {
            val saved = settings.nickname.first()
            if (saved.isNotEmpty() && nickname.isEmpty()) nickname = saved
        }
        viewModelScope.launch {
            val saved = settings.signalUrl.first()
            if (saved.isNotEmpty()) signalUrl = saved
        }
    }

    fun createPortal() {
        if (nickname.isBlank()) return
        commitNickname()
        MeshService.start(getApplication(), "Portal yaratilmoqda…")
        val mesh = ensureMesh()
        mesh.createPortal()
        observeForRecent(asOwner = true, joinedId = "", joinedCode = "")
    }

    fun joinPortal(portalId: String, code: String) {
        if (nickname.isBlank() || portalId.isBlank() || code.isBlank()) return
        commitNickname()
        MeshService.start(getApplication(), "Portalga ulanmoqda…")
        val mesh = ensureMesh()
        mesh.joinPortal(portalId, code)
        observeForRecent(asOwner = false, joinedId = portalId, joinedCode = code)
    }

    fun rejoinRecent(entry: RecentPortal) {
        if (entry.role == "owner") {
            // Owner portals are always created fresh (the server destroys
            // them when the owner disconnects). Best we can do is reuse
            // the saved nickname.
            nickname = entry.nickname
            createPortal()
        } else {
            nickname = entry.nickname
            joinPortal(entry.portalId, entry.code)
        }
    }

    fun forgetRecent(portalId: String) {
        viewModelScope.launch { settings.clearRecent(portalId) }
    }

    fun sendChat(text: String) {
        val msg = text.trim()
        if (msg.isEmpty()) return
        _mesh.value?.sendChat(msg)
    }

    /** Send the file pointed to by [uri] to [peerId]. Resolves the file's
     * display name + size + mime via [android.content.ContentResolver],
     * then hands an InputStream to the engine. */
    fun sendFile(peerId: String, uri: Uri) {
        val engine = _engine.value ?: return
        val app = getApplication<Application>()
        viewModelScope.launch {
            val (name, size, mime) = withContext(Dispatchers.IO) {
                val cr = app.contentResolver
                var displayName: String? = null
                var bytes: Long = -1L
                cr.query(uri, null, null, null, null)?.use { cursor ->
                    if (cursor.moveToFirst()) {
                        val nameIdx = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
                        val sizeIdx = cursor.getColumnIndex(OpenableColumns.SIZE)
                        if (nameIdx >= 0) displayName = cursor.getString(nameIdx)
                        if (sizeIdx >= 0) bytes = cursor.getLong(sizeIdx)
                    }
                }
                Triple(displayName ?: "untitled", bytes, cr.getType(uri))
            }
            val input = withContext(Dispatchers.IO) {
                runCatching { app.contentResolver.openInputStream(uri) }.getOrNull()
            } ?: return@launch
            engine.sendFile(
                peerId = peerId,
                manifest = TransferManifest(name = name, size = size.coerceAtLeast(-1L), mime = mime),
                input = input,
            )
        }
    }

    fun leave() {
        _mesh.value?.leave()
        _mesh.value?.transferHandler = null
        _mesh.value = null
        _engine.value = null
        _chatLog.clear()
        MeshService.stop(getApplication())
    }

    private fun commitNickname() {
        viewModelScope.launch { settings.setNickname(nickname.trim()) }
    }

    /** Watch the next Ready transition and persist the portal so the user
     * can quick-rejoin from the welcome screen. */
    private fun observeForRecent(asOwner: Boolean, joinedId: String, joinedCode: String) {
        viewModelScope.launch {
            // Take the first Ready emission to avoid accidentally re-saving
            // every state tick.
            meshState.collect { st ->
                if (st is MeshState.Ready) {
                    val info = st.portal
                    settings.rememberPortal(
                        RecentPortal(
                            portalId = info.portalId,
                            code = if (asOwner) info.code else joinedCode,
                            role = if (asOwner) "owner" else "joiner",
                            nickname = nickname,
                            lastUsedMs = System.currentTimeMillis(),
                        ),
                    )
                    return@collect
                }
            }
        }
    }

    private fun ensureMesh(): MeshManager {
        _mesh.value?.let { return it }
        val app = getApplication<Application>()
        val mesh = MeshManager(
            appContext = app,
            scope = viewModelScope,
            nickname = nickname,
            signalingUrl = signalUrl,
        )
        viewModelScope.launch {
            mesh.chats.collect { _chatLog.add(it) }
        }
        // Wire up the file-transfer engine. Save dir is app-private external
        // storage so the user can browse/share via a file manager; no
        // runtime permission is required on any minSdk-26+ device.
        val saveDir = app.getExternalFilesDir(Environment.DIRECTORY_DOWNLOADS)
            ?: File(app.filesDir, "Downloads").also { it.mkdirs() }
        val engine = TransferEngine(
            saveDir = saveDir,
            mesh = object : TransferSink {
                override fun sendTransferFrame(peerId: String, payload: ByteArray) =
                    mesh.sendTransferFrame(peerId, payload)
                override fun nicknameOf(peerId: String) = mesh.nicknameOf(peerId)
            },
            scope = viewModelScope,
        )
        mesh.transferHandler = { peerId, payload -> engine.handleFrame(peerId, payload) }
        _engine.value = engine
        _mesh.value = mesh
        return mesh
    }

    override fun onCleared() {
        _mesh.value?.close()
        MeshService.stop(getApplication())
        super.onCleared()
    }
}
