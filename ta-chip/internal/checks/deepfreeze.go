package checks

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"golang.org/x/sys/windows/registry"
)

const dfcPath = `C:\Windows\SysWOW64\DFC.exe`

type DeepFreezeResult struct {
	Frozen     string // "V" frozen, "X" thawed, "N/A" not installed
	PolicyName string // from registry; "N/A" if not found
	Detail     string
}

func CheckDeepFreeze() DeepFreezeResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use exit code: 0 = Thawed, 1 = Frozen
	cmd := exec.CommandContext(ctx, dfcPath, "get", "/ISFROZEN")
	err := cmd.Run()

	if err != nil {
		if ctx.Err() != nil {
			return DeepFreezeResult{Frozen: "X", PolicyName: "N/A", Detail: "DFC.exe timed out"}
		}
		// exec.ExitError means the process ran but returned non-zero
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			if code == 1 {
				return DeepFreezeResult{
					Frozen:     "V",
					PolicyName: getDFPolicyName(),
					Detail:     "Frozen",
				}
			}
			return DeepFreezeResult{
				Frozen:     "X",
				PolicyName: "N/A",
				Detail:     fmt.Sprintf("DFC exit %d", code),
			}
		}
		// Process could not start — DFC.exe not found
		return DeepFreezeResult{Frozen: "N/A", PolicyName: "N/A", Detail: "DFC.exe not found"}
	}

	// Exit code 0 = Thawed
	return DeepFreezeResult{
		Frozen:     "X",
		PolicyName: getDFPolicyName(),
		Detail:     "Thawed",
	}
}

// getDFPolicyName reads the policy/config name from the Deep Freeze registry key.
// Returns "N/A" if not found (no CLI command exposes this).
func getDFPolicyName() string {
	// Deep Freeze stores its config under one of these paths depending on version
	regPaths := []string{
		`SOFTWARE\Faronics\Deep Freeze 6`,
		`SOFTWARE\Faronics\Deep Freeze`,
		`SOFTWARE\WOW6432Node\Faronics\Deep Freeze 6`,
		`SOFTWARE\WOW6432Node\Faronics\Deep Freeze`,
	}
	valueNames := []string{"PolicyName", "ConfigName", "Policy", "WorkstationPolicy"}

	for _, path := range regPaths {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			continue
		}
		for _, name := range valueNames {
			val, _, err := k.GetStringValue(name)
			if err == nil && val != "" {
				k.Close()
				return val
			}
		}
		k.Close()
	}
	return "N/A"
}
