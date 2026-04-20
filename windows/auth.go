package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Firebase / Google OAuth constants – replace with your project values.
const (
	firebaseAPIKey    = "AIzaSyDKjqPzxhE3JEsOpOjz_FFCejiK-mSPbOQ"
	googleClientID    = "1473173017-f22r9trf854gm47kb10msqmv8lr5f4ja.apps.googleusercontent.com"
	googleRedirectURI = "http://localhost"
)

// signIn performs Google OAuth2 PKCE flow, exchanges for a Firebase ID token,
// and returns a populated Config.
func signIn(relayURL, deviceID string) Config {
	verifier := pkceVerifier()
	challenge := pkceChallenge(verifier)

	// Find a free local port for the redirect callback.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("cannot bind local port: %v", err))
	}
	port := ln.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/cb", port)

	// Build Google OAuth URL.
	authURL := "https://accounts.google.com/o/oauth2/v2/auth?" + url.Values{
		"client_id":             {googleClientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid email"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}.Encode()

	openBrowser(authURL)

	// Wait for the callback containing the authorization code.
	codeCh := make(chan string, 1)
	srv := &http.Server{}
	http.HandleFunc("/cb", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		fmt.Fprint(w, `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Wormhole — вход выполнен</title>
<link rel="icon" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'%3E%3Crect width='100' height='100' rx='22' fill='%236366f1'/%3E%3Ccircle cx='50' cy='50' r='34' fill='none' stroke='white' stroke-width='6'/%3E%3Ccircle cx='50' cy='50' r='22' fill='none' stroke='white' stroke-width='4' opacity='.6'/%3E%3Ccircle cx='50' cy='50' r='10' fill='none' stroke='white' stroke-width='3' opacity='.35'/%3E%3Ccircle cx='50' cy='50' r='4' fill='white'/%3E%3C/svg%3E">
<style>
  *{box-sizing:border-box;margin:0;padding:0}
  body{background:#0f1117;color:#e8eaf0;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;
       display:flex;align-items:center;justify-content:center;min-height:100vh;padding:24px}
  .card{background:#1a1d27;border:1px solid #2a2d3a;border-radius:16px;padding:40px 48px;
        max-width:480px;width:100%;text-align:center;box-shadow:0 8px 32px rgba(0,0,0,.4)}
  .icon{width:72px;height:72px;margin:0 auto 20px;border-radius:16px;overflow:hidden}
  .icon svg{width:100%;height:100%}
  h1{font-size:22px;font-weight:700;margin-bottom:8px;color:#fff}
  .badge{display:inline-block;background:#22c55e22;color:#22c55e;border:1px solid #22c55e44;
         border-radius:20px;padding:4px 14px;font-size:13px;font-weight:600;margin-bottom:24px}
  .divider{height:1px;background:#2a2d3a;margin:24px 0}
  h2{font-size:14px;font-weight:600;color:#9ca3af;text-transform:uppercase;letter-spacing:.06em;margin-bottom:14px}
  ol{text-align:left;padding-left:20px;line-height:1.8;color:#c9cdd8;font-size:14px}
  ol li{margin-bottom:6px}
  kbd{background:#252836;border:1px solid #3a3d4a;border-radius:5px;padding:1px 6px;
      font-size:12px;font-family:monospace;color:#e2e8f0}
  .close-hint{margin-top:24px;font-size:13px;color:#6b7280}
</style>
</head>
<body>
<div class="card">
  <div class="icon">
    <svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="g" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#6366f1"/>
          <stop offset="100%" stop-color="#7c3aed"/>
        </linearGradient>
        <radialGradient id="void" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="#1e1b4b"/>
          <stop offset="100%" stop-color="#1e1b4b" stop-opacity="0"/>
        </radialGradient>
      </defs>
      <rect width="100" height="100" rx="22" fill="url(#g)"/>
      <circle cx="50" cy="50" r="34" fill="none" stroke="#fff" stroke-width="6" opacity=".95"/>
      <circle cx="50" cy="50" r="23" fill="none" stroke="#fff" stroke-width="4" opacity=".65"/>
      <circle cx="50" cy="50" r="12" fill="none" stroke="#fff" stroke-width="3" opacity=".38"/>
      <circle cx="50" cy="50" r="34" fill="url(#void)"/>
      <circle cx="50" cy="50" r="4"  fill="#fff" opacity=".9"/>
    </svg>
  </div>
  <h1>Wormhole</h1>
  <div class="badge">✓ Вход выполнен</div>
  <div class="divider"></div>
  <h2>Как пользоваться</h2>
  <ol>
    <li>Найдите файл в Проводнике</li>
    <li>Щёлкните правой кнопкой → <kbd>Отправить</kbd> → <kbd>Wormhole</kbd></li>
    <li>Файл автоматически отправится на все ваши устройства</li>
    <li>На телефоне появится уведомление — нажмите «Принять»</li>
  </ol>
  <p class="close-hint">Это окно можно закрыть.</p>
</div>
</body>
</html>`)
		codeCh <- code
	})
	go srv.Serve(ln) //nolint:errcheck

	var authCode string
	select {
	case authCode = <-codeCh:
	case <-time.After(3 * time.Minute):
		panic("sign-in timed out")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx) //nolint:errcheck

	// Exchange auth code for Google ID token (no client secret — PKCE only).
	googleToken := exchangeGoogleCode(authCode, verifier, redirectURI)

	// Exchange Google ID token for Firebase ID token + refresh token.
	uid, refreshToken, idToken := firebaseSignIn(googleToken)

	return Config{
		UID:          uid,
		DeviceID:     deviceID,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		RelayURL:     relayURL,
	}
}

// refreshIDToken uses the stored refresh token to obtain a fresh Firebase ID token.
func refreshIDToken() (string, error) {
	resp, err := http.PostForm(
		"https://securetoken.googleapis.com/v1/token?key="+firebaseAPIKey,
		url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {cfg.RefreshToken},
		},
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		IDToken string `json:"id_token"`
	}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	return result.IDToken, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func exchangeGoogleCode(code, verifier, redirectURI string) string {
	resp, err := http.PostForm("https://oauth2.googleapis.com/token", url.Values{
		"client_id":     {googleClientID},
		"code":          {code},
		"code_verifier": {verifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
	})
	if err != nil {
		panic(fmt.Sprintf("token exchange: %v", err))
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		IDToken string `json:"id_token"`
	}
	json.Unmarshal(body, &result) //nolint:errcheck
	return result.IDToken
}

func firebaseSignIn(googleIDToken string) (uid, refreshToken, idToken string) {
	payload := map[string]interface{}{
		"postBody":          "id_token=" + googleIDToken + "&providerId=google.com",
		"requestUri":        "http://localhost",
		"returnIdpCredential": true,
		"returnSecureToken": true,
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(
		"https://identitytoolkit.googleapis.com/v1/accounts:signInWithIdp?key="+firebaseAPIKey,
		"application/json",
		strings.NewReader(string(b)),
	)
	if err != nil {
		panic(fmt.Sprintf("firebase sign-in: %v", err))
	}
	defer resp.Body.Close()
	var result struct {
		LocalID      string `json:"localId"`
		RefreshToken string `json:"refreshToken"`
		IDToken      string `json:"idToken"`
	}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	return result.LocalID, result.RefreshToken, result.IDToken
}

func pkceVerifier() string {
	b := make([]byte, 32)
	rand.Read(b) //nolint:errcheck
	return base64.RawURLEncoding.EncodeToString(b)
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func openBrowser(u string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start() //nolint:errcheck
	case "linux":
		exec.Command("xdg-open", u).Start() //nolint:errcheck
	case "darwin":
		exec.Command("open", u).Start() //nolint:errcheck
	}
}
