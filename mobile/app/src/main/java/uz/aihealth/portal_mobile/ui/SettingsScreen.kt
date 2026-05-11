package uz.aihealth.portal_mobile.ui

import android.content.Intent
import android.net.Uri
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
import androidx.compose.foundation.layout.wrapContentSize
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import uz.aihealth.portal_mobile.BuildConfig
import uz.aihealth.portal_mobile.i18n.Lang
import uz.aihealth.portal_mobile.i18n.LocalLang
import uz.aihealth.portal_mobile.i18n.t
import uz.aihealth.portal_mobile.signaling.DEFAULT_SIGNALING_URL
import uz.aihealth.portal_mobile.turn.CloudflareTurnConfig
import uz.aihealth.portal_mobile.turn.FREE_TURN
import uz.aihealth.portal_mobile.turn.ManualTurnConfig

private const val GITHUB_URL = "https://github.com/Yaxyobek0877/portal_traffic"
private const val PRIVACY_URL = "https://github.com/Yaxyobek0877/portal_traffic/blob/main/PRIVACY.md"
private const val TERMS_URL = "https://github.com/Yaxyobek0877/portal_traffic/blob/main/TERMS.md"
private const val LICENSE_URL = "https://github.com/Yaxyobek0877/portal_traffic/blob/main/LICENSE"
private const val PCP_URL = "https://github.com/Yaxyobek0877/portal_traffic/blob/main/client/crypt/pcp/SPEC.md"

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    vm: PortalViewModel,
    onBack: () -> Unit,
) {
    val lang = LocalLang.current
    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(t("settings.title")) },
                navigationIcon = {
                    TextButton(onClick = onBack) { Text("← ${t("common.back")}") }
                },
                actions = {
                    LangPicker(selected = lang, onSelect = { vm.setLang(it) })
                    Spacer(Modifier.width(12.dp))
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            AccountSection(vm)
            NetworkSection(vm)
            CloudflareTurnSection(vm)
            ManualTurnSection(vm)
            AboutSection()
        }
    }
}

/**
 * Server-account block. In mandatory-login mode this is always shown
 * with the active user — Settings is unreachable while signed out.
 * Kept as the first card so a user who wants to switch accounts has
 * the sign-out button at the top, not buried under network config.
 */
@Composable
private fun AccountSection(vm: PortalViewModel) {
    val user by vm.signedInUser.collectAsState()
    val username = user?.username ?: ""
    Section(title = t("auth.signed_in_as", username)) {
        OutlinedButton(
            onClick = { vm.signOut() },
            modifier = Modifier.fillMaxWidth(),
        ) { Text(t("auth.signout")) }
    }
}

@Composable
private fun LangPicker(selected: Lang, onSelect: (Lang) -> Unit) {
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
                Text(l.label, style = MaterialTheme.typography.labelSmall, color = fg, fontFamily = FontFamily.Monospace)
            }
        }
    }
}

@Composable
private fun NetworkSection(vm: PortalViewModel) {
    var draft by remember { mutableStateOf(vm.signalUrl) }
    var error by remember { mutableStateOf<String?>(null) }
    var saved by remember { mutableStateOf(false) }
    LaunchedEffect(vm.signalUrl) { draft = vm.signalUrl }

    // Pre-resolve localized strings used inside non-composable lambdas
    // (onClick is `() -> Unit`, not `@Composable () -> Unit`, so we
    // can't call t() from inside it).
    val urlValidationMsg = t("settings.url_validation")

    Section(title = t("settings.network")) {
        OutlinedTextField(
            value = draft,
            onValueChange = {
                draft = it.trim()
                error = null
                saved = false
            },
            label = { Text(t("settings.signal_url")) },
            placeholder = { Text(DEFAULT_SIGNALING_URL) },
            singleLine = true,
            isError = error != null,
            supportingText = {
                when {
                    error != null -> Text(error!!, color = MaterialTheme.colorScheme.error)
                    saved -> Text(t("common.saved"), color = MaterialTheme.colorScheme.primary)
                    else -> Text(t("settings.signal_url_hint"))
                }
            },
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        Text(
            t("settings.default", DEFAULT_SIGNALING_URL),
            style = MaterialTheme.typography.bodySmall,
            fontFamily = FontFamily.Monospace,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(12.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Button(
                onClick = {
                    val target = draft.ifBlank { DEFAULT_SIGNALING_URL }
                    when {
                        !target.startsWith("ws://") && !target.startsWith("wss://") ->
                            error = urlValidationMsg
                        else -> {
                            vm.saveSignalUrl(target)
                            draft = target
                            saved = true
                            error = null
                        }
                    }
                },
                modifier = Modifier.weight(1f),
            ) { Text(t("common.save")) }
            TextButton(
                onClick = {
                    draft = DEFAULT_SIGNALING_URL
                    vm.saveSignalUrl(DEFAULT_SIGNALING_URL)
                    saved = true
                    error = null
                },
            ) { Text(t("settings.reset")) }
        }
    }
}

@Composable
private fun CloudflareTurnSection(vm: PortalViewModel) {
    val saved by vm.cfTurn.collectAsState()
    val testResult by vm.cfTurnTest.collectAsState()
    val testing by vm.cfTurnTesting.collectAsState()
    var tokenId by remember(saved.tokenId) { mutableStateOf(saved.tokenId) }
    var apiToken by remember(saved.apiToken) { mutableStateOf(saved.apiToken) }
    var savedAt by remember { mutableStateOf<Long?>(null) }

    Section(title = t("settings.cf_turn")) {
        Text(
            t("settings.cf_turn.intro"),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(6.dp))
        Text(
            t("settings.cf_turn.steps"),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = tokenId,
            onValueChange = { tokenId = it.trim() },
            label = { Text(t("settings.cf_turn.token_id")) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = apiToken,
            onValueChange = { apiToken = it.trim() },
            label = { Text(t("settings.cf_turn.api_token")) },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(12.dp))
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Button(
                onClick = {
                    vm.saveCloudflareTurn(CloudflareTurnConfig(tokenId, apiToken))
                    savedAt = System.currentTimeMillis()
                },
            ) { Text(t("common.save")) }
            OutlinedButton(
                onClick = {
                    // Persist whatever the user typed before testing — that
                    // way you don't get a "tested values are different from
                    // saved" surprise if the test passes but they forgot to
                    // hit Save.
                    vm.saveCloudflareTurn(CloudflareTurnConfig(tokenId, apiToken))
                    vm.testCloudflareTurn()
                },
                enabled = !testing && tokenId.isNotBlank() && apiToken.isNotBlank(),
            ) { Text(if (testing) t("settings.cf_turn.testing") else t("settings.cf_turn.test")) }
            if (savedAt != null && testResult == null) {
                Spacer(Modifier.weight(1f))
                Text(t("common.saved") + " ✓", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary)
            }
        }
        testResult?.let { r ->
            Spacer(Modifier.height(12.dp))
            ResultPanel(ok = r.ok, text = r.message, sub = if (r.urls.isNotEmpty()) "URLs: ${r.urls.joinToString(", ")} (${r.gatherMs} ms)" else "${r.gatherMs} ms")
        }
    }
}

@Composable
private fun ManualTurnSection(vm: PortalViewModel) {
    val saved by vm.manualTurn.collectAsState()
    var url by remember(saved.url) { mutableStateOf(saved.url) }
    var username by remember(saved.username) { mutableStateOf(saved.username) }
    var credential by remember(saved.credential) { mutableStateOf(saved.credential) }
    var savedAt by remember { mutableStateOf<Long?>(null) }

    Section(title = t("settings.manual_turn")) {
        Text(
            t("settings.manual_turn.intro"),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = url,
            onValueChange = { url = it },
            label = { Text(t("settings.manual_turn.url_label")) },
            modifier = Modifier.fillMaxWidth().height(112.dp),
            placeholder = { Text("turn:turn.example.com:3478\nturns:turn.example.com:5349") },
        )
        Spacer(Modifier.height(8.dp))
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            modifier = Modifier.fillMaxWidth(),
        ) {
            OutlinedTextField(
                value = username,
                onValueChange = { username = it },
                label = { Text(t("settings.manual_turn.username")) },
                singleLine = true,
                modifier = Modifier.weight(1f),
            )
            OutlinedTextField(
                value = credential,
                onValueChange = { credential = it },
                label = { Text(t("settings.manual_turn.credential")) },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                modifier = Modifier.weight(1f),
            )
        }
        Spacer(Modifier.height(12.dp))
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Button(
                onClick = {
                    vm.saveManualTurn(ManualTurnConfig(url.trim(), username.trim(), credential.trim()))
                    savedAt = System.currentTimeMillis()
                },
            ) { Text(t("common.save")) }
            OutlinedButton(
                onClick = {
                    url = FREE_TURN.url
                    username = FREE_TURN.username
                    credential = FREE_TURN.credential
                    vm.saveManualTurn(FREE_TURN)
                    savedAt = System.currentTimeMillis()
                },
            ) { Text(t("settings.manual_turn.free")) }
            if (url.isNotBlank()) {
                TextButton(
                    onClick = {
                        url = ""; username = ""; credential = ""
                        vm.saveManualTurn(ManualTurnConfig())
                        savedAt = System.currentTimeMillis()
                    },
                ) { Text(t("common.clear")) }
            }
        }
        if (savedAt != null) {
            Spacer(Modifier.height(6.dp))
            Text(t("common.saved") + " ✓", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary)
        }
        Spacer(Modifier.height(8.dp))
        Text(
            t("settings.manual_turn.applies_next"),
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun AboutSection() {
    val ctx = LocalContext.current
    Section(title = t("settings.about")) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                t("settings.about.version"),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.weight(1f))
            // Pulled from BuildConfig (see app/build.gradle.kts:buildConfig=true).
            // The build number lets us distinguish two installs that share a
            // versionName during a dev cycle without inspecting the APK.
            Text(
                "v${BuildConfig.VERSION_NAME} (${BuildConfig.VERSION_CODE})",
                style = MaterialTheme.typography.titleMedium,
                fontFamily = FontFamily.Monospace,
            )
        }
        Spacer(Modifier.height(12.dp))
        Text(
            t("settings.about.docs"),
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(6.dp))
        val links = listOf(
            "GitHub" to GITHUB_URL,
            "PCP-1 Spec" to PCP_URL,
            "Privacy" to PRIVACY_URL,
            "Terms" to TERMS_URL,
            "License (MIT)" to LICENSE_URL,
        )
        links.chunked(2).forEach { row ->
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxWidth(),
            ) {
                row.forEach { (label, url) ->
                    OutlinedButton(
                        onClick = { openUrl(ctx, url) },
                        modifier = Modifier.weight(1f),
                    ) { Text(label, style = MaterialTheme.typography.labelMedium) }
                }
                if (row.size == 1) Spacer(Modifier.weight(1f))
            }
            Spacer(Modifier.height(4.dp))
        }
    }
}

private fun openUrl(ctx: android.content.Context, url: String) {
    runCatching {
        ctx.startActivity(
            Intent(Intent.ACTION_VIEW, Uri.parse(url)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
        )
    }
}

@Composable
private fun Section(title: String, content: @Composable () -> Unit) {
    Card(
        shape = RoundedCornerShape(14.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f),
        ),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text(
                title,
                style = MaterialTheme.typography.titleMedium,
            )
            Spacer(Modifier.height(10.dp))
            content()
        }
    }
}

@Composable
private fun ResultPanel(ok: Boolean, text: String, sub: String) {
    val color = if (ok) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.error
    Card(
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(
            containerColor = color.copy(alpha = 0.08f),
        ),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Text(
                (if (ok) "✓ " else "✗ ") + text,
                style = MaterialTheme.typography.bodyMedium,
                color = color,
            )
            if (sub.isNotBlank()) {
                Spacer(Modifier.height(2.dp))
                Text(
                    sub,
                    style = MaterialTheme.typography.labelSmall,
                    fontFamily = FontFamily.Monospace,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}
