$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$ico = Join-Path $root "internal\ui\icons\tray.ico"
$syso = Join-Path $root "cmd\spotify-ad-free\rsrc_windows_amd64.syso"
$rsrc = Join-Path (go env GOPATH) "bin\rsrc.exe"
if (-not (Test-Path $rsrc)) {
    go install github.com/akavel/rsrc@latest
}
& $rsrc -ico $ico -arch amd64 -o $syso

New-Item -ItemType Directory -Force -Path (Join-Path $root "bin") | Out-Null

$out = "bin\Spotify AdFree.exe"
go build -ldflags "-s -w -H windowsgui" -o $out .\cmd\spotify-ad-free
Write-Host "built $out ($((Get-Item $out).Length) bytes)"
