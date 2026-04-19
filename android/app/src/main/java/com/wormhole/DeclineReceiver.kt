package com.wormhole

import android.app.NotificationManager
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

class DeclineReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        val codeId  = intent.getStringExtra(EXTRA_CODE_ID)  ?: return
        val notifId = intent.getIntExtra(EXTRA_NOTIF_ID, -1)

        CoroutineScope(Dispatchers.IO).launch {
            RelayClient.ackCode(context, codeId)
        }

        if (notifId != -1) {
            context.getSystemService(NotificationManager::class.java).cancel(notifId)
        }
    }

    companion object {
        const val EXTRA_CODE_ID  = "code_id"
        const val EXTRA_NOTIF_ID = "notif_id"
    }
}
