package uz.aihealth.portal_mobile

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.Alignment
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import uz.aihealth.portal_mobile.mesh.MeshState
import uz.aihealth.portal_mobile.ui.AuthState
import uz.aihealth.portal_mobile.ui.JoinScreen
import uz.aihealth.portal_mobile.ui.LockScreen
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

@Composable
fun PortalApp(modifier: Modifier = Modifier) {
    val vm: PortalViewModel = viewModel()
    val authState by vm.authState.collectAsState()

    // Auth-state gate sits ABOVE the nav graph: while we don't know
    // whether the saved cookie is still valid, render a tiny spinner;
    // once we do, show either LockScreen or the regular Welcome→…
    // nav graph. Putting the gate above the NavHost (rather than as
    // a route) means signing out from any screen automatically tears
    // the whole stack down and rebuilds — no leftover Portal screen
    // peeking through behind the lock.
    when (val s = authState) {
        is AuthState.Loading -> AuthLoadingScreen(modifier)
        is AuthState.Anonymous -> LockScreen(vm = vm)
        is AuthState.Authenticated -> SignedInApp(vm = vm, modifier = modifier)
    }
}

/**
 * Pure spinner scene shown for the half-second between "the cached
 * cookie exists" and "the /api/me probe finished".
 */
@Composable
private fun AuthLoadingScreen(modifier: Modifier = Modifier) {
    Box(
        modifier = modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background),
        contentAlignment = Alignment.Center,
    ) {
        CircularProgressIndicator()
    }
}

/**
 * The original WELCOME → JOIN → PORTAL → SETTINGS nav graph, only
 * mounted once the user is signed in. Lifted out of [PortalApp] so
 * the auth gate can swap the whole subtree atomically.
 */
@Composable
private fun SignedInApp(vm: PortalViewModel, modifier: Modifier) {
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
