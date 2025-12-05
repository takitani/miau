package main

import (
	"embed"
	"os"

	"github.com/opik/miau/internal/desktop"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	// Check --devtools flag
	var openDevtools = false
	for _, arg := range os.Args[1:] {
		if arg == "--devtools" {
			openDevtools = true
			break
		}
	}

	// Create an instance of the app structure
	var app = desktop.NewApp()

	// Create application with options
	var err = wails.Run(&options.App{
		Title:     "miau",
		Width:     1200,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		OnDomReady:       app.DomReady,
		OnBeforeClose:    app.BeforeClose,
		Bind: []interface{}{
			app,
		},
		// Enable devtools (F12 to open, --devtools flag to auto-open)
		Debug: options.Debug{
			OpenInspectorOnStartup: openDevtools,
		},
		// Linux specific options
		Linux: &linux.Options{
			Icon:                icon,
			WindowIsTranslucent: false,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyAlways,
			ProgramName:         "miau",
		},
		// Windows specific options
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
			WebviewUserDataPath:               "",
			WebviewBrowserPath:                "",
			Theme:                             windows.SystemDefault,
		},
		// Mac specific options
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
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "miau",
				Message: "Mail Intelligence Assistant Utility\n\nA local-first email client with AI integration.",
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
