package com.wormhole

import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.os.Environment
import android.os.IBinder
import androidx.core.app.NotificationCompat
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

/**
 * Foreground service that runs the wormhole receive operation.
 * Started from a notification action tap; auto-cancels when done.
 * Minimal resource footprint: runs only during the active transfer.
 */
class ReceiveService : Service() {

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val code     = intent?.getStringExtra(EXTRA_CODE)     ?: return START_NOT_STICKY
        val codeId   = intent.getStringExtra(EXTRA_CODE_ID)   ?: ""
        val filename = intent.getStringExtra(EXTRA_FILENAME)  ?: "файл"
        val notifId  = intent.getIntExtra(EXTRA_NOTIF_ID, FcmService.NOTIF_ID)

        // Cancel the "Accept/Decline" notification immediately.
        getSystemService(NotificationManager::class.java).cancel(notifId)

        // Show a minimal "Receiving…" foreground notification.
        startForeground(PROGRESS_NOTIF_ID, buildProgressNotification(filename))

        CoroutineScope(Dispatchers.IO).launch {
            val downloadsDir = Environment
                .getExternalStoragePublicDirectory(Environment.DIRECTORY_DOWNLOADS)
                .absolutePath

            WormholeLib.receiveFile(code, downloadsDir, object : WormholeLib.ReceiveCallback {
                override fun onProgress(received: Long, total: Long) {}

                override fun onError(msg: String) {
                    showResult("❌ Ошибка: $msg", filename)
                    stopSelf()
                }

                override fun onDone(savedPath: String) {
                    RelayClient.ackCode(applicationContext, codeId)
                    showResult("✅ Получено: $filename", "Сохранён в Загрузки")
                    stopSelf()
                }
            })
        }

        return START_NOT_STICKY
    }

    private fun buildProgressNotification(filename: String) =
        NotificationCompat.Builder(this, WormholeApp.CHANNEL_PROGRESS)
            .setSmallIcon(R.drawable.ic_wormhole)
            .setContentTitle("Получение файла…")
            .setContentText(filename)
            .setOngoing(true)
            .build()

    private fun showResult(title: String, text: String) {
        val nm = getSystemService(NotificationManager::class.java)
        nm.cancel(PROGRESS_NOTIF_ID)
        val n = NotificationCompat.Builder(this, WormholeApp.CHANNEL_INCOMING)
            .setSmallIcon(R.drawable.ic_wormhole)
            .setContentTitle(title)
            .setContentText(text)
            .setAutoCancel(true)
            .build()
        nm.notify(RESULT_NOTIF_ID, n)
    }

    companion object {
        const val EXTRA_CODE      = "code"
        const val EXTRA_CODE_ID   = "code_id"
        const val EXTRA_FILENAME  = "filename"
        const val EXTRA_NOTIF_ID  = "notif_id"
        const val PROGRESS_NOTIF_ID = 1002
        const val RESULT_NOTIF_ID   = 1003
    }
}
