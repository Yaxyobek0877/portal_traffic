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
import uz.aihealth.portal_mobile.data.AuthApi
import uz.aihealth.portal_mobile.data.AuthSession
import uz.aihealth.portal_mobile.data.PortalSettings
import uz.aihealth.portal_mobile.data.RecentPortal
import uz.aihealth.portal_mobile.data.deriveApiBaseUrl
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

/**
 * Auth status for the lock screen / dashboard gate.
 *
 *  - [Loading]      — boot-time /api/me probe is in flight; UI shows
 *                     a brief skeleton instead of bouncing the user
 *                     back to login on every cold start.
 *  - [Anonymous]    — no valid session; show LockScreen.
 *  - [Authenticated] — show the regular Welcome → Portal flow.
 */
sealed class AuthState {
    object Loading : AuthState()
    object Anonymous : AuthState()
    data class Authenticated(val username: String) : AuthState()
}

@OptIn(ExperimentalCoroutinesApi::class)
class PortalViewModel(app: Application) : AndroidViewModel(app) {

    private val settings = PortalSettings(app)
    private val authSession = AuthSession(app)
    private val authApi = AuthApi(authSession) {
        // Resolve the API base URL from the user-configurable override
        // first, then derive it from the signaling URL as a fallback.
        // signalUrl is mutable Compose state so we read it lazily.
        authSession.resolveApiBaseUrl(deriveApiBaseUrl(signalUrl))
    }

    var nickname by mutableStateOf("")
    var signalUrl by mutableStateOf(DEFAULT_SIGNALING_URL)

    // ---- Web-account auth state ----

    private val _authState = MutableStateFlow<AuthState>(AuthState.Loading)
    val authState: StateFlow<AuthState> = _authState.asStateFlow()

    private val _authError = MutableStateFlow<String?>(null)
    /** Last sign-in/sign-up error code (e.g. "invalid_credentials"). UI maps to localised text. */
    val authError: StateFlow<String?> = _authError.asStateFlow()

    private val _authLockoutSeconds = MutableStateFlow(0)
    /** Server-supplied lockout countdown after too many failed attempts. */
    val authLockoutSeconds: StateFlow<Int> = _authLockoutSeconds.asStateFlow()

    private val _authBusy = MutableStateFlow(false)
    val authBusy: StateFlow<Boolean> = _authBusy.asStateFlow()

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
        // Boot-time auth probe. If we have a saved session cookie, ask
        // /api/me whether it's still valid; on success we land directly
        // on Welcome, on 401 the cookie is cleared and we show Lock.
        viewModelScope.launch {
            val cached = authSession.username.first()
            val token = authSession.currentToken()
            if (token.isNullOrEmpty()) {
                _authState.value = AuthState.Anonymous
                return@launch
            }
            // Optimistically render the cached username while the
            // network probe runs — feels instantaneous and degrades
            // gracefully on a slow network.
            cached?.let { _authState.value = AuthState.Authenticated(it) }
            when (val r = authApi.me()) {
                is AuthApi.AuthResult.Success -> {
                    nickname = if (nickname.isBlank()) r.data.username else nickname
                    _authState.value = AuthState.Authenticated(r.data.username)
                }
                is AuthApi.AuthResult.Failure -> {
                    // Network failures shouldn't bounce the user out of
                    // the app — keep the optimistic Authenticated state
                    // if we already showed it. Only auth-level failures
                    // (401, 403, server-emitted error codes) drop us
                    // back to Anonymous.
                    if (r.code != "network") {
                        _authState.value = AuthState.Anonymous
                    } else if (cached == null) {
                        _authState.value = AuthState.Anonymous
                    }
                }
            }
        }
    }

    // ---- Auth actions ----

    fun signUp(username: String, password: String) {
        if (_authBusy.value) return
        _authError.value = null
        _authLockoutSeconds.value = 0
        viewModelScope.launch {
            _authBusy.value = true
            try {
                when (val r = authApi.signUp(username.trim(), password)) {
                    is AuthApi.AuthResult.Success -> {
                        // Username on a brand-new account becomes the
                        // default Portal nickname so the user doesn't
                        // have to retype it on Welcome.
                        if (nickname.isBlank()) {
                            nickname = r.data.username
                            settings.setNickname(r.data.username)
                        }
                        _authState.value = AuthState.Authenticated(r.data.username)
                    }
                    is AuthApi.AuthResult.Failure -> {
                        _authError.value = r.code
                    }
                }
            } finally {
                _authBusy.value = false
            }
        }
    }

    fun signIn(username: String, password: String) {
        if (_authBusy.value) return
        _authError.value = null
        _authLockoutSeconds.value = 0
        viewModelScope.launch {
            _authBusy.value = true
            try {
                when (val r = authApi.signIn(username.trim(), password)) {
                    is AuthApi.AuthResult.Success -> {
                        if (nickname.isBlank()) nickname = r.data.username
                        _authState.value = AuthState.Authenticated(r.data.username)
                    }
                    is AuthApi.AuthResult.Failure -> {
                        _authError.value = r.code
                        if (r.code == "locked_out") {
                            _authLockoutSeconds.value = r.lockoutSeconds.coerceAtLeast(1)
                            tickLockoutDown()
                        }
                    }
                }
            } finally {
                _authBusy.value = false
            }
        }
    }

    fun signOut() {
        viewModelScope.launch {
            // Tear down any active mesh first — the user explicitly
            // asked to sign out, leaving a portal connected behind a
            // locked UI is a footgun.
            try { _mesh.value?.leave() } catch (_: Throwable) {}
            authApi.signOut()
            _authState.value = AuthState.Anonymous
        }
    }

    fun clearAuthError() {
        _authError.value = null
    }

    /**
     * Drive the lockout countdown to zero. Called once when the server
     * returns a lockout; subsequent ticks happen here on the VM scope
     * so the UI just reads the StateFlow.
     */
    private fun tickLockoutDown() {
        viewModelScope.launch {
            while (_authLockoutSeconds.value > 0) {
                kotlinx.coroutines.delay(1000)
                _authLockoutSeconds.value = (_authLockoutSeconds.value - 1).coerceAtLeast(0)
            }
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

    fun saveSignalUrl(value: String) {
        signalUrl = value
        viewModelScope.launch { settings.setSignalUrl(value) }
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
