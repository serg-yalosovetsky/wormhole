package com.wormhole

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.view.Gravity
import android.view.View
import android.widget.Button
import android.widget.LinearLayout
import android.widget.TextView
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.google.android.gms.auth.api.signin.GoogleSignIn
import com.google.android.gms.auth.api.signin.GoogleSignInOptions
import com.google.android.gms.common.api.ApiException
import com.google.firebase.auth.FirebaseAuth
import com.google.firebase.auth.GoogleAuthProvider
import kotlinx.coroutines.launch

/**
 * Shown for initial Google sign-in and as a status screen when already signed in.
 * Everything else happens via the share sheet and notifications.
 */
class MainActivity : AppCompatActivity() {

    private val RC_SIGN_IN = 9001
    private val auth by lazy { FirebaseAuth.getInstance() }
    private val currentPending = mutableListOf<RelayClient.PendingCode>()

    private val notifPermLauncher = registerForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) { /* granted or denied — system manages the setting */ }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)
        requestNotificationPermission()

        if (auth.currentUser != null) {
            showSignedInState()
        } else {
            showSignInState()
        }
    }

    override fun onResume() {
        super.onResume()
        if (auth.currentUser != null) {
            refreshPending()
        }
    }

    private fun showSignedInState() {
        findViewById<View>(R.id.signInGroup).visibility = View.GONE
        findViewById<View>(R.id.signedInGroup).visibility = View.VISIBLE
        findViewById<TextView>(R.id.tvUserEmail).text = auth.currentUser?.email ?: ""
        findViewById<Button>(R.id.btnSignOut).setOnClickListener { signOut() }
        // Ensure FCM token is registered in case it was refreshed while not signed in.
        RelayClient.registerDevice(this)
        refreshPending()
    }

    private fun showSignInState() {
        findViewById<View>(R.id.signInGroup).visibility = View.VISIBLE
        findViewById<View>(R.id.signedInGroup).visibility = View.GONE
        findViewById<Button>(R.id.btnSignIn).setOnClickListener { startSignIn() }
    }

    private fun refreshPending() {
        lifecycleScope.launch {
            val codes = RelayClient.pollPending(this@MainActivity)
            renderPending(codes)
        }
    }

    private fun renderPending(codes: List<RelayClient.PendingCode>) {
        currentPending.clear()
        currentPending.addAll(codes)

        val list   = findViewById<LinearLayout>(R.id.pendingList)
        val status = findViewById<TextView>(R.id.tvPendingStatus)

        list.removeAllViews()

        if (codes.isEmpty()) {
            status.text = "Нет ожидающих файлов"
            status.visibility = View.VISIBLE
            list.visibility = View.GONE
            return
        }

        status.visibility = View.GONE
        list.visibility = View.VISIBLE

        val dp8 = (8 * resources.displayMetrics.density).toInt()

        for (pending in codes) {
            val row = LinearLayout(this).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT
                ).also { it.topMargin = dp8 }
            }

            val tvName = TextView(this).apply {
                text = pending.filename
                textSize = 14f
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
            }

            val btnAccept = Button(this).apply {
                text = "Принять"
                setOnClickListener { acceptFile(pending) }
            }

            val btnDecline = Button(this).apply {
                text = "Отклонить"
                setOnClickListener { declineFile(pending) }
            }

            row.addView(tvName)
            row.addView(btnAccept)
            row.addView(btnDecline)
            list.addView(row)
        }
    }

    private fun acceptFile(pending: RelayClient.PendingCode) {
        // Ack immediately so the item disappears from the queue on all devices.
        // ReceiveService will also call ackCode on success — that second call is a no-op.
        RelayClient.ackCode(this, pending.id)
        val intent = Intent(this, ReceiveService::class.java).apply {
            putExtra(ReceiveService.EXTRA_CODE, pending.code)
            putExtra(ReceiveService.EXTRA_CODE_ID, pending.id)
            putExtra(ReceiveService.EXTRA_FILENAME, pending.filename)
        }
        ContextCompat.startForegroundService(this, intent)
        removeFromList(pending)
    }

    private fun declineFile(pending: RelayClient.PendingCode) {
        RelayClient.ackCode(this, pending.id)
        removeFromList(pending)
    }

    private fun removeFromList(pending: RelayClient.PendingCode) {
        currentPending.remove(pending)
        renderPending(currentPending)
    }

    private fun requestNotificationPermission() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) {
            notifPermLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
        }
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
