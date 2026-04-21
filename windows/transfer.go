package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	ww "github.com/psanford/wormhole-william/wormhole"
)

type recvProgressEvent struct {
	Filename string `json:"filename,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Bytes    int64  `json:"bytes,omitempty"`
	Done     bool   `json:"done,omitempty"`
	Error    string `json:"error,omitempty"`
}

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

// runReceive opens a browser progress window and downloads the file via wormhole.
func runReceive(code, codeID, filename string) {
	if cfg.UID == "" {
		loadConfig()
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		showErrorToast("Wormhole", "Не удалось открыть окно загрузки: "+err.Error())
		return
	}
	port := ln.Addr().(*net.TCPAddr).Port

	progressCh := make(chan recvProgressEvent, 64)
	done := make(chan struct{}, 1)

	var outDirMu sync.Mutex
	var outDir string

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, receiverHTML)
	})

	mux.HandleFunc("/progress", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		for {
			select {
			case ev := <-progressCh:
				b, _ := json.Marshal(ev)
				fmt.Fprintf(w, "data: %s\n\n", b)
				flusher.Flush()
				if ev.Done || ev.Error != "" {
					return
				}
			case <-r.Context().Done():
				return
			}
		}
	})

	mux.HandleFunc("/openfolder", func(w http.ResponseWriter, r *http.Request) {
		outDirMu.Lock()
		dir := outDir
		outDirMu.Unlock()
		if dir != "" {
			openFolder(dir)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	go srv.Serve(ln) //nolint:errcheck

	go func() {
		select {
		case <-done:
			time.Sleep(4 * time.Second)
		case <-time.After(30 * time.Minute):
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Shutdown(ctx) //nolint:errcheck
	}()

	openBrowser(fmt.Sprintf("http://localhost:%d/", port))

	go func() {
		defer func() { done <- struct{}{} }()

		c := ww.Client{}
		ctx := context.Background()

		msg, err := c.Receive(ctx, code)
		if err != nil {
			progressCh <- recvProgressEvent{Error: err.Error()}
			showErrorToast("Ошибка получения", err.Error())
			return
		}

		progressCh <- recvProgressEvent{
			Filename: msg.Name,
			Size:     int64(msg.TransferBytes64),
		}

		destDir := desktopDir()
		outPath := filepath.Join(destDir, msg.Name)

		outDirMu.Lock()
		outDir = destDir
		outDirMu.Unlock()

		f, err := os.Create(outPath)
		if err != nil {
			progressCh <- recvProgressEvent{Error: err.Error()}
			showErrorToast("Ошибка создания файла", err.Error())
			return
		}
		defer f.Close()

		buf := make([]byte, 32*1024)
		var received int64
		const reportEvery = 512 * 1024

		for {
			n, err := msg.Read(buf)
			if n > 0 {
				if _, werr := f.Write(buf[:n]); werr != nil {
					progressCh <- recvProgressEvent{Error: werr.Error()}
					showErrorToast("Ошибка записи", werr.Error())
					return
				}
				received += int64(n)
				// Emit progress event each time we cross a 512 KB boundary.
				if received/reportEvery > (received-int64(n))/reportEvery {
					progressCh <- recvProgressEvent{Bytes: received}
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				progressCh <- recvProgressEvent{Error: err.Error()}
				showErrorToast("Ошибка чтения", err.Error())
				return
			}
		}

		ackCode(codeID)
		progressCh <- recvProgressEvent{Done: true}
		showReceivedToast(msg.Name, outPath)
	}()
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
