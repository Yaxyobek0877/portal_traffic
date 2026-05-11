// Self-replacing auto-update.
//
// CheckForUpdate already finds the right asset URL for the running
// OS/arch (Result.AssetForOS). InstallUpdate downloads it, swaps the
// running binary / .app for the new one, and relaunches.
//
// The swap is the tricky bit: the OS won't let us replace a binary
// that's currently executing (Linux's open-but-deleted semantics is
// the exception). The pattern Sparkle / WinSparkle / Electron-builder
// all use is:
//
//   1. Download the new asset.
//   2. Extract to a staging directory.
//   3. Write a tiny shell / batch script that sleeps a beat (so the
//      current process can exit), then `rm -rf old; mv new old; open`.
//   4. Spawn the script DETACHED so it survives our exit.
//   5. Quit the running app (caller's job — we hand back the script
//      handle so the Wails layer can call runtime.Quit).
//
// This file implements (1) → (4) per OS. The handoff signal is the
// returned PendingInstall whose Apply() launches the script and
// returns. Callers should immediately quit the app after Apply() so
// the script's swap step doesn't race a still-alive process.

package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// PendingInstall is the result of staging an update download. The
// caller calls Apply() to kick off the swap script and then quits
// the app; the script then replaces the binary and relaunches.
type PendingInstall struct {
	// Path to the staged update artefact (the new .app or binary).
	StagedPath string
	// Path to the swap script the OS-specific helper wrote out.
	ScriptPath string
	// Where the running binary lives — the swap target.
	CurrentPath string
	// Optional download progress reports go here; closed when the
	// download finishes.
	progress chan int
}

// Apply runs the swap script detached and returns. The caller must
// then quit the current process so the script's `rm -rf old` step
// can proceed cleanly. On Windows the script is a .bat invoked via
// cmd /c start /min; on macOS / Linux it's a /bin/bash spawn with
// stdin/out/err detached.
func (p *PendingInstall) Apply() error {
	if p == nil {
		return errors.New("update: nothing staged")
	}
	switch runtime.GOOS {
	case "windows":
		// Use cmd /c start so the .bat survives our exit. /b would
		// keep it attached to our console; we want full detachment.
		cmd := exec.Command("cmd.exe", "/c", "start", "/min", "", p.ScriptPath)
		return cmd.Start()
	default:
		cmd := exec.Command("/bin/bash", p.ScriptPath)
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		// Detach from our process group so killing us doesn't kill
		// the script. setSid is set in install_unix.go via a build
		// constraint to keep this file portable.
		setDetached(cmd)
		return cmd.Start()
	}
}

// PrepareInstall downloads the update asset and stages the swap.
// Returns a PendingInstall whose Apply() the caller should call
// followed by an immediate runtime.Quit.
//
// progress is optional — when non-nil, percentages 0..100 are sent
// during the download so the UI can render a bar.
func PrepareInstall(ctx context.Context, assetURL string, currentPath string, progress chan<- int) (*PendingInstall, error) {
	if assetURL == "" {
		return nil, errors.New("update: empty asset URL")
	}
	if currentPath == "" {
		// Default to the running executable. Callers usually pass
		// a more useful path (e.g. the .app bundle on macOS).
		exe, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("update: locate self: %w", err)
		}
		currentPath = exe
	}

	// 1) Pick a staging dir that survives the swap. We use a
	//    timestamped dir under the OS temp so successive install
	//    attempts don't collide.
	tmpRoot := filepath.Join(os.TempDir(), fmt.Sprintf("portal-update-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tmpRoot, 0o755); err != nil {
		return nil, fmt.Errorf("update: mkdir: %w", err)
	}

	// 2) Download.
	archivePath := filepath.Join(tmpRoot, baseURL(assetURL))
	if err := downloadFile(ctx, assetURL, archivePath, progress); err != nil {
		return nil, fmt.Errorf("update: download: %w", err)
	}

	// 3) Extract.
	extractDir := filepath.Join(tmpRoot, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return nil, fmt.Errorf("update: mkdir extract: %w", err)
	}
	switch ext := strings.ToLower(filepath.Ext(archivePath)); ext {
	case ".zip":
		if err := unzip(archivePath, extractDir); err != nil {
			return nil, fmt.Errorf("update: unzip: %w", err)
		}
	case ".gz", ".tgz":
		if err := untargz(archivePath, extractDir); err != nil {
			return nil, fmt.Errorf("update: untar: %w", err)
		}
	default:
		return nil, fmt.Errorf("update: unsupported archive %q", archivePath)
	}

	// 4) Locate the staged artefact inside the extracted tree.
	staged, err := stagedArtefact(extractDir)
	if err != nil {
		return nil, fmt.Errorf("update: locate artefact: %w", err)
	}

	// 5) Write the OS-specific swap script.
	scriptPath, err := writeSwapScript(tmpRoot, staged, currentPath)
	if err != nil {
		return nil, fmt.Errorf("update: write swap script: %w", err)
	}

	return &PendingInstall{
		StagedPath:  staged,
		ScriptPath:  scriptPath,
		CurrentPath: currentPath,
	}, nil
}

// baseURL returns the file name component of a URL. Falls back to
// 'portal-update' when the URL doesn't end in something useful.
func baseURL(u string) string {
	idx := strings.LastIndex(u, "/")
	if idx < 0 || idx == len(u)-1 {
		return "portal-update"
	}
	name := u[idx+1:]
	if q := strings.Index(name, "?"); q >= 0 {
		name = name[:q]
	}
	if name == "" {
		return "portal-update"
	}
	return name
}

// downloadFile fetches the URL into dst with a 5-minute deadline.
// Reports progress as integer 0..100 percentages on the optional
// channel. Closes the channel on completion or error so the UI
// knows the stream is done.
func downloadFile(ctx context.Context, url, dst string, progress chan<- int) error {
	if progress != nil {
		defer close(progress)
	}
	dctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(dctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "portal-updater/1.0")
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	written := int64(0)
	buf := make([]byte, 64*1024)
	lastPct := -1
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if progress != nil && total > 0 {
				pct := int(float64(written) / float64(total) * 100)
				if pct > 100 {
					pct = 100
				}
				if pct != lastPct {
					select {
					case progress <- pct:
					default:
					}
					lastPct = pct
				}
			}
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			return rerr
		}
	}
}

func unzip(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		// Sanitise path to stop zip-slip — never extract above dst.
		target := filepath.Join(dst, f.Name)
		if !strings.HasPrefix(target, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("zip slip: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()|0o111); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return err
		}
		rc.Close()
		out.Close()
	}
	return nil
}

func untargz(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dst, hdr.Name)
		if !strings.HasPrefix(target, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("tar slip: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)|0o111); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
}

// stagedArtefact picks the right swap target inside the extracted
// archive. We look for: a Portal.app dir (macOS), a Portal.exe file
// (Windows), or a Portal binary (Linux). Top-level only — the CI
// archives flatten to those names.
func stagedArtefact(dir string) (string, error) {
	candidates := []string{"Portal.app", "Portal.exe", "Portal"}
	for _, name := range candidates {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	// Some Linux .tar.gz layouts wrap into a single subdir; recurse one level.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		for _, name := range candidates {
			p := filepath.Join(dir, e.Name(), name)
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}
	return "", errors.New("no Portal.app / Portal.exe / Portal found in archive")
}

// writeSwapScript emits the OS-specific shell / batch that performs
// the rm/mv/start dance after the parent process exits. Returns the
// path to the script.
func writeSwapScript(tmpRoot, staged, current string) (string, error) {
	if runtime.GOOS == "windows" {
		path := filepath.Join(tmpRoot, "portal-update.bat")
		// Wait for parent process exit, replace, relaunch. The ping
		// trick is the canonical Windows way to sleep without
		// shell-builtin-sleep being reliably present.
		body := fmt.Sprintf("@echo off\r\n"+
			"ping 127.0.0.1 -n 2 >nul\r\n"+
			"del /F /Q %q\r\n"+
			"move /Y %q %q\r\n"+
			"start \"\" %q\r\n",
			current, staged, current, current,
		)
		return path, os.WriteFile(path, []byte(body), 0o755)
	}
	path := filepath.Join(tmpRoot, "portal-update.sh")
	// macOS / Linux. /bin/sleep is universally present; rm -rf
	// handles both .app bundles (dirs) and bare binaries. open
	// is macOS-specific; on Linux we re-exec the binary directly.
	relaunch := fmt.Sprintf("%q &", current)
	if runtime.GOOS == "darwin" {
		relaunch = fmt.Sprintf("/usr/bin/open %q", current)
	}
	body := fmt.Sprintf("#!/bin/bash\n"+
		"sleep 1\n"+
		"rm -rf %q\n"+
		"mv %q %q\n"+
		"chmod +x %q 2>/dev/null || true\n"+
		"%s\n",
		current, staged, current, current, relaunch,
	)
	return path, os.WriteFile(path, []byte(body), 0o755)
}
