$RepoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$GoRoot = Join-Path $RepoRoot ".tools\go"
$GoBin = Join-Path $GoRoot "bin"
$GoPath = Join-Path $RepoRoot ".tools\gopath"
$GoInstallBin = Join-Path $RepoRoot ".tools\bin"
$GoCache = Join-Path $RepoRoot ".tools\gocache"

if (-not (Test-Path $GoBin)) {
    Write-Error "Go toolchain not found at $GoBin"
    exit 1
}

New-Item -ItemType Directory -Force $GoPath | Out-Null
New-Item -ItemType Directory -Force $GoInstallBin | Out-Null
New-Item -ItemType Directory -Force $GoCache | Out-Null

$env:GOROOT = $GoRoot
$env:GOPATH = $GoPath
$env:GOBIN = $GoInstallBin
$env:GOCACHE = $GoCache

$pathParts = $env:PATH -split ';'
if ($pathParts -notcontains $GoBin) {
    $env:PATH = "$GoBin;$env:PATH"
}
if ($pathParts -notcontains $GoInstallBin) {
    $env:PATH = "$GoInstallBin;$env:PATH"
}

Write-Host "Go activated for this shell:"
go version
