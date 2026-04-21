@echo off
REM Build the Windows tray app.
REM Prerequisites: Go 1.21+, go mod tidy already run.
setlocal

set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0

set "SECRETS_FILE=%~dp0auth.secrets.json"
set "GENERATED_FILE=%~dp0z_auth_embedded.go"

if not exist "%SECRETS_FILE%" (
  echo Missing %SECRETS_FILE%
  echo Copy windows\auth.secrets.json.template to windows\auth.secrets.json and fill in real values.
  exit /b 1
)

powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "$cfg = Get-Content '%SECRETS_FILE%' -Raw | ConvertFrom-Json; " ^
  "$missing = @(); " ^
  "if ([string]::IsNullOrWhiteSpace($cfg.firebase_api_key)) { $missing += 'firebase_api_key' }; " ^
  "if ([string]::IsNullOrWhiteSpace($cfg.google_client_id)) { $missing += 'google_client_id' }; " ^
  "if ($missing.Count -gt 0) { throw ('Missing required fields: ' + ($missing -join ', ')) }; " ^
  "$secret = if ($cfg.google_client_secret) { [string]$cfg.google_client_secret } else { '' }; " ^
  "$content = @(" ^
  "  '//go:build authembed'," ^
  "  ''," ^
  "  'package main'," ^
  "  ''," ^
  "  'func embeddedAuthSettings() authSettings {'," ^
  "  '    return authSettings{'," ^
  "  ('        FirebaseAPIKey: ' + (ConvertTo-Json ([string]$cfg.firebase_api_key) -Compress) + ',')," ^
  "  ('        GoogleClientID: ' + (ConvertTo-Json ([string]$cfg.google_client_id) -Compress) + ',')," ^
  "  ('        GoogleClientSecret: ' + (ConvertTo-Json $secret -Compress) + ',')," ^
  "  '    }'," ^
  "  '}'" ^
  "); " ^
  "[System.IO.File]::WriteAllLines('%GENERATED_FILE%', $content)"
if errorlevel 1 exit /b 1

go build -tags authembed -ldflags="-s -w -H=windowsgui" -o wormhole-windows-amd64.exe .
set BUILD_STATUS=%ERRORLEVEL%

if exist "%GENERATED_FILE%" del /q "%GENERATED_FILE%"
if not "%BUILD_STATUS%"=="0" exit /b %BUILD_STATUS%

echo Built wormhole-windows-amd64.exe
echo Run with --install to add SendTo shortcut and protocol handler.
