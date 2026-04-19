package com.wormhole

import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import androidx.core.app.NotificationCompat
import com.google.firebase.auth.FirebaseAuth
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage

class FcmService : FirebaseMessagingService() {

    /** Called when a new FCM token is generated (first launch or token refresh). */
    override fun onNewToken(token: String) {
        RelayClient.registerDevice(applicationContext, token)
    }

    /** Called for data messages — both foreground and background (with high priority). */
    override fun onMessageReceived(msg: RemoteMessage) {
        if (FirebaseAuth.getInstance().currentUser == null) return

        val code     = msg.data["code"]     ?: return
        val filename = msg.data["filename"] ?: "файл"
        val codeId   = msg.data["code_id"]  ?: ""

        showIncomingNotification(code, codeId, filename)
    }

    private fun showIncomingNotification(code: String, codeId: String, filename: String) {
        val nm = getSystemService(NotificationManager::class.java)

        // "Принять" action starts ReceiveService.
        val acceptIntent = Intent(this, ReceiveService::class.java).apply {
            putExtra(ReceiveService.EXTRA_CODE,     code)
            putExtra(ReceiveService.EXTRA_CODE_ID,  codeId)
            putExtra(ReceiveService.EXTRA_FILENAME, filename)
            putExtra(ReceiveService.EXTRA_NOTIF_ID, NOTIF_ID)
        }
        val acceptPi = PendingIntent.getService(
            this, 0, acceptIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        // "Отклонить" action calls /ack and cancels the notification.
        val declineIntent = Intent(this, DeclineReceiver::class.java).apply {
            putExtra(DeclineReceiver.EXTRA_CODE_ID,  codeId)
            putExtra(DeclineReceiver.EXTRA_NOTIF_ID, NOTIF_ID)
        }
        val declinePi = PendingIntent.getBroadcast(
            this, 1, declineIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(this, WormholeApp.CHANNEL_INCOMING)
            .setSmallIcon(R.drawable.ic_wormhole)
            .setContentTitle("📥 Входящий файл")
            .setContentText(filename)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setAutoCancel(false)
            .addAction(0, "Принять", acceptPi)
            .addAction(0, "Отклонить", declinePi)
            .build()

        nm.notify(NOTIF_ID, notification)
    }

    companion object {
        const val NOTIF_ID = 1001
    }
}
