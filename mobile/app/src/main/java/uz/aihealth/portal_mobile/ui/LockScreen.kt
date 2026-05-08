package uz.aihealth.portal_mobile.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.input.KeyboardType

/**
 * Web-account lock screen. Sits in front of [WelcomeScreen] until the
 * /api/me probe in [PortalViewModel] confirms a valid session.
 *
 * Two tabs (Sign-In / Sign-Up) share one form. Sign-Up adds a confirm-
 * password field and a live 5-rule strength checklist that mirrors the
 * server-side validatePasswordStrength rules — what turns green here
 * is what the API will accept on submit. Lockout is server-driven:
 * after 5 wrong sign-in attempts the API returns `locked_out` with a
 * countdown, the submit button disables, and the banner ticks down.
 */
@Composable
fun LockScreen(
    vm: PortalViewModel,
) {
    val busy by vm.authBusy.collectAsState()
    val error by vm.authError.collectAsState()
    val lockoutSec by vm.authLockoutSeconds.collectAsState()

    var tab by remember { mutableStateOf(LockTab.SignIn) }
    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    // Two independent reveal flags so the password and confirm fields
    // toggle separately — flipping one shouldn't unmask the other.
    var showPassword by remember { mutableStateOf(false) }
    var showConfirm by remember { mutableStateOf(false) }
    var confirm by remember { mutableStateOf("") }
    var localValidation by remember { mutableStateOf<String?>(null) }

    val strength = remember(password) { evalStrength(password) }
    val allStrong = strength.length && strength.lower &&
        strength.upper && strength.digit && strength.special

    fun submit() {
        localValidation = null
        vm.clearAuthError()
        if (tab == LockTab.SignUp) {
            if (username.isBlank()) {
                localValidation = "username_invalid"; return
            }
            if (!allStrong) {
                localValidation = if (password.length < 8) "password_too_short" else "password_weak"
                return
            }
            if (password != confirm) {
                localValidation = "password_mismatch"; return
            }
            vm.signUp(username, password)
        } else {
            if (username.isBlank() || password.isBlank()) {
                localValidation = "invalid_credentials"; return
            }
            vm.signIn(username, password)
        }
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 24.dp, vertical = 32.dp),
        ) {
            // Logo + brand
            Box(
                modifier = Modifier
                    .size(72.dp)
                    .background(
                        brush = androidx.compose.ui.graphics.Brush.linearGradient(
                            listOf(Color(0xFF8B5CF6), Color(0xFF22D3EE)),
                        ),
                        shape = RoundedCornerShape(18.dp),
                    )
                    .align(Alignment.CenterHorizontally),
            )
            Spacer(Modifier.height(12.dp))
            Text(
                "Portal",
                style = MaterialTheme.typography.headlineMedium.copy(fontWeight = FontWeight.ExtraBold),
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(20.dp))

            Card(
                shape = RoundedCornerShape(16.dp),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f),
                ),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Column(modifier = Modifier.padding(20.dp)) {
                    // Tab segmented control.
                    Surface(
                        color = Color.Black.copy(alpha = 0.25f),
                        shape = RoundedCornerShape(10.dp),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Row(modifier = Modifier.padding(4.dp)) {
                            TabPill(
                                label = "Kirish",
                                active = tab == LockTab.SignIn,
                                enabled = !busy,
                                onClick = {
                                    tab = LockTab.SignIn
                                    password = ""; confirm = ""
                                    localValidation = null
                                    vm.clearAuthError()
                                },
                                modifier = Modifier.weight(1f),
                            )
                            TabPill(
                                label = "Ro'yxatdan o'tish",
                                active = tab == LockTab.SignUp,
                                enabled = !busy,
                                onClick = {
                                    tab = LockTab.SignUp
                                    password = ""; confirm = ""
                                    localValidation = null
                                    vm.clearAuthError()
                                },
                                modifier = Modifier.weight(1f),
                            )
                        }
                    }
                    Spacer(Modifier.height(16.dp))

                    Text(
                        if (tab == LockTab.SignUp)
                            "Akkaunt yaratib qo'ying — keyingi safar parol bilan kirasiz."
                        else
                            "Akkauntingizga kiring.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.fillMaxWidth(),
                        textAlign = TextAlign.Center,
                    )
                    Spacer(Modifier.height(16.dp))

                    OutlinedTextField(
                        value = username,
                        onValueChange = { username = it.take(24) },
                        label = { Text("Foydalanuvchi nomi") },
                        singleLine = true,
                        enabled = !busy,
                        keyboardOptions = KeyboardOptions(
                            keyboardType = KeyboardType.Text,
                            capitalization = KeyboardCapitalization.None,
                        ),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Spacer(Modifier.height(10.dp))
                    OutlinedTextField(
                        value = password,
                        onValueChange = { password = it.take(128) },
                        label = { Text("Parol") },
                        singleLine = true,
                        enabled = !busy,
                        visualTransformation = if (showPassword) VisualTransformation.None
                                               else PasswordVisualTransformation(),
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                        trailingIcon = {
                            IconButton(onClick = { showPassword = !showPassword }) {
                                Icon(
                                    imageVector = if (showPassword) Icons.Filled.VisibilityOff
                                                  else Icons.Filled.Visibility,
                                    contentDescription = if (showPassword) "Parolni yashirish"
                                                         else "Parolni ko'rsatish",
                                )
                            }
                        },
                        modifier = Modifier.fillMaxWidth(),
                    )

                    if (tab == LockTab.SignUp) {
                        Spacer(Modifier.height(10.dp))
                        OutlinedTextField(
                            value = confirm,
                            onValueChange = { confirm = it.take(128) },
                            label = { Text("Parolni takrorlang") },
                            singleLine = true,
                            enabled = !busy,
                            visualTransformation = if (showConfirm) VisualTransformation.None
                                                   else PasswordVisualTransformation(),
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                            trailingIcon = {
                                IconButton(onClick = { showConfirm = !showConfirm }) {
                                    Icon(
                                        imageVector = if (showConfirm) Icons.Filled.VisibilityOff
                                                      else Icons.Filled.Visibility,
                                        contentDescription = if (showConfirm) "Parolni yashirish"
                                                             else "Parolni ko'rsatish",
                                    )
                                }
                            },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        Spacer(Modifier.height(10.dp))
                        StrengthChecklist(strength)
                    }

                    val errCode = localValidation ?: error
                    if (errCode != null || lockoutSec > 0) {
                        Spacer(Modifier.height(12.dp))
                        ErrorBanner(
                            text = if (lockoutSec > 0) {
                                "Juda ko'p urinish. ${lockoutSec} soniyadan so'ng qayta urining."
                            } else {
                                localizeAuthError(errCode)
                            },
                        )
                    }

                    Spacer(Modifier.height(16.dp))
                    Button(
                        onClick = ::submit,
                        enabled = !busy && lockoutSec == 0,
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(48.dp),
                    ) {
                        Text(
                            when {
                                busy -> if (tab == LockTab.SignUp) "Saqlanmoqda..." else "Tekshirilmoqda..."
                                lockoutSec > 0 -> "Kirish (${lockoutSec}s)"
                                tab == LockTab.SignUp -> "Davom etish"
                                else -> "Kirish"
                            },
                        )
                    }

                    if (tab == LockTab.SignUp) {
                        Spacer(Modifier.height(8.dp))
                        Text(
                            "Eslatma: parol unutsangiz, tiklash mumkin emas.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            textAlign = TextAlign.Center,
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }
                }
            }
        }
    }
}

private enum class LockTab { SignIn, SignUp }

private data class StrengthState(
    val length: Boolean,
    val lower: Boolean,
    val upper: Boolean,
    val digit: Boolean,
    val special: Boolean,
)

/**
 * Mirror of validatePasswordStrength in server/userstore.go and
 * client/auth.go. Keep the Regex literals in sync if those rules drift.
 */
private fun evalStrength(pwd: String): StrengthState = StrengthState(
    length = pwd.length >= 8,
    lower = pwd.any { it in 'a'..'z' },
    upper = pwd.any { it in 'A'..'Z' },
    digit = pwd.any { it in '0'..'9' },
    special = pwd.any { ch ->
        !(ch in 'a'..'z' || ch in 'A'..'Z' || ch in '0'..'9' || ch.isWhitespace())
    },
)

@Composable
private fun TabPill(
    label: String,
    active: Boolean,
    enabled: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val bg = if (active) MaterialTheme.colorScheme.primary.copy(alpha = 0.30f) else Color.Transparent
    val fg = if (active) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurfaceVariant
    TextButton(
        onClick = onClick,
        enabled = enabled,
        modifier = modifier.height(36.dp),
        colors = androidx.compose.material3.ButtonDefaults.textButtonColors(
            containerColor = bg,
            contentColor = fg,
        ),
        shape = RoundedCornerShape(8.dp),
    ) {
        Text(label, style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.SemiBold))
    }
}

@Composable
private fun StrengthChecklist(s: StrengthState) {
    Card(
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(
            containerColor = Color.Black.copy(alpha = 0.20f),
        ),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 10.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            Text(
                "Parol mezonlari:",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(2.dp))
            StrengthRow("Kamida 8 ta belgi", s.length)
            StrengthRow("Kichik harf (a-z)", s.lower)
            StrengthRow("Katta harf (A-Z)", s.upper)
            StrengthRow("Raqam (0-9)", s.digit)
            StrengthRow("Maxsus belgi (!@#\$...)", s.special)
        }
    }
}

@Composable
private fun StrengthRow(label: String, ok: Boolean) {
    val color = if (ok) Color(0xFF34D399) else MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.6f)
    Row(verticalAlignment = Alignment.CenterVertically) {
        Box(
            modifier = Modifier
                .size(14.dp)
                .border(width = 1.dp, color = color, shape = CircleShape)
                .background(if (ok) color.copy(alpha = 0.15f) else Color.Transparent, CircleShape),
            contentAlignment = Alignment.Center,
        ) {
            if (ok) {
                Text("✓", color = color, style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Bold))
            }
        }
        Spacer(Modifier.width(8.dp))
        Text(label, color = color, style = MaterialTheme.typography.bodySmall)
    }
}

@Composable
private fun ErrorBanner(text: String) {
    Card(
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(
            containerColor = Color(0xFFEF4444).copy(alpha = 0.10f),
        ),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Text(
            text,
            color = Color(0xFFFCA5A5),
            style = MaterialTheme.typography.bodySmall,
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
        )
    }
}

/**
 * Map server-side error codes (and a couple of client-side validation
 * codes) to user-facing Uzbek text. Kept here rather than in a strings
 * resource for now because the rest of the mobile app is hardcoded
 * Uzbek too — if/when we add a string-resource-driven i18n layer this
 * is the first thing to migrate.
 */
private fun localizeAuthError(code: String?): String = when (code) {
    null -> ""
    "username_invalid" -> "Foydalanuvchi nomi 3-24 belgi, faqat harf/raqam/_/-."
    "username_taken" -> "Bu nom band — boshqa nom tanlang."
    "password_too_short" -> "Parol kamida 8 ta belgidan iborat bo'lsin."
    "password_weak" -> "Parol kuchsiz. Mezonlar bajarilsin."
    "password_mismatch" -> "Parollar mos kelmadi."
    "invalid_credentials" -> "Foydalanuvchi nomi yoki parol noto'g'ri."
    "no_session" -> "Sessiya tugagan. Qaytadan kiring."
    "network" -> "Server bilan ulanish bo'lmadi."
    "server_error" -> "Serverda xato."
    else -> code
}
