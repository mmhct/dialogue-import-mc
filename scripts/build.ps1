$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
Push-Location $projectRoot
$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
$oldCGO = $env:CGO_ENABLED
try {
    New-Item -ItemType Directory -Force dist | Out-Null
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    go build -trimpath -ldflags '-s -w -H=windowsgui' -o dist/DialogueForge.exe ./cmd/dialogueforge
    if ($LASTEXITCODE -ne 0) { throw 'Windows build failed' }
    Write-Output 'Built dist/DialogueForge.exe'
} finally {
    $env:GOOS = $oldGOOS
    $env:GOARCH = $oldGOARCH
    $env:CGO_ENABLED = $oldCGO
    Pop-Location
}
