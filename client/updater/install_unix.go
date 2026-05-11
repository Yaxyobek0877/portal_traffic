// Unix-specific helper for detaching the swap script from our own
// process group, so the script survives our exit cleanly. Lives
// behind a build tag because syscall.SysProcAttr.Setsid is
// platform-specific.

//go:build darwin || linux

package updater

import (
	"os/exec"
	"syscall"
)

func setDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
