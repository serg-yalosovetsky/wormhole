package com.wormhole

import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.provider.OpenableColumns
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import com.google.firebase.auth.FirebaseAuth
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.io.File

/**
 * Invisible activity registered as a share target.
 * Receives ACTION_SEND, copies the file to cache, starts wormhole send,
 * notifies other devices, then finishes immediately.
 */
class ShareActivity : AppCompatActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        if (FirebaseAuth.getInstance().currentUser == null) {
            Toast.makeText(this, "Сначала войдите в аккаунт", Toast.LENGTH_LONG).show()
            startActivity(Intent(this, MainActivity::class.java))
            finish()
            return
        }

        val uri: Uri? = intent.getParcelableExtra(Intent.EXTRA_STREAM)
        if (uri == null) {
            finish()
            return
        }

        Toast.makeText(this, "Wormhole: отправляем…", Toast.LENGTH_SHORT).show()

        CoroutineScope(Dispatchers.IO).launch {
            val file = copyToCache(uri) ?: run {
                withContext(Dispatchers.Main) {
                    Toast.makeText(this@ShareActivity, "Не удалось прочитать файл", Toast.LENGTH_LONG).show()
                }
                return@launch
            }

            WormholeLib.sendFile(file.absolutePath, object : WormholeLib.SendCallback {
                override fun onCode(code: String) {
                    // Notify other devices as soon as we have the code.
                    RelayClient.notifyDevices(applicationContext, code, file.name)
                    CoroutineScope(Dispatchers.Main).launch {
                        Toast.makeText(applicationContext, "Код: $code", Toast.LENGTH_LONG).show()
                    }
                }
                override fun onProgress(sent: Long, total: Long) {}
                override fun onError(msg: String) {
                    CoroutineScope(Dispatchers.Main).launch {
                        Toast.makeText(applicationContext, "Ошибка: $msg", Toast.LENGTH_LONG).show()
                    }
                }
                override fun onDone() {
                    file.delete()
                }
            })
        }

        finish() // Activity returns immediately; transfer runs in coroutine.
    }

    private fun copyToCache(uri: Uri): File? {
        val name = queryFileName(uri) ?: "wormhole_file"
        val dest = File(cacheDir, name)
        return try {
            contentResolver.openInputStream(uri)?.use { input ->
                dest.outputStream().use { output -> input.copyTo(output) }
            }
            dest
        } catch (e: Exception) {
            null
        }
    }

    private fun queryFileName(uri: Uri): String? {
        val cursor = contentResolver.query(uri, null, null, null, null) ?: return null
        return cursor.use {
            if (it.moveToFirst()) {
                val idx = it.getColumnIndex(OpenableColumns.DISPLAY_NAME)
                if (idx >= 0) it.getString(idx) else null
            } else null
        }
    }
}
