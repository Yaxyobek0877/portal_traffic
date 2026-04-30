package uz.aihealth.portal_mobile.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import uz.aihealth.portal_mobile.MainActivity
import uz.aihealth.portal_mobile.R

/**
 * Lightweight foreground service whose only job is to keep this process
 * alive while the user is in a portal. Without it, Android will
 * kill the app within ~30 s of being backgrounded — the WebSocket and
 * every WebRTC peer connection would die with it.
 *
 * The mesh logic itself stays in [uz.aihealth.portal_mobile.mesh.MeshManager]
 * (held by the ViewModel). The service is just a process-priority
 * anchor with a user-visible notification.
 */
class MeshService : Service() {

    override fun onCreate() {
        super.onCreate()
        ensureChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val text = intent?.getStringExtra(EXTRA_STATUS) ?: "Portal ulangan"
        val notif = buildNotification(text)
        // Q+ requires the foregroundServiceType arg; we use dataSync since
        // we're transferring application data over the network. Older OSes
        // ignore the type argument.
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            startForeground(NOTIF_ID, notif, ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC)
        } else {
            startForeground(NOTIF_ID, notif)
        }
        return START_NOT_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun ensureChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val mgr = getSystemService(NotificationManager::class.java)
        if (mgr.getNotificationChannel(CHANNEL_ID) != null) return
        val channel = NotificationChannel(
            CHANNEL_ID,
            "Portal mesh",
            NotificationManager.IMPORTANCE_LOW,
        ).apply {
            description = "Portalga ulanganda ko'rinadi"
            setShowBadge(false)
        }
        mgr.createNotificationChannel(channel)
    }

    private fun buildNotification(text: String): Notification {
        val tap = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java).apply {
                addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP)
            },
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
        )
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("Portal")
            .setContentText(text)
            .setSmallIcon(R.mipmap.ic_launcher)
            .setContentIntent(tap)
            .setOngoing(true)
            .setSilent(true)
            .setCategory(NotificationCompat.CATEGORY_SERVICE)
            .build()
    }

    companion object {
        const val CHANNEL_ID = "portal_mesh"
        const val NOTIF_ID = 1
        const val EXTRA_STATUS = "status"

        fun start(context: Context, status: String? = null) {
            val intent = Intent(context, MeshService::class.java).apply {
                if (status != null) putExtra(EXTRA_STATUS, status)
            }
            ContextCompat.startForegroundService(context, intent)
        }

        fun stop(context: Context) {
            context.stopService(Intent(context, MeshService::class.java))
        }
    }
}
