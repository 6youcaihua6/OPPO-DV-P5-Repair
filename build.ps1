$ErrorActionPreference = 'Stop'
Push-Location $PSScriptRoot
try {
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Tests failed' }
    New-Item -ItemType Directory -Path ./dist -Force | Out-Null
    go build -trimpath -ldflags '-H=windowsgui -s -w' -o ./dist/OPPO_DV_P5_Repair_0.1BETA.exe .
    if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
} finally { Pop-Location }
