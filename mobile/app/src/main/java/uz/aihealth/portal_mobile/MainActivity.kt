package uz.aihealth.portal_mobile

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import uz.aihealth.portal_mobile.i18n.LocalLang
import uz.aihealth.portal_mobile.mesh.MeshState
import uz.aihealth.portal_mobile.ui.AuthScreen
import uz.aihealth.portal_mobile.ui.JoinScreen
import uz.aihealth.portal_mobile.ui.Logo
import uz.aihealth.portal_mobile.ui.PortalScreen
import uz.aihealth.portal_mobile.ui.PortalViewModel
import uz.aihealth.portal_mobile.ui.SettingsScreen
import uz.aihealth.portal_mobile.ui.WelcomeScreen
import uz.aihealth.portal_mobile.ui.theme.Portal_mobileTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            Portal_mobileTheme {
                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    PortalApp(modifier = Modifier.padding(innerPadding))
                }
            }
        }
    }
}

private object Routes {
    const val WELCOME = "welcome"
    const val JOIN = "join"
    const val PORTAL = "portal"
    const val SETTINGS = "settings"
}

/**
 * Top-level composition. Mandatory-login mode means the auth gate is
 * the outermost decision: until the user has a valid session, *only*
 * AuthScreen is reachable. The internal NavHost (welcome/join/portal/
 * settings) is mounted only after sign-in.
 *
 * Why a top-level if/else and not just a NavHost route — the gate is
 * stateful and global; routing AUTH inside the same nav graph as the
 * portal screens would let back-stack history accumulate and create
 * weird back-button behaviour ("the user signs in then presses back
 * to land on the auth screen they thought they'd left"). Splitting
 * the gate from the rest of the app keeps the back stack clean.
 */
@Composable
fun PortalApp(modifier: Modifier = Modifier) {
    val vm: PortalViewModel = viewModel()
    val lang by vm.lang.collectAsState()
    val authReady by vm.authReady.collectAsState()
    val signedInUser by vm.signedInUser.collectAsState()

    CompositionLocalProvider(LocalLang provides lang) {
        when {
            !authReady -> SplashScreen(modifier)
            signedInUser == null -> AuthScreen(
                vm = vm,
                // onDone fires when the Auth screen wants to dismiss
                // itself, but we don't pop anywhere — the gate flips
                // automatically once vm.signedInUser becomes non-null
                // and this branch stops being chosen.
                onDone = {},
                // Pass the Scaffold's innerPadding through so AuthScreen
                // doesn't draw under the status bar / system gestures.
                modifier = modifier,
            )
            else -> SignedInApp(vm = vm, modifier = modifier)
        }
    }
}

/**
 * Brief splash while we read the cached session and (best-effort) hit
 * /api/me. Usually < 200 ms; long enough on a cold cache to avoid
 * flashing the AuthScreen for a frame before the cached identity loads.
 */
@Composable
private fun SplashScreen(modifier: Modifier = Modifier) {
    Box(
        modifier = modifier.fillMaxSize().padding(48.dp),
        contentAlignment = Alignment.Center,
    ) {
        Logo(size = 120.dp)
    }
}

@Composable
private fun SignedInApp(vm: PortalViewModel, modifier: Modifier = Modifier) {
    val nav = rememberNavController()
    val meshState by vm.meshState.collectAsState()

    // Navigate to portal automatically once we leave Idle, and back to
    // welcome on Closed. Failed states still show on the portal screen
    // so the user can read the error.
    LaunchedEffect(meshState) {
        when (meshState) {
            is MeshState.Connecting,
            is MeshState.Ready,
            is MeshState.Reconnecting -> {
                if (nav.currentDestination?.route != Routes.PORTAL) {
                    nav.navigate(Routes.PORTAL) {
                        popUpTo(Routes.WELCOME)
                    }
                }
            }
            is MeshState.Closed -> {
                nav.popBackStack(Routes.WELCOME, inclusive = false)
            }
            else -> {}
        }
    }

    NavHost(
        navController = nav,
        startDestination = Routes.WELCOME,
        modifier = modifier,
    ) {
        composable(Routes.WELCOME) {
            WelcomeScreen(
                vm = vm,
                onCreate = { vm.createPortal() },
                onGoToJoin = { nav.navigate(Routes.JOIN) },
                onGoToSettings = { nav.navigate(Routes.SETTINGS) },
            )
        }
        composable(Routes.JOIN) {
            JoinScreen(
                vm = vm,
                onJoin = { /* state-driven nav handles this */ },
                onBack = { nav.popBackStack() },
            )
        }
        composable(Routes.PORTAL) {
            PortalScreen(
                vm = vm,
                onLeave = { /* state-driven nav handles this */ },
            )
        }
        composable(Routes.SETTINGS) {
            SettingsScreen(
                vm = vm,
                onBack = { nav.popBackStack() },
            )
        }
    }
}
