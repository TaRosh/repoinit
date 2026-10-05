//go:build windows

package setup

import (
	"os/exec"
	"syscall"
)

// createNoWindow — флаг WinAPI CREATE_NO_WINDOW: без него GUI-приложение
// при каждом запуске git показывало бы мелькающее окно консоли.
const createNoWindow = 0x08000000

func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
