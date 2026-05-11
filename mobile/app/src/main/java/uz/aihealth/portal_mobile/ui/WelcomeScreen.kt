package uz.aihealth.portal_mobile.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import uz.aihealth.portal_mobile.data.RecentPortal
import uz.aihealth.portal_mobile.i18n.Lang
import uz.aihealth.portal_mobile.i18n.LocalLang
import uz.aihealth.portal_mobile.i18n.t
import kotlin.math.max

@Composable
fun WelcomeScreen(
    vm: PortalViewModel,
    onCreate: () -> Unit,
    onGoToJoin: () -> Unit,
    onGoToSettings: () -> Unit,
) {
    val recents by vm.recentPortals.collectAsState()
    val signedInUser by vm.signedInUser.collectAsState()
    val lang = LocalLang.current

    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 24.dp),
        verticalArrangement = Arrangement.spacedBy(0.dp),
    ) {
        // Top row: account greeting + language toggle + settings shortcut.
        // Mandatory-login mode means the gate above us has guaranteed
        // signedInUser is non-null by the time this screen is composed —
        // we still defensively guard, but there's no "Sign in" fallback.
        item {
            Spacer(Modifier.height(16.dp))
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    t("auth.greeting", signedInUser?.username.orEmpty()),
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.primary,
                )
                Spacer(Modifier.weight(1f))
                LangToggle(
                    selected = lang,
                    onSelect = { vm.setLang(it) },
                )
                Spacer(Modifier.width(8.dp))
                IconButton(onClick = onGoToSettings) { Text("⚙") }
            }
            Spacer(Modifier.height(8.dp))
        }

        // Logo + title block — visually centered like the desktop welcome.
        item {
            Column(
                modifier = Modifier.fillMaxWidth(),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Logo(size = 160.dp)
                Spacer(Modifier.height(8.dp))
                Text(
                    "Portal",
                    style = MaterialTheme.typography.displayMedium,
                    fontWeight = FontWeight.ExtraBold,
                )
                Spacer(Modifier.height(8.dp))
                Text(
                    t("welcome.tagline"),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Spacer(Modifier.height(32.dp))
            }
        }

        item {
            OutlinedTextField(
                value = vm.nickname,
                onValueChange = { vm.nickname = it.take(32) },
                label = { Text(t("welcome.nickname.label")) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(20.dp))
            Button(
                onClick = onCreate,
                enabled = vm.nickname.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) { Text(t("welcome.create")) }
            Spacer(Modifier.height(12.dp))
            OutlinedButton(
                onClick = onGoToJoin,
                enabled = vm.nickname.isNotBlank(),
                modifier = Modifier.fillMaxWidth(),
            ) { Text(t("welcome.join")) }
            Spacer(Modifier.height(24.dp))
        }
        if (recents.isNotEmpty()) {
            item {
                Text(
                    t("welcome.recent"),
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

/** UZ/EN segmented switch — matches the desktop's titlebar pill. */
@Composable
private fun LangToggle(selected: Lang, onSelect: (Lang) -> Unit) {
    Row(
        modifier = Modifier
            .clip(RoundedCornerShape(6.dp))
            .border(
                width = 1.dp,
                color = MaterialTheme.colorScheme.outline.copy(alpha = 0.3f),
                shape = RoundedCornerShape(6.dp),
            ),
    ) {
        Lang.values().forEach { l ->
            val active = l == selected
            val bg = if (active) MaterialTheme.colorScheme.primary.copy(alpha = 0.18f) else Color.Transparent
            val fg = if (active) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant
            Box(
                modifier = Modifier
                    .background(bg)
                    .clickable { onSelect(l) }
                    .padding(horizontal = 10.dp, vertical = 4.dp),
            ) {
                Text(
                    l.label,
                    style = MaterialTheme.typography.labelSmall,
                    color = fg,
                    fontFamily = FontFamily.Monospace,
                )
            }
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
                        if (entry.role == "owner") t("common.host") else t("common.joiner"),
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

@Composable
private fun relativeTime(thenMs: Long): String {
    val seconds = max(0L, (System.currentTimeMillis() - thenMs) / 1000)
    return when {
        seconds < 60 -> t("common.now")
        seconds < 3600 -> t("common.min_ago", seconds / 60)
        seconds < 86400 -> t("common.hour_ago", seconds / 3600)
        else -> t("common.day_ago", seconds / 86400)
    }
}
