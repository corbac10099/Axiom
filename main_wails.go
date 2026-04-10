//go:build wails

package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	axiomApp := NewApp()

	app := application.New(application.Options{
		Name:        "Axiom IDE",
		Description: "Modular Software Engineering Platform",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	// Initialiser le moteur Axiom AVANT app.Run()
	axiomApp.Startup(app)
	defer axiomApp.Shutdown()

	// Fenêtre principale
	app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:            "Axiom IDE",
		Width:            1400,
		Height:           900,
		BackgroundColour: application.NewRGBA(30, 30, 30, 255),
		URL:              "/",
	})

	// Exposer les méthodes Go au JS
	app.Bind(axiomApp)

	if err := app.Run(); err != nil {
		log.Fatal("axiom: wails run failed:", err)
	}
}