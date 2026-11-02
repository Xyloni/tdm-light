param([switch]$StabilityOnly)
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path $PSScriptRoot -Parent
$auditDir = Join-Path $projectRoot '.gocache/audit'
New-Item -ItemType Directory -Force -Path $auditDir | Out-Null
$previousCache = $env:GOCACHE
$previousModuleCache = $env:GOMODCACHE
Push-Location $projectRoot
try {
    $env:GOCACHE = Join-Path $projectRoot '.gocache'
    $env:GOMODCACHE = Join-Path $projectRoot '.gocache/mod'
    if ($StabilityOnly) {
        & go test ./... -run 'TestReview(Upload|Cancel|Concurrent|500)|Test500AccountsNeverWaitForStalledBot|TestDiscord|TestGateway' -count 10 -timeout 120s
    } else {
        & go test ./... -run TestReview -timeout 60s
    }
    $testExitCode = $LASTEXITCODE
} finally {
    Pop-Location
    $env:GOCACHE = $previousCache
    $env:GOMODCACHE = $previousModuleCache
}
exit $testExitCode
