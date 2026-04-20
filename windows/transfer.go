package main

import (
	"context"
	"io"
	"os"
	"path/filepath"

	ww "github.com/psanford/wormhole-william/wormhole"
)

// runSend is called when the app is launched via SendTo (subprocess mode).
func runSend(filePath string) {
	if cfg.UID == "" {
		loadConfig()
	}

	f, err := os.Open(filePath)
	if err != nil {
		showErrorToast("Ошибка открытия файла", err.Error())
		return
	}
	defer f.Close()

	c := ww.Client{}
	ctx := context.Background()
	name := filepath.Base(filePath)

	code, statusCh, err := c.SendFile(ctx, name, f)
	if err != nil {
		showErrorToast("Ошибка отправки", err.Error())
		return
	}

	// Notify user and all other devices immediately once code is known.
	showSendingToast(name, code)
	go notifyDevices(code, name)

	s := <-statusCh
	if s.Error != nil {
		showErrorToast("Передача прервана", s.Error.Error())
		return
	}
	showSentToast(name)
}

// runReceive is called when the user clicks "Принять" on a toast notification.
func runReceive(code, codeID, filename string) {
	if cfg.UID == "" {
		loadConfig()
	}

	c := ww.Client{}
	ctx := context.Background()

	msg, err := c.Receive(ctx, code)
	if err != nil {
		showErrorToast("Ошибка получения", err.Error())
		return
	}

	destDir := desktopDir()
	outPath := filepath.Join(destDir, msg.Name)

	f, err := os.Create(outPath)
	if err != nil {
		showErrorToast("Ошибка создания файла", err.Error())
		return
	}
	defer f.Close()

	if _, err := io.Copy(f, msg); err != nil {
		showErrorToast("Ошибка записи", err.Error())
		return
	}

	ackCode(codeID)
	showReceivedToast(msg.Name, outPath)
}

func desktopDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}
	d := filepath.Join(home, "Desktop")
	if _, err := os.Stat(d); err == nil {
		return d
	}
	return filepath.Join(home, "Downloads")
}
