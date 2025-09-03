package main

import (
	"database/sql"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/internal/lightgallery"
	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/SayaAndy/saya-today-web/internal/tailwind"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/redirect"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

var (
	md = goldmark.New(
		goldmark.WithExtensions(
			lightgallery.NewLightGalleryExtension(),
			tailwind.NewTailwindExtension(),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			gmhtml.WithXHTML(),
		),
	)
	b2Client           *b2.B2Client
	configPath         = flag.String("c", "config.yaml", "Path to the configuration file (in YAML format)")
	availableLanguages = make([]string, 0)
	localization       = make(map[string]*locale.LocaleConfig)
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

	db, err := sql.Open(cfg.Auth.Db.Type, cfg.Auth.Db.Cfg.DSN)
	if err != nil {
		slog.Error("fail to initialize db", slog.String("error", err.Error()))
		os.Exit(1)
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		slog.Error("fail to initialize driver for migrating db", slog.String("error", err.Error()))
		os.Exit(1)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		cfg.Auth.Db.Type, driver)
	if err != nil {
		slog.Error("fail to initialize migration client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err = m.Up(); err != nil && err == errors.New("no change") {
		slog.Error("fail to apply migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info("successfully applied migrations")

	b2Client, err = b2.NewB2Client(&cfg.BlogPages.Storage.Config)
	if err != nil {
		slog.Error("fail to initialize b2 client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	for _, lang := range cfg.AvailableLanguages {
		localeCfg, err := locale.InitConfig(cfg.LocalePath + lang.LocFile)
		if err != nil {
			slog.Warn("fail to initialize a locale", slog.String("locale", lang.Name), slog.String("error", err.Error()))
			continue
		}
		availableLanguages = append(availableLanguages, lang.Name)
		localization[lang.Name] = localeCfg
	}

	app := fiber.New(fiber.Config{
		ProxyHeader: "X-Forwarded-For",
	})

	app.Use(redirect.New(redirect.Config{
		Rules: map[string]string{
			"/ru": "/",
			"/en": "/",
		},
		StatusCode: 301,
	}))

	router.CCache, err = router.NewClientCache(db, []byte(cfg.Auth.Salt))
	if err != nil {
		slog.Error("fail to initialize client cache", slog.String("error", err.Error()))
		os.Exit(1)
	}

	router.PCache, err = ristretto.NewCache(&ristretto.Config[string, []byte]{
		NumCounters: 1e6,     // 1,000,000
		MaxCost:     1 << 29, // 512 MB
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		slog.Error("fail to initialize page cache", slog.String("error", err.Error()))
		os.Exit(1)
	}

	app.Get("/", router.Root(cfg.AvailableLanguages))
	app.Get("/:lang/map", router.Lang_Map(localization, availableLanguages, b2Client))
	app.Get("/:lang/blog", router.Api_V1_GeneralPage(localization, availableLanguages))
	app.Get("/:lang/blog/:title", router.Api_V1_GeneralPage(localization, availableLanguages))

	app.Get("/api/v1/tz", router.Api_V1_TZ())
	app.Get("/api/v1/blog-search", router.Api_V1_BlogSearch(localization, availableLanguages, b2Client))
	app.Get("/api/v1/like", router.Api_V1_Like_Get(localization, b2Client))
	app.Put("/api/v1/like", router.Api_V1_Like_Put(localization, b2Client))
	app.Get("/api/v1/general-page/header", router.Api_V1_GeneralPage_Header(localization, availableLanguages, b2Client))
	app.Get("/api/v1/general-page/body", router.Api_V1_GeneralPage_Body(localization, availableLanguages, b2Client, md))
	app.Get("/api/v1/general-page/footer", router.Api_V1_GeneralPage_Footer(localization, availableLanguages))
	app.Get("/api/v1/general-page/top-embeds", router.Api_V1_GeneralPage_TopEmbeds(localization, availableLanguages))
	app.Get("/api/v1/general-page/bottom-embeds", router.Api_V1_GeneralPage_BottomEmbeds(localization, availableLanguages, b2Client))

	app.Static("/", "./static")

	go func() {
		if err := app.Listen(":3000"); err != nil {
			slog.Error("error while running fiber server", slog.String("error", err.Error()))
			panic(err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	slog.Info("gracefully shutting down...")
	if err = app.Shutdown(); err != nil {
		slog.Error("fail to shutdown fiber server", slog.String("error", err.Error()))
	}
	if err = router.CCache.Close(); err != nil {
		slog.Error("fail to dump cache", slog.String("error", err.Error()))
	}
	if err = db.Close(); err != nil {
		slog.Error("fail to close db connection", slog.String("error", err.Error()))
	}
	router.PCache.Close()
	db.Close()
}
