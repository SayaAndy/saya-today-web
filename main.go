package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/router"

	_ "github.com/SayaAndy/saya-today-web/internal/router/handlers"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

var (
	configPath = flag.String("c", "config.yaml", "Path to the configuration file (in YAML format)")
)

func main() {
	var err error

	flag.Parse()

	cfg := &config.Config{}
	if err := config.LoadConfig(*configPath, cfg); err != nil {
		slog.Error("fail to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.SetLogLoggerLevel(cfg.LogLevel)
	slog.Info("starting sayana-web server...")

	app, err := router.NewRouter(cfg)
	if err != nil {
		slog.Error("fail to initialize router app", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err = app.InitRoutes(); err != nil {
		slog.Error("fail to initialize routes inside router app", slog.String("error", err.Error()))
		os.Exit(1)
	}

	routerShutdown := make(chan struct{}, 1)

	go func() {
		if err := app.Listen(":3000"); err != nil {
			slog.Error("error while running fiber server", slog.String("error", err.Error()))
			if nestedErr := app.Close(); nestedErr != nil {
				slog.Error("error while shutting down fiber server", slog.String("error", err.Error()))
			}
			<-routerShutdown
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		slog.Info("gracefully shutting down...")
		if err = app.Close(); err != nil {
			slog.Error("error while shutting down fiber server", slog.String("error", err.Error()))
		}
	case <-routerShutdown:
		slog.Info("gracefully shutting down...")
	}
}
