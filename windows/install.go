package main

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// installShortcuts registers:
//  1. A .bat launcher in the Windows SendTo folder so users can right-click any
//     file → Send to → Wormhole.
//  2. A "wormhole:" URI protocol handler so toast action buttons can launch the
//     exe with --receive / --decline / --openfolder arguments.
func installShortcuts() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get exe path: %w", err)
	}

	if err := installSendTo(exePath); err != nil {
		return fmt.Errorf("SendTo: %w", err)
	}
	if err := installProtocol(exePath); err != nil {
		return fmt.Errorf("protocol handler: %w", err)
	}
	return nil
}

// installSendTo creates %APPDATA%\Microsoft\Windows\SendTo\Wormhole.bat
func installSendTo(exePath string) error {
	sendTo := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "SendTo")
	batPath := filepath.Join(sendTo, "Wormhole.bat")
	content := fmt.Sprintf("@echo off\n\"%s\" \"%%~1\"\n", exePath)
	return os.WriteFile(batPath, []byte(content), 0644)
}

// installProtocol registers HKCU\Software\Classes\wormhole so that
// wormhole:receive:... URIs launch the exe.
func installProtocol(exePath string) error {
	// HKCU\Software\Classes\wormhole
	k, _, err := registry.CreateKey(
		registry.CURRENT_USER,
		`Software\Classes\wormhole`,
		registry.SET_VALUE,
	)
	if err != nil {
		return err
	}
	defer k.Close()
	k.SetStringValue("", "URL:Wormhole Protocol")              //nolint:errcheck
	k.SetStringValue("URL Protocol", "")                       //nolint:errcheck

	// HKCU\Software\Classes\wormhole\shell\open\command
	cmd, _, err := registry.CreateKey(
		registry.CURRENT_USER,
		`Software\Classes\wormhole\shell\open\command`,
		registry.SET_VALUE,
	)
	if err != nil {
		return err
	}
	defer cmd.Close()
	// Windows passes the full URI as the first argument.
	// main.go parseURI() will decode it.
	return cmd.SetStringValue("", fmt.Sprintf(`"%s" --uri "%%1"`, exePath))
}
