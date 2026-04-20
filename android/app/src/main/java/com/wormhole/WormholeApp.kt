package com.wormhole

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import android.os.Build
import com.google.firebase.FirebaseApp
import io.sentry.android.core.SentryAndroid

class WormholeApp : Application() {

    override fun onCreate() {
        super.onCreate()
        try {
            SentryAndroid.init(this) { options ->
                options.dsn = "https://d3c95e3fc6f8be0d32b42244de016180@o4504272346480640.ingest.us.sentry.io/4511254231973888"
            }
        } catch (_: Exception) {}
        FirebaseApp.initializeApp(this)
        createNotificationChannels()
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
