package uz.aihealth.portal_mobile.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay
import uz.aihealth.portal_mobile.auth.AuthErrorCode
import uz.aihealth.portal_mobile.auth.AuthResult
import uz.aihealth.portal_mobile.auth.apiBaseFromSignal
import uz.aihealth.portal_mobile.i18n.t

private enum class AuthMode { SIGN_IN, SIGN_UP }

/**
 * Mapping of a server [AuthResult] onto where its message should land
 * in the UI:
 *
 *   - field-specific errors (username_taken, password_weak, ...) attach
 *     to the relevant field's `supportingText`, so the user sees the
 *     fix exactly where they typed the broken input;
 *   - account-level errors (invalid_credentials, locked_out, network)
 *     surface as a banner under the action button — they aren't tied
 *     to a single field.
 *
 * Returning empty strings via null lets the field reuse its hint.
 */
private data class FieldErrors(
    val username: String? = null,
    val password: String? = null,
    val banner: String? = null,
)

/**
 * Sign-in / sign-up gate. In mandatory-login mode this is the outer
 * composable shown until the user has a valid session — there's no
 * "skip" path.
 *
 * Sign-up shows a second password field (confirm). Submit is disabled
 * until the local form is valid (username ≥ 3, password ≥ 8, and the
 * two passwords match in sign-up). The server is the source of truth
 * for everything else — username uniqueness, password complexity, and
 * the per-username lockout — and surfaces errors via [authResult] which
 * we route via [resolveFieldErrors].
 */
@Composable
fun AuthScreen(
    vm: PortalViewModel,
    onDone: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val signedInUser by vm.signedInUser.collectAsState()
    val authResult by vm.authResult.collectAsState()
    val busy by vm.authBusy.collectAsState()

    LaunchedEffect(signedInUser) {
        if (signedInUser != null) onDone()
    }

    var mode by remember { mutableStateOf(AuthMode.SIGN_IN) }
    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var passwordConfirm by remember { mutableStateOf("") }
    var showPassword by remember { mutableStateOf(false) }
    val focus = LocalFocusManager.current

    // Switching mode resets server-side errors and the confirm field;
    // a stale "wrong password" while staring at the sign-up form is
    // disorienting, and the confirm value is meaningless across modes.
    LaunchedEffect(mode) {
        vm.clearAuthResult()
        passwordConfirm = ""
    }

    val errs = resolveFieldErrors(authResult)

    // Local validation — sets isError on the confirm field once the user
    // has typed enough to compare. We don't show "doesn't match" on an
    // empty confirm, only after they've put a value in.
    val confirmMismatch =
        mode == AuthMode.SIGN_UP && passwordConfirm.isNotEmpty() && passwordConfirm != password
    val localPasswordsOk = mode == AuthMode.SIGN_IN || password == passwordConfirm
    val canSubmit = !busy && username.length >= 3 && password.length >= 8 && localPasswordsOk

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 24.dp),
    ) {
        // Brand strip — small logo + wordmark, sits below the system
        // status bar (the modifier carries the inset).
        Spacer(Modifier.height(20.dp))
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Logo(size = 32.dp, animated = false)
            Spacer(Modifier.width(8.dp))
            Text(
                "Portal",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
            )
        }

        Spacer(Modifier.height(48.dp))

        Text(
            when (mode) {
                AuthMode.SIGN_IN -> t("auth.signin.title")
                AuthMode.SIGN_UP -> t("auth.signup.title")
            },
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            t("auth.banner.required"),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(28.dp))

        OutlinedTextField(
            value = username,
            onValueChange = {
                username = it.trim().take(24)
                // Clear the server "username_taken" / "username_invalid"
                // error as soon as the user edits the field — staring
                // at a red error after fixing it is confusing.
                if (errs.username != null) vm.clearAuthResult()
            },
            label = { Text(t("auth.username")) },
            singleLine = true,
            isError = errs.username != null,
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Ascii,
                imeAction = ImeAction.Next,
            ),
            supportingText = {
                if (errs.username != null) {
                    Text(errs.username, color = MaterialTheme.colorScheme.error)
                } else {
                    Text(t("auth.username_hint"))
                }
            },
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(8.dp))

        OutlinedTextField(
            value = password,
            onValueChange = {
                password = it.take(128)
                if (errs.password != null) vm.clearAuthResult()
            },
            label = { Text(t("auth.password")) },
            singleLine = true,
            isError = errs.password != null,
            visualTransformation = if (showPassword) {
                VisualTransformation.None
            } else {
                PasswordVisualTransformation()
            },
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Password,
                imeAction = if (mode == AuthMode.SIGN_UP) ImeAction.Next else ImeAction.Done,
            ),
            trailingIcon = {
                TextButton(
                    onClick = { showPassword = !showPassword },
                    contentPadding = PaddingValues(horizontal = 10.dp),
                ) {
                    Text(
                        if (showPassword) t("auth.hide_password") else t("auth.show_password"),
                        style = MaterialTheme.typography.labelMedium,
                    )
                }
            },
            supportingText = {
                if (errs.password != null) {
                    Text(errs.password, color = MaterialTheme.colorScheme.error)
                } else {
                    Text(t("auth.password_hint"))
                }
            },
            modifier = Modifier.fillMaxWidth(),
        )

        // Confirm-password field — sign-up only. Mirrors the password
        // field's show/hide toggle so the user can verify both at once.
        if (mode == AuthMode.SIGN_UP) {
            Spacer(Modifier.height(8.dp))
            OutlinedTextField(
                value = passwordConfirm,
                onValueChange = { passwordConfirm = it.take(128) },
                label = { Text(t("auth.password_confirm")) },
                singleLine = true,
                isError = confirmMismatch,
                visualTransformation = if (showPassword) {
                    VisualTransformation.None
                } else {
                    PasswordVisualTransformation()
                },
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.Password,
                    imeAction = ImeAction.Done,
                ),
                supportingText = {
                    when {
                        confirmMismatch -> Text(
                            t("auth.password_mismatch"),
                            color = MaterialTheme.colorScheme.error,
                        )
                        else -> Text(t("auth.password_confirm_hint"))
                    }
                },
                modifier = Modifier.fillMaxWidth(),
            )
        }

        Spacer(Modifier.height(20.dp))

        Button(
            onClick = {
                focus.clearFocus()
                when (mode) {
                    AuthMode.SIGN_IN -> vm.signIn(username, password)
                    AuthMode.SIGN_UP -> vm.signUp(username, password)
                }
            },
            enabled = canSubmit,
            modifier = Modifier
                .fillMaxWidth()
                .height(52.dp),
        ) {
            Text(
                if (busy) t("auth.busy") else when (mode) {
                    AuthMode.SIGN_IN -> t("auth.signin.cta")
                    AuthMode.SIGN_UP -> t("auth.signup.cta")
                },
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.SemiBold,
            )
        }
        Spacer(Modifier.height(4.dp))
        TextButton(
            onClick = {
                mode = if (mode == AuthMode.SIGN_IN) AuthMode.SIGN_UP else AuthMode.SIGN_IN
            },
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(
                when (mode) {
                    AuthMode.SIGN_IN -> t("auth.toggle_to_signup")
                    AuthMode.SIGN_UP -> t("auth.toggle_to_signin")
                },
            )
        }

        // Account-level / network errors (i.e. errors that aren't tied to
        // a specific field) live in a banner below the action.
        if (errs.banner != null) {
            Spacer(Modifier.height(16.dp))
            ErrorBanner(text = errs.banner)
        }

        // The lockout countdown lives outside the banner — it'd be
        // wasted recomposition to redraw the banner card every second.
        if (authResult is AuthResult.Err &&
            (authResult as AuthResult.Err).code == AuthErrorCode.LOCKED_OUT &&
            (authResult as AuthResult.Err).lockoutSeconds > 0
        ) {
            LockoutCountdownEffect(authResult as AuthResult.Err)
        }

        Spacer(Modifier.weight(1f, fill = true))
        Spacer(Modifier.height(32.dp))
        Text(
            t("auth.connecting_to", apiBaseFromSignal(vm.signalUrl)),
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            fontFamily = FontFamily.Monospace,
            textAlign = TextAlign.Center,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(20.dp))
    }
}

/** Localizes the [AuthResult] and routes it to the right slot. */
@Composable
private fun resolveFieldErrors(r: AuthResult?): FieldErrors {
    if (r == null) return FieldErrors()
    return when (r) {
        is AuthResult.Ok -> FieldErrors()
        is AuthResult.NetworkError -> FieldErrors(banner = t("auth.err.network", r.message))
        is AuthResult.Err -> when (r.code) {
            AuthErrorCode.USERNAME_TAKEN -> FieldErrors(username = t("auth.err.username_taken"))
            AuthErrorCode.USERNAME_INVALID -> FieldErrors(username = t("auth.err.username_invalid"))
            AuthErrorCode.PASSWORD_TOO_SHORT -> FieldErrors(password = t("auth.err.password_too_short"))
            AuthErrorCode.PASSWORD_WEAK -> FieldErrors(password = t("auth.err.password_weak"))
            AuthErrorCode.INVALID_CREDENTIALS -> FieldErrors(banner = t("auth.err.invalid_credentials"))
            AuthErrorCode.LOCKED_OUT -> FieldErrors(banner = t("auth.err.locked_out", r.lockoutSeconds))
            AuthErrorCode.NO_SESSION -> FieldErrors(banner = t("auth.err.no_session"))
            AuthErrorCode.BAD_REQUEST -> FieldErrors(banner = t("auth.err.bad_request"))
            AuthErrorCode.SERVER_OUTDATED -> FieldErrors(banner = t("auth.err.server_outdated"))
            AuthErrorCode.UNKNOWN -> FieldErrors(banner = t("auth.err.unknown"))
        }
    }
}

@Composable
private fun ErrorBanner(text: String) {
    Card(
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.errorContainer,
        ),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Box(modifier = Modifier.padding(12.dp)) {
            Text(
                "✗ $text",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onErrorContainer,
            )
        }
    }
}

@Composable
private fun LockoutCountdownEffect(err: AuthResult.Err) {
    var remaining by remember(err) { mutableStateOf(err.lockoutSeconds) }
    LaunchedEffect(err) {
        while (remaining > 0) {
            delay(1_000)
            remaining -= 1
        }
    }
}
