package uz.aihealth.portal_mobile.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

@Composable
fun WelcomeScreen(
    vm: PortalViewModel,
    onCreate: () -> Unit,
    onGoToJoin: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(
            text = "Portal",
            style = MaterialTheme.typography.displayMedium,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = "To'g'ridan-to'g'ri ulanish. Orada server yo'q.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(48.dp))
        OutlinedTextField(
            value = vm.nickname,
            onValueChange = { vm.nickname = it.take(32) },
            label = { Text("Taxallusingiz") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(24.dp))
        Button(
            onClick = onCreate,
            enabled = vm.nickname.isNotBlank(),
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text("Yangi portal yaratish")
        }
        Spacer(Modifier.height(12.dp))
        OutlinedButton(
            onClick = onGoToJoin,
            enabled = vm.nickname.isNotBlank(),
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text("Mavjud portalga qo'shilish")
        }
    }
}
