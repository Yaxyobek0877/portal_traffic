package uz.aihealth.portal_mobile.ui

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import uz.aihealth.portal_mobile.data.RecentPortal
import kotlin.math.max

@Composable
fun WelcomeScreen(
    vm: PortalViewModel,
    onCreate: () -> Unit,
    onGoToJoin: () -> Unit,
    onGoToSettings: () -> Unit,
) {
    val recents by vm.recentPortals.collectAsState()

    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 24.dp),
        verticalArrangement = Arrangement.spacedBy(0.dp),
    ) {
        item {
            Spacer(Modifier.height(48.dp))
            Text("Portal", style = MaterialTheme.typography.displayMedium)
            Spacer(Modifier.height(8.dp))
            Text(
                "To'g'ridan-to'g'ri ulanish. Orada server yo'q.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(40.dp))
        }
        item {
            OutlinedTextField(
                value = vm.nickname,
                onValueChange = { vm.nickname = it.take(32) },
                label = { Text("Taxallusingiz") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(20.dp))
            Button(
                onClick = onCreate,
                enabled = vm.nickname.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Yangi portal yaratish") }
            Spacer(Modifier.height(12.dp))
            OutlinedButton(
                onClick = onGoToJoin,
                enabled = vm.nickname.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Mavjud portalga qo'shilish") }
            Spacer(Modifier.height(8.dp))
            TextButton(
                onClick = onGoToSettings,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Sozlamalar") }
            Spacer(Modifier.height(24.dp))
        }
        if (recents.isNotEmpty()) {
            item {
                Text(
                    "Yaqindagi portallar",
                    style = MaterialTheme.typography.titleSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Spacer(Modifier.height(8.dp))
            }
            items(recents, key = { it.portalId + it.lastUsedMs }) { entry ->
                RecentRow(
                    entry = entry,
                    onTap = { vm.rejoinRecent(entry) },
                    onForget = { vm.forgetRecent(entry.portalId) },
                )
                Spacer(Modifier.height(8.dp))
            }
            item { Spacer(Modifier.height(24.dp)) }
        }
    }
}

@Composable
private fun RecentRow(entry: RecentPortal, onTap: () -> Unit, onForget: () -> Unit) {
    Card(
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant,
        ),
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onTap),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 10.dp),
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        entry.portalId,
                        style = MaterialTheme.typography.titleMedium,
                        fontFamily = FontFamily.Monospace,
                        fontWeight = FontWeight.SemiBold,
                    )
                    Spacer(Modifier.width(8.dp))
                    Text(
                        if (entry.role == "owner") "host" else "joiner",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                Text(
                    entry.nickname + " · " + relativeTime(entry.lastUsedMs),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            IconButton(onClick = onForget) {
                Text("×", style = MaterialTheme.typography.titleLarge)
            }
        }
    }
}

private fun relativeTime(thenMs: Long): String {
    val seconds = max(0L, (System.currentTimeMillis() - thenMs) / 1000)
    return when {
        seconds < 60 -> "hozir"
        seconds < 3600 -> "${seconds / 60} daq oldin"
        seconds < 86400 -> "${seconds / 3600} soat oldin"
        else -> "${seconds / 86400} kun oldin"
    }
}
