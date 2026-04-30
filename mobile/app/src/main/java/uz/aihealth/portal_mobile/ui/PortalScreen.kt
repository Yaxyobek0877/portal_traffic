package uz.aihealth.portal_mobile.ui

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import uz.aihealth.portal_mobile.mesh.ChatMessage
import uz.aihealth.portal_mobile.mesh.MeshState
import uz.aihealth.portal_mobile.mesh.PeerSnapshot
import uz.aihealth.portal_mobile.peer.PeerState
import uz.aihealth.portal_mobile.transfer.Direction
import uz.aihealth.portal_mobile.transfer.FileTransfer

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PortalScreen(
    vm: PortalViewModel,
    onLeave: () -> Unit,
) {
    val state by vm.meshState.collectAsState()
    val peers by vm.peers.collectAsState()
    val transfers by vm.transfers.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        when (val s = state) {
                            is MeshState.Ready ->
                                if (s.portal.isOwner)
                                    "Portal: ${s.portal.portalId}"
                                else
                                    "Portal ${s.portal.portalId}"
                            MeshState.Connecting -> "Ulanmoqda…"
                            is MeshState.Reconnecting -> "Qayta ulanmoqda… (${s.attempt}/8)"
                            is MeshState.Failed -> "Xatolik"
                            else -> "Portal"
                        },
                    )
                },
                actions = {
                    TextButton(onClick = {
                        vm.leave()
                        onLeave()
                    }) {
                        Text("Chiqish")
                    }
                },
            )
        },
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            HeaderCard(state = state)
            HorizontalDivider()
            PeerListSection(peers = peers, vm = vm, modifier = Modifier.fillMaxWidth())
            if (transfers.isNotEmpty()) {
                HorizontalDivider()
                TransferListSection(transfers = transfers, modifier = Modifier.fillMaxWidth())
            }
            HorizontalDivider()
            ChatSection(
                vm = vm,
                modifier = Modifier
                    .fillMaxWidth()
                    .weight(1f),
            )
        }
    }
}

@Composable
private fun HeaderCard(state: MeshState) {
    when (state) {
        is MeshState.Ready -> {
            val info = state.portal
            var qrOpen by remember { mutableStateOf(false) }
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(16.dp),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.primaryContainer,
                ),
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    if (info.isOwner) {
                        ItemRow("Portal ID", info.portalId, mono = true)
                        Spacer(Modifier.height(4.dp))
                        ItemRow("Kod", info.code, mono = true)
                        Spacer(Modifier.height(12.dp))
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                "Bu kodni do'stlaringiz bilan ulashing.",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onPrimaryContainer,
                                modifier = Modifier.weight(1f),
                            )
                            OutlinedButton(onClick = { qrOpen = true }) { Text("QR") }
                        }
                    } else {
                        ItemRow("Portal ID", info.portalId, mono = true)
                        Spacer(Modifier.height(4.dp))
                        ItemRow("Sizning IP", info.ownVip, mono = true)
                    }
                }
            }
            if (qrOpen) {
                QrInviteDialog(
                    portalId = info.portalId,
                    code = info.code,
                    onDismiss = { qrOpen = false },
                )
            }
        }

        is MeshState.Failed -> {
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(16.dp),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.errorContainer,
                ),
            ) {
                Text(
                    text = "Xato: ${state.reason}",
                    modifier = Modifier.padding(16.dp),
                    color = MaterialTheme.colorScheme.onErrorContainer,
                )
            }
        }

        MeshState.Connecting -> {
            Text(
                "Signal serveriga ulanmoqda…",
                modifier = Modifier.padding(16.dp),
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }

        is MeshState.Reconnecting -> {
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(16.dp),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.tertiaryContainer,
                ),
            ) {
                Text(
                    "Tarmoq vaqtinchalik uzildi. Urinish ${state.attempt}/8…",
                    modifier = Modifier.padding(16.dp),
                    color = MaterialTheme.colorScheme.onTertiaryContainer,
                )
            }
        }

        else -> {}
    }
}

@Composable
private fun ItemRow(label: String, value: String, mono: Boolean) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text(
            "$label: ",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onPrimaryContainer,
        )
        Text(
            value,
            style = MaterialTheme.typography.titleMedium,
            fontFamily = if (mono) FontFamily.Monospace else FontFamily.Default,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.onPrimaryContainer,
        )
    }
}

@Composable
private fun PeerListSection(
    peers: List<PeerSnapshot>,
    vm: PortalViewModel,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier.padding(16.dp)) {
        Text(
            text = if (peers.isEmpty()) "Hech kim ulangan emas" else "Peer'lar (${peers.size})",
            style = MaterialTheme.typography.titleSmall,
        )
        Spacer(Modifier.height(8.dp))
        peers.forEach { p ->
            PeerRow(p, onSendFile = { uri -> vm.sendFile(p.id, uri) })
            Spacer(Modifier.height(4.dp))
        }
    }
}

@Composable
private fun PeerRow(peer: PeerSnapshot, onSendFile: (android.net.Uri) -> Unit) {
    val picker = rememberLauncherForActivityResult(ActivityResultContracts.OpenDocument()) { uri ->
        if (uri != null) onSendFile(uri)
    }
    val (color, label) = when (peer.state) {
        PeerState.CONNECTED -> Color(0xFF4CAF50) to "ulangan"
        PeerState.CONNECTING -> Color(0xFFFFC107) to "ulanmoqda"
        PeerState.FAILED -> Color(0xFFF44336) to "xato"
        PeerState.CLOSED -> Color(0xFF9E9E9E) to "yopildi"
    }
    Row(verticalAlignment = Alignment.CenterVertically) {
        Box(modifier = Modifier.size(10.dp).clip(CircleShape).background(color))
        Spacer(Modifier.width(8.dp))
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = peer.nickname.ifBlank { peer.id.take(8) },
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = if (peer.isOwner) FontWeight.Bold else FontWeight.Normal,
            )
            Text(
                text = peer.virtualIp,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                fontFamily = FontFamily.Monospace,
            )
        }
        Text(
            text = if (peer.rttMs > 0) "${peer.rttMs} ms" else label,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.width(8.dp))
        TextButton(
            onClick = { picker.launch(arrayOf("*/*")) },
            enabled = peer.state == PeerState.CONNECTED,
        ) {
            Text("📎")
        }
    }
}

@Composable
private fun TransferListSection(transfers: List<FileTransfer>, modifier: Modifier = Modifier) {
    Column(modifier = modifier.padding(horizontal = 16.dp, vertical = 8.dp)) {
        Text(
            text = "Fayllar (${transfers.size})",
            style = MaterialTheme.typography.titleSmall,
        )
        Spacer(Modifier.height(4.dp))
        transfers.forEach { t ->
            TransferRow(t)
            Spacer(Modifier.height(4.dp))
        }
    }
}

@Composable
private fun TransferRow(t: FileTransfer) {
    val arrow = if (t.direction == Direction.SEND) "↑" else "↓"
    val pct = if (t.manifest.size > 0) {
        (t.bytes.toDouble() / t.manifest.size.toDouble() * 100).coerceIn(0.0, 100.0).toInt()
    } else 0
    val sub = when {
        t.error != null -> "xato: ${t.error}"
        t.done && t.savePath != null -> "tayyor — ${t.savePath}"
        t.done -> "tayyor"
        t.manifest.size > 0 -> "$pct% · ${humanBytes(t.bytes)} / ${humanBytes(t.manifest.size)}"
        else -> humanBytes(t.bytes)
    }
    Column {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                "$arrow ${t.manifest.name}",
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.weight(1f),
            )
            Text(
                t.peerNickname,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        Text(
            sub,
            style = MaterialTheme.typography.bodySmall,
            color = if (t.error != null) MaterialTheme.colorScheme.error
            else MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

private fun humanBytes(n: Long): String {
    if (n < 0) return "?"
    val units = arrayOf("B", "KB", "MB", "GB")
    var v = n.toDouble()
    var i = 0
    while (v >= 1024.0 && i < units.lastIndex) {
        v /= 1024.0
        i++
    }
    return if (i == 0) "${n} ${units[0]}" else String.format("%.1f %s", v, units[i])
}

@Composable
private fun ChatSection(vm: PortalViewModel, modifier: Modifier = Modifier) {
    val log = vm.chatLog
    val listState = rememberLazyListState()
    LaunchedEffect(log.size) {
        if (log.isNotEmpty()) listState.animateScrollToItem(log.lastIndex)
    }
    var input by remember { mutableStateOf("") }

    Column(modifier = modifier) {
        LazyColumn(
            state = listState,
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
            modifier = Modifier.weight(1f),
        ) {
            items(log) { msg -> ChatBubble(msg, isMine = msg.fromPeerId.isEmpty() || msg.fromNickname == vm.nickname) }
        }
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 8.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            OutlinedTextField(
                value = input,
                onValueChange = { input = it },
                placeholder = { Text("Xabar yozing…") },
                singleLine = true,
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(8.dp))
            Button(
                onClick = {
                    vm.sendChat(input)
                    input = ""
                },
                enabled = input.isNotBlank(),
            ) {
                Text("Yuborish")
            }
        }
    }
}

@Composable
private fun QrInviteDialog(portalId: String, code: String, onDismiss: () -> Unit) {
    val invite = inviteText(portalId, code)
    val bmp = remember(invite) { generateQrBitmap(invite, 600).asImageBitmap() }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Portal taklifi") },
        text = {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .aspectRatio(1f)
                        .background(Color.White),
                    contentAlignment = Alignment.Center,
                ) {
                    Image(
                        bitmap = bmp,
                        contentDescription = "QR taklif",
                        modifier = Modifier.fillMaxWidth().padding(8.dp),
                    )
                }
                Spacer(Modifier.height(12.dp))
                Text(
                    "$portalId · $code",
                    style = MaterialTheme.typography.titleMedium,
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.SemiBold,
                )
                Spacer(Modifier.height(4.dp))
                Text(
                    "QR ni do'stingizning kamerasiga tutsangiz, portalga to'g'ridan-to'g'ri kiradi.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        },
        confirmButton = {
            TextButton(onClick = onDismiss) { Text("Yopish") }
        },
    )
}

@Composable
private fun ChatBubble(msg: ChatMessage, isMine: Boolean) {
    val bg = if (isMine) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceVariant
    val fg = if (isMine) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurfaceVariant
    val align = if (isMine) Alignment.End else Alignment.Start
    Column(
        modifier = Modifier.fillMaxWidth(),
        horizontalAlignment = align,
    ) {
        if (!isMine) {
            Text(
                text = msg.fromNickname.ifBlank { msg.fromPeerId.take(8) },
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        Box(
            modifier = Modifier
                .clip(RoundedCornerShape(12.dp))
                .background(bg)
                .padding(horizontal = 12.dp, vertical = 8.dp),
        ) {
            Text(text = msg.text, color = fg, fontSize = 14.sp)
        }
    }
}
