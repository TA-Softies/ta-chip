package checks

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// CheckWallpaper reads the current wallpaper from registry.
// If expected is non-empty, checks for exact match.
// If empty, checks that a non-default wallpaper is set.
func CheckWallpaper(expected string) (status, detail string) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Desktop`, registry.QUERY_VALUE)
	if err != nil {
		return "X", "cannot read registry"
	}
	defer k.Close()

	val, _, err := k.GetStringValue("Wallpaper")
	if err != nil || val == "" {
		return "X", "no wallpaper set"
	}

	if expected != "" {
		if strings.EqualFold(val, expected) {
			return "V", filepath.Base(val)
		}
		return "X", "wrong wallpaper: " + filepath.Base(val)
	}

	// Default Windows wallpaper paths
	lower := strings.ToLower(val)
	if strings.Contains(lower, `windows\web\wallpaper\windows`) ||
		strings.HasSuffix(lower, "img0.jpg") ||
		val == "" {
		return "X", "default Windows wallpaper"
	}
	return "V", filepath.Base(val)
}

// CheckOffice returns V if any Office installation is detected.
func CheckOffice() (status, detail string) {
	// Check Click-to-Run config key
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Office\ClickToRun\Configuration`,
		registry.QUERY_VALUE)
	if err == nil {
		k.Close()
		return "V", "Office (Click-to-Run)"
	}

	// Check classic MSI install
	k2, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Office`,
		registry.ENUMERATE_SUB_KEYS)
	if err == nil {
		defer k2.Close()
		subs, _ := k2.ReadSubKeyNames(-1)
		for _, s := range subs {
			if s != "ClickToRun" && s != "Common" && s != "Delivery" {
				return "V", "Office " + s
			}
		}
	}

	// Fallback: check for winword.exe
	paths := []string{
		`C:\Program Files\Microsoft Office\root\Office16\WINWORD.EXE`,
		`C:\Program Files (x86)\Microsoft Office\root\Office16\WINWORD.EXE`,
		`C:\Program Files\Microsoft Office\Office16\WINWORD.EXE`,
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return "V", "Office 2016+"
		}
	}

	return "X", "not found"
}

// CheckTeams returns V if Microsoft Teams is installed.
func CheckTeams() (status, detail string) {
	localApp := os.Getenv("LOCALAPPDATA")

	paths := []string{
		`C:\Program Files\Microsoft\Teams\current\Teams.exe`,
		`C:\Program Files (x86)\Microsoft\Teams\current\Teams.exe`,
		filepath.Join(localApp, `Microsoft\Teams\current\Teams.exe`),
		// Teams 2.x (new Teams)
		`C:\Program Files\WindowsApps\MSTeams_*`,
	}

	for _, p := range paths {
		if strings.Contains(p, "*") {
			// glob check for new Teams
			matches, _ := filepath.Glob(p)
			if len(matches) > 0 {
				return "V", "Teams (new)"
			}
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return "V", "Teams"
		}
	}

	// Registry check for Teams machine-wide install
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Teams`,
		registry.QUERY_VALUE)
	if err == nil {
		k.Close()
		return "V", "Teams (registry)"
	}

	return "X", "not found"
}

// CheckBrowser returns V if any modern browser is found.
func CheckBrowser() (status, detail string) {
	progFiles := os.Getenv("ProgramFiles")
	progFiles86 := os.Getenv("ProgramFiles(x86)")
	localApp := os.Getenv("LOCALAPPDATA")

	browsers := map[string][]string{
		"Chrome": {
			filepath.Join(progFiles, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(progFiles86, `Google\Chrome\Application\chrome.exe`),
			filepath.Join(localApp, `Google\Chrome\Application\chrome.exe`),
		},
		"Edge": {
			filepath.Join(progFiles, `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(progFiles86, `Microsoft\Edge\Application\msedge.exe`),
		},
		"Firefox": {
			filepath.Join(progFiles, `Mozilla Firefox\firefox.exe`),
			filepath.Join(progFiles86, `Mozilla Firefox\firefox.exe`),
		},
	}

	var found []string
	for name, paths := range browsers {
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				found = append(found, name)
				break
			}
		}
	}

	if len(found) == 0 {
		return "X", "no browser found"
	}
	return "V", strings.Join(found, ", ")
}
