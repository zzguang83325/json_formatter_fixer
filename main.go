package main

import (
	"embed"
	"net"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Check for existing instance
	conn, err := net.Dial("tcp", "127.0.0.1:52109")
	if err == nil {
		// Existing instance found, send the file path if provided
		if len(os.Args) > 1 {
			conn.Write([]byte(os.Args[1]))
		}
		conn.Close()
		os.Exit(0)
	}

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "Json Formatter & Fixer",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
