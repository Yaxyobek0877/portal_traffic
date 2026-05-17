// Portal desktop app — Wails entry point.
//
// The frontend lives under client/frontend/ as a Vite + React + TS app.
// At build time `wails build` invokes `npm run build` which produces
// frontend/dist; we embed it via go:embed and Wails serves it through
// the platform's native webview.
//
// `wails dev` is the recommended development workflow: it boots the
// Vite dev server, hot-reloads the frontend on changes, and rebuilds
// the Go side as needed.
package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"portal_traffic_client/crashreport"
)

// Version is the canonical product version. Bump for each release;
// embedded into Wails app metadata, the Settings → About tab, and
// the User-Agent string sent to the GitHub Releases API.
//
// IMPORTANT: this constant MUST match `wails.json:productVersion` and
// the git tag (`vX.Y.Z`). v0.5.0 → v0.5.5 shipped with this number
// stuck at "0.5.4", which made the in-app updater see itself as
// permanently outdated against the GitHub `releases/latest` tag and
// re-prompt for install on every relaunch.
const Version = "0.5.7"

// keep slog import alive even if main.go shrinks
var _ = slog.LevelInfo

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := initLogger()
	app := NewApp(logger)
	app.crashCatcher = crashreport.New("", Version)
	// Capture any panic on the main goroutine into a structured local
	// report before re-panicking to let the runtime print + exit. Other
	// goroutines arrange their own recover blocks.
	defer crashreport.InstallGlobal(app.crashCatcher)

	err := wails.Run(&options.App{
		Title:             "Portal",
		Width:             1280,
		Height:            820,
		MinWidth:          900,
		MinHeight:         600,
		Frameless:         false,
		BackgroundColour:  &options.RGBA{R: 10, G: 14, B: 26, A: 255},
		EnableDefaultContextMenu: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  true, // we draw our own header; macOS title text was overlapping the ID/KOD
				HideTitleBar:               false,
				FullSizeContent:            true, // content under the title bar — we add 28px padding via .titlebar-pad
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		OnStartup:  app.Startup,
		OnShutdown: app.Shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		slog.Error("wails run failed", "err", err)
		os.Exit(1)
	}
}
