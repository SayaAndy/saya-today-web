package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/internal/frontmatter"
	"github.com/SayaAndy/saya-today-web/internal/lightgallery"
	"github.com/SayaAndy/saya-today-web/internal/tailwind"
	"github.com/SayaAndy/saya-today-web/internal/templatemanager"
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

	tm, err := templatemanager.NewTemplateManager([]templatemanager.TemplateManagerTemplates{
		{Name: "blog-page", Files: []string{"views/layouts/general-page.html", "views/pages/blog-page.html"}},
		{Name: "blog-catalogue", Files: []string{"views/layouts/general-page.html", "views/pages/blog-catalogue.html"}},
		{Name: "index", Files: []string{"views/index.html"}},
		{Name: "catalogue-blog-cards", Files: []string{"views/partials/catalogue-blog-cards.html"}},
	})
	if err != nil {
		slog.Error("fail to initialize template manager", slog.String("error", err.Error()))
		os.Exit(1)
	}

	app := fiber.New(fiber.Config{})

	app.Use(redirect.New(redirect.Config{
		Rules: map[string]string{
			"/blog": "/",
		},
		StatusCode: 301,
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		content, err := tm.Render("index", fiber.Map{
			"AvailableLanguages": cfg.AvailableLanguages,
		})
		if err != nil {
			slog.Warn("failed to generate page", slog.String("page", "/"), slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate page")
		}

		return c.Type("html").Send(content)
	})

	app.Get("/blog/:lang", func(c *fiber.Ctx) error {
		lang := c.Params("lang")
		if !slices.Contains(availableLanguages, lang) {
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("server does not support '%s' language... yet??", lang))
		}

		querySort := c.Query("sort")
		if querySort == "" {
			querySort = "titleAsc"
		}

		encodedQuery := c.Request().URI().QueryString()
		re, err := regexp.Compile(`tags\[\]=([\w]+)`)
		if err != nil {
			slog.Warn("failed to generate regex for tags gathering", slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate regex for tags gathering")
		}
		decodedQuery, err := url.QueryUnescape(string(encodedQuery))
		matches := re.FindAllStringSubmatch(decodedQuery, -1)

		queryTags := make([]string, 0, len(matches))
		for _, match := range matches {
			queryTags = append(queryTags, string(match[1]))
		}

		pages, err := b2Client.Scan(lang + "/")
		status := fiber.StatusOK
		if err != nil {
			status = fiber.StatusPartialContent
			pages = []*b2.BlogPage{}
		}

		tagsSet := make(map[string]struct{})
		for _, page := range pages {
			slog.Debug("enlist page for catalogue", slog.Any("page", page), slog.String("endpoint", "/"))
			for _, tag := range page.Metadata.Tags {
				tagsSet[tag] = struct{}{}
			}
		}

		tagsArray := make([]string, 0, len(tagsSet))
		for tag := range tagsSet {
			tagsArray = append(tagsArray, tag)
		}
		slices.Sort(tagsArray)

		content, err := tm.Render("blog-catalogue", fiber.Map{
			"QuerySort":     querySort,
			"QueryTags":     strings.Join(queryTags, ","),
			"Tags":          tagsArray,
			"Lang":          lang,
			"L":             localization[lang],
			"PublishedYear": "2025",
			"Title":         localization[lang].BlogSearch.Header,
		})
		if err != nil {
			slog.Warn("failed to generate page", slog.String("page", "/blog/"+lang), slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate page")
		}

		return c.Type("html").Status(status).Send(content)
	})

	app.Get("/blog/:lang/:title", func(c *fiber.Ctx) error {
		lang := c.Params("lang")
		if !slices.Contains(availableLanguages, lang) {
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("server does not support '%s' language... yet??", lang))
		}

		metadata, parsedMarkdown, err := readBlogPost(lang + "/" + c.Params("title"))
		if err != nil {
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("failed to find '%s' post", c.Params("title")))
		}

		geolocationParts := strings.Split(metadata.Geolocation, " ")
		var x, y, areaError string
		if len(geolocationParts) >= 2 {
			x = geolocationParts[0]
			y = geolocationParts[1]
		}
		if len(geolocationParts) >= 3 {
			areaError = geolocationParts[2]
		}

		content, err := tm.Render("blog-page", fiber.Map{
			"Title":                 metadata.Title,
			"PublishedDate":         metadata.PublishedTime.Format("2006-01-02 15:04:05 -07:00"),
			"PublishedYear":         strconv.Itoa(metadata.PublishedTime.Year()),
			"ParsedMarkdown":        template.HTML(parsedMarkdown),
			"MapLocationX":          x,
			"MapLocationY":          y,
			"MapLocationAreaMeters": areaError,
			"Lang":                  lang,
		})
		if err != nil {
			slog.Warn("failed to generate page", slog.String("page", "/blog/"+lang+c.Params("title")), slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate page")
		}

		return c.Type("html").Send(content)
	})

	app.Get("/api/v1/blog-search", func(c *fiber.Ctx) error {
		sort := c.Query("sort")
		lang := c.Query("lang")

		if !slices.Contains(availableLanguages, lang) {
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("server does not support '%s' language", lang))
		}

		pages, err := b2Client.Scan(lang + "/")
		if err != nil {
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString(fmt.Sprintf("failed to scan pages for '%s' lang: %v", lang, err))
		}

		encodedQuery := c.Request().URI().QueryString()
		re, err := regexp.Compile(`tags\[\]=([\w]+)`)
		if err != nil {
			slog.Warn("failed to generate regex for tags gathering", slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate regex for tags gathering")
		}
		decodedQuery, err := url.QueryUnescape(string(encodedQuery))
		matches := re.FindAllStringSubmatch(decodedQuery, -1)

		tags := make([]string, 0, len(matches))
		for _, match := range matches {
			tags = append(tags, string(match[1]))
		}

		pageMeta := make([]map[string]string, 0, len(pages))
		for _, page := range pages {
			for _, tag := range page.Metadata.Tags {
				if len(tags) == 0 || slices.Contains(tags, tag) {
					pageMeta = append(pageMeta, map[string]string{
						"Link":             page.Link,
						"Title":            page.Metadata.Title,
						"PublishedTime":    page.Metadata.PublishedTime.Format("2006-01-02 15:04:05 -0700"),
						"ActionDate":       page.Metadata.ActionDate,
						"ShortDescription": page.Metadata.ShortDescription,
						"Thumbnail":        page.Metadata.Thumbnail,
						"Tags":             strings.Join(page.Metadata.Tags, ", "),
					})
					break
				}
			}
		}

		slices.SortFunc(pageMeta, func(a, b map[string]string) int {
			switch sort {
			case "titleAsc":
				return strings.Compare(a["Title"], b["Title"])
			case "titleDesc":
				return strings.Compare(b["Title"], a["Title"])
			case "actionDateAsc":
				return strings.Compare(a["ActionDate"], b["ActionDate"])
			case "actionDateDesc":
				return strings.Compare(b["ActionDate"], a["ActionDate"])
			case "publicationDateAsc":
				publishedTimeA, _ := time.Parse(time.RFC3339, a["PublishedTime"])
				publishedTimeB, _ := time.Parse(time.RFC3339, b["PublishedTime"])
				return publishedTimeA.Compare(publishedTimeB)
			case "publicationDateDesc":
				publishedTimeA, _ := time.Parse(time.RFC3339, a["PublishedTime"])
				publishedTimeB, _ := time.Parse(time.RFC3339, b["PublishedTime"])
				return publishedTimeB.Compare(publishedTimeA)
			}
			return 0
		})

		content, err := tm.Render("catalogue-blog-cards", fiber.Map{
			"BlogPages": pageMeta,
			"L":         localization[lang],
		})
		if err != nil {
			slog.Warn("failed to generate div", slog.String("page", "/blog/"+lang), slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
		}

		return c.Type("html").Send(content)
	})

	app.Static("/", "./static")

	log.Fatal(app.Listen(":3000"))
}

func readBlogPost(sourceName string) (metadata *frontmatter.Metadata, html string, err error) {
	metadata, markdown, err := b2Client.ReadFrontmatter(sourceName + ".md")
	if err != nil {
		return nil, "", fmt.Errorf("failed to read a frontmatter file: %w", err)
	}

	var buf bytes.Buffer
	if err := md.Convert(markdown, &buf); err != nil {
		return nil, "", fmt.Errorf("convert source context from md to html: %w", err)
	}

	return metadata, buf.String(), nil
}
