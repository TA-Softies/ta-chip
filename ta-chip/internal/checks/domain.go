package checks

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	advapi32    = windows.NewLazySystemDLL("advapi32.dll")
	logonUserW  = advapi32.NewProc("LogonUserW")
)

const (
	logon32LogonNetwork   = 3
	logon32ProviderDefault = 0
)

// CheckDomainMembership compares USERDOMAIN env var to the expected domain name.
func CheckDomainMembership(domainName string) (status, detail string) {
	current := os.Getenv("USERDOMAIN")
	if current == "" {
		return "X", "USERDOMAIN not set"
	}
	if current == domainName {
		return "V", current
	}
	return "X", fmt.Sprintf("joined to %q, expected %q", current, domainName)
}

// TestDomainLogin attempts a network logon via LogonUserW in advapi32.dll.
func TestDomainLogin(domain, username, password string) (ok bool, detail string) {
	userPtr, err := syscall.UTF16PtrFromString(username)
	if err != nil {
		return false, fmt.Sprintf("encode error: %v", err)
	}
	domainPtr, err := syscall.UTF16PtrFromString(domain)
	if err != nil {
		return false, fmt.Sprintf("encode error: %v", err)
	}
	passPtr, err := syscall.UTF16PtrFromString(password)
	if err != nil {
		return false, fmt.Sprintf("encode error: %v", err)
	}

	var token syscall.Handle
	r1, _, callErr := logonUserW.Call(
		uintptr(unsafe.Pointer(userPtr)),
		uintptr(unsafe.Pointer(domainPtr)),
		uintptr(unsafe.Pointer(passPtr)),
		logon32LogonNetwork,
		logon32ProviderDefault,
		uintptr(unsafe.Pointer(&token)),
	)

	// Zero the password from memory
	passLen := len(password)
	if passLen > 0 {
		passSlice := (*[1 << 20]uint16)(unsafe.Pointer(passPtr))[:passLen+1 : passLen+1]
		for i := range passSlice {
			passSlice[i] = 0
		}
	}

	if r1 == 0 {
		return false, fmt.Sprintf("auth failed: %v", callErr)
	}
	syscall.CloseHandle(token)
	return true, fmt.Sprintf("%s\\%s authenticated", domain, username)
}
