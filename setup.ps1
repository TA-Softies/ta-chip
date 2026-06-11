<#
    .SYNOPSIS
    TA CHIP Setup Wizard
    Flashes CircuitPython, installs HID library, and deploys ta-chip payload.
#>
param (
    [string]$CodeFile     = "code.py",
    [string]$BootFile     = "boot.py",
    [string]$SourceFolder = "ROOT",
    [string]$FirmwareDir  = "firmware",
    [string]$NukeFile     = "flash_nuke.uf2"
)

# --- SELF-ELEVATION ---
if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Start-Process powershell -ArgumentList "-ExecutionPolicy Bypass -File `"$PSCommandPath`"" -Verb RunAs
    exit
}

$ErrorActionPreference = "Stop"
$ScriptPath   = $PSScriptRoot
$RootPath     = Join-Path $ScriptPath $SourceFolder
$FirmwarePath = Join-Path $ScriptPath $FirmwareDir
$NukePath     = Join-Path $FirmwarePath $NukeFile
$BootPath     = Join-Path $ScriptPath $BootFile
$CodePath     = Join-Path $ScriptPath $CodeFile

# ==========================================
# HELPERS
# ==========================================
function Show-Header {
    Clear-Host
    Write-Host "  ============================================" -ForegroundColor DarkCyan
    Write-Host "       TA CHIP  |  Setup Wizard              " -ForegroundColor Cyan
    Write-Host "  ============================================" -ForegroundColor DarkCyan
    Write-Host ""
}

function Write-Step  { param([string]$M) Write-Host "  [>>] $M" -ForegroundColor Cyan }
function Write-OK    { param([string]$M) Write-Host "  [OK] $M" -ForegroundColor Green }
function Write-Warn  { param([string]$M) Write-Host "  [!!] $M" -ForegroundColor Yellow }
function Write-Fatal { param([string]$M) Write-Host "  [XX] $M" -ForegroundColor Red }

function Wait-ForDrive {
    param([string]$Label, [int]$TimeoutSec = 90, [string]$Prompt = "")
    if ($Prompt) { Write-Step $Prompt }
    $t = 0
    while ($t -lt $TimeoutSec) {
        $vol = Get-Volume -ErrorAction SilentlyContinue | Where-Object { $_.FileSystemLabel -eq $Label }
        if ($vol) { return $vol }
        Start-Sleep 1
        $t++
        if ($t % 10 -eq 0) { Write-Host "  ($t`s)" -ForegroundColor DarkGray }
    }
    return $null
}

$ExcludeDirs = @('.venv', '__pycache__', '.git', 'dist', '.pytest_cache')

function Copy-WithProgress {
    param([string]$Source, [string]$Destination, [string]$Label = "Copying")
    if (Test-Path -LiteralPath $Source -PathType Container) {
        $files = Get-ChildItem -Path $Source -Recurse -File | Where-Object {
            $parts = $_.FullName.Substring($Source.Length).TrimStart('\') -split '\\'
            -not ($parts | Where-Object { $ExcludeDirs -contains $_ })
        }
        $total = $files.Count
        $i = 0
        foreach ($f in $files) {
            $i++
            $rel = $f.FullName.Substring($Source.Length).TrimStart("\")
            $dst = Join-Path $Destination $rel
            $dir = Split-Path $dst -Parent
            if (-not (Test-Path $dir)) { New-Item -Path $dir -ItemType Directory -Force | Out-Null }
            Write-Progress -Activity $Label -Status $rel -PercentComplete ([math]::Round(($i / $total) * 100))
            Copy-Item -Path $f.FullName -Destination $dst -Force
        }
    } else {
        Write-Progress -Activity $Label -Status (Split-Path $Source -Leaf) -PercentComplete 50
        Copy-Item -Path $Source -Destination $Destination -Force
    }
    Write-Progress -Activity $Label -Completed
}

# ==========================================
# STARTUP CHECKS
# ==========================================
Show-Header

$missing = @()
if (-not (Test-Path $FirmwarePath)) { $missing += "firmware\ folder" }
if (-not (Test-Path $BootPath))     { $missing += $BootFile }
if (-not (Test-Path $CodePath))     { $missing += $CodeFile }
if (-not (Test-Path $RootPath))     { $missing += "ROOT\ folder" }
if (-not (Test-Path $NukePath))     { $missing += "firmware\$NukeFile" }

if ($missing.Count -gt 0) {
    Write-Fatal "Missing required files:"
    $missing | ForEach-Object { Write-Host "       - $_" -ForegroundColor Red }
    Write-Host ""
    Read-Host "  Press Enter to exit"
    exit 1
}

# --- SCAN FIRMWARE PAIRS ---
$allZips = Get-ChildItem -Path $FirmwarePath -Filter "*.zip" -ErrorAction SilentlyContinue
$allUf2s = Get-ChildItem -Path $FirmwarePath -Filter "*.uf2"  -ErrorAction SilentlyContinue
$pairs   = @{}

foreach ($z in $allZips) {
    if ($z.Name -match "bundle-(\d+)\.x") {
        $v = [int]$Matches[1]
        if (-not $pairs[$v]) { $pairs[$v] = @{} }
        $pairs[$v]["Zip"] = $z
    }
}
foreach ($u in $allUf2s) {
    if ($u.Name -notmatch "nuke" -and $u.Name -match "[-_](\d+)\.\d+\.\d+") {
        $v = [int]$Matches[1]
        if ($pairs[$v]) { $pairs[$v]["Uf2"] = $u }
    }
}

$bestPair = $pairs.GetEnumerator() |
            Where-Object { $_.Value["Zip"] -and $_.Value["Uf2"] } |
            Sort-Object Key -Descending |
            Select-Object -First 1

if (-not $bestPair) {
    Write-Fatal "No matching CircuitPython .uf2 + library .zip found in firmware\."
    Write-Warn  "Filename must contain a version like '-9.x' and '-9.0.0'."
    Read-Host "  Press Enter to exit"
    exit 1
}

$fwVer = $bestPair.Key
$fwZip = $bestPair.Value["Zip"]
$fwUf2 = $bestPair.Value["Uf2"]
Write-OK "Firmware: CircuitPython v$fwVer.x  ($($fwUf2.Name))"
Write-OK "Library:  $($fwZip.Name)"
Write-Host ""

# ==========================================
# MENU
# ==========================================
Write-Host "  SELECT OPERATION:" -ForegroundColor Yellow
Write-Host "  [1]  SMART INSTALL   - Auto-detects device mode and installs everything"
Write-Host "  [2]  FORCE RESET     - Wipe + full reinstall (use if device is stuck)"
Write-Host "  [3]  UPDATE PAYLOAD  - Deploy code.py + ROOT only (skips firmware and libs)"
Write-Host "  [Q]  QUIT"
Write-Host ""

$validChoices = @("1","2","3","Q")
do {
    $choice = (Read-Host "  >").Trim().ToUpper()
    if ($choice -notin $validChoices) { Write-Warn "Enter 1, 2, 3, or Q." }
} while ($choice -notin $validChoices)

if ($choice -eq "Q") { exit 0 }

# ==========================================
# DEVICE ACQUISITION
# ==========================================
Show-Header
$targetDrive = $null
$installMode = ""   # FULL | UPDATE | PAYLOAD_ONLY

if ($choice -eq "2") {
    # --- FORCE RESET ---
    Write-Warn "FORCE RESET will erase everything on the device."
    $confirm = (Read-Host "  Type YES to confirm").Trim().ToUpper()
    if ($confirm -ne "YES") { Write-Host "  Aborted."; Read-Host; exit 0 }

    Write-Host ""
    Write-Step "Hold BOOT on the board, plug it in, then release BOOT."
    $bl = Wait-ForDrive -Label "RPI-RP2" -TimeoutSec 120 -Prompt "Waiting for RPI-RP2 bootloader drive..."
    Write-Host ""
    if (-not $bl) {
        Write-Fatal "RPI-RP2 not found within 120s. Check cable and BOOT button."
        Read-Host; exit 1
    }
    Write-OK "RPI-RP2 on $($bl.DriveLetter):"

    Write-Step "Flashing nuke..."
    Copy-WithProgress -Source $NukePath -Destination "$($bl.DriveLetter):\" -Label "Flashing Nuke"
    Write-OK "Nuke complete. Waiting for board to reboot..."
    Start-Sleep 4

    $bl2 = Wait-ForDrive -Label "RPI-RP2" -TimeoutSec 60 -Prompt "Waiting for RPI-RP2 to reappear..."
    Write-Host ""
    if (-not $bl2) {
        Write-Fatal "Board did not come back as RPI-RP2. Try unplugging and replug in BOOT mode."
        Read-Host; exit 1
    }
    Write-OK "Board ready - $($bl2.DriveLetter):"
    $targetDrive = $bl2
    $installMode = "FULL"
}

elseif ($choice -eq "1") {
    # --- SMART INSTALL ---
    Write-Step "Plug in the board (normal for update, BOOT-mode for fresh firmware)."
    Write-Host "  Watching for RPI-RP2 or CIRCUITPY..." -ForegroundColor DarkGray
    $t = 0
    while ($t -lt 120) {
        $bl = Get-Volume -ErrorAction SilentlyContinue | Where-Object { $_.FileSystemLabel -eq "RPI-RP2" }
        $cp = Get-Volume -ErrorAction SilentlyContinue | Where-Object { $_.FileSystemLabel -eq "CIRCUITPY" }
        if ($bl) { $targetDrive = $bl; $installMode = "FULL";   break }
        if ($cp) { $targetDrive = $cp; $installMode = "UPDATE"; break }
        Start-Sleep 1; $t++
        if ($t % 10 -eq 0) { Write-Host "  ($t`s)" -ForegroundColor DarkGray }
    }
    Write-Host ""
    if (-not $targetDrive) {
        Write-Fatal "No device found after 120s."
        Read-Host; exit 1
    }
    Write-OK "Detected: $($targetDrive.FileSystemLabel) on $($targetDrive.DriveLetter):"
    if ($installMode -eq "UPDATE") {
        Write-Warn "Found CIRCUITPY - firmware will be skipped. Use [2] to force a clean install."
    }
}

elseif ($choice -eq "3") {
    # --- PAYLOAD ONLY ---
    $cp = Get-Volume -ErrorAction SilentlyContinue | Where-Object { $_.FileSystemLabel -eq "CIRCUITPY" }
    if (-not $cp) {
        Write-Step "Plug in the device, then press Enter..."
        Read-Host | Out-Null
        $cp = Get-Volume -ErrorAction SilentlyContinue | Where-Object { $_.FileSystemLabel -eq "CIRCUITPY" }
    }
    if (-not $cp) {
        Write-Fatal "CIRCUITPY drive not found."
        Read-Host; exit 1
    }
    Write-OK "CIRCUITPY on $($cp.DriveLetter):"
    $targetDrive = $cp
    $installMode = "PAYLOAD_ONLY"
}

$dest = "$($targetDrive.DriveLetter):\"

# ==========================================
# STEP 0: CLEAR DIRTY BIT (if present)
# ==========================================
$driveLetter = $targetDrive.DriveLetter
$dirtyResult = fsutil dirty query "${driveLetter}:" 2>&1
if ($dirtyResult -match "is Dirty") {
    Write-Host ""
    Write-Step "Dirty bit detected on ${driveLetter}: - running chkdsk /f..."
    try {
        $chkOutput = & chkdsk "${driveLetter}:" /f /x 2>&1
        $dirtyCheck = fsutil dirty query "${driveLetter}:" 2>&1
        if ($dirtyCheck -match "is NOT Dirty") {
            Write-OK "Dirty bit cleared."
        } else {
            Write-Warn "chkdsk ran but dirty bit may still be set. Continuing anyway."
        }
    } catch {
        Write-Warn "Could not run chkdsk (drive may be locked). Continuing anyway."
    }
}

# ==========================================
# STEP 1: FLASH FIRMWARE (FULL only)
# ==========================================
if ($installMode -eq "FULL") {
    Write-Host ""
    Write-Step "Flashing CircuitPython $($fwUf2.Name)..."
    Copy-WithProgress -Source $fwUf2.FullName -Destination $dest -Label "Flashing Firmware"
    Write-OK "Firmware written. Waiting for CIRCUITPY..."

    $cp = Wait-ForDrive -Label "CIRCUITPY" -TimeoutSec 60 -Prompt ""
    Write-Host ""
    if (-not $cp) {
        Write-Fatal "CIRCUITPY did not appear after firmware flash (60s timeout)."
        Write-Warn  "Try unplugging and re-running with option [2]."
        Read-Host; exit 1
    }
    Write-OK "CIRCUITPY ready on $($cp.DriveLetter):"
    $targetDrive = $cp
    $dest = "$($cp.DriveLetter):\"
}

# ==========================================
# STEP 2: INSTALL HID LIBRARY (FULL + UPDATE)
# ==========================================
if ($installMode -in @("FULL","UPDATE")) {
    Write-Host ""
    Write-Step "Installing adafruit_hid library..."

    $tmp = Join-Path $env:TEMP "ta_chip_$([System.IO.Path]::GetRandomFileName().Replace('.',''))"
    Expand-Archive -Path $fwZip.FullName -DestinationPath $tmp -Force

    $hidSrc = Get-ChildItem -Path $tmp -Recurse -Directory |
              Where-Object { $_.Name -eq "adafruit_hid" } |
              Select-Object -First 1

    if (-not $hidSrc) {
        Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
        Write-Fatal "adafruit_hid folder not found inside $($fwZip.Name)."
        Read-Host; exit 1
    }

    $hidDest = Join-Path $dest "lib\adafruit_hid"
    if (Test-Path $hidDest) { Remove-Item $hidDest -Recurse -Force }
    New-Item -Path $hidDest -ItemType Directory -Force | Out-Null

    Copy-WithProgress -Source $hidSrc.FullName -Destination $hidDest -Label "Installing adafruit_hid"
    Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
    Write-OK "adafruit_hid installed."
}

# ==========================================
# STEP 3: DEPLOY PAYLOAD
# ==========================================
Write-Host ""
Write-Step "Deploying payload..."

Copy-WithProgress -Source $BootPath -Destination (Join-Path $dest "boot.py") -Label "Copying boot.py"
Write-OK "boot.py"

Copy-WithProgress -Source $CodePath -Destination (Join-Path $dest "code.py") -Label "Copying code.py"
Write-OK "code.py"

$destRoot = Join-Path $dest "ROOT"
if (Test-Path $destRoot) {
    Write-Step "Removing old ROOT..."
    cmd /c rmdir /s /q "$destRoot"
}
Copy-WithProgress -Source $RootPath -Destination $destRoot -Label "Copying ROOT folder"
Write-OK "ROOT folder"

# ==========================================
# DONE
# ==========================================
Write-Host ""
Write-Host "  ============================================" -ForegroundColor DarkCyan
Write-Host "  [OK] SETUP COMPLETE" -ForegroundColor Green
Write-Host "  ============================================" -ForegroundColor DarkCyan
Write-Host ""

$steps = switch ($installMode) {
    "FULL"         { "Firmware, HID library, payload" }
    "UPDATE"       { "HID library, payload" }
    "PAYLOAD_ONLY" { "Payload only (code.py + ROOT)" }
}
Write-Host "  Installed: $steps" -ForegroundColor White
Write-Host ""
Write-Host "  Safely eject the drive, then unplug and replug to run." -ForegroundColor Yellow
Write-Host ""
Read-Host "  Press Enter to exit"
