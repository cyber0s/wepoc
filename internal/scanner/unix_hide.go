//go:build !windows
// +build !windows

package scanner

import (
	"os/exec"
)

// hideWindowOnWindows is a no-op on non-Windows systems
func hideWindowOnWindows(cmd *exec.Cmd) {
	// No-op on non-Windows systems
}
