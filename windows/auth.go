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
	firebaseAPIKey    = "YOUR_FIREBASE_WEB_API_KEY"
	googleClientID    = "YOUR_GOOGLE_OAUTH_CLIENT_ID"
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
		fmt.Fprint(w, "<html><body><h2>Вход выполнен. Можно закрыть окно.</h2></body></html>")
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
