@echo off
REM Build the Windows tray app.
REM Prerequisites: Go 1.21+, go mod tidy already run.

set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0

go build -ldflags="-s -w -H=windowsgui" -o wormhole-windows-amd64.exe .

echo Built wormhole-windows-amd64.exe
echo Run with --install to add SendTo shortcut and protocol handler.
