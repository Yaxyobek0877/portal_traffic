package uz.aihealth.portal_mobile.ui

import android.app.Application
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.flatMapLatest
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import uz.aihealth.portal_mobile.mesh.ChatMessage
import uz.aihealth.portal_mobile.mesh.MeshManager
import uz.aihealth.portal_mobile.mesh.MeshState
import uz.aihealth.portal_mobile.mesh.PeerSnapshot

@OptIn(ExperimentalCoroutinesApi::class)
class PortalViewModel(app: Application) : AndroidViewModel(app) {

    var nickname by mutableStateOf("")

    private val _mesh = MutableStateFlow<MeshManager?>(null)

    val meshState: StateFlow<MeshState> = _mesh
        .flatMapLatest { it?.state ?: flowOf(MeshState.Idle) }
        .stateIn(viewModelScope, SharingStarted.Eagerly, MeshState.Idle)

    val peers: StateFlow<List<PeerSnapshot>> = _mesh
        .flatMapLatest { it?.peerList ?: flowOf(emptyList()) }
        .stateIn(viewModelScope, SharingStarted.Eagerly, emptyList())

    private val _chatLog = mutableStateListOf<ChatMessage>()
    val chatLog: List<ChatMessage> get() = _chatLog

    fun createPortal() {
        if (nickname.isBlank()) return
        val mesh = ensureMesh()
        mesh.createPortal()
    }

    fun joinPortal(portalId: String, code: String) {
        if (nickname.isBlank() || portalId.isBlank() || code.isBlank()) return
        val mesh = ensureMesh()
        mesh.joinPortal(portalId, code)
    }

    fun sendChat(text: String) {
        val msg = text.trim()
        if (msg.isEmpty()) return
        _mesh.value?.sendChat(msg)
    }

    fun leave() {
        _mesh.value?.leave()
        _mesh.value = null
        _chatLog.clear()
    }

    private fun ensureMesh(): MeshManager {
        _mesh.value?.let { return it }
        val mesh = MeshManager(
            appContext = getApplication(),
            scope = viewModelScope,
            nickname = nickname,
        )
        // Tail the chat flow into the in-memory log used by the UI.
        viewModelScope.launch {
            mesh.chats.collect { _chatLog.add(it) }
        }
        _mesh.value = mesh
        return mesh
    }

    override fun onCleared() {
        _mesh.value?.close()
        super.onCleared()
    }
}
