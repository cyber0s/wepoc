//go:build windows
// +build windows

package scanner

import (
	"os/exec"
	"syscall"
)

// hideWindowOnWindows hides the command window on Windows
func hideWindowOnWindows(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}
