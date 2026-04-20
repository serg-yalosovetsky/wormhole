package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	UID          string `json:"uid"`
	DeviceID     string `json:"device_id"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token,omitempty"`
	RelayURL     string `json:"relay_url"`
}

var cfg Config

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

func loadConfig() {
	data, err := os.ReadFile(configPath())
	if err == nil {
		json.Unmarshal(data, &cfg) //nolint:errcheck
	}
	if cfg.RelayURL == "" {
		cfg.RelayURL = "https://wormhole.ibotz.fun"
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = newDeviceID()
	}
	if cfg.UID == "" || cfg.RefreshToken == "" {
		cfg = signIn(cfg.RelayURL, cfg.DeviceID)
		if cfg.UID == "" || cfg.RefreshToken == "" {
			panic("sign-in completed but returned no credentials — check Firebase configuration")
		}
	}
	saveConfig()
}

func saveConfig() {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	path := configPath()
	// Atomic write: save to a temp file then rename.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		showErrorToast("Wormhole", "Не удалось сохранить конфигурацию: "+err.Error())
		return
	}
	os.Rename(tmp, path) //nolint:errcheck
}

func newDeviceID() string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	return "win-" + hex.EncodeToString(b)
}
