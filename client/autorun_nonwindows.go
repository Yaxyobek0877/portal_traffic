// Non-Windows stub for the auto-run Windows-specific helpers.
// Returning an error here would surprise the user — the
// SetAutoRun dispatch in autorun.go already routes to the
// per-OS implementation, so these stubs are only reached if
// someone changes the dispatch table without flipping this file
// too. Returning nil keeps a misconfiguration from being a hard
// failure.

//go:build !windows

package main

func installAutoRunWindows(string) error { return nil }
func removeAutoRunWindows() error        { return nil }
