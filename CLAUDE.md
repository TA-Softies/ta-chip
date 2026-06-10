# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Important

Never add `Co-Authored-By:` attribution to git commits.

## Build

```powershell
# From ta-chip/ (Go module root)
cd ta-chip

# Build for Windows (production — what gets deployed)
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -ldflags="-X ta-chip/version.Version=vX.Y.Z" -o ..\ROOT\ta-chip.exe .

# Build native (for local syntax/type checking — won't run on non-Windows)
go build ./...

# Tidy dependencies
go mod tidy
```

There are no tests. The binary only runs on Windows (uses `advapi32.dll`, `golang.org/x/sys/windows/registry`).

## Release

```bash
git tag v1.2.3
git push origin v1.2.3
```

GitHub Actions (`.github/workflows/release.yml`) cross-compiles a Windows amd64 exe and publishes it as a release asset with an auto-generated changelog (commits + file diff since the previous tag).

## Architecture

### Two distinct layers

**RP2040 (CircuitPython)** — `boot.py` + `code.py` at the CIRCUITPY drive root. Acts as a blind HID keyboard: locks the screen (`Win+L`), types credentials, then uses `Win+R` to invoke `Launch.ps1` via an elevated PowerShell run. No feedback channel — it cannot detect screen state.

**Windows host** — `ROOT/` is deployed to `CIRCUITPY:\ROOT\`. `Launch.ps1` checks GitHub releases for a newer `ta-chip.exe`, downloads if available, then runs it. `ta-chip.exe` is the Go binary.

### Go binary (`ta-chip/`)

Module path: `ta-chip`. Single `main.go` entry point; all logic is under `internal/`.

| Package | Role |
|---------|------|
| `internal/config` | Loads `config.json` from the same directory as the running exe (`os.Executable()` relative). Single `Config` struct. |
| `internal/checks` | Four files: `system.go` (hostname, NTP via raw UDP), `software.go` (registry + filesystem checks for Office/Teams/browser/wallpaper), `deepfreeze.go` (runs `DFC.exe get /ISFROZEN`, checks exit code; reads policy name from registry), `domain.go` (USERDOMAIN env check + `LogonUserW` via `advapi32.dll` lazy proc). |
| `internal/ui` | Bubble Tea TUI. `app.go` is an 11-screen state machine (`screen` iota). `keyboard.go` tracks key presses for the interactive keyboard test. `hardware.go` renders V/Y/X prompts. `styles.go` holds all Lipgloss constants. |
| `internal/submit` | Single `Submit()` function — marshals `InspectionData` to JSON, POSTs to the AppScript URL, returns the sheet row number. |
| `version` | Single `var Version = "dev"` injected at build via `-ldflags`. Printed by `--version` flag (used by `Launch.ps1` for version comparison). |

### TUI screen flow

```
Banner → AutoChecks (goroutine) → Rounder → Hardware×4 → KeyboardTest
       → SoftwareReview → DomainTest → Remarks → Review → Submit → Done
```

Auto-checks run in a background `tea.Cmd` goroutine; results arrive as a `checksCompleteMsg`. All results are collected into `model` fields before the Review screen assembles the final `submit.InspectionData`.

### Config file

`ROOT/config.json` lives alongside the exe on the CIRCUITPY drive. The two fields that must be filled before first use: `github_repo` and `appscript_url`. All other fields have sensible defaults or are optional.

### DeepFreeze

`DFC.exe get /ISFROZEN` — exit code `1` = Frozen (V), `0` = Thawed (X), process not found = N/A. Policy name has no CLI; read from `HKLM\SOFTWARE\Faronics\Deep Freeze 6` registry (key name varies by version).

### `keyboard.go` naming note

The file is named `keyboard.go`, not `keyboard_test.go` — Go treats `*_test.go` as test files and excludes them from normal builds.
