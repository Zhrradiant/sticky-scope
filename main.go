package main

import (
	"embed"
	"os"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// Populated from CLI args before wails.Run, then consumed in startup().
var (
	stickyPath string  // non-empty when this process is a pinned sticky note
	stickyX    int     // window X position offset (0 = default)
	stickyY    int     // window Y position offset (0 = default)
)

func main() {
	// Parse CLI flags before Wails consumes os.Args.
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, "--project-path=") {
			stickyPath = strings.TrimPrefix(a, "--project-path=")
		}
		if strings.HasPrefix(a, "--x=") {
			stickyX, _ = strconv.Atoi(strings.TrimPrefix(a, "--x="))
		}
		if strings.HasPrefix(a, "--y=") {
			stickyY, _ = strconv.Atoi(strings.TrimPrefix(a, "--y="))
		}
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Sticky Scope",
		Width:     330,
		Height:    480,
		MinWidth:  330,
		MinHeight: 480,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// Beige background matching CSS --bg to eliminate load-time flash.
		BackgroundColour: &options.RGBA{R: 253, G: 248, B: 240, A: 1},
		// Frameless window — DWM shadows + Win11 rounded corners are preserved
		// because DisableFramelessWindowDecorations is not set (default false).
		Frameless:    true,
		AlwaysOnTop:  true,
		OnStartup:    app.startup,
		OnDomReady:   app.domReady,
		OnShutdown:   app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			DisableWindowIcon: true,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}