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
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	app := NewApp(logger)

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
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
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
