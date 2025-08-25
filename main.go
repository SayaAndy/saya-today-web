package main

import (
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/internal/lightgallery"
	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/SayaAndy/saya-today-web/internal/tailwind"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/redirect"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
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

	router.CCache = router.NewClientCache([]byte(cfg.Auth.Salt))

	app.Get("/", router.Root(cfg.AvailableLanguages))
	app.Get("/:lang/map", router.Lang_Map(localization, availableLanguages, b2Client))
	app.Get("/:lang/blog", router.Lang_Blog(localization, availableLanguages, b2Client))
	app.Get("/:lang/blog/:title", router.Lang_Blog_Title(localization, availableLanguages, b2Client, md))
	app.Get("/api/v1/tz", router.Api_V1_TZ())
	app.Get("/api/v1/blog-search", router.Api_V1_BlogSearch(localization, availableLanguages, b2Client))

	app.Static("/", "./static")

	log.Fatal(app.Listen(":3000"))
}
