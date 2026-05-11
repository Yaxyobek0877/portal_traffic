// Windows-specific auto-run hook — writes a value under
// HKCU\Software\Microsoft\Windows\CurrentVersion\Run with the
// path to Portal.exe. The Run key is read by Explorer at logon
// and each value listed there is launched once. Lives in a
// separate file behind a //go:build windows tag because the
// registry import only resolves on Windows.

//go:build windows

package main

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

const winRunKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const winRunValue = "Portal"

func installAutoRunWindows(exe string) error {
	k, _, err := registry.CreateKey(
		registry.CURRENT_USER,
		winRunKey,
		registry.SET_VALUE,
	)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(winRunValue, `"`+exe+`"`)
}

func removeAutoRunWindows() error {
	k, err := registry.OpenKey(
		registry.CURRENT_USER,
		winRunKey,
		registry.SET_VALUE,
	)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	defer k.Close()
	if err := k.DeleteValue(winRunValue); err != nil {
		// "value does not exist" is the no-op success case.
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}
