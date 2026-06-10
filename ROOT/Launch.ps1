<#
    .SYNOPSIS
    ta-chip Launcher — checks for updates then runs ta-chip.exe

    .DESCRIPTION
    1. Self-elevates to admin if needed
    2. Checks GitHub for a newer ta-chip.exe release
    3. Downloads if newer (or missing)
    4. Launches ta-chip.exe from the same directory

    This file is static — edit config.json instead.
#>

# ── Require Administrator ──────────────────────────────────────────────────────
if (!([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
    [Security.Principal.WindowsBuiltInRole]"Administrator")) {
    Start-Process powershell.exe "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`"" -Verb RunAs
    Exit
}

# ── Configuration ──────────────────────────────────────────────────────────────
$ErrorActionPreference = "Stop"
$ScriptRoot  = $PSScriptRoot
$ExeName     = "ta-chip.exe"
$TAChipExe   = Join-Path $ScriptRoot $ExeName
$ConfigFile  = Join-Path $ScriptRoot "config.json"
$LogFile     = Join-Path $env:TEMP "ta-chip_launch.log"

# Read github_repo from config.json (fallback hardcoded if config missing)
$GitHubRepo = "YOUR_ORG/ta-chip"
if (Test-Path $ConfigFile) {
    try {
        $cfg = Get-Content $ConfigFile -Raw | ConvertFrom-Json
        if ($cfg.github_repo -and $cfg.github_repo -notlike "*YOUR_ORG*") {
            $GitHubRepo = $cfg.github_repo
        }
    } catch {}
}

# ── Console Setup ──────────────────────────────────────────────────────────────
chcp 65001 | Out-Null
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$Host.UI.RawUI.WindowTitle = "TA CHIP  |  PC Health Inspector"
try {
    $Host.UI.RawUI.BackgroundColor = "Black"
    $Host.UI.RawUI.ForegroundColor = "White"
    $w = 120; $h = 42
    $buf = $Host.UI.RawUI.BufferSize
    if ($buf.Width  -lt $w) { $buf.Width  = $w; $Host.UI.RawUI.BufferSize = $buf }
    if ($buf.Height -lt $h) { $buf.Height = $h; $Host.UI.RawUI.BufferSize = $buf }
    $win = $Host.UI.RawUI.WindowSize
    $win.Width = $w; $win.Height = $h
    $Host.UI.RawUI.WindowSize = $win
    $buf.Width = $w; $buf.Height = $h
    $Host.UI.RawUI.BufferSize = $buf
} catch {}
Clear-Host

# ── Logging ────────────────────────────────────────────────────────────────────
function Write-Log { param([string]$L,[string]$M)
    "$(Get-Date -f 'yyyy-MM-dd HH:mm:ss')  [$L]  $M" |
        Out-File -FilePath $LogFile -Append -Encoding UTF8 -ErrorAction SilentlyContinue
}
Write-Log "START" "ta-chip Launch  |  Repo: $GitHubRepo"

# ── Styled Output ──────────────────────────────────────────────────────────────
function Write-Step { param([string]$I,[string]$M,[string]$C="White")
    Write-Host "  $I " -NoNewline -ForegroundColor Yellow
    Write-Host $M -ForegroundColor $C
    Write-Log "INFO" $M
}

function Write-Banner {
    Write-Host ""
    Write-Host "  ╔══════════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "  ║   TA CHIP  •  PC Health Inspector            ║" -ForegroundColor Cyan
    Write-Host "  ╚══════════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
}

# ── Version Helpers ────────────────────────────────────────────────────────────
function Get-CurrentVersion {
    if (-not (Test-Path $TAChipExe)) { return [System.Version]"0.0.0" }
    try {
        $raw = (& $TAChipExe --version 2>&1) | Select-Object -First 1
        $clean = ($raw -replace '[^0-9\.]','').Trim()
        if ($clean -match '^\d+\.\d+') { return [System.Version]$clean }
    } catch {}
    return [System.Version]"0.0.0"
}

function Get-LatestRelease {
    $url = "https://api.github.com/repos/$GitHubRepo/releases/latest"
    try {
        $ProgressPreference = 'SilentlyContinue'
        $rel = Invoke-RestMethod -Uri $url -UseBasicParsing -ErrorAction Stop
        $ProgressPreference = 'Continue'
        return $rel
    } catch {
        Write-Log "WARN" "GitHub API failed: $($_.Exception.Message)"
        return $null
    }
}

function Download-Exe { param($Release)
    $asset = $Release.assets | Where-Object { $_.name -eq $ExeName } | Select-Object -First 1
    if (-not $asset) { Write-Log "WARN" "No $ExeName asset in release."; return $false }
    try {
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $TAChipExe -UseBasicParsing -ErrorAction Stop
        $ProgressPreference = 'Continue'
        Write-Step "OK" "$ExeName downloaded ($($Release.tag_name))" "Green"
        Write-Log "INFO" "Downloaded $($asset.browser_download_url)"
        return $true
    } catch {
        Write-Log "WARN" "Download failed: $($_.Exception.Message)"
        return $false
    }
}

# ══════════════════════════════════════════════════════════════════════════════
# MAIN
# ══════════════════════════════════════════════════════════════════════════════
Write-Banner
Set-Location $ScriptRoot

# ── Check for updates ──────────────────────────────────────────────────────────
$currentVer = Get-CurrentVersion
Write-Step ">>" "Current version: $currentVer" "Gray"

$release = Get-LatestRelease
if ($release) {
    $latestRaw  = ($release.tag_name -replace '[^0-9\.]','').Trim()
    if ($latestRaw -match '^\d+\.\d+') {
        $latestVer = [System.Version]$latestRaw
        if ($latestVer -gt $currentVer) {
            Write-Step ">>" "Update available: $latestVer  (current: $currentVer)" "Cyan"
            $null = Download-Exe -Release $release
        } else {
            Write-Step "OK" "Up to date ($currentVer)" "Green"
        }
    }
} else {
    Write-Step "!!" "Could not reach GitHub — skipping update check" "Yellow"
}

# ── Launch ─────────────────────────────────────────────────────────────────────
if (-not (Test-Path $TAChipExe)) {
    Write-Host ""
    Write-Host "  ta-chip.exe not found and could not be downloaded." -ForegroundColor Red
    Write-Host "  Place ta-chip.exe in: $ScriptRoot" -ForegroundColor Yellow
    Write-Host ""
    Write-Log "ERROR" "ta-chip.exe missing after update attempt."
    $null = [Console]::ReadKey($true)
    Exit 1
}

Write-Host ""
Write-Step ">>" "Launching ta-chip..." "Cyan"
Write-Log "INFO" "Launching $TAChipExe"

Start-Process -FilePath $TAChipExe -WorkingDirectory $ScriptRoot

Write-Log "INFO" "ta-chip exited."
Exit 0
