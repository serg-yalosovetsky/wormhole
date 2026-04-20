// Package wormhole provides gomobile-compatible bindings to wormhole-william.
// Build for Android: gomobile bind -target android -o wormhole.aar .
package wormhole

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	ww "github.com/psanford/wormhole-william/wormhole"
)

// SendCallback receives events during a send operation.
// gomobile converts this Go interface to a Java/Kotlin interface automatically.
type SendCallback interface {
	OnCode(code string)
	OnProgress(sent int64, total int64)
	OnError(msg string)
	OnDone()
}

// ReceiveCallback receives events during a receive operation.
type ReceiveCallback interface {
	OnProgress(received int64, total int64)
	OnError(msg string)
	OnDone(savedPath string)
}

// SendFile sends the file at path via a new wormhole transfer.
// cb.OnCode is called as soon as the wormhole code is known so the caller
// can display it / notify other devices before the transfer completes.
func SendFile(path string, cb SendCallback) {
	f, err := os.Open(path)
	if err != nil {
		cb.OnError(fmt.Sprintf("open: %v", err))
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		cb.OnError(fmt.Sprintf("stat: %v", err))
		return
	}

	c := ww.Client{}
	ctx := context.Background()

	code, statusCh, err := c.SendFile(ctx, filepath.Base(path), f)
	if err != nil {
		cb.OnError(fmt.Sprintf("send: %v", err))
		return
	}

	cb.OnCode(code)

	// wormhole-william doesn't stream per-byte progress, but we can report
	// a single progress update once the transfer is acknowledged.
	s := <-statusCh
	if s.Error != nil {
		cb.OnError(s.Error.Error())
		return
	}
	cb.OnProgress(fi.Size(), fi.Size())
	cb.OnDone()
}

// ReceiveFile receives a file using code and saves it to destDir.
func ReceiveFile(code, destDir string, cb ReceiveCallback) {
	c := ww.Client{}
	ctx := context.Background()

	msg, err := c.Receive(ctx, code)
	if err != nil {
		cb.OnError(fmt.Sprintf("receive: %v", err))
		return
	}

	if msg.Type != ww.TransferFile {
		cb.OnError("unexpected transfer type (expected file)")
		return
	}

	outPath := filepath.Join(destDir, msg.Name)
	f, err := os.Create(outPath)
	if err != nil {
		cb.OnError(fmt.Sprintf("create: %v", err))
		return
	}
	defer f.Close()

	total := msg.TransferBytes64
	buf := make([]byte, 32*1024)
	var received int64

	for {
		n, err := msg.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				cb.OnError(fmt.Sprintf("write: %v", werr))
				return
			}
			received += int64(n)
			cb.OnProgress(received, total)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			cb.OnError(fmt.Sprintf("read: %v", err))
			return
		}
	}

	cb.OnDone(outPath)
}
