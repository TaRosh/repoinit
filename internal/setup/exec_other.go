//go:build !windows

package setup

import "os/exec"

// На Linux и macOS консольных окон нет — ничего делать не нужно.
func hideWindow(*exec.Cmd) {}
