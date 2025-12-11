package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/opik/miau/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
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

	// Create the desktop app service
	var desktopApp = desktop.NewApp()

	// Create application with options
	app := application.New(application.Options{
		Name:        "miau",
		Description: "Mail Intelligence Assistant Utility",
		Services: []application.Service{
			application.NewService(desktopApp),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Logger: application.DefaultLogger(slog.LevelInfo),
		OnShutdown: func() {
			desktopApp.Shutdown()
		},
	})

	// Store app reference in desktop for events/dialogs
	desktopApp.SetApplication(app)

	// Create main window
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "miau",
		Width:     1200,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		URL:       "/",
		BackgroundColour: application.NewRGBA(27, 38, 54, 255),
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		Windows: application.WindowsWindow{
			BackdropType: application.Mica,
		},
		Linux: application.LinuxWindow{
			Icon:                icon,
			WindowIsTranslucent: false,
			WebviewGpuPolicy:    application.WebviewGpuPolicyAlways,
		},
		DevToolsEnabled:        true,
		OpenInspectorOnStartup: openDevtools,
	})

	// Run the application
	if err := app.Run(); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}
