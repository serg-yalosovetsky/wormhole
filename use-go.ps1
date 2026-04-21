$RepoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$GoRoot = Join-Path $RepoRoot ".tools\go"
$GoBin = Join-Path $GoRoot "bin"
$GoPath = Join-Path $RepoRoot ".tools\gopath"
$GoInstallBin = Join-Path $RepoRoot ".tools\bin"
$GoCache = Join-Path $RepoRoot ".tools\gocache"
$BrokenProxy = "http://127.0.0.1:9"

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

$clearedProxy = $false
foreach ($proxyVar in @("HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY")) {
    $proxyValue = (Get-Item "Env:$proxyVar" -ErrorAction SilentlyContinue).Value
    if ($proxyValue -eq $BrokenProxy) {
        Remove-Item "Env:$proxyVar" -ErrorAction SilentlyContinue
        $clearedProxy = $true
    }
}

$noProxyValue = (Get-Item "Env:NO_PROXY" -ErrorAction SilentlyContinue).Value
if ($clearedProxy -and $noProxyValue -eq "localhost,127.0.0.1,::1") {
    Remove-Item "Env:NO_PROXY" -ErrorAction SilentlyContinue
}

$pathParts = $env:PATH -split ';'
if ($pathParts -notcontains $GoBin) {
    $env:PATH = "$GoBin;$env:PATH"
}
if ($pathParts -notcontains $GoInstallBin) {
    $env:PATH = "$GoInstallBin;$env:PATH"
}

Write-Host "Go activated for this shell:"
if ($clearedProxy) {
    Write-Host "Cleared broken proxy env vars for Go downloads."
}
go version
