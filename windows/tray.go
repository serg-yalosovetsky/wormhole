package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/getlantern/systray"
	"github.com/go-toast/toast"
	"github.com/sqweek/dialog"
)

//go:embed assets/tray.ico
var trayIcon []byte

func runTray() {
	systray.Run(onTrayReady, onTrayExit)
}

func onTrayReady() {
	systray.SetIcon(trayIcon)
	systray.SetTooltip("Wormhole")

	mSend := systray.AddMenuItem("Отправить файл…", "Выбрать файл для отправки")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выйти", "Закрыть Wormhole")

	go func() {
		for {
			select {
			case <-mSend.ClickedCh:
				go pickAndSend()
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func onTrayExit() {}

// pickAndSend opens a native file-open dialog and sends the chosen file.
// No window stays open — dialog is modal and closes on its own.
func pickAndSend() {
	filePath, err := dialog.File().
		Title("Выберите файл для отправки через Wormhole").
		Load()
	if err != nil {
		// User cancelled — do nothing.
		return
	}
	runSend(filePath)
}

// ── Toast notifications ───────────────────────────────────────────────────────

func showSendingToast(filename, code string) {
	n := toast.Notification{
		AppID:   "Wormhole",
		Title:   "📤 Отправка: " + filename,
		Message: "Код: " + code + "\nОжидание получателя…",
	}
	n.Push() //nolint:errcheck
}

func showSentToast(filename string) {
	n := toast.Notification{
		AppID:   "Wormhole",
		Title:   "✅ Отправлено",
		Message: filename + " успешно передан.",
	}
	n.Push() //nolint:errcheck
}

func showReceiveToast(codeID, code, filename string) {
	// Clicking "Принять" re-launches the exe with --receive CODE:CODEID:FILENAME.
	exePath, _ := os.Executable()
	arg := fmt.Sprintf("%s:%s:%s", code, codeID, filename)

	n := toast.Notification{
		AppID:   "Wormhole",
		Title:   "📥 Входящий файл",
		Message: filename,
		Actions: []toast.Action{
			{
				Type:      "protocol",
				Label:     "Принять",
				Arguments: "wormhole:receive:" + arg,
			},
			{
				Type:      "protocol",
				Label:     "Отклонить",
				Arguments: "wormhole:decline:" + codeID,
			},
		},
	}
	_ = exePath // registered via protocol handler in install.go
	n.Push()    //nolint:errcheck
}

func showReceivedToast(filename, path string) {
	folder := filepath.Dir(path)
	n := toast.Notification{
		AppID:   "Wormhole",
		Title:   "✅ Получено: " + filename,
		Message: "Сохранён в " + folder,
		Actions: []toast.Action{
			{
				Type:      "protocol",
				Label:     "Открыть папку",
				Arguments: "wormhole:openfolder:" + folder,
			},
		},
	}
	n.Push() //nolint:errcheck
}

func showErrorToast(title, msg string) {
	n := toast.Notification{
		AppID:   "Wormhole",
		Title:   "❌ " + title,
		Message: msg,
	}
	n.Push() //nolint:errcheck
}

// openFolder is called when user clicks "Открыть папку" action.
func openFolder(path string) {
	exec.Command("explorer", path).Start() //nolint:errcheck
}
