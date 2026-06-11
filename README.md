# ta-chip

**Technical Assistant Computer Health Inspection Platform**

A RP2040 USB device that automatically logs into a lab PC and launches a guided inspection tool. The tool runs automated checks, walks through hardware and software items, then submits the results directly to a Google Sheet.

---

## How it works

1. Plug the RP2040 into a lab PC
2. The device signs out the current user, signs in as `.\student`, then launches the inspection tool automatically via an elevated PowerShell run
3. The tool runs automated checks in parallel:
   - System: hostname, time sync (NTP), disk space, last reboot, Windows version, RAM
   - Software: lockscreen wallpaper, Microsoft Office, Teams, internet (TCP), DeepFreeze frozen/policy, Windows Defender, Windows Activation
   - Hardware detection: monitor, keyboard, mouse, audio device, camera
   - Domain: membership check + test login
4. You confirm hardware items (display, Kensington lock, cable management, tidiness) and run a keyboard/mouse test
5. Software results are reviewed with the option to override any status
6. Results are submitted to Google Sheets automatically

---

## Requirements

- RP2040 board (VCC-GND YD RP2040 or similar) with CircuitPython 9.x
- `adafruit_hid` CircuitPython library installed on the device
- Windows 10/11 lab PC joined to the `TECHLAB` domain
- A Google account to host the AppScript
- Go 1.22+ (only needed if building from source)

---

## Setup

### 1. Google Sheets

1. Create a new Google Sheet
2. Go to **Extensions → Apps Script**
3. Paste the contents of [`appscript/Code.gs`](appscript/Code.gs)
4. Click **Deploy → New deployment**
   - Type: **Web app**
   - Execute as: **Me**
   - Who has access: **Anyone**
5. Copy the `/exec` URL — you'll need it in the next step

### 2. Config file

Edit `ROOT/config.json`:

```json
{
  "github_repo": "TA-Softies/ta-chip",
  "appscript_url": "https://script.google.com/macros/s/YOUR_SCRIPT_ID/exec",
  "domain_name": "TECHLAB",
  "domain_test_user": "student",
  "domain_test_password": "",
  "expected_wallpaper": "",
  "ntp_tolerance_seconds": 300,
  "credentials": {
    "login_user": ".\\student",
    "login_pass": "student"
  }
}
```

| Field | Description |
|-------|-------------|
| `github_repo` | GitHub repo used by the launcher to check for updates |
| `appscript_url` | The `/exec` URL from step 1 |
| `domain_name` | Domain to verify against (`USERDOMAIN` env var) |
| `domain_test_user` | Account used to test domain login |
| `domain_test_password` | Password for the domain test account (blank if none) |
| `expected_wallpaper` | Full path to the required lockscreen image. Leave blank to just check that any custom wallpaper is set |
| `ntp_tolerance_seconds` | Clock drift tolerance before marking as `Y` (default 300 = 5 min) |
| `credentials` | Username and password the RP2040 types during auto-login |

### 3. Flash the RP2040

Run the setup wizard — it handles everything automatically:

```
Right-click setup.ps1 → Run with PowerShell
```

| Option | When to use |
|--------|-------------|
| **[1] Smart Install** | First-time setup or normal update. Auto-detects whether the board is in bootloader (RPI-RP2) or normal (CIRCUITPY) mode |
| **[2] Force Reset** | Board is stuck or corrupted. Wipes the flash, reflashes firmware, reinstalls everything |
| **[3] Update Payload** | You only changed `code.py` or `ROOT/` contents and the firmware is already correct |

The wizard flashes CircuitPython, installs `adafruit_hid`, and copies `boot.py`, `code.py`, and `ROOT/` to the device. The `firmware/` folder already contains the required files.

> **After `boot.py` is installed:** the CIRCUITPY drive will no longer appear on the host PC (by design — `boot.py` disables it). To edit files again, hold the BOOT button while plugging in to skip `boot.py`.

---

## Usage

1. Plug the RP2040 into a lab PC that has a user already logged in
2. The onboard LED lights up while the login sequence runs
3. After the LED turns off the inspection tool opens — work through each screen in order:

| Step | Screen | What to do |
|------|--------|------------|
| 1/7 | **Auto Checks** | Wait for checks to complete, then press Enter |
| 2/7 | **Rounder** | Type your name and press Enter |
| 3/7 | **Hardware** | Press `V`, `Y`, or `X` for each item (or arrow keys + Enter) |
| 4/7 | **Keyboard & Mouse Test** | Press keys and click mouse buttons, then press Enter |
| 5/7 | **Software Review** | Review auto-check results; navigate with `↑↓`, override with `V`/`Y`/`X` |
| 6/7 | **Domain Test** | Shows membership + login test; override with `V`/`Y`/`X` if needed |
| 7/7 | **Remarks** | Optional notes; press Tab to skip |
| Review | **Review** | Confirm everything looks correct, then press Enter to submit |

4. Results appear in the Google Sheet instantly

### Software review key actions

When a row is focused in the Software Review screen, extra keys are available:

| Key | Row | Action |
|-----|-----|--------|
| `F` | Lockscreen Wallpaper | Downloads and applies the correct lockscreen image |
| `B` | Audio | Plays a short beep to confirm audio output is working |
| `L` | Camera | Opens the Windows Camera app to verify the camera |

### Status codes

| Code | Meaning |
|------|---------|
| `V` | Working / present / pass |
| `Y` | Partially working / minor issue |
| `X` | Faulty / missing / fail |

---

## Auto-update

`Launch.ps1` checks the GitHub releases API every time it runs. If a newer `ta-chip.exe` is available it downloads and replaces the local copy before launching. No manual updates needed on lab PCs.

---

## Building from source

```powershell
cd ta-chip
$env:GOOS="windows"; $env:GOARCH="amd64"
go build -ldflags="-X ta-chip/version.Version=v0.3.0" -o ..\ROOT\ta-chip.exe .
```

Requires Go 1.22+. The binary is Windows-only (uses `advapi32.dll` and Windows registry APIs).

When running locally without `-ldflags` the version shows as `dev-<githash>`.

---

## Releasing

Tag the commit and push — GitHub Actions cross-compiles the exe and publishes the release automatically:

```bash
git tag v0.4.0
git push origin v0.4.0
```

---

## DeepFreeze notes

- Calls `C:\Windows\SysWOW64\DFC.exe get /ISFROZEN`
- Exit code `1` = Frozen (V), `0` = Thawed (X), not found = N/A
- Policy name is read from `HKLM\SOFTWARE\Faronics\Deep Freeze 6` — the exact key name varies by version; run `reg query "HKLM\SOFTWARE\Faronics" /s` on a lab PC if it shows N/A

---

## Project structure

```
ta-chip/
├── boot.py                    CircuitPython: enable HID keyboard, disable USB drive
├── code.py                    CircuitPython: auto sign-out/sign-in + launch sequence
├── setup.ps1                  Board setup wizard (flash firmware, install libs, deploy)
├── firmware/
│   ├── flash_nuke.uf2         Wipes RP2040 flash (used by Force Reset)
│   ├── adafruit-circuitpython-vcc_gnd_yd_rp2040-*.uf2   CircuitPython firmware
│   └── adafruit-circuitpython-bundle-9.x-*.zip          Library bundle (adafruit_hid)
├── ROOT/                      Deployed to CIRCUITPY:\ROOT\ by setup.ps1
│   ├── Launch.ps1             Auto-update launcher
│   ├── config.json            Runtime configuration
│   └── ta-chip.exe            Windows inspection tool (download from Releases)
├── ta-chip/                   Go source
│   ├── main.go
│   └── internal/
│       ├── checks/
│       │   ├── system.go      Hostname, NTP time check
│       │   ├── sysinfo.go     Disk, RAM, reboot time, Windows version, Defender,
│       │   │                  Activation, hardware WMI queries, beep
│       │   ├── software.go    Office, Teams, internet, wallpaper checks
│       │   ├── deepfreeze.go  DFC.exe + registry policy name
│       │   └── domain.go      Domain membership + LogonUserW test
│       ├── config/            Config loading
│       ├── submit/            Google Sheets submission
│       └── ui/
│           ├── app.go         11-screen state machine
│           ├── hardware.go    Hardware prompt screens
│           ├── keyboard.go    Keyboard/mouse test screen
│           ├── set_lockscreen.ps1   Embedded lockscreen fix script
│           └── styles.go      Lipgloss styles
├── appscript/
│   └── Code.gs                Google Apps Script (sheet writer + Discord webhook)
└── .github/workflows/
    └── release.yml            Build + publish on version tag
```
