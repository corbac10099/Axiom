//go:build wails

package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	axiomApp := NewApp()

	err := wails.Run(&options.App{
		Title:  "Axiom IDE",
		Width:  1400,
		Height: 900,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 30, G: 30, B: 30, A: 255},
		OnStartup:        axiomApp.OnStartup,
		OnShutdown:       axiomApp.OnShutdown,
		Bind: []interface{}{
			axiomApp,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisablePinchZoom:                  true,
			IsZoomControlEnabled:              false,
			EnableSwipeGestures:               false,
		},
	})
	if err != nil {
		log.Fatal("axiom: wails run failed:", err)
	}
}