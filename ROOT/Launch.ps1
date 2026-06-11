<#
    .SYNOPSIS
    ta-chip Launcher -- downloads ta-chip-latest.exe from R2 and runs it

    .DESCRIPTION
    1. Self-elevates to admin if needed
    2. Downloads ta-chip-latest.exe from Cloudflare R2
    3. Writes config.json to the same temp directory
    4. Launches ta-chip.exe

    To update config or the launcher itself, edit this file and push a new tag.
#>

# -- Config (edit here, then push a new tag to update R2) ----------------------
$R2BaseUrl    = "https://ta-chip.lrxrn.dev"
$AppScriptUrl = "https://script.google.com/macros/s/AKfycbxqryHNoNEWGeHaoq61n24OAf1-liP3a7Of6jCXthOZ_iEWLRyhEpzS8oFOJ9_BvC7v/exec"
$DomainName   = "TECHLAB"
$DomainUser   = "student"
$DomainPass   = ""
$NTPTolerance = 300
$LoginUser    = ".\student"
$LoginPass    = "student"

# -- Require Administrator -----------------------------------------------------
if (!([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
    [Security.Principal.WindowsBuiltInRole]"Administrator")) {
    Start-Process powershell.exe "-NoProfile -ExecutionPolicy Bypass -C `"iex (irm '$R2BaseUrl/Launch.ps1')`"" -Verb RunAs
    Exit
}

# -- Paths ---------------------------------------------------------------------
$ErrorActionPreference = "Stop"
$TempDir  = Join-Path $env:TEMP "ta-chip"
$ExeDest  = Join-Path $TempDir  "ta-chip.exe"
$CfgDest  = Join-Path $TempDir  "config.json"
$LogFile  = Join-Path $env:TEMP "ta-chip_launch.log"

if (-not (Test-Path $TempDir)) { New-Item -ItemType Directory $TempDir | Out-Null }

# -- Console Setup -------------------------------------------------------------
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

# -- Logging -------------------------------------------------------------------
function Write-Log { param([string]$L,[string]$M)
    "$(Get-Date -f 'yyyy-MM-dd HH:mm:ss')  [$L]  $M" |
        Out-File -FilePath $LogFile -Append -Encoding UTF8 -ErrorAction SilentlyContinue
}
Write-Log "START" "ta-chip Launch  |  R2: $R2BaseUrl"

# -- Styled Output -------------------------------------------------------------
function Write-Step { param([string]$I,[string]$M,[string]$C="White")
    Write-Host "  $I " -NoNewline -ForegroundColor Yellow
    Write-Host $M -ForegroundColor $C
    Write-Log "INFO" $M
}

function Write-Banner {
    Write-Host ""
    Write-Host "  +================================================+" -ForegroundColor Cyan
    Write-Host "  |   TA CHIP  *  PC Health Inspector              |" -ForegroundColor Cyan
    Write-Host "  +================================================+" -ForegroundColor Cyan
    Write-Host ""
}

# =============================================================================
# MAIN
# =============================================================================
Write-Banner

# -- Download exe from R2 ------------------------------------------------------
Write-Step ">>" "Downloading ta-chip-latest.exe..." "Cyan"
try {
    $ProgressPreference = "SilentlyContinue"
    Invoke-WebRequest -Uri "$R2BaseUrl/ta-chip-latest.exe" -OutFile $ExeDest -UseBasicParsing -ErrorAction Stop
    $ProgressPreference = "Continue"
    Write-Step "OK" "Downloaded ta-chip-latest.exe" "Green"
    Write-Log "INFO" "Downloaded $R2BaseUrl/ta-chip-latest.exe -> $ExeDest"
} catch {
    $ProgressPreference = "Continue"
    Write-Log "WARN" "Download failed: $($_.Exception.Message)"
    if (-not (Test-Path $ExeDest)) {
        Write-Host ""
        Write-Host "  ta-chip.exe could not be downloaded." -ForegroundColor Red
        Write-Host "  Check your internet connection and try again." -ForegroundColor Yellow
        Write-Host ""
        Write-Log "ERROR" "ta-chip.exe missing after download failure."
        $null = [Console]::ReadKey($true)
        Exit 1
    }
    Write-Step "!!" "Download failed -- using cached copy" "Yellow"
}

# -- Write config.json ---------------------------------------------------------
$cfg = [ordered]@{
    appscript_url         = $AppScriptUrl
    domain_name           = $DomainName
    domain_test_user      = $DomainUser
    domain_test_password  = $DomainPass
    ntp_tolerance_seconds = $NTPTolerance
    credentials           = [ordered]@{ login_user = $LoginUser; login_pass = $LoginPass }
}
$json = $cfg | ConvertTo-Json -Depth 3
[System.IO.File]::WriteAllText($CfgDest, $json, [System.Text.UTF8Encoding]::new($false))
Write-Log "INFO" "Wrote config.json to $CfgDest"

# -- Launch --------------------------------------------------------------------
Write-Host ""
Write-Step ">>" "Launching ta-chip..." "Cyan"
Write-Log "INFO" "Launching $ExeDest"

Start-Process -FilePath $ExeDest -WorkingDirectory $TempDir

Write-Log "INFO" "ta-chip launched."
Exit 0
