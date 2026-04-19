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
	d := filepath.Join(os.Getenv("APPDATA"), "Wormhole")
	os.MkdirAll(d, 0700)
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
	}
	saveConfig()
}

func saveConfig() {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(configPath(), data, 0600) //nolint:errcheck
}

func newDeviceID() string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	return "win-" + hex.EncodeToString(b)
}
