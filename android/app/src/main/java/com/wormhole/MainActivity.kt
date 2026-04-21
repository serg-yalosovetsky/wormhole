package com.wormhole

import android.content.Intent
import android.os.Bundle
import android.view.View
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
 * Shown for initial Google sign-in and as a status screen when already signed in.
 * Everything else happens via the share sheet and notifications.
 */
class MainActivity : AppCompatActivity() {

    private val RC_SIGN_IN = 9001
    private val auth by lazy { FirebaseAuth.getInstance() }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        if (auth.currentUser != null) {
            showSignedInState()
        } else {
            showSignInState()
        }
    }

    private fun showSignedInState() {
        findViewById<View>(R.id.signInGroup).visibility = View.GONE
        findViewById<View>(R.id.signedInGroup).visibility = View.VISIBLE
        findViewById<TextView>(R.id.tvUserEmail).text = auth.currentUser?.email ?: ""
        findViewById<Button>(R.id.btnSignOut).setOnClickListener { signOut() }
    }

    private fun showSignInState() {
        findViewById<View>(R.id.signInGroup).visibility = View.VISIBLE
        findViewById<View>(R.id.signedInGroup).visibility = View.GONE
        findViewById<Button>(R.id.btnSignIn).setOnClickListener { startSignIn() }
    }

    private fun signOut() {
        auth.signOut()
        GoogleSignIn.getClient(this, GoogleSignInOptions.DEFAULT_SIGN_IN).signOut()
        showSignInState()
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
                    RelayClient.registerDevice(this)
                    showSignedInState()
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
