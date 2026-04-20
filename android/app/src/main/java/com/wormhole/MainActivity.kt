package com.wormhole

import android.content.Intent
import android.os.Bundle
import android.widget.Button
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import com.google.android.gms.auth.api.signin.GoogleSignIn
import com.google.android.gms.auth.api.signin.GoogleSignInOptions
import com.google.android.gms.common.api.ApiException
import com.google.firebase.auth.FirebaseAuth
import com.google.firebase.auth.GoogleAuthProvider

/**
 * Shown only for initial sign-in. After that the user never needs to open the app —
 * everything happens via the share sheet and notifications.
 */
class MainActivity : AppCompatActivity() {

    private val RC_SIGN_IN = 9001
    private val auth by lazy { FirebaseAuth.getInstance() }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Already signed in → nothing to show, go straight to the background.
        if (auth.currentUser != null) {
            finish()
            return
        }

        setContentView(R.layout.activity_main)

        findViewById<Button>(R.id.btnSignIn).setOnClickListener { startSignIn() }
        updateStatus()
    }

    private fun updateStatus() {
        val user = auth.currentUser
        findViewById<TextView>(R.id.tvStatus).text =
            if (user != null) "Вошли как ${user.email}" else "Не вошли"
    }

    private fun startSignIn() {
        val gso = GoogleSignInOptions.Builder(GoogleSignInOptions.DEFAULT_SIGN_IN)
            .requestIdToken(getString(R.string.default_web_client_id))
            .requestEmail()
            .build()
        val client = GoogleSignIn.getClient(this, gso)
        startActivityForResult(client.signInIntent, RC_SIGN_IN)
    }

    @Deprecated("Deprecated in Java")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode != RC_SIGN_IN) return

        val task = GoogleSignIn.getSignedInAccountFromIntent(data)
        try {
            val account = task.getResult(ApiException::class.java)
            val credential = GoogleAuthProvider.getCredential(account.idToken, null)
            auth.signInWithCredential(credential).addOnCompleteListener(this) { t ->
                if (t.isSuccessful) {
                    Toast.makeText(this, "Готово! Теперь можно закрыть приложение.", Toast.LENGTH_LONG).show()
                    updateStatus()
                    // Register FCM token now that we have a UID.
                    RelayClient.registerDevice(this)
                    finish()
                } else {
                    Toast.makeText(this, "Ошибка входа: ${t.exception?.message}", Toast.LENGTH_LONG).show()
                    t.exception?.let { io.sentry.Sentry.captureException(it) }
                }
            }
        } catch (e: ApiException) {
            Toast.makeText(this, "Ошибка Google: ${e.statusCode}", Toast.LENGTH_LONG).show()
            io.sentry.Sentry.captureException(e)
        }
    }
}
