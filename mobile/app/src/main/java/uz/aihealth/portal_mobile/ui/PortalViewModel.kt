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
import uz.aihealth.portal_mobile.auth.AuthApi
import uz.aihealth.portal_mobile.auth.AuthResult
import uz.aihealth.portal_mobile.auth.MeResult
import uz.aihealth.portal_mobile.auth.UserInfo
import uz.aihealth.portal_mobile.auth.apiBaseFromSignal
import uz.aihealth.portal_mobile.data.PortalSettings
import uz.aihealth.portal_mobile.data.RecentPortal
import uz.aihealth.portal_mobile.i18n.Lang
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
import uz.aihealth.portal_mobile.turn.CloudflareTurn
import uz.aihealth.portal_mobile.turn.CloudflareTurnConfig
import uz.aihealth.portal_mobile.turn.IceServerResolver
import uz.aihealth.portal_mobile.turn.ManualTurnConfig
import uz.aihealth.portal_mobile.turn.TurnTestResult
import java.io.File

@OptIn(ExperimentalCoroutinesApi::class)
class PortalViewModel(app: Application) : AndroidViewModel(app) {

    private val settings = PortalSettings(app)

    var nickname by mutableStateOf("")
    var signalUrl by mutableStateOf(DEFAULT_SIGNALING_URL)

    val recentPortals: StateFlow<List<RecentPortal>> = settings.recentPortals
        .stateIn(viewModelScope, SharingStarted.Eagerly, emptyList())

    /**
     * Active language, also persisted. Reflects the user's pick across
     * process restarts; default is UZ.
     */
    private val _lang = MutableStateFlow(Lang.UZ)
    val lang: StateFlow<Lang> = _lang.asStateFlow()

    /** Persisted Cloudflare TURN credentials. Empty until the user enters them. */
    private val _cfTurn = MutableStateFlow(CloudflareTurnConfig())
    val cfTurn: StateFlow<CloudflareTurnConfig> = _cfTurn.asStateFlow()

    /** Last [CloudflareTurn.test] result; null while idle. */
    private val _cfTurnTest = MutableStateFlow<TurnTestResult?>(null)
    val cfTurnTest: StateFlow<TurnTestResult?> = _cfTurnTest.asStateFlow()

    /** True while a Cloudflare TURN test request is in flight. */
    private val _cfTurnTesting = MutableStateFlow(false)
    val cfTurnTesting: StateFlow<Boolean> = _cfTurnTesting.asStateFlow()

    /** Persisted manual TURN config (URL/user/pass). */
    private val _manualTurn = MutableStateFlow(ManualTurnConfig())
    val manualTurn: StateFlow<ManualTurnConfig> = _manualTurn.asStateFlow()

    // ------------------------------------------------------------------------
    // Account (server-side login)
    //
    // Optional layer — Portal's P2P features all work without an account.
    // Signing in lets the user later get a "my portals" dashboard via the
    // server's /api/portals endpoint and (eventually) recovery if they
    // lose their device. Sign-in is exposed via [signIn]/[signUp]; the
    // resulting session token is cookie material only — we never display
    // it. Sign-out clears it locally and on the server.
    // ------------------------------------------------------------------------

    private val authApi = AuthApi()

    /** Currently signed-in user, or null. Restored from settings on init,
     *  re-validated against the server in the background. */
    private val _signedInUser = MutableStateFlow<UserInfo?>(null)
    val signedInUser: StateFlow<UserInfo?> = _signedInUser.asStateFlow()

    /**
     * False until the cold-start auth resolution finishes. Mandatory-login
     * mode renders a brief splash while this is false so we don't flash
     * the AuthScreen for one frame before the cached session loads.
     */
    private val _authReady = MutableStateFlow(false)
    val authReady: StateFlow<Boolean> = _authReady.asStateFlow()

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
        // Pre-fill nickname / signal URL / lang / TURN from saved settings.
        viewModelScope.launch {
            val saved = settings.nickname.first()
            if (saved.isNotEmpty() && nickname.isEmpty()) nickname = saved
        }
        viewModelScope.launch {
            val saved = settings.signalUrl.first()
            if (saved.isNotEmpty()) signalUrl = saved
        }
        viewModelScope.launch {
            settings.lang.collect { _lang.value = Lang.fromCode(it) }
        }
        viewModelScope.launch {
            settings.cloudflareTurn.collect { _cfTurn.value = it }
        }
        viewModelScope.launch {
            settings.manualTurn.collect { _manualTurn.value = it }
        }
        // Restore the cached user immediately for a smooth "Salom, X"
        // welcome on cold start, then re-validate against the server.
        // For mandatory-login mode the rules are:
        //   - cached identity present + token valid (200)  → Ok, stay in
        //   - cached identity present + 401 from server    → evict, force re-auth
        //   - cached identity present + network blip       → keep cached (offline ok)
        //   - no cached identity                           → boot to AuthScreen
        viewModelScope.launch {
            val cachedUsername = settings.accountUsername.first()
            val cachedUserId = settings.accountUserId.first()
            if (cachedUsername.isNotBlank() && cachedUserId.isNotBlank()) {
                _signedInUser.value = UserInfo(id = cachedUserId, username = cachedUsername)
            }
            val token = settings.sessionToken.first()
            if (token.isNotBlank()) {
                val apiBase = apiBaseFromSignal(signalUrl)
                when (val r = authApi.fetchMe(apiBase, token)) {
                    is MeResult.Ok -> {
                        _signedInUser.value = r.user
                        settings.setSession(token, r.user.username, r.user.id)
                    }
                    MeResult.Unauthorized -> {
                        // Server says the token is dead. Clear local state
                        // so the gate falls back to AuthScreen.
                        settings.clearSession()
                        _signedInUser.value = null
                    }
                    MeResult.NetworkError -> {
                        // Don't punish the user for a flaky network — keep
                        // whatever was cached. Next /api/me call (after
                        // they retry an action) will reconcile.
                    }
                }
            }
            _authReady.value = true
        }
    }

    /**
     * Last AuthResult — null while idle. Read by AuthScreen to render
     * server-side errors / lockout countdown / network failures. Cleared
     * on a fresh attempt or when leaving the screen.
     */
    private val _authResult = MutableStateFlow<AuthResult?>(null)
    val authResult: StateFlow<AuthResult?> = _authResult.asStateFlow()

    private val _authBusy = MutableStateFlow(false)
    val authBusy: StateFlow<Boolean> = _authBusy.asStateFlow()

    fun signIn(username: String, password: String) {
        if (_authBusy.value) return
        viewModelScope.launch {
            _authBusy.value = true
            _authResult.value = null
            val r = authApi.signIn(apiBaseFromSignal(signalUrl), username, password)
            applyAuth(r)
            _authBusy.value = false
        }
    }

    fun signUp(username: String, password: String) {
        if (_authBusy.value) return
        viewModelScope.launch {
            _authBusy.value = true
            _authResult.value = null
            val r = authApi.signUp(apiBaseFromSignal(signalUrl), username, password)
            applyAuth(r)
            _authBusy.value = false
        }
    }

    fun signOut() {
        viewModelScope.launch {
            // Mandatory-login mode: signing out kicks the user back to the
            // gate, so we also tear down any active mesh — leaving the
            // foreground service alive while the user is at AuthScreen
            // would be confusing and waste battery.
            if (_mesh.value != null) {
                leave()
            }
            val token = settings.sessionToken.first()
            // Fire-and-forget the server call — local sign-out should
            // happen even if the network drops.
            launch { authApi.signOut(apiBaseFromSignal(signalUrl), token) }
            settings.clearSession()
            _signedInUser.value = null
            _authResult.value = null
        }
    }

    fun clearAuthResult() {
        _authResult.value = null
    }

    private suspend fun applyAuth(r: AuthResult) {
        _authResult.value = r
        if (r is AuthResult.Ok) {
            _signedInUser.value = r.user
            settings.setSession(r.sessionToken, r.user.username, r.user.id)
        }
    }

    // ------------------------------------------------------------------------
    // Portal lifecycle
    //
    // Both create/join now run inside viewModelScope.launch so we can `await`
    // the Cloudflare TURN credential fetch before constructing the mesh — a
    // synchronous mesh.createPortal() while the resolver suspends would
    // either block the main thread or skip TURN entirely.
    // ------------------------------------------------------------------------

    fun createPortal() {
        if (nickname.isBlank()) return
        viewModelScope.launch {
            commitNickname()
            MeshService.start(getApplication(), "Portal yaratilmoqda…")
            val mesh = ensureMesh()
            mesh.createPortal()
            observeForRecent(asOwner = true, joinedId = "", joinedCode = "")
        }
    }

    fun joinPortal(portalId: String, code: String) {
        if (nickname.isBlank() || portalId.isBlank() || code.isBlank()) return
        viewModelScope.launch {
            commitNickname()
            MeshService.start(getApplication(), "Portalga ulanmoqda…")
            val mesh = ensureMesh()
            mesh.joinPortal(portalId, code)
            observeForRecent(asOwner = false, joinedId = portalId, joinedCode = code)
        }
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

    // ------------------------------------------------------------------------
    // Settings: language, signal URL, TURN
    // ------------------------------------------------------------------------

    fun setLang(value: Lang) {
        _lang.value = value
        viewModelScope.launch { settings.setLang(value.code) }
    }

    fun saveSignalUrl(value: String) {
        signalUrl = value
        viewModelScope.launch { settings.setSignalUrl(value) }
    }

    fun saveCloudflareTurn(cfg: CloudflareTurnConfig) {
        _cfTurn.value = cfg
        _cfTurnTest.value = null
        CloudflareTurn.invalidateCache()
        viewModelScope.launch { settings.setCloudflareTurn(cfg) }
    }

    fun saveManualTurn(cfg: ManualTurnConfig) {
        _manualTurn.value = cfg
        viewModelScope.launch { settings.setManualTurn(cfg) }
    }

    fun testCloudflareTurn() {
        if (_cfTurnTesting.value) return
        viewModelScope.launch {
            _cfTurnTesting.value = true
            _cfTurnTest.value = null
            try {
                _cfTurnTest.value = CloudflareTurn.test(_cfTurn.value)
            } finally {
                _cfTurnTesting.value = false
            }
        }
    }

    private suspend fun commitNickname() {
        settings.setNickname(nickname.trim())
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

    private suspend fun ensureMesh(): MeshManager {
        _mesh.value?.let { return it }
        // Resolve ICE servers (STUN + Cloudflare-or-Manual TURN) before
        // constructing the mesh. The Cloudflare path can take ~200ms;
        // resolveOffline is the synchronous fallback if the user isn't
        // configured for it.
        val ice = IceServerResolver.resolve(_cfTurn.value, _manualTurn.value)
        val app = getApplication<Application>()
        val mesh = MeshManager(
            appContext = app,
            scope = viewModelScope,
            nickname = nickname,
            signalingUrl = signalUrl,
            iceServers = ice,
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
