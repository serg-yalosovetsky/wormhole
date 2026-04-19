package com.wormhole

import android.content.Context
import com.google.firebase.auth.FirebaseAuth
import com.google.firebase.messaging.FirebaseMessaging
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import java.util.UUID

object RelayClient {

    // Set this to your deployed backend URL.
    private const val RELAY_URL = "https://wormhole.ibotz.fun"

    private val http = OkHttpClient()
    private val JSON = "application/json".toMediaType()

    private fun deviceId(context: Context): String {
        val prefs = context.getSharedPreferences("wormhole", Context.MODE_PRIVATE)
        return prefs.getString("device_id", null) ?: run {
            val id = "android-${UUID.randomUUID()}"
            prefs.edit().putString("device_id", id).apply()
            id
        }
    }

    /** Register or refresh the FCM token on the relay backend. */
    fun registerDevice(context: Context, fcmToken: String? = null) {
        CoroutineScope(Dispatchers.IO).launch {
            val uid = FirebaseAuth.getInstance().currentUser?.uid ?: return@launch
            val token = fcmToken ?: runCatching { FirebaseMessaging.getInstance().token.await() }
                .getOrNull() ?: return@launch

            post("/register", mapOf(
                "uid"       to uid,
                "device_id" to deviceId(context),
                "fcm_token" to token,
                "platform"  to "android"
            ))
        }
    }

    /** Notify other devices that this device has a wormhole code ready. */
    fun notifyDevices(context: Context, code: String, filename: String) {
        CoroutineScope(Dispatchers.IO).launch {
            val uid = FirebaseAuth.getInstance().currentUser?.uid ?: return@launch
            post("/notify", mapOf(
                "uid"              to uid,
                "sender_device_id" to deviceId(context),
                "code"             to code,
                "filename"         to filename
            ))
        }
    }

    /** Mark a pending code as handled so it won't appear on other devices. */
    fun ackCode(context: Context, codeId: String) {
        CoroutineScope(Dispatchers.IO).launch {
            val uid = FirebaseAuth.getInstance().currentUser?.uid ?: return@launch
            post("/ack", mapOf(
                "uid"       to uid,
                "device_id" to deviceId(context),
                "code_id"   to codeId
            ))
        }
    }

    private fun post(path: String, body: Map<String, String>) {
        runCatching {
            val json = JSONObject(body).toString()
            val req = Request.Builder()
                .url(RELAY_URL + path)
                .post(json.toRequestBody(JSON))
                .build()
            http.newCall(req).execute().close()
        }
    }
}
