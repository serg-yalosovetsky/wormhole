package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	UID      string `json:"uid"`
	DeviceID string `json:"device_id"`
	RelayURL string `json:"relay_url"`
}

type Credentials struct {
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token,omitempty"`
}

var cfg Config
var creds Credentials

func configDir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = os.Getenv("APPDATA")
	}
	if base == "" {
		home, _ := os.UserHomeDir()
		base = home
	}
	d := filepath.Join(base, "Wormhole")
	os.MkdirAll(d, 0700) //nolint:errcheck
	return d
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func credentialsPath() string {
	return filepath.Join(configDir(), "credentials.json")
}

func loadConfig() {
	mustSave := false

	if err := loadJSON(configPath(), &cfg); err != nil && !errors.Is(err, os.ErrNotExist) {
		showErrorToast("Wormhole", "Не удалось прочитать конфигурацию: "+err.Error())
	}
	if err := loadJSON(credentialsPath(), &creds); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Legacy builds stored credentials inside config.json; keep them for migration.
			var legacy struct {
				RefreshToken string `json:"refresh_token"`
				IDToken      string `json:"id_token"`
			}
			if err := loadJSON(configPath(), &legacy); err == nil && legacy.RefreshToken != "" {
				creds.RefreshToken = legacy.RefreshToken
				creds.IDToken = legacy.IDToken
				mustSave = true
			}
		} else {
			showErrorToast("Wormhole", "Не удалось прочитать креды: "+err.Error())
		}
	}
	if cfg.RelayURL == "" {
		cfg.RelayURL = "https://wormhole.ibotz.fun"
		mustSave = true
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = newDeviceID()
		mustSave = true
	}

	restoreAttempted := false
	restoreFailed := false
	if creds.RefreshToken != "" && (cfg.UID == "" || creds.IDToken == "") {
		restoreAttempted = true
		if err := restoreSession(); err == nil {
			mustSave = true
		} else {
			restoreFailed = true
		}
	}

	if cfg.UID == "" || creds.RefreshToken == "" || (restoreAttempted && restoreFailed && creds.IDToken == "") {
		cfg, creds = signIn(cfg.RelayURL, cfg.DeviceID)
		if cfg.UID == "" || creds.RefreshToken == "" {
			panic("sign-in completed but returned no credentials — check Firebase configuration")
		}
		mustSave = true
	}
	if mustSave {
		saveConfig()
	}
}

func saveConfig() {
	if err := saveJSON(configPath(), cfg); err != nil {
		showErrorToast("Wormhole", "Не удалось сохранить конфигурацию: "+err.Error())
		return
	}
	if err := saveJSON(credentialsPath(), creds); err != nil {
		showErrorToast("Wormhole", "Не удалось сохранить креды: "+err.Error())
	}
}

func loadJSON(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func saveJSON(path string, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write: save to a temp file then rename.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			_ = os.Remove(tmp)
			return err
		}
		if retryErr := os.Rename(tmp, path); retryErr != nil {
			_ = os.Remove(tmp)
			return retryErr
		}
	}
	return nil
}

func restoreSession() error {
	result, err := refreshIDToken()
	if err != nil {
		return err
	}
	creds.IDToken = result.IDToken
	if result.RefreshToken != "" {
		creds.RefreshToken = result.RefreshToken
	}
	if cfg.UID == "" && result.UID != "" {
		cfg.UID = result.UID
	}
	return nil
}

func newDeviceID() string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	return "win-" + hex.EncodeToString(b)
}
