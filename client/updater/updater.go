// Package updater polls the GitHub Releases API for newer Portal
// versions. The result is fed to the frontend so the user can decide
// whether to download — we never auto-install.
//
// Auto-update flow lives entirely on the client side: there's no
// Portal-controlled update server. The GitHub Releases API serves as
// our "is there a newer version?" oracle and the release artifacts
// double as the download endpoint.
//
// Privacy: the GitHub API call carries no Portal-specific identifier
// — it's the same request `curl https://api.github.com/...` would
// make. GitHub sees a hit from your IP; that's it.
package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Default release source — change here if the repo or org moves.
const (
	defaultOwner = "Yaxyobek0877"
	defaultRepo  = "portal_traffic"
	apiBaseURL   = "https://api.github.com"
)

// Result describes what CheckForUpdate found. `Available` is true
// only when LatestVersion strictly exceeds the running version, ignoring
// pre-release tags unless the running version is itself a pre-release.
type Result struct {
	Available      bool      `json:"available"`
	CurrentVersion string    `json:"currentVersion"`
	LatestVersion  string    `json:"latestVersion"`
	ReleaseURL     string    `json:"releaseUrl"`
	ReleaseNotes   string    `json:"releaseNotes"`
	PublishedAt    time.Time `json:"publishedAt"`
	// AssetForOS contains the download URL for the running OS/arch
	// when the release exposes one we recognise. Empty otherwise.
	AssetForOS string `json:"assetForOs"`
	// CheckedAt is set so the UI can display "checked X minutes ago".
	CheckedAt time.Time `json:"checkedAt"`
	// Error captures non-fatal issues (no network, rate-limited)
	// without forcing the caller to handle Go errors at the JSON edge.
	Error string `json:"error,omitempty"`
}

// Config tells CheckForUpdate which repo to query and what to compare
// against. Owner and Repo default to the canonical values; CurrentVersion
// must be supplied — typically the wails.json productVersion.
type Config struct {
	Owner          string
	Repo           string
	CurrentVersion string
	HTTPClient     *http.Client // optional; defaults to a 10-second timeout
}

// CheckForUpdate fetches the latest release from GitHub and compares
// versions. Failures (network, 404, rate limit) are returned via
// Result.Error rather than as Go errors, so the UI can show a polite
// message either way.
func CheckForUpdate(ctx context.Context, cfg Config) Result {
	if cfg.Owner == "" {
		cfg.Owner = defaultOwner
	}
	if cfg.Repo == "" {
		cfg.Repo = defaultRepo
	}
	res := Result{
		CurrentVersion: cfg.CurrentVersion,
		CheckedAt:      time.Now(),
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBaseURL, cfg.Owner, cfg.Repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	// GitHub recommends sending a User-Agent. Keep it generic so we
	// don't fingerprint individual installs.
	req.Header.Set("User-Agent", "portal-updater/1.0")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		res.Error = "network: " + err.Error()
		return res
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		// Rate-limited or auth required — not fatal, surface as Error.
		res.Error = "rate-limited (try again later)"
		return res
	}
	if resp.StatusCode == http.StatusNotFound {
		// No releases yet for this repo. Treat as "no update", not an error.
		return res
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		res.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		return res
	}

	var ghr githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&ghr); err != nil {
		res.Error = "decode: " + err.Error()
		return res
	}

	res.LatestVersion = strings.TrimPrefix(ghr.TagName, "v")
	res.ReleaseURL = ghr.HTMLURL
	res.ReleaseNotes = trimReleaseNotes(ghr.Body, 800)
	res.PublishedAt = ghr.PublishedAt
	res.AssetForOS = pickAssetForOS(ghr.Assets)
	res.Available = isNewer(res.CurrentVersion, res.LatestVersion)
	return res
}

// githubRelease is the subset of fields we read from /releases/latest.
type githubRelease struct {
	TagName     string         `json:"tag_name"`
	HTMLURL     string         `json:"html_url"`
	Body        string         `json:"body"`
	PublishedAt time.Time      `json:"published_at"`
	Prerelease  bool           `json:"prerelease"`
	Assets      []githubAsset  `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// pickAssetForOS picks the asset whose name matches the OS/arch the
// updater is running on. Falls back to an empty string when no asset
// fits. Names follow our release pipeline:
//   Portal-vX.Y.Z-darwin-arm64.zip
//   Portal-vX.Y.Z-darwin-amd64.zip
//   Portal-vX.Y.Z-windows-amd64.zip
//   Portal-vX.Y.Z-linux-amd64.tar.gz
func pickAssetForOS(assets []githubAsset) string {
	target := runtime.GOOS + "-" + runtime.GOARCH
	for _, a := range assets {
		if strings.Contains(a.Name, target) {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

// trimReleaseNotes truncates GitHub-flavoured-Markdown body to a
// reasonable length so the in-app preview stays compact. Cut on a
// newline boundary when possible.
func trimReleaseNotes(body string, max int) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	if len(body) <= max {
		return body
	}
	cutoff := strings.LastIndex(body[:max], "\n")
	if cutoff < max/2 {
		cutoff = max
	}
	return body[:cutoff] + "\n…"
}

// isNewer compares semver-ish strings. Returns true when remote is
// strictly greater than local. Pre-release identifiers (-rc1, -beta)
// are treated as older than the same numeric version without one,
// matching SemVer 2.0.0 §11.
//
// We deliberately keep this small and dependency-free; if we ever
// need more sophistication, switch to golang.org/x/mod/semver.
func isNewer(local, remote string) bool {
	if remote == "" {
		return false
	}
	if local == "" {
		return true
	}
	la, lpre := splitVersion(local)
	ra, rpre := splitVersion(remote)
	for i := 0; i < 3; i++ {
		if ra[i] != la[i] {
			return ra[i] > la[i]
		}
	}
	// Numeric parts equal: a release > a pre-release of the same number.
	switch {
	case lpre == "" && rpre == "":
		return false
	case lpre == "" && rpre != "":
		// remote is a pre-release of the same version -> remote is older
		return false
	case lpre != "" && rpre == "":
		// local is a pre-release; remote is the release -> remote is newer
		return true
	default:
		// Both pre-release: lexicographic gives a useful ordering for
		// the common "rc1 < rc2 < beta < final" pattern even if it isn't
		// strict SemVer. Good enough for our update-prompt use case.
		return rpre > lpre
	}
}

// splitVersion turns "1.2.3-rc1" into ([1 2 3], "rc1"). Missing parts
// default to 0; non-numeric parts are clamped to 0 so a malformed tag
// can't confuse the comparison.
func splitVersion(v string) ([3]int, string) {
	v = strings.TrimPrefix(v, "v")
	pre := ""
	if i := strings.Index(v, "-"); i >= 0 {
		pre = v[i+1:]
		v = v[:i]
	}
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		n, err := atoi(p)
		if err != nil {
			out[i] = 0
		} else {
			out[i] = n
		}
	}
	return out, pre
}

// atoi is a minimal positive-integer parser. We avoid strconv.Atoi
// only to dodge importing strconv solely for this — ymmv.
func atoi(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty")
	}
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, errors.New("non-digit")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
