# Technical Assistant: Lockscreen Update (embedded — called by ta-chip)
# Version: 1.5-GA
param([string]$ScriptDir = "")

$ErrorActionPreference = 'Stop'
$lockscreenFolder = "C:\ProgramData\Wallpaper"
$lockscreenPath   = Join-Path $lockscreenFolder 'lockscreen.png'
$registryPath     = "HKLM:\SOFTWARE\Policies\Microsoft\Windows\Personalization"
$serverHost       = "wallflare.lrxrn.workers.dev"
$baseUrl          = "https://$serverHost"
$cacheMaxAgeHours = 24

function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    if (-not (Test-Path $lockscreenFolder)) { New-Item -Path $lockscreenFolder -ItemType Directory -Force | Out-Null }
    "[$(Get-Date -Format 'HH:mm:ss')] [$Level] $Message" | Add-Content -Path "$lockscreenFolder\wallpaper.log"
}

function Clear-WallpaperOverrides {
    $userProfiles = Get-CimInstance -ClassName Win32_UserProfile | Where-Object { $_.Special -eq $false }

    foreach ($profile in $userProfiles) {
        $sid = $profile.SID

        # Remove per-user lock screen registry keys (image + creative/spotlight)
        foreach ($key in @(
            "Software\Microsoft\Windows\CurrentVersion\Lock Screen",
            "Software\Microsoft\Windows\CurrentVersion\Lock Screen\Creative",
            "Software\Microsoft\Windows\CurrentVersion\Authentication\LogonUI\Creative"
        )) {
            if (Test-Path "Registry::HKEY_USERS\$sid\$key") {
                Remove-Item -Path "Registry::HKEY_USERS\$sid\$key" -Recurse -Force -ErrorAction SilentlyContinue
            }
        }

        # Disable Windows Spotlight and content delivery per user
        $cdm = "Registry::HKEY_USERS\$sid\SOFTWARE\Microsoft\Windows\CurrentVersion\ContentDeliveryManager"
        if (Test-Path $cdm) {
            @(
                "RotatingLockScreenEnabled",
                "RotatingLockScreenOverlayEnabled",
                "SubscribedContent-338387Enabled",
                "SubscribedContent-338388Enabled",
                "SubscribedContent-353696Enabled",
                "SoftLandingEnabled",
                "SystemPaneSuggestionsEnabled"
            ) | ForEach-Object {
                try { Set-ItemProperty -Path $cdm -Name $_ -Value 0 -Type DWord -Force -ErrorAction SilentlyContinue } catch {}
            }
        }

        # Remove cached transitioned wallpaper images in user themes folder
        $themesDir = "C:\Users\$($profile.LocalPath | Split-Path -Leaf)\AppData\Roaming\Microsoft\Windows\Themes"
        if (Test-Path $themesDir) {
            @("TranscodedWallpaper", "TranscodedWallpaper.jpg") | ForEach-Object {
                $f = Join-Path $themesDir $_
                if (Test-Path $f) { Remove-Item -Path $f -Force -ErrorAction SilentlyContinue }
            }
            Get-ChildItem -Path $themesDir -Filter "CachedFiles" -ErrorAction SilentlyContinue |
                Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
    Write-Log "Per-user overrides cleared."

    # Windows system lock screen image cache (under SystemData)
    $systemDataRoot = "C:\ProgramData\Microsoft\Windows\SystemData"
    if (Test-Path $systemDataRoot) {
        Get-ChildItem -Path $systemDataRoot -Recurse -ErrorAction SilentlyContinue |
            Where-Object { $_.Name -like "LockScreen_*" -or $_.Name -like "lockscreen_*" } |
            Remove-Item -Force -ErrorAction SilentlyContinue
    }

    # Windows ContentDelivery spotlight asset cache
    $spotlightCache = "$env:LocalAppData\Packages\Microsoft.Windows.ContentDeliveryManager_cw5n1h2txyewy\LocalState\Assets"
    if (Test-Path $spotlightCache) {
        Get-ChildItem -Path $spotlightCache -File -ErrorAction SilentlyContinue |
            Where-Object { $_.Length -gt 100KB -and $_.Extension -eq "" } |
            Remove-Item -Force -ErrorAction SilentlyContinue
    }
    Write-Log "System lock screen caches cleared."

    # Disable Windows Spotlight and cloud content features via HKLM policy
    $cloudContent = "HKLM:\SOFTWARE\Policies\Microsoft\Windows\CloudContent"
    if (-not (Test-Path $cloudContent)) { New-Item -Path $cloudContent -Force | Out-Null }
    @{
        "DisableWindowsSpotlightFeatures"     = 1
        "DisableLockScreenAppNotifications"   = 1
        "DisableWindowsConsumerFeatures"      = 1
        "DisableCloudOptimizedContent"        = 1
    }.GetEnumerator() | ForEach-Object {
        Set-ItemProperty -Path $cloudContent -Name $_.Key -Value $_.Value -Type DWord -Force
    }

    # Disable Spotlight on HKLM Personalization policy
    if (-not (Test-Path $registryPath)) { New-Item -Path $registryPath -Force | Out-Null }
    Set-ItemProperty -Path $registryPath -Name "NoChangingLockScreen"   -Value 1 -Type DWord -Force
    Set-ItemProperty -Path $registryPath -Name "NoLockScreenCamera"     -Value 1 -Type DWord -Force
    Set-ItemProperty -Path $registryPath -Name "NoLockScreenSlideshow"  -Value 1 -Type DWord -Force

    Write-Log "Spotlight and lock screen change policies applied."
}

function Set-LockScreen {
    param([string]$WallpaperName)
    $uri = "$baseUrl/$WallpaperName"

    if (-not (Test-Path $lockscreenFolder)) { New-Item -Path $lockscreenFolder -ItemType Directory -Force | Out-Null }

    # Cache check: skip download if file is fresh (< 24 h)
    $needsDownload = $true
    if (Test-Path $lockscreenPath) {
        $ageHours = ((Get-Date) - (Get-Item $lockscreenPath).LastWriteTime).TotalHours
        if ($ageHours -lt $cacheMaxAgeHours) {
            $needsDownload = $false
            Write-Log "Using cached wallpaper (age: $([math]::Round($ageHours, 1)) h)."
        }
    }

    if ($needsDownload) {
        $tmpFile = Join-Path $env:TEMP ("lockscreen-" + [guid]::NewGuid().ToString('N') + ".png")
        try {
            Invoke-WebRequest -Uri $uri -UseBasicParsing -OutFile $tmpFile -ErrorAction Stop -TimeoutSec 10
            Move-Item -LiteralPath $tmpFile -Destination $lockscreenPath -Force
            Write-Log "Downloaded and cached $WallpaperName."
        } catch {
            Write-Log "Download failed ($($_.Exception.Message))." "WARN"

            # Local file alongside exe (passed via -ScriptDir)
            $localFile = if ($ScriptDir) { Join-Path $ScriptDir $WallpaperName } else { $null }
            if ($localFile -and (Test-Path $localFile)) {
                Copy-Item -Path $localFile -Destination $lockscreenPath -Force
                Write-Log "Used local file: $localFile"
            } elseif (Test-Path $lockscreenPath) {
                Write-Log "No new download — keeping existing cached file." "WARN"
            } else {
                throw "No network, no local file, and no cached copy available."
            }
        }
    }

    # Set registry lockscreen policy
    if (-not (Test-Path $registryPath)) { New-Item -Path $registryPath -Force | Out-Null }
    Set-ItemProperty -Path $registryPath -Name "LockScreenImage"     -Value $lockscreenPath -Type String -Force
    Set-ItemProperty -Path $registryPath -Name "NoChangingLockScreen" -Value 1              -Type DWord  -Force
    Write-Log "Registry policy set."

    # Full override clearing
    Clear-WallpaperOverrides

    # Refresh computer policy
    $gp = Start-Process gpupdate -ArgumentList "/Target:Computer", "/Force" -Wait -PassThru -WindowStyle Hidden
    Write-Log "GPUpdate exited ($($gp.ExitCode))."

    # Cycle LogonUI for immediate visual effect
    Stop-Process -Name "LogonUI" -Force -ErrorAction SilentlyContinue
    Write-Log "LogonUI cycled."
}

# --- EXECUTION ---
$hostname = $env:COMPUTERNAME

# ModernWorkspace@APU labs: deployment blocked
if ($hostname -like "LAB-*") {
    Write-Log "BLOCKED: ModernWorkspace@APU machine ($hostname)." "WARN"
    exit 1
}

$winBuild      = [int](Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion").CurrentBuildNumber
$wallpaperName = if ($winBuild -gt 19045) { "lockscreen11.png" } else { "lockscreen10.png" }

# Exception labs keep their own wallpapers
# TL03-07 (Robotics Lab), S-04-02, TL05-TR (Financial Trading Centre)
if (($hostname -like "TL03-ROBO-*") -or ($hostname -like "S-04-02-*") -or ($hostname -like "TL05-TR-*")) {
    $wallpaperName = if ($winBuild -gt 19045) { "original_lockscreen11.png" } else { "original_lockscreen10.png" }
    Write-Log "Exception lab ($hostname): using $wallpaperName."
}

try {
    Set-LockScreen -WallpaperName $wallpaperName
    Write-Log "Lockscreen update complete."
    exit 0
} catch {
    Write-Log "FAILED: $($_.Exception.Message)" "ERROR"
    exit 1
}
