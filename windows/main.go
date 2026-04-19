package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	receiveFlag := flag.String("receive", "", "wormhole code to receive (code:codeID:filename)")
	uriFlag     := flag.String("uri", "", "wormhole: URI dispatched by protocol handler")
	installFlag := flag.Bool("install", false, "install SendTo shortcut and protocol handler")
	flag.Parse()

	// Launched by a toast action via the wormhole: protocol handler.
	if *uriFlag != "" {
		handleURI(*uriFlag)
		return
	}

	// Launched by toast "Accept" button: --receive CODE:CODEID:FILENAME
	if *receiveFlag != "" {
		parts := strings.SplitN(*receiveFlag, ":", 3)
		if len(parts) != 3 {
			fmt.Fprintln(os.Stderr, "invalid --receive value")
			os.Exit(1)
		}
		runReceive(parts[0], parts[1], parts[2])
		return
	}

	// Launched from SendTo context menu: first non-flag arg is the file path
	if args := flag.Args(); len(args) > 0 {
		runSend(args[0])
		return
	}

	// First-time setup
	if *installFlag {
		if err := installShortcuts(); err != nil {
			fmt.Fprintf(os.Stderr, "install error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Installed successfully.")
		return
	}

	// Default: load config (triggers sign-in if needed), then run tray.
	loadConfig()
	go registerWithBackend()
	go pollLoop()
	runTray() // blocks until quit
}

// handleURI dispatches wormhole:<action>:<payload> URIs from toast buttons.
func handleURI(raw string) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return
	}
	// scheme = "wormhole", opaque = "action:payload"
	parts := strings.SplitN(parsed.Opaque, ":", 2)
	if len(parts) < 1 {
		return
	}
	action := parts[0]
	payload := ""
	if len(parts) == 2 {
		payload = parts[1]
	}

	loadConfig()

	switch action {
	case "receive":
		// payload = CODE:CODEID:FILENAME
		p := strings.SplitN(payload, ":", 3)
		if len(p) == 3 {
			runReceive(p[0], p[1], p[2])
		}
	case "decline":
		// payload = CODEID
		ackCode(payload)
	case "openfolder":
		openFolder(filepath.Clean(payload))
	}
}

// suppress "imported and not used" for fmt when building on non-Windows
var _ = fmt.Sprintf
