package main

import (
	"embed"
	"flag"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Parse command line flags (passed by bootstrap binary)
	serverURL := flag.String("server", "", "Immich server URL")
	setupToken := flag.String("token", "", "Setup token for authentication")
	flag.Parse()

	// Create app instance
	app := NewApp(*serverURL, *setupToken)

	// Create Wails application
	err := wails.Run(&options.App{
		Title:  "Immich Google Photos Importer",
		Width:  900,
		Height: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatal("Error:", err.Error())
	}
}
