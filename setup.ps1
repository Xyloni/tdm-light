$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$envPath = Join-Path $root 'config.env'
$jsonPath = Join-Path $root 'config.json'

$script:envLines = [System.Collections.Generic.List[string]]::new()
if (Test-Path -LiteralPath $envPath) {
    foreach ($line in [System.IO.File]::ReadAllLines($envPath)) {
        $script:envLines.Add($line)
    }
}

function Get-Setting([string]$key) {
    $pattern = '^\s*' + [regex]::Escape($key) + '\s*=(.*)$'
    foreach ($line in $script:envLines) {
        if ($line -match $pattern) { return $Matches[1].Trim() }
    }
    if (Test-Path -LiteralPath $jsonPath) {
        try {
            $config = [System.IO.File]::ReadAllText($jsonPath) | ConvertFrom-Json
            $property = $key.ToLowerInvariant()
            $value = $config.$property
            if ($null -ne $value) { return [string]$value }
        } catch { }
    }
    return ''
}

function Set-Setting([string]$key, [string]$value) {
    $pattern = '^\s*' + [regex]::Escape($key) + '\s*='
    $found = $false
    for ($i = 0; $i -lt $script:envLines.Count; $i++) {
        if ($script:envLines[$i] -match $pattern) {
            $script:envLines[$i] = $key + '=' + $value
            $found = $true
            break
        }
    }
    if (-not $found) { $script:envLines.Add($key + '=' + $value) }
    [System.IO.File]::WriteAllLines($envPath, $script:envLines, [System.Text.UTF8Encoding]::new($false))
}

function Convert-SecureStringToPlainText([System.Security.SecureString]$secure) {
    $ptr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
    try { return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($ptr) }
    finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr) }
}

$token = Get-Setting 'TELEGRAM_BOT_TOKEN'
$validToken = $token -match '^\d+:[A-Za-z0-9_-]{20,}$' -and $token -notmatch '^1234567890:AAx+'
if (-not $validToken) {
    Write-Host 'Telegram-бот: токен вводится скрыто и сохраняется в config.env.'
    $secureToken = Read-Host 'Введите токен Telegram-бота (Enter — пропустить)' -AsSecureString
    $newToken = Convert-SecureStringToPlainText $secureToken
    if (-not [string]::IsNullOrWhiteSpace($newToken)) {
        if ($newToken -notmatch '^\d+:[A-Za-z0-9_-]{20,}$') {
            Write-Host '[ERROR] Формат токена не похож на токен Telegram.' -ForegroundColor Red
            exit 1
        }
        Set-Setting 'TELEGRAM_BOT_TOKEN' $newToken
    } else {
        Set-Setting 'TELEGRAM_BOT_TOKEN' ''
    }
}

$adminID = Get-Setting 'TELEGRAM_ADMIN_ID'
$validAdmin = $adminID -match '^-?\d+$' -and $adminID -ne '123456789'
if (-not $validAdmin) {
    $newAdminID = Read-Host 'Введите ваш числовой Telegram ID (Enter — пропустить)'
    if (-not [string]::IsNullOrWhiteSpace($newAdminID)) {
        if ($newAdminID -notmatch '^-?\d+$') {
            Write-Host '[ERROR] Telegram ID должен быть числом.' -ForegroundColor Red
            exit 1
        }
        Set-Setting 'TELEGRAM_ADMIN_ID' $newAdminID.Trim()
    } else {
        Set-Setting 'TELEGRAM_ADMIN_ID' ''
    }
}

$currentGame = Get-Setting 'TARGET_GAME'
if ([string]::IsNullOrWhiteSpace($currentGame)) {
    $newGame = Read-Host 'Введите целевую игру, например Hunt: Showdown 1896'
    if ([string]::IsNullOrWhiteSpace($newGame)) {
        Write-Host '[ERROR] Название игры обязательно.' -ForegroundColor Red
        exit 1
    }
    Set-Setting 'TARGET_GAME' $newGame.Trim()
} else {
    Write-Host "Текущая игра: $currentGame"
    $changeGame = Read-Host 'Оставить её? [Y/n]'
    if ($changeGame -match '^(n|no|н|нет)$') {
        $newGame = Read-Host 'Введите новую целевую игру'
        if ([string]::IsNullOrWhiteSpace($newGame)) {
            Write-Host '[ERROR] Игра не изменена: пустое название.' -ForegroundColor Red
            exit 1
        }
        Set-Setting 'TARGET_GAME' $newGame.Trim()
    }
}

Write-Host 'Настройки запуска сохранены.'
