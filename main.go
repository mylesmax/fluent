package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Read icon file
	iconData, err := os.ReadFile("frontend/src/assets/images/flntribbon.png")
	if err != nil {
		println("Error reading icon:", err.Error())
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "fluent",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			TitleBar:             mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "Fluent",
				Message: "© 2025 mylesmax",
				Icon:    iconData,
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			DisableWindowIcon:    false,
			Theme:                windows.SystemDefault,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
