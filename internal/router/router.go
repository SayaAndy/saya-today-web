package router

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/internal/blogtrigger"
	"github.com/SayaAndy/saya-today-web/internal/factgiver"
	"github.com/SayaAndy/saya-today-web/internal/glightbox"
	"github.com/SayaAndy/saya-today-web/internal/mailer"
	"github.com/SayaAndy/saya-today-web/internal/tailwind"
	"github.com/SayaAndy/saya-today-web/internal/templatemanager"
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
)

type CacheSetting int

const (
	Disabled CacheSetting = iota
	ByUrlOnly
	ByUrlAndQuery
)

type LangSetting int

const (
	NotRequired LangSetting = iota
	InPath
	InForm
	InReferer
)

var (
	Routes = make([]Route, 0)
)

type Route interface {
	Filter() (method string, path string)
	IsTemplated() bool
	ToCache() CacheSetting
	CacheDuration() time.Duration
	ToValidateLang() LangSetting
	TemplatesToInject() []string
	Render(c *fiber.Ctx, supplements *Supplements, lang string, templateMap fiber.Map) (statusCode int, err error)
	RenderHeader(c *fiber.Ctx, supplements *Supplements, lang string, templateMap fiber.Map) (statusCode int, err error)
	RenderBody(c *fiber.Ctx, supplements *Supplements, lang string, templateMap fiber.Map) (statusCode int, err error)
	RenderFooter(c *fiber.Ctx, supplements *Supplements, lang string, templateMap fiber.Map) (statusCode int, err error)
	RenderTopEmbeds(c *fiber.Ctx, supplements *Supplements, lang string, templateMap fiber.Map) (statusCode int, err error)
	RenderBottomEmbeds(c *fiber.Ctx, supplements *Supplements, lang string, templateMap fiber.Map) (statusCode int, err error)
}

type Supplements struct {
	DB                 *sql.DB
	B2Client           *b2.B2Client
	Localization       map[string]*locale.LocaleConfig
	AvailableLanguages []config.AvailableLanguageConfig
	ClientCache        *ClientCache
	PageCache          *ristretto.Cache[string, []byte]
	FactGiver          *factgiver.FactGiver
	Mailer             *mailer.Mailer
	BlogTrigger        *blogtrigger.BlogTriggerScheduler
	TemplateManager    *templatemanager.TemplateManager
	MarkdownRenderer   goldmark.Markdown
}

type Router struct {
	supplements          *Supplements
	app                  *fiber.App
	templatedRoutes      map[string]map[string]Route
	templatedPathMatcher *PathMatcher
}

func NewRouter(cfg *config.Config) (*Router, error) {
	supplements := &Supplements{
		AvailableLanguages: cfg.AvailableLanguages,
	}

	var err error

	supplements.DB, err = sql.Open(cfg.Auth.Db.Type, cfg.Auth.Db.Cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("fail to initialize db: %w", err)
	}

	driver, err := sqlite3.WithInstance(supplements.DB, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("fail to initialize driver for migrating db: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		cfg.Auth.Db.Type, driver)
	if err != nil {
		return nil, fmt.Errorf("fail to initialize migration client: %w", err)
	}

	if err = m.Up(); err != nil && err == errors.New("no change") {
		return nil, fmt.Errorf("fail to apply migrations: %w", err)
	}
	slog.Debug("successfully applied migrations")

	supplements.B2Client, err = b2.NewB2Client(&cfg.BlogPages.Storage.Config)
	if err != nil {
		return nil, fmt.Errorf("fail to initialize b2 client: %w", err)
	}

	supplements.Localization = make(map[string]*locale.LocaleConfig, len(cfg.AvailableLanguages))
	for _, lang := range cfg.AvailableLanguages {
		localeCfg, err := locale.InitConfig(cfg.LocalePath + lang.LocFile)
		if err != nil {
			return nil, fmt.Errorf("fail to initialize a locale: %w", err)
		}
		supplements.Localization[lang.Name] = localeCfg
	}

	supplements.MarkdownRenderer = goldmark.New(
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

	supplements.ClientCache, err = NewClientCache(supplements.DB, []byte(cfg.Auth.Salt))
	if err != nil {
		return nil, fmt.Errorf("fail to initialize client cache: %w", err)
	}

	supplements.PageCache, err = ristretto.NewCache(&ristretto.Config[string, []byte]{
		NumCounters: 1e6,     // 1,000,000
		MaxCost:     1 << 29, // 512 MB
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		return nil, fmt.Errorf("fail to initialize page cache: %w", err)
	}

	supplements.FactGiver, err = factgiver.NewFactGiver(&cfg.FactGiver, supplements.AvailableLanguages)
	if err != nil {
		return nil, fmt.Errorf("fail to initialize fact giver: %w", err)
	}

	supplements.Mailer, err = mailer.NewMailer(supplements.DB, cfg.Mail.ClientHost, cfg.Mail.MailHost,
		cfg.Mail.PublicName, cfg.Mail.MailAddress, cfg.Mail.Username, cfg.Mail.Password, []byte(cfg.Mail.Salt), supplements.Localization)
	if err != nil {
		return nil, fmt.Errorf("fail to initialize mailer: %w", err)
	}

	supplements.BlogTrigger, err = blogtrigger.NewBlogTriggerScheduler(supplements.B2Client, cfg.AvailableLanguages, cfg.Mail.Trigger.OnNewPost,
		func(bp []*b2.BlogPage) error {
			for _, post := range bp {
				if err := supplements.Mailer.NewPost(post); err != nil {
					return err
				}
			}
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("fail to initialize blog trigger: %w", err)
	}

	supplements.TemplateManager, err = templatemanager.NewTemplateManager()
	if err != nil {
		return nil, fmt.Errorf("fail to initialize template manager: %w", err)
	}

	enablePrintRoutes := false
	if cfg.LogLevel <= slog.LevelDebug {
		enablePrintRoutes = true
	}

	app := fiber.New(fiber.Config{
		EnablePrintRoutes: enablePrintRoutes,
		ProxyHeader:       "X-Forwarded-For",
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "https://f003.backblazeb2.com",
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	templatedRoutes := make(map[string]map[string]Route)
	templatedPathMatcher := NewPathMatcher()

	return &Router{supplements, app, templatedRoutes, templatedPathMatcher}, nil
}

func (r *Router) InitRoutes() (err error) {
	r.supplements.TemplateManager.Add("general-page", "views/layouts/general-page.html")
	for _, route := range Routes {
		method, match := route.Filter()

		if err = r.supplements.TemplateManager.Add(method+" "+match, route.TemplatesToInject()...); err != nil {
			return fmt.Errorf("failed to add '%s %s' route into template manager: %w", method, match, err)
		}

		if route.IsTemplated() {
			if _, ok := r.templatedRoutes[method]; !ok {
				r.templatedRoutes[method] = make(map[string]Route)
			}
			r.templatedRoutes[method][match] = route
			r.templatedPathMatcher.AddRoute(method, match)

			currentRoute := route
			r.app.Add(method, match, func(c *fiber.Ctx) error {
				lang, err := r.getAndValidateLang(c, currentRoute.ToValidateLang())
				if err != nil {
					return err
				}
				err = r.generalPage(c, currentRoute, lang)
				return err
			})
		} else {
			currentRoute := route
			r.app.Add(method, match, func(c *fiber.Ctx) error {
				lang, err := r.getAndValidateLang(c, currentRoute.ToValidateLang())
				if err != nil {
					return err
				}

				var cacheKey string
				trimmedPath := strings.Trim(c.Path(), "/")
				queryString := c.Request().URI().QueryString()

				switch currentRoute.ToCache() {
				case ByUrlOnly:
					cacheKey = fmt.Sprintf("%s.full-page.%s", method, trimmedPath)
				case ByUrlAndQuery:
					cacheKey = fmt.Sprintf("%s.full-page.%s.%s", method, trimmedPath, queryString)
				}

				defaultMap := fiber.Map{
					"L":           r.supplements.Localization[lang],
					"Lang":        lang,
					"Path":        trimmedPath,
					"QueryString": queryString,
				}

				statusCode, err := currentRoute.Render(c, r.supplements, lang, defaultMap)
				method := c.Method()
				_, match := currentRoute.Filter()
				if err != nil {
					slog.Error("failed to finish rendering a page",
						slog.Int("status_code", statusCode),
						slog.String("method", method),
						slog.String("path", c.Path()),
						slog.String("match", match),
						slog.String("query", string(queryString)),
						slog.String("error", err.Error()),
					)
					c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
					return c.Status(statusCode).SendString(err.Error())
				}

				content, err := r.supplements.TemplateManager.Render(method+" "+match, defaultMap)
				if err != nil {
					slog.Error("failed to generate div",
						slog.String("method", method),
						slog.String("path", c.Path()),
						slog.String("match", match),
						slog.String("query", string(queryString)),
						slog.String("error", err.Error()),
					)
					c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
					return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
				}
				if val, ok := defaultMap["Output"]; ok && len(content) == 0 {
					content = val.([]byte)
				}

				if statusCode >= 200 && statusCode < 300 && route.ToCache() != Disabled {
					go r.supplements.PageCache.SetWithTTL(cacheKey, content, int64(len(content)), route.CacheDuration())
				}

				c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
				return c.Status(statusCode).Type("html").Send(content)
			})
		}
	}

	segments := []string{"header", "body", "footer", "top-embeds", "bottom-embeds"}

	r.app.Get("/api/v1/general-page/:part", func(c *fiber.Ctx) error {
		part := c.Params("part")
		if !slices.Contains(segments, part) {
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrBadRequest.Code).SendString(fmt.Sprintf("unknown segment '%s'", part))
		}
		err := r.generalPageSegment(c, part)
		return err
	})

	for _, segment := range segments {
		r.supplements.TemplateManager.Add("general-page-"+segment, "views/partials/general-page-"+segment+".html")
	}

	r.app.Static("/", "./static")

	Routes = make([]Route, 0)

	return nil
}

func (r *Router) Listen(endpoint string) error {
	if err := r.app.Listen(endpoint); err != nil {
		return fmt.Errorf("error while running fiber server: %w", err)
	}
	return nil
}

func (r *Router) Close() (err error) {
	allErrors := make([]error, 0)
	if err = r.supplements.BlogTrigger.Close(); err != nil {
		allErrors = append(allErrors, fmt.Errorf("fail to shutdown blog trigger scheduler: %w", err))
	}
	if err = r.app.Shutdown(); err != nil {
		allErrors = append(allErrors, fmt.Errorf("fail to shutdown fiber server: %w", err))
	}
	if err = r.supplements.ClientCache.Close(); err != nil {
		allErrors = append(allErrors, fmt.Errorf("fail to dump cache: %w", err))
	}
	if err = r.supplements.DB.Close(); err != nil {
		allErrors = append(allErrors, fmt.Errorf("fail to close db connection: %w", err))
	}
	r.supplements.PageCache.Close()
	return errors.Join(allErrors...)
}

func (r *Router) generalPage(c *fiber.Ctx, route Route, lang string) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)

	path := c.Path()
	method := c.Method()
	trimmedPath := strings.Trim(path, "/")
	cacheKey := ""
	queryString := c.Request().URI().QueryString()

	switch route.ToCache() {
	case ByUrlOnly:
		cacheKey = fmt.Sprintf("%s.general-page.%s", method, trimmedPath)
	case ByUrlAndQuery:
		cacheKey = fmt.Sprintf("%s.general-page.%s.%s", method, trimmedPath, queryString)
	}

	if val, ok := r.supplements.PageCache.Get(cacheKey); val != nil && ok {
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return c.Status(fiber.StatusOK).Type("html").Send(val)
	}

	content, err := r.supplements.TemplateManager.Render("general-page", fiber.Map{
		"L":           r.supplements.Localization[lang],
		"Lang":        lang,
		"QueryString": queryString,
	})
	if err != nil {
		slog.Warn("failed to generate div", slog.String("path", path), slog.String("error", err.Error()))
		return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
	}

	go r.supplements.PageCache.SetWithTTL(cacheKey, content, int64(len(content)), route.CacheDuration())
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.Status(fiber.StatusOK).Type("html").Send(content)
}

func (r *Router) generalPageSegment(c *fiber.Ctx, part string) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)

	path, pathParts, queryString, err := GetPathFromReferer(c)
	if err != nil {
		return err
	}

	method := c.Method()
	pattern, _, matched := r.templatedPathMatcher.MatchPath(method, path)
	if !matched {
		return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("fail to get templated page for '%s %s': not found", method, path))
	}
	route := r.templatedRoutes[method][pattern]

	lang := ""

	switch route.ToValidateLang() {
	case InPath, InReferer:
		if len(pathParts) > 0 {
			lang = pathParts[0]
		}
		if _, err := r.getAndValidateLang(c, NotRequired, lang); err != nil {
			return err
		}
	case InForm:
		if lang, err = r.getAndValidateLang(c, InForm); err != nil {
			return err
		}
	}

	cacheKey := ""
	trimmedPath := strings.Trim(path, "/")
	switch route.ToCache() {
	case ByUrlOnly:
		cacheKey = fmt.Sprintf("%s.%s.%s", method, part, trimmedPath)
	case ByUrlAndQuery:
		cacheKey = fmt.Sprintf("%s.%s.%s.%s", method, part, trimmedPath, queryString)
	}

	if route.ToCache() != Disabled {
		if val, ok := r.supplements.PageCache.Get(cacheKey); val != nil && ok {
			c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
			return c.Status(fiber.StatusOK).Type("html").Send(val)
		}
	}

	var statusCode int
	defaultMap := fiber.Map{
		"L":           r.supplements.Localization[lang],
		"Lang":        lang,
		"Path":        strings.Trim(path, "/"),
		"QueryString": queryString,
	}

	switch part {
	case "body":
		statusCode, err = route.RenderBody(c, r.supplements, lang, defaultMap)
	case "header":
		statusCode, err = route.RenderHeader(c, r.supplements, lang, defaultMap)
	case "footer":
		statusCode, err = route.RenderFooter(c, r.supplements, lang, defaultMap)
	case "top-embeds":
		statusCode, err = route.RenderTopEmbeds(c, r.supplements, lang, defaultMap)
	case "bottom-embeds":
		statusCode, err = route.RenderBottomEmbeds(c, r.supplements, lang, defaultMap)
	}
	if err != nil {
		slog.Error(err.Error(),
			slog.String("method", method),
			slog.String("path", path),
			slog.String("query", string(queryString)),
			slog.String("segment", part),
			slog.String("error", err.Error()),
		)
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		return c.Status(statusCode).SendString(err.Error())
	}

	content, err := r.supplements.TemplateManager.Render("general-page-"+part, defaultMap, route.TemplatesToInject()...)
	if err != nil {
		slog.Error("failed to generate div",
			slog.String("method", method),
			slog.String("path", path),
			slog.String("query", string(queryString)),
			slog.String("segment", part),
			slog.String("error", err.Error()),
		)
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
	}

	if statusCode >= 200 && statusCode < 300 && route.ToCache() != Disabled {
		go r.supplements.PageCache.SetWithTTL(cacheKey, content, int64(len(content)), route.CacheDuration())
	}

	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.Status(statusCode).Type("html").Send(content)
}

func (r *Router) getAndValidateLang(c *fiber.Ctx, langSetting LangSetting, defaultLang ...string) (string, error) {
	var lang string
	if len(defaultLang) > 0 {
		lang = defaultLang[0]
	}

	switch langSetting {
	case NotRequired:
		return lang, nil

	case InPath:
		path := c.Path()

		pathParts := strings.Split(strings.Trim(path, "/"), "/")
		if len(pathParts) == 1 && pathParts[0] == "" {
			pathParts = []string{}
		}
		if len(pathParts) > 0 && len(pathParts[0]) == 2 {
			lang = pathParts[0]
		}

	case InForm:
		lang = c.FormValue("lang")

	case InReferer:
		_, pathParts, _, err := GetPathFromReferer(c)
		if err != nil {
			return "", err
		}

		if len(pathParts) >= 1 {
			lang = pathParts[0]
		}
	}

	for _, availableLang := range r.supplements.AvailableLanguages {
		if availableLang.Name == lang {
			return lang, nil
		}
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	return "", c.Status(fiber.ErrBadRequest.Code).SendString(fmt.Sprintf("lang value is invalid: '%s' is not considered an available language", lang))
}

func GetPathFromReferer(c *fiber.Ctx) (path string, pathParts []string, queryString string, err error) {
	referer := c.Get("Referer")
	if referer == "" {
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		return "", nil, "", c.Status(fiber.ErrBadRequest.Code).SendString("'Referer' header is empty")
	}
	urlStruct, err := url.ParseRequestURI(referer)
	if err != nil {
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		return "", nil, "", c.Status(fiber.ErrBadRequest.Code).SendString(fmt.Sprintf("'Referer' header is invalid: %s", err.Error()))
	}

	path = urlStruct.EscapedPath()
	pathParts = strings.Split(strings.Trim(path, "/"), "/")
	queryString = urlStruct.RawQuery
	return
}
