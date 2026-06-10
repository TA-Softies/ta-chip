package checks

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	kernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procGetDiskFreeSpaceExW  = kernel32.NewProc("GetDiskFreeSpaceExW")
	procGetTickCount64       = kernel32.NewProc("GetTickCount64")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	procBeep                 = kernel32.NewProc("Beep")
)

// GetDiskSpace returns free and total GB for the C: drive.
func GetDiskSpace() (freeGB, totalGB float64) {
	pathPtr, err := syscall.UTF16PtrFromString(`C:\`)
	if err != nil {
		return 0, 0
	}
	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	r1, _, _ := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if r1 == 0 {
		return 0, 0
	}
	const gb = 1024 * 1024 * 1024
	return float64(freeBytesAvailable) / gb, float64(totalBytes) / gb
}

// GetLastReboot returns how long ago the PC was last started.
func GetLastReboot() string {
	ms, _, _ := procGetTickCount64.Call()
	if ms == 0 {
		return "unknown"
	}
	d := time.Duration(ms) * time.Millisecond
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh ago", days, hours)
	}
	return fmt.Sprintf("%dh %dm ago", hours, mins)
}

// GetWindowsVersion reads the OS product name, display version, and build from registry.
func GetWindowsVersion() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "unknown"
	}
	defer k.Close()
	product, _, _ := k.GetStringValue("ProductName")
	display, _, _ := k.GetStringValue("DisplayVersion")
	build, _, _ := k.GetStringValue("CurrentBuildNumber")
	if product == "" {
		return "unknown"
	}
	if display != "" && build != "" {
		return fmt.Sprintf("%s %s (build %s)", product, display, build)
	}
	if build != "" {
		return fmt.Sprintf("%s (build %s)", product, build)
	}
	return product
}

type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

// GetRAM returns the total physical RAM as a human-readable string.
func GetRAM() string {
	var ms memoryStatusEx
	ms.dwLength = uint32(unsafe.Sizeof(ms))
	r1, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms)))
	if r1 == 0 {
		return "unknown"
	}
	gb := float64(ms.ullTotalPhys) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.0f GB", gb)
}

// CheckDefender checks whether Windows Defender real-time protection is enabled.
func CheckDefender() (status, detail string) {
	check := func(hive registry.Key, path string) (found bool, disabled bool) {
		k, err := registry.OpenKey(hive, path, registry.QUERY_VALUE)
		if err != nil {
			return false, false
		}
		defer k.Close()
		val, _, err := k.GetIntegerValue("DisableRealtimeMonitoring")
		return true, err == nil && val != 0
	}

	if found, disabled := check(registry.LOCAL_MACHINE,
		`SOFTWARE\Policies\Microsoft\Windows Defender\Real-Time Protection`); found {
		if disabled {
			return "X", "disabled by policy"
		}
		return "V", "real-time protection on"
	}

	if found, disabled := check(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows Defender\Real-Time Protection`); found {
		if disabled {
			return "X", "real-time protection disabled"
		}
		return "V", "real-time protection on"
	}

	return "V", "real-time protection on"
}

// CheckActivation checks Windows license status via PowerShell CIM.
func CheckActivation() (status, detail string) {
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		`(Get-CimInstance -ClassName SoftwareLicensingProduct `+
			`-Filter "ApplicationId='55c92734-d682-4d71-983e-d6ec3f16059f' AND PartialProductKey IS NOT NULL" `+
			`| Select-Object -First 1).LicenseStatus`).Output()
	if err != nil {
		return "Y", "check unavailable"
	}
	switch strings.TrimSpace(string(out)) {
	case "1":
		return "V", "activated"
	case "":
		return "Y", "status unknown"
	default:
		return "X", "not activated"
	}
}

// PlayBeep plays a short 880 Hz tone through the system audio output.
func PlayBeep() {
	procBeep.Call(880, 400)
}

// HardwareInfo holds detected hardware device names.
type HardwareInfo struct {
	Monitor  string
	Keyboard string
	Mouse    string
	Audio    string
	Camera   string
}

// GetHardwareInfo queries WMI via PowerShell for device names.
// All five queries run concurrently.
func GetHardwareInfo() HardwareInfo {
	type kv struct{ key, val string }
	ch := make(chan kv, 5)

	queries := []struct{ key, ps string }{
		{"monitor", `(Get-CimInstance Win32_DesktopMonitor | Select-Object -First 1).Name`},
		{"keyboard", `(Get-CimInstance Win32_Keyboard | Select-Object -First 1).Name`},
		{"mouse", `(Get-CimInstance Win32_PointingDevice | Select-Object -First 1).Name`},
		{"audio", `(Get-CimInstance Win32_SoundDevice | Select-Object -First 1).Name`},
		{"camera", `(Get-CimInstance Win32_PnPEntity -Filter "PNPClass='Camera'" | Select-Object -First 1).Name`},
	}

	var wg sync.WaitGroup
	for _, q := range queries {
		q := q
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := exec.Command("powershell.exe",
				"-NoProfile", "-NonInteractive", "-Command", q.ps).Output()
			val := ""
			if err == nil {
				val = strings.TrimSpace(string(out))
			}
			ch <- kv{q.key, val}
		}()
	}
	go func() { wg.Wait(); close(ch) }()

	var hw HardwareInfo
	for r := range ch {
		switch r.key {
		case "monitor":
			hw.Monitor = r.val
		case "keyboard":
			hw.Keyboard = r.val
		case "mouse":
			hw.Mouse = r.val
		case "audio":
			hw.Audio = r.val
		case "camera":
			hw.Camera = r.val
		}
	}
	return hw
}
