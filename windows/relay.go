package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var (
	notifMu    sync.Mutex
	notifCount = map[string]int{} // codeID → times shown
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// registerWithBackend tells the relay server our FCM token (we use polling on
// Windows, so we register with an empty fcm_token and platform="windows").
func registerWithBackend() {
	body := map[string]string{
		"uid":       cfg.UID,
		"device_id": cfg.DeviceID,
		"fcm_token": "",
		"platform":  "windows",
	}
	if err := postJSON("/register", body, nil); err != nil {
		fmt.Printf("register error: %v\n", err)
	}
}

// notifyDevices tells the relay to push the wormhole code to other devices.
func notifyDevices(code, filename string) {
	body := map[string]string{
		"uid":              cfg.UID,
		"sender_device_id": cfg.DeviceID,
		"code":             code,
		"filename":         filename,
	}
	if err := postJSON("/notify", body, nil); err != nil {
		fmt.Printf("notify error: %v\n", err)
	}
}

type pendingCode struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Filename string `json:"filename"`
}

// pollIncoming fetches pending codes from the relay for this Windows device.
func pollIncoming() ([]pendingCode, error) {
	url := fmt.Sprintf("%s/poll/%s/%s", cfg.RelayURL, cfg.UID, cfg.DeviceID)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Codes []pendingCode `json:"codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Codes, nil
}

// ackCode marks a pending code as handled.
func ackCode(codeID string) {
	notifMu.Lock()
	delete(notifCount, codeID)
	notifMu.Unlock()

	body := map[string]string{
		"uid":       cfg.UID,
		"device_id": cfg.DeviceID,
		"code_id":   codeID,
	}
	postJSON("/ack", body, nil) //nolint:errcheck
}

// pollLoop runs in a background goroutine.
func pollLoop() {
	for {
		time.Sleep(30 * time.Second)
		codes, err := pollIncoming()
		if err != nil {
			continue
		}
		for _, c := range codes {
			notifMu.Lock()
			count := notifCount[c.ID]
			if count < 3 {
				notifCount[c.ID] = count + 1
				notifMu.Unlock()
				showReceiveToast(c.ID, c.Code, c.Filename)
			} else {
				notifMu.Unlock()
			}
		}
	}
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

func postJSON(path string, body, out interface{}) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := httpClient.Post(cfg.RelayURL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, raw)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
