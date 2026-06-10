# ta-chip

**Technical Assistant Computer Health Inspection Platform**

A RP2040 USB device that automatically logs into a lab PC and launches a guided inspection tool. The tool walks through hardware and software checks, then submits the results directly to a Google Sheet.

---

## How it works

1. Plug the RP2040 into a lab PC
2. It locks the screen, signs in as `.\student`, then launches the inspection tool automatically
3. The tool runs automated software checks (Office, Teams, browser, DeepFreeze, domain, time sync, wallpaper)
4. You confirm hardware items (display, Kensington lock, cable management, tidiness) and run a keyboard/mouse test
5. Results are submitted to Google Sheets with a single keypress

---

## Requirements

- RP2040 board (Raspberry Pi Pico or similar) with CircuitPython 9.x
- `adafruit_hid` CircuitPython library on the device
- Windows 10/11 lab PC joined to the `TECHLAB` domain
- A Google account to host the AppScript
- Go 1.22+ (only needed if building from source)

---

## Setup

### 1. Google Sheets

1. Create a new Google Sheet
2. Go to **Extensions → Apps Script**
3. Delete any existing code and paste the contents of [`appscript/Code.gs`](appscript/Code.gs)
4. Click **Deploy → New deployment**
   - Type: **Web app**
   - Execute as: **Me**
   - Who has access: **Anyone**
5. Click **Deploy** and copy the `/exec` URL — you'll need it in the next step

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
| `expected_wallpaper` | Full path to the required wallpaper file. Leave blank to just check that *any* custom wallpaper is set |
| `ntp_tolerance_seconds` | How many seconds of clock drift is acceptable before marking as `Y` (default 300 = 5 min) |
| `credentials` | Username and password the RP2040 types during auto-login |

### 3. Flash the RP2040

1. Download CircuitPython 9.x for your board from [circuitpython.org](https://circuitpython.org/downloads)
2. Hold BOOTSEL and plug in the RP2040 — it mounts as `RPI-RP2`
3. Drag the `.uf2` file onto the drive — it reboots as `CIRCUITPY`
4. Install the `adafruit_hid` library:
   - Download the CircuitPython library bundle from [circuitpython.org/libraries](https://circuitpython.org/libraries)
   - Copy `adafruit_hid/` from the bundle into `CIRCUITPY/lib/`
5. Copy `boot.py` and `code.py` to the **root** of the `CIRCUITPY` drive

> **Note:** After copying `boot.py`, the CIRCUITPY drive will no longer appear as a USB drive on the host PC. To edit files on it again, hold a button (or short GP0 to GND on a Pico) while plugging in, which skips `boot.py`.

### 4. Deploy to the device

Copy the entire `ROOT/` folder to the root of the `CIRCUITPY` drive:

```
CIRCUITPY/
├── boot.py
├── code.py
└── ROOT/
    ├── Launch.ps1
    ├── config.json
    └── ta-chip.exe        ← download from Releases, or build from source
```

`ta-chip.exe` is downloaded automatically by `Launch.ps1` on first run if internet is available. To pre-load it, grab the latest `.exe` from the [Releases](../../releases) page and place it in `ROOT/`.

---

## Usage

1. Plug the RP2040 into any lab PC
2. The onboard LED lights up — the device is running the login sequence
3. After the LED turns off, the inspection tool opens automatically
4. Follow the on-screen prompts:
   - **Auto-checks screen** — wait for all checks to complete, then press Enter
   - **Rounder** — type your name and press Enter
   - **Hardware checks** — press `V`, `Y`, or `X` for each item (or use arrow keys + Enter)
   - **Keyboard test** — press every key you want to test, click each mouse button, then press Enter
   - **Software review** — auto-check results are shown; use arrow keys to navigate and `V`/`Y`/`X` to override any result
   - **Domain test** — shows membership and login test result; override with `V`/`Y`/`X` if needed
   - **Remarks** — optional notes; press Tab to skip
   - **Review** — check everything looks correct, then press Enter to submit
5. Results appear instantly in the Google Sheet

### Status codes

| Code | Meaning |
|------|---------|
| `V` | Working / present / pass |
| `Y` | Partially working / unsecured / minor issue |
| `X` | Faulty / missing / fail |

---

## Auto-update

`Launch.ps1` checks the GitHub releases API every time it runs. If a newer `ta-chip.exe` is available, it downloads and replaces the local copy automatically before launching. No manual updates needed.

---

## Building from source

```powershell
cd ta-chip
go build -ldflags="-X ta-chip/version.Version=v1.0.0" -o ..\ROOT\ta-chip.exe .
```

Requires Go 1.22+. The binary is Windows-only (`GOOS=windows GOARCH=amd64`).

---

## Releasing

Tag the commit and push — GitHub Actions builds the exe and publishes the release automatically:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The release includes a changelog of all commits since the previous tag.

---

## DeepFreeze notes

- `ta-chip.exe` calls `C:\Windows\SysWOW64\DFC.exe get /ISFROZEN`
- Exit code `1` = Frozen (pass), `0` = Thawed (fail)
- Policy name is read from the registry under `HKLM\SOFTWARE\Faronics\Deep Freeze 6`; the exact key name varies by version — check with `reg query "HKLM\SOFTWARE\Faronics" /s` on a lab PC if it shows `N/A`

---

## Project structure

```
ta-chip/
├── boot.py                  CircuitPython: enable HID keyboard, disable serial
├── code.py                  CircuitPython: auto-login + launch sequence
├── ROOT/                    Deploy this folder to CIRCUITPY:\ROOT\
│   ├── Launch.ps1           PowerShell launcher (auto-update + run)
│   └── config.json          Runtime configuration
├── ta-chip/                 Go source
│   ├── main.go
│   └── internal/
│       ├── checks/          Automated checks (system, software, DeepFreeze, domain)
│       ├── config/          Config loading
│       ├── submit/          Google Sheets submission
│       └── ui/              Bubble Tea TUI
├── appscript/
│   └── Code.gs              Google Apps Script for the Sheet
└── .github/workflows/
    └── release.yml          Build + publish on version tag
```
