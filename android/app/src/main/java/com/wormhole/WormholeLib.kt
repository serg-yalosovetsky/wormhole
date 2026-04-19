package com.wormhole

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

/**
 * Thin Kotlin wrapper around the gomobile-generated wormhole AAR.
 * All callbacks arrive on a background thread; callers must switch to Main if needed.
 */
object WormholeLib {

    interface SendCallback {
        fun onCode(code: String)
        fun onProgress(sent: Long, total: Long)
        fun onError(msg: String)
        fun onDone()
    }

    interface ReceiveCallback {
        fun onProgress(received: Long, total: Long)
        fun onError(msg: String)
        fun onDone(savedPath: String)
    }

    // gomobile wraps the Go interfaces:
    //   wormhole.SendCallback  → Wormhole.SendCallback (Java interface)
    //   wormhole.ReceiveCallback → Wormhole.ReceiveCallback (Java interface)

    fun sendFile(path: String, cb: SendCallback) {
        CoroutineScope(Dispatchers.IO).launch {
            wormhole.Wormhole.sendFile(path, object : wormhole.SendCallback {
                override fun onCode(code: String)              = cb.onCode(code)
                override fun onProgress(sent: Long, total: Long) = cb.onProgress(sent, total)
                override fun onError(msg: String)              = cb.onError(msg)
                override fun onDone()                          = cb.onDone()
            })
        }
    }

    fun receiveFile(code: String, destDir: String, cb: ReceiveCallback) {
        CoroutineScope(Dispatchers.IO).launch {
            wormhole.Wormhole.receiveFile(code, destDir, object : wormhole.ReceiveCallback {
                override fun onProgress(received: Long, total: Long) = cb.onProgress(received, total)
                override fun onError(msg: String)                    = cb.onError(msg)
                override fun onDone(savedPath: String)               = cb.onDone(savedPath)
            })
        }
    }
}
