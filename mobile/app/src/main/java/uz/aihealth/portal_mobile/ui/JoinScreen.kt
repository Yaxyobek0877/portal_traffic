package uz.aihealth.portal_mobile.ui

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.journeyapps.barcodescanner.ScanContract
import com.journeyapps.barcodescanner.ScanOptions

@Composable
fun JoinScreen(
    vm: PortalViewModel,
    onJoin: () -> Unit,
    onBack: () -> Unit,
) {
    var portalId by remember { mutableStateOf("") }
    var code by remember { mutableStateOf("") }
    var scanError by remember { mutableStateOf<String?>(null) }

    val scanner = rememberLauncherForActivityResult(ScanContract()) { result ->
        val raw = result.contents
        if (raw.isNullOrBlank()) {
            // user cancelled or scan failed silently
            return@rememberLauncherForActivityResult
        }
        val parsed = parseInvite(raw)
        if (parsed == null) {
            scanError = "QR ichida 6 raqamli ID va kod topilmadi"
            return@rememberLauncherForActivityResult
        }
        portalId = parsed.first
        code = parsed.second
        scanError = null
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text("Portalga qo'shilish", style = MaterialTheme.typography.headlineMedium)
        Spacer(Modifier.height(24.dp))

        OutlinedButton(
            onClick = {
                val opts = ScanOptions()
                    .setDesiredBarcodeFormats(ScanOptions.QR_CODE)
                    .setPrompt("QR ni kameraga tuting")
                    .setBeepEnabled(false)
                    .setOrientationLocked(false)
                    .setBarcodeImageEnabled(false)
                scanner.launch(opts)
            },
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text("QR kodni skanerlash")
        }
        if (scanError != null) {
            Spacer(Modifier.height(8.dp))
            Text(
                scanError!!,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
            )
        }
        Spacer(Modifier.height(20.dp))
        Text(
            "yoki qo'l bilan kiriting",
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = portalId,
            onValueChange = { portalId = it.filter { c -> c.isDigit() }.take(6) },
            label = { Text("Portal ID (6 raqam)") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(12.dp))
        OutlinedTextField(
            value = code,
            onValueChange = { code = it.filter { c -> c.isDigit() }.take(6) },
            label = { Text("Kod (6 raqam)") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(24.dp))
        Button(
            onClick = {
                vm.joinPortal(portalId, code)
                onJoin()
            },
            enabled = portalId.length == 6 && code.length == 6,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text("Qo'shilish")
        }
        Spacer(Modifier.height(8.dp))
        TextButton(onClick = onBack, modifier = Modifier.fillMaxWidth()) {
            Text("Orqaga")
        }
    }
}
