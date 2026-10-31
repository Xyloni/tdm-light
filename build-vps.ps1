$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$bundle = Join-Path $root 'vps-linux'

foreach ($file in @('install.sh', 'dropfarm.service', 'README.md')) {
    if (-not (Test-Path -LiteralPath (Join-Path $bundle $file))) {
        throw "Missing VPS bundle file: $file"
    }
}

Copy-Item -LiteralPath (Join-Path $root 'config.env.example') -Destination (Join-Path $bundle 'config.env.example') -Force
Copy-Item -LiteralPath (Join-Path $root 'config.json.example') -Destination (Join-Path $bundle 'config.json.example') -Force

$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
$oldCGO = $env:CGO_ENABLED
$env:GOOS = 'linux'
$env:CGO_ENABLED = '0'
try {
    foreach ($target in @(@{ Arch = 'amd64'; Name = 'dropfarm-linux-amd64' }, @{ Arch = 'arm64'; Name = 'dropfarm-linux-arm64' })) {
        $env:GOARCH = $target.Arch
        Write-Host "Building Linux/$($target.Arch)..."
        & go build -trimpath -ldflags '-s -w' -o (Join-Path $bundle $target.Name) .
        if ($LASTEXITCODE -ne 0) { throw "Build failed for Linux/$($target.Arch)." }
    }
} finally {
    $env:GOOS = $oldGOOS
    $env:GOARCH = $oldGOARCH
    $env:CGO_ENABLED = $oldCGO
}

$archive = Join-Path $root 'dropfarm-vps-ubuntu.tar.gz'
$members = @(
    'vps-linux/install.sh',
    'vps-linux/dropfarm.service',
    'vps-linux/README.md',
    'vps-linux/config.env.example',
    'vps-linux/config.json.example',
    'vps-linux/dropfarm-linux-amd64',
    'vps-linux/dropfarm-linux-arm64'
)
& tar -czf $archive -C $root @members
if ($LASTEXITCODE -ne 0) { throw 'Could not create VPS archive.' }

Write-Host "VPS bundle ready: $bundle"
Write-Host "Archive ready: $archive"
