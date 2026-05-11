// Auto-run on system startup.
//
// User asked Portal to come back automatically after a reboot — same
// experience the user gets from a chat / video / dock app. Each OS
// has its own hook:
//
//   - macOS: ~/Library/LaunchAgents/uz.1pro.portal.plist
//     (a LaunchAgent that runs Portal.app/Contents/MacOS/Portal at
//     login)
//   - Windows: HKCU\Software\Microsoft\Windows\CurrentVersion\Run
//     (a string value pointing at Portal.exe)
//   - Linux:   ~/.config/autostart/portal.desktop
//     (a freedesktop.org .desktop file that the desktop session
//     reads on login)
//
// The hook points at the running binary's own path (os.Executable),
// so reinstalling Portal to a different location and toggling auto-
// run again rewrites the entry to the new path automatically. The
// auto-run flag is also persisted in the local SQLite store
// (KeyAutoRun) so the Settings UI can read the current state without
// a roundtrip to the OS.

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"portal_traffic_client/storage"
)

// IsAutoRun returns the persisted preference. Reads storage; safe
// to call without a portal session active.
func (a *App) IsAutoRun() bool {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return false
	}
	v, _ := store.GetSetting(storage.KeyAutoRun)
	return v == "1"
}

// SetAutoRun enables or disables the OS-level startup hook AND
// persists the preference. Errors propagate to the frontend so the
// Settings toggle can surface a user-friendly failure ('couldn't
// write LaunchAgent — your home directory is read-only?').
func (a *App) SetAutoRun(enabled bool) error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if enabled {
		if err := installAutoRun(); err != nil {
			return err
		}
	} else {
		if err := removeAutoRun(); err != nil {
			return err
		}
	}
	if store != nil {
		flag := "0"
		if enabled {
			flag = "1"
		}
		_ = store.PutSetting(storage.KeyAutoRun, flag)
	}
	return nil
}

func installAutoRun() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("auto-run: locate self: %w", err)
	}
	switch runtime.GOOS {
	case "darwin":
		return installAutoRunMac(exe)
	case "windows":
		return installAutoRunWindows(exe)
	case "linux":
		return installAutoRunLinux(exe)
	default:
		return fmt.Errorf("auto-run not supported on %s", runtime.GOOS)
	}
}

func removeAutoRun() error {
	switch runtime.GOOS {
	case "darwin":
		return removeAutoRunMac()
	case "windows":
		return removeAutoRunWindows()
	case "linux":
		return removeAutoRunLinux()
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// macOS — LaunchAgent
// ---------------------------------------------------------------------------

const macLaunchAgentLabel = "uz.1pro.portal"

func macLaunchAgentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", macLaunchAgentLabel+".plist"), nil
}

// macAppPathFromExecutable walks up from the inner Mach-O binary to
// the .app bundle. Wails builds put Portal at:
//
//   /Applications/Portal.app/Contents/MacOS/Portal
//
// LaunchAgents launch a .app via `open`, not the Mach-O directly,
// so we strip back to the Portal.app path. Falls back to the
// executable path if the .app structure isn't recognised (someone
// running a bare binary).
func macAppPathFromExecutable(exe string) string {
	dir := filepath.Dir(exe)            // .../Contents/MacOS
	contents := filepath.Dir(dir)        // .../Contents
	app := filepath.Dir(contents)        // .../Portal.app
	if filepath.Ext(app) == ".app" {
		return app
	}
	return exe
}

func installAutoRunMac(exe string) error {
	plistPath, err := macLaunchAgentPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("auto-run: mkdir LaunchAgents: %w", err)
	}
	app := macAppPathFromExecutable(exe)
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key><string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/bin/open</string>
        <string>-W</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key><true/>
</dict>
</plist>
`, macLaunchAgentLabel, app)
	return os.WriteFile(plistPath, []byte(plist), 0o644)
}

func removeAutoRunMac() error {
	plistPath, err := macLaunchAgentPath()
	if err != nil {
		return err
	}
	if err := os.Remove(plistPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Linux — XDG autostart .desktop
// ---------------------------------------------------------------------------

func linuxAutostartPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "autostart", "portal.desktop"), nil
}

func installAutoRunLinux(exe string) error {
	p, err := linuxAutostartPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("auto-run: mkdir autostart: %w", err)
	}
	desktop := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Portal
Comment=Direct connections. Zero servers between you.
Exec=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`, exe)
	return os.WriteFile(p, []byte(desktop), 0o644)
}

func removeAutoRunLinux() error {
	p, err := linuxAutostartPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
