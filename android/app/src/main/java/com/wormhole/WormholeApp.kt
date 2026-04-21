package com.wormhole

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.os.Build
import com.google.firebase.auth.FirebaseAuth
import com.google.firebase.FirebaseApp

class WormholeApp : Application() {

    override fun onCreate() {
        super.onCreate()
        // Sentry is auto-initialized from AndroidManifest.xml meta-data
        // before this point via SentryInitProvider ContentProvider.
        try {
            FirebaseApp.initializeApp(this)
        } catch (e: Exception) {
            io.sentry.Sentry.captureException(e)
        }
        try {
            createNotificationChannels()
        } catch (e: Exception) {
            io.sentry.Sentry.captureException(e)
        }
        try {
            if (FirebaseAuth.getInstance().currentUser != null) {
                // Re-register on every app start so relay state survives backend/db resets.
                RelayClient.registerDevice(applicationContext)
            }
        } catch (e: Exception) {
            io.sentry.Sentry.captureException(e)
        }
    }

    private fun createNotificationChannels() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val nm = getSystemService(NotificationManager::class.java)

        nm.createNotificationChannel(
            NotificationChannel(
                CHANNEL_INCOMING,
                "Входящие файлы",
                NotificationManager.IMPORTANCE_HIGH
            ).apply { description = "Уведомления о входящих wormhole-передачах" }
        )
        nm.createNotificationChannel(
            NotificationChannel(
                CHANNEL_PROGRESS,
                "Прогресс передачи",
                NotificationManager.IMPORTANCE_LOW
            ).apply { description = "Прогресс текущей передачи файла" }
        )
    }

    companion object {
        const val CHANNEL_INCOMING = "incoming"
        const val CHANNEL_PROGRESS = "progress"
    }
}
