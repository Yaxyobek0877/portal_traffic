package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		local  string
		remote string
		want   bool
	}{
		{"0.3.0", "0.4.0", true},
		{"0.4.0", "0.4.0", false},
		{"0.4.0", "0.3.9", false},
		{"0.4.0", "1.0.0", true},
		{"1.0.0", "0.9.9", false},
		{"v0.3.0", "0.4.0", true}, // tolerates "v" prefix
		{"0.4.0", "v0.4.1", true},
		// Pre-release: rc1 < final
		{"0.4.0-rc1", "0.4.0", true},
		{"0.4.0", "0.4.0-rc1", false},
		// Pre-release ordering
		{"0.4.0-rc1", "0.4.0-rc2", true},
		{"0.4.0-rc2", "0.4.0-rc1", false},
		// Empty handling
		{"", "0.4.0", true},
		{"0.4.0", "", false},
		// Malformed local tolerated
		{"garbage", "0.4.0", true},
	}
	for _, c := range cases {
		got := isNewer(c.local, c.remote)
		if got != c.want {
			t.Errorf("isNewer(local=%q, remote=%q) = %v, want %v", c.local, c.remote, got, c.want)
		}
	}
}

func TestPickAssetForOS(t *testing.T) {
	target := runtime.GOOS + "-" + runtime.GOARCH
	assets := []githubAsset{
		{Name: "Portal-v0.4.0-darwin-arm64.zip", BrowserDownloadURL: "https://example/darwin-arm64.zip"},
		{Name: "Portal-v0.4.0-darwin-amd64.zip", BrowserDownloadURL: "https://example/darwin-amd64.zip"},
		{Name: "Portal-v0.4.0-windows-amd64.zip", BrowserDownloadURL: "https://example/windows-amd64.zip"},
		{Name: "Portal-v0.4.0-linux-amd64.tar.gz", BrowserDownloadURL: "https://example/linux-amd64.tar.gz"},
	}
	got := pickAssetForOS(assets)
	if got == "" {
		t.Skipf("no asset for runtime target %q (test running on uncommon GOOS/GOARCH)", target)
	}
	if !strings.Contains(got, target) {
		t.Errorf("pickAssetForOS = %q, expected substring %q", got, target)
	}
}

func TestPickAssetForOS_NoMatch(t *testing.T) {
	got := pickAssetForOS([]githubAsset{
		{Name: "Portal-v0.4.0-something-else.zip", BrowserDownloadURL: "https://example/x.zip"},
	})
	if got != "" {
		t.Errorf("pickAssetForOS with no match = %q, want empty", got)
	}
}

func TestCheckForUpdate_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(githubRelease{
			TagName:     "v0.4.0",
			HTMLURL:     "https://example/releases/v0.4.0",
			Body:        "## What's new\n- PCP-1\n- i18n",
			PublishedAt: time.Now().Add(-2 * time.Hour),
			Assets: []githubAsset{
				{Name: "Portal-v0.4.0-darwin-arm64.zip", BrowserDownloadURL: "https://example/darwin-arm64.zip"},
				{Name: "Portal-v0.4.0-windows-amd64.zip", BrowserDownloadURL: "https://example/windows-amd64.zip"},
				{Name: "Portal-v0.4.0-linux-amd64.tar.gz", BrowserDownloadURL: "https://example/linux-amd64.tar.gz"},
				{Name: "Portal-v0.4.0-darwin-amd64.zip", BrowserDownloadURL: "https://example/darwin-amd64.zip"},
			},
		})
	}))
	defer srv.Close()

	res := checkForUpdateAt(context.Background(), srv.URL, "0.3.0")
	if res.Error != "" {
		t.Fatalf("unexpected error: %s", res.Error)
	}
	if !res.Available {
		t.Errorf("expected Available=true (0.3.0 < 0.4.0), got false")
	}
	if res.LatestVersion != "0.4.0" {
		t.Errorf("LatestVersion = %q, want 0.4.0", res.LatestVersion)
	}
	if res.ReleaseURL != "https://example/releases/v0.4.0" {
		t.Errorf("ReleaseURL = %q, unexpected", res.ReleaseURL)
	}
	if res.ReleaseNotes == "" {
		t.Errorf("ReleaseNotes empty")
	}
}

func TestCheckForUpdate_NoNewer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(githubRelease{TagName: "v0.4.0"})
	}))
	defer srv.Close()

	res := checkForUpdateAt(context.Background(), srv.URL, "0.4.0")
	if res.Error != "" {
		t.Fatalf("unexpected error: %s", res.Error)
	}
	if res.Available {
		t.Errorf("expected Available=false when current == latest")
	}
}

func TestCheckForUpdate_404IsNotError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	res := checkForUpdateAt(context.Background(), srv.URL, "0.4.0")
	if res.Error != "" {
		t.Errorf("404 should not produce an Error, got %q", res.Error)
	}
	if res.Available {
		t.Errorf("404 should not flag an update")
	}
}

func TestCheckForUpdate_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	res := checkForUpdateAt(context.Background(), srv.URL, "0.4.0")
	if !strings.Contains(res.Error, "rate-limit") {
		t.Errorf("expected rate-limit error, got %q", res.Error)
	}
}

func TestTrimReleaseNotes(t *testing.T) {
	short := "small body"
	if got := trimReleaseNotes(short, 800); got != short {
		t.Errorf("trim of short body changed it: %q", got)
	}
	long := strings.Repeat("a", 2000)
	got := trimReleaseNotes(long, 800)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected ellipsis on truncated body, got %q", got[len(got)-10:])
	}
	// 800 cap + "\n" + "…" (3 bytes in UTF-8) = 804 bytes max.
	if len(got) > 808 {
		t.Errorf("trimmed length = %d, want ≤ 808", len(got))
	}
}

// checkForUpdateAt is a test-only seam that calls CheckForUpdate with
// a redirected base URL. Hits the same code paths as the public
// function but lets us point at httptest servers.
func checkForUpdateAt(ctx context.Context, base string, current string) Result {
	// Quick & dirty: rewrite the URL by injecting a custom RoundTripper
	// that maps api.github.com → base.
	transport := http.RoundTripper(&urlRewriter{base: base})
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	return CheckForUpdate(ctx, Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: current,
		HTTPClient:     client,
	})
}

type urlRewriter struct{ base string }

func (u *urlRewriter) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), apiBaseURL) {
		// Replace https://api.github.com with the test base URL.
		newURL, err := req.URL.Parse(u.base + strings.TrimPrefix(req.URL.String(), apiBaseURL))
		if err != nil {
			return nil, err
		}
		req2 := req.Clone(req.Context())
		req2.URL = newURL
		req2.Host = newURL.Host
		req = req2
	}
	return http.DefaultTransport.RoundTrip(req)
}
