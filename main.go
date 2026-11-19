package main

import (
	"embed"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"outlook_descargas_adjuntos/internal/config"
	"outlook_descargas_adjuntos/internal/logging"
)

//go:embed all:frontend/dist
var assets embed.FS

func setupLog() (*slog.Logger, io.Closer) {
	dir, err := logging.LogDir()
	if err == nil {
		if log, closer, err := logging.Setup(dir, time.Now()); err == nil {
			return log, closer
		}
	}
	return slog.New(slog.NewTextHandler(os.Stderr, nil)), io.NopCloser(nil)
}

func main() {
	log, closer := setupLog()
	defer closer.Close()

	cfgDir, err := config.Dir()
	if err != nil {
		log.Error("no se encontró el directorio de configuración", "error", err)
	}
	cfg, err := config.Load(cfgDir)
	if err != nil {
		log.Warn("config inválida, se usan valores por defecto", "error", err)
	}

	src, err := selectSource(os.Args)
	if err != nil {
		log.Error("fuente de correos no disponible", "error", err)
	}
	app := NewApp(src, log, cfgDir, cfg)

	err = wails.Run(&options.App{
		Title:     appTitle,
		Width:     cfg.WindowWidth,
		Height:    cfg.WindowHeight,
		MinWidth:  1100,
		MinHeight: 680,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: backgroundFor(cfg.Theme),
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.beforeClose,
		Bind:             []interface{}{app},
	})
	if err != nil {
		log.Error("error de Wails", "error", err)
	}
}

// backgroundFor es el color de fondo de la ventana mientras carga el WebView, según el tema guardado.
func backgroundFor(theme string) *options.RGBA {
	if theme == config.ThemeDark {
		return &options.RGBA{R: 13, G: 15, B: 34, A: 1}
	}
	return &options.RGBA{R: 248, G: 250, B: 252, A: 1}
}
