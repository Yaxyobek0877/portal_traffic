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
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import uz.aihealth.portal_mobile.mesh.MeshState
import uz.aihealth.portal_mobile.ui.JoinScreen
import uz.aihealth.portal_mobile.ui.PortalScreen
import uz.aihealth.portal_mobile.ui.PortalViewModel
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
}

@Composable
fun PortalApp(modifier: Modifier = Modifier) {
    val nav = rememberNavController()
    val vm: PortalViewModel = viewModel()
    val meshState by vm.meshState.collectAsState()

    // Navigate to portal automatically once we leave Idle, and back to
    // welcome on Closed. Failed states still show on the portal screen
    // so the user can read the error.
    LaunchedEffect(meshState) {
        when (meshState) {
            is MeshState.Connecting, is MeshState.Ready -> {
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
    }
}
