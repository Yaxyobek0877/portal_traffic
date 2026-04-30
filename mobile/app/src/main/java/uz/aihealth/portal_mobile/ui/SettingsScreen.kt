package uz.aihealth.portal_mobile.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import uz.aihealth.portal_mobile.signaling.DEFAULT_SIGNALING_URL

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    vm: PortalViewModel,
    onBack: () -> Unit,
) {
    var draft by remember { mutableStateOf(vm.signalUrl) }
    var error by remember { mutableStateOf<String?>(null) }
    var saved by remember { mutableStateOf(false) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Sozlamalar") },
                navigationIcon = {
                    TextButton(onClick = onBack) { Text("← Orqaga") }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(20.dp),
        ) {
            Text("Tarmoq", style = MaterialTheme.typography.titleMedium)
            Spacer(Modifier.height(12.dp))

            OutlinedTextField(
                value = draft,
                onValueChange = {
                    draft = it.trim()
                    error = null
                    saved = false
                },
                label = { Text("Signal serveri URL") },
                placeholder = { Text(DEFAULT_SIGNALING_URL) },
                singleLine = true,
                isError = error != null,
                supportingText = {
                    when {
                        error != null -> Text(error!!, color = MaterialTheme.colorScheme.error)
                        saved -> Text("Saqlandi", color = MaterialTheme.colorScheme.primary)
                        else -> Text("wss:// yoki ws:// bilan boshlanishi shart")
                    }
                },
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(8.dp))
            Text(
                "Standart: $DEFAULT_SIGNALING_URL",
                style = MaterialTheme.typography.bodySmall,
                fontFamily = FontFamily.Monospace,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )

            Spacer(Modifier.height(16.dp))
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
                                error = "URL wss:// yoki ws:// bilan boshlanishi kerak"
                            else -> {
                                vm.saveSignalUrl(target)
                                draft = target
                                saved = true
                                error = null
                            }
                        }
                    },
                    modifier = Modifier.weight(1f),
                ) { Text("Saqlash") }
                TextButton(
                    onClick = {
                        draft = DEFAULT_SIGNALING_URL
                        vm.saveSignalUrl(DEFAULT_SIGNALING_URL)
                        saved = true
                        error = null
                    },
                ) { Text("Standartga qaytarish") }
            }
        }
    }
}
