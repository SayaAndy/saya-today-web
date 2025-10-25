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
	"github.com/SayaAndy/saya-today-web/internal/blogtrigger"
	"github.com/SayaAndy/saya-today-web/internal/factgiver"
	"github.com/SayaAndy/saya-today-web/internal/glightbox"
	"github.com/SayaAndy/saya-today-web/internal/mailer"
	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/SayaAndy/saya-today-web/internal/tailwind"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

var (
	md = goldmark.New(
		goldmark.WithExtensions(
			glightbox.NewGLightboxExtension(),
			tailwind.NewTailwindExtension(),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
		goldmark.WithRenderer(
			renderer.NewRenderer(
				renderer.WithNodeRenderers(
					util.Prioritized(tailwind.NewCustomLinkRenderer(html.WithUnsafe(), html.WithXHTML()), 50),
					util.Prioritized(html.NewRenderer(html.WithXHTML()), 100),
				),
			),
		),
	)
	b2Client           *b2.B2Client
	configPath         = flag.String("c", "config.yaml", "Path to the configuration file (in YAML format)")
	availableLanguages = make([]config.AvailableLanguageConfig, 0)
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
		availableLanguages = append(availableLanguages, lang)
		localization[lang.Name] = localeCfg
	}

	app := fiber.New(fiber.Config{
		ProxyHeader: "X-Forwarded-For",
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://f003.backblazeb2.com",
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept",
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

	router.FactGiver, err = factgiver.NewFactGiver(&cfg.FactGiver, availableLanguages)
	if err != nil {
		slog.Error("fail to initialize fact giver", slog.String("error", err.Error()))
		os.Exit(1)
	}

	router.Mailer, err = mailer.NewMailer(db, cfg.Mail.ClientHost, cfg.Mail.MailHost,
		cfg.Mail.PublicName, cfg.Mail.MailAddress, cfg.Mail.Username, cfg.Mail.Password, []byte(cfg.Mail.Salt), localization)
	if err != nil {
		slog.Error("fail to initialize mailer", slog.String("error", err.Error()))
		os.Exit(1)
	}

	router.BlogTrigger, err = blogtrigger.NewBlogTriggerScheduler(b2Client, cfg.AvailableLanguages, cfg.Mail.Trigger.OnNewPost, func(bp []*b2.BlogPage) error {
		for _, post := range bp {
			if err := router.Mailer.NewPost(post); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		slog.Error("fail to initialize blog trigger", slog.String("error", err.Error()))
		os.Exit(1)
	}

	app.Get("/", router.Api_V1_GeneralPage(localization, availableLanguages))
	app.Get("/:lang<len(2)>", router.Api_V1_GeneralPage(localization, availableLanguages))
	app.Get("/:lang/map", router.Lang_Map(localization, availableLanguages, b2Client))
	app.Get("/:lang/user", router.Api_V1_GeneralPage(localization, availableLanguages))
	app.Get("/:lang/user/unsubscribe", router.Lang_User_Unsubscribe(localization, availableLanguages))
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
	app.Post("/api/v1/email/send-verification-code", router.Api_V1_Email_SendVerificationCode(localization))
	app.Post("/api/v1/email/verify", router.Api_V1_Email_Verify(localization))
	app.Get("/api/v1/email/is-in-verification", router.Api_V1_Email_IsInVerification(localization))
	app.Put("/api/v1/subs", router.Api_V1_Subs_Put(localization))

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
	if err = router.BlogTrigger.Close(); err != nil {
		slog.Error("fail to shutdown blog trigger scheduler", slog.String("error", err.Error()))
	}
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
