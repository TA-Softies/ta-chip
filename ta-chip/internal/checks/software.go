package checks

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

// CheckInternet verifies internet connectivity by dialing Google DNS.
func CheckInternet() (status, detail string) {
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 3*time.Second)
	if err != nil {
		return "X", "no internet"
	}
	conn.Close()
	return "V", "connected"
}

// CheckWallpaper checks the lockscreen wallpaper via GPO/MDM registry keys.
func CheckWallpaper(expected string) (status, detail string) {
	type entry struct {
		hive registry.Key
		key  string
		val  string
	}
	entries := []entry{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\PersonalizationCSP`, "LockScreenImagePath"},
		{registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\Personalization`, "LockScreenImage"},
	}

	for _, e := range entries {
		k, err := registry.OpenKey(e.hive, e.key, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		val, _, err := k.GetStringValue(e.val)
		k.Close()
		if err != nil || val == "" {
			continue
		}
		if expected != "" {
			if strings.EqualFold(val, expected) {
				return "V", filepath.Base(val)
			}
			return "X", "wrong: " + filepath.Base(val)
		}
		return "V", filepath.Base(val)
	}

	return "X", "lockscreen wallpaper not set"
}

// CheckOffice returns V if any Office installation is detected.
func CheckOffice() (status, detail string) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Office\ClickToRun\Configuration`,
		registry.QUERY_VALUE)
	if err == nil {
		k.Close()
		return "V", "Click-to-Run — verify login & launch"
	}

	k2, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Office`,
		registry.ENUMERATE_SUB_KEYS)
	if err == nil {
		defer k2.Close()
		subs, _ := k2.ReadSubKeyNames(-1)
		for _, s := range subs {
			if s != "ClickToRun" && s != "Common" && s != "Delivery" {
				return "V", "Office " + s + " — verify login & launch"
			}
		}
	}

	paths := []string{
		`C:\Program Files\Microsoft Office\root\Office16\WINWORD.EXE`,
		`C:\Program Files (x86)\Microsoft Office\root\Office16\WINWORD.EXE`,
		`C:\Program Files\Microsoft Office\Office16\WINWORD.EXE`,
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return "V", "Office 2016+ — verify login & launch"
		}
	}

	return "X", "not found"
}

// CheckTeams returns V if Microsoft Teams is installed, Y if installed but no session found.
func CheckTeams() (status, detail string) {
	localApp := os.Getenv("LOCALAPPDATA")
	appData := os.Getenv("APPDATA")

	paths := []string{
		`C:\Program Files\Microsoft\Teams\current\Teams.exe`,
		`C:\Program Files (x86)\Microsoft\Teams\current\Teams.exe`,
		filepath.Join(localApp, `Microsoft\Teams\current\Teams.exe`),
		`C:\Program Files\WindowsApps\MSTeams_*`,
	}

	installed := false
	for _, p := range paths {
		if strings.Contains(p, "*") {
			matches, _ := filepath.Glob(p)
			if len(matches) > 0 {
				installed = true
				break
			}
			continue
		}
		if _, err := os.Stat(p); err == nil {
			installed = true
			break
		}
	}

	if !installed {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Teams`, registry.QUERY_VALUE)
		if err == nil {
			k.Close()
			installed = true
		}
	}

	if !installed {
		return "X", "not found"
	}

	// Check for an existing session (classic Teams storage)
	sessionFile := filepath.Join(appData, `Microsoft\Teams\storage.json`)
	if _, err := os.Stat(sessionFile); err == nil {
		return "V", "Installed, session found"
	}

	return "Y", "Installed — verify sign-in"
}
