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
	"time"

	ww "github.com/psanford/wormhole-william/wormhole"
)

type progressEvent struct {
	Code  string `json:"code,omitempty"`
	Done  bool   `json:"done,omitempty"`
	Error string `json:"error,omitempty"`
}

// openSenderUI starts a local HTTP server, opens a browser drag-and-drop window,
// and manages the full send flow including progress feedback.
func openSenderUI() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		showErrorToast("Wormhole", "Не удалось открыть окно отправки: "+err.Error())
		return
	}
	port := ln.Addr().(*net.TCPAddr).Port

	progressCh := make(chan progressEvent, 8)
	done := make(chan struct{}, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, senderHTML)
	})

	mux.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(4 << 30); err != nil { // 4 GB
			jsonError(w, "Ошибка чтения файла: "+err.Error())
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			jsonError(w, err.Error())
			return
		}
		defer file.Close()

		// Save to a temp file; wormhole-william needs an io.ReadSeeker.
		tmp, err := os.CreateTemp("", "wormhole-*")
		if err != nil {
			jsonError(w, err.Error())
			return
		}
		tmpPath := tmp.Name()
		if _, err = io.Copy(tmp, file); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			jsonError(w, err.Error())
			return
		}
		tmp.Close()

		// Respond immediately so the browser can open the SSE connection.
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)

		name := filepath.Base(header.Filename)
		go func() {
			defer os.Remove(tmpPath)
			defer func() { done <- struct{}{} }()
			defer func() {
				if r := recover(); r != nil {
					progressCh <- progressEvent{Error: fmt.Sprintf("%v", r)}
				}
			}()

			f, err := os.Open(tmpPath)
			if err != nil {
				progressCh <- progressEvent{Error: err.Error()}
				return
			}
			defer f.Close()

			c := ww.Client{}
			ctx := context.Background()

			code, statusCh, err := c.SendFile(ctx, name, f)
			if err != nil {
				progressCh <- progressEvent{Error: err.Error()}
				showErrorToast("Ошибка отправки", err.Error())
				return
			}

			progressCh <- progressEvent{Code: code}
			go notifyDevices(code, name)
			showSendingToast(name, code)

			s := <-statusCh
			if s.Error != nil {
				progressCh <- progressEvent{Error: s.Error.Error()}
				showErrorToast("Передача прервана", s.Error.Error())
				return
			}
			progressCh <- progressEvent{Done: true}
			showSentToast(name)
		}()
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

	go srv.Serve(ln) //nolint:errcheck

	// Shut down after transfer completes (+ short delay for browser to show result)
	// or after a 30-minute idle timeout.
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
}

func jsonError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	b, _ := json.Marshal(map[string]string{"error": msg})
	w.Write(b) //nolint:errcheck
}

const senderHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Wormhole — отправить файл</title>
<link rel="icon" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'%3E%3Crect width='100' height='100' rx='22' fill='%236366f1'/%3E%3Ccircle cx='50' cy='50' r='34' fill='none' stroke='white' stroke-width='6'/%3E%3Ccircle cx='50' cy='50' r='22' fill='none' stroke='white' stroke-width='4' opacity='.6'/%3E%3Ccircle cx='50' cy='50' r='10' fill='none' stroke='white' stroke-width='3' opacity='.35'/%3E%3Ccircle cx='50' cy='50' r='4' fill='white'/%3E%3C/svg%3E">
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{background:#0f1117;color:#e8eaf0;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;
     display:flex;align-items:center;justify-content:center;min-height:100vh;padding:24px}
.card{background:#1a1d27;border:1px solid #2a2d3a;border-radius:16px;padding:40px 48px;
      max-width:520px;width:100%;text-align:center;box-shadow:0 8px 32px rgba(0,0,0,.4)}
.logo{width:56px;height:56px;margin:0 auto 14px;border-radius:14px;overflow:hidden}
.logo svg{width:100%;height:100%}
h1{font-size:20px;font-weight:700;margin-bottom:28px;color:#fff}
.drop-zone{border:2px dashed #374151;border-radius:12px;padding:52px 24px;cursor:pointer;
           transition:border-color .2s,background .2s}
.drop-zone.over{border-color:#6366f1;background:#6366f108}
.drop-zone:hover{border-color:#4b5563}
.drop-icon{font-size:44px;margin-bottom:14px;display:block}
.drop-hint{color:#9ca3af;font-size:14px;margin:6px 0 18px}
.divider{display:flex;align-items:center;gap:12px;color:#4b5563;font-size:12px;
         text-transform:uppercase;letter-spacing:.05em;margin:14px 0}
.divider::before,.divider::after{content:'';flex:1;height:1px;background:#252836}
.btn{background:#6366f1;color:#fff;border:none;border-radius:8px;
     padding:11px 24px;font-size:14px;font-weight:600;cursor:pointer}
.btn:hover{background:#5558e3}
#progress-view{display:none}
.fname{font-size:15px;font-weight:500;color:#e2e8f0;margin-bottom:20px;
       overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:100%}
.bar-wrap{background:#252836;border-radius:100px;height:8px;overflow:hidden;margin-bottom:20px}
.bar-fill{height:100%;background:linear-gradient(90deg,#6366f1,#a78bfa);
          transition:width .5s ease;border-radius:100px;width:0%}
.code-wrap{display:none;margin-bottom:18px}
.code-label{font-size:11px;color:#6b7280;text-transform:uppercase;letter-spacing:.08em;margin-bottom:6px}
.code{background:#252836;border-radius:8px;padding:12px 20px;font-family:'Courier New',monospace;
      font-size:22px;letter-spacing:.18em;color:#a78bfa;user-select:all}
.status{font-size:14px;color:#9ca3af;min-height:20px}
.status.ok{color:#22c55e;font-weight:600}
.status.err{color:#ef4444}
</style>
</head>
<body>
<div class="card">
  <div class="logo">
    <svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="g" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#6366f1"/><stop offset="100%" stop-color="#7c3aed"/>
        </linearGradient>
        <radialGradient id="void" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="#1e1b4b"/>
          <stop offset="100%" stop-color="#1e1b4b" stop-opacity="0"/>
        </radialGradient>
      </defs>
      <rect width="100" height="100" rx="22" fill="url(#g)"/>
      <circle cx="50" cy="50" r="34" fill="none" stroke="#fff" stroke-width="6" opacity=".95"/>
      <circle cx="50" cy="50" r="23" fill="none" stroke="#fff" stroke-width="4" opacity=".65"/>
      <circle cx="50" cy="50" r="12" fill="none" stroke="#fff" stroke-width="3" opacity=".38"/>
      <circle cx="50" cy="50" r="34" fill="url(#void)"/>
      <circle cx="50" cy="50" r="4"  fill="#fff" opacity=".9"/>
    </svg>
  </div>
  <h1>Wormhole</h1>

  <div id="pick-view">
    <div class="drop-zone" id="dz">
      <span class="drop-icon">📂</span>
      <div style="font-size:15px;color:#e2e8f0;margin-bottom:6px">Перетащите файл сюда</div>
      <div class="drop-hint">Файл мгновенно появится на всех ваших устройствах</div>
      <div class="divider">или</div>
      <button class="btn" onclick="document.getElementById('fi').click()">Выбрать файл</button>
      <input type="file" id="fi" style="display:none">
    </div>
  </div>

  <div id="progress-view">
    <div class="fname" id="fname"></div>
    <div class="bar-wrap"><div class="bar-fill" id="bar"></div></div>
    <div class="code-wrap" id="code-wrap">
      <div class="code-label">Код для получателя</div>
      <div class="code" id="code"></div>
    </div>
    <div class="status" id="status">Подготовка…</div>
  </div>
</div>
<script>
const dz=document.getElementById('dz'),fi=document.getElementById('fi');
dz.addEventListener('dragover',e=>{e.preventDefault();dz.classList.add('over')});
dz.addEventListener('dragleave',()=>dz.classList.remove('over'));
dz.addEventListener('drop',e=>{e.preventDefault();dz.classList.remove('over');
  const f=e.dataTransfer.files[0];if(f)upload(f)});
fi.addEventListener('change',()=>{if(fi.files[0])upload(fi.files[0])});

function upload(file){
  document.getElementById('pick-view').style.display='none';
  document.getElementById('progress-view').style.display='block';
  document.getElementById('fname').textContent=file.name;
  bar(15);status('Загрузка…');

  const fd=new FormData();fd.append('file',file);
  fetch('/send',{method:'POST',body:fd})
    .then(r=>r.json())
    .then(d=>{if(d.error)status(d.error,'err')})
    .catch(()=>status('Ошибка соединения','err'));

  const es=new EventSource('/progress');
  es.onmessage=e=>{
    const d=JSON.parse(e.data);
    if(d.code){
      document.getElementById('code-wrap').style.display='block';
      document.getElementById('code').textContent=d.code;
      bar(60);status('Ожидание получателя…');
    }
    if(d.done){bar(100);status('✅ Файл успешно отправлен!','ok');es.close();
      setTimeout(()=>window.close(),3500)}
    if(d.error){status('❌ '+d.error,'err');es.close()}
  };
}
function bar(p){document.getElementById('bar').style.width=p+'%'}
function status(t,c){const s=document.getElementById('status');s.textContent=t;s.className='status'+(c?' '+c:'')}
</script>
</body>
</html>`
