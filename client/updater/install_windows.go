// Windows stub for the detach hook. We rely on `cmd.exe /c start`
// to detach on Windows; SysProcAttr.Setsid doesn't exist there, so
// the helper is a no-op and the OS-specific Apply() path uses
// `start /min` instead.

//go:build windows

package updater

import "os/exec"

func setDetached(*exec.Cmd) {}
