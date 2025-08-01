package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/internal/frontmatter"
	"github.com/SayaAndy/saya-today-web/internal/lightgallery"
	"github.com/SayaAndy/saya-today-web/internal/tailwind"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
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
	b2Client   *b2.B2Client
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

	b2Client, err = b2.NewB2Client(&cfg.BlogPages.Storage.Config)
	if err != nil {
		slog.Error("fail to initialize b2 client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	engine := html.New("./views", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		pages, err := b2Client.Scan()
		status := fiber.StatusOK
		if err != nil {
			status = fiber.StatusPartialContent
			pages = []*b2.BlogPage{}
		}

		pageMeta := make([]map[string]string, 0, len(pages))
		for _, page := range pages {
			slog.Debug("enlist page for catalogue", slog.Any("page", page), slog.String("endpoint", "/"))
			pageMeta = append(pageMeta, map[string]string{
				"Link":             page.Link,
				"Title":            page.Metadata.Title,
				"PublishedTime":    page.Metadata.PublishedTime.Format("2006-01-02 15:04:05 -0700"),
				"ActionDate":       page.Metadata.ActionDate,
				"ShortDescription": page.Metadata.ShortDescription,
				"Thumbnail":        page.Metadata.Thumbnail,
				"Tags":             strings.Join(page.Metadata.Tags, ", "),
			})
		}

		return c.Status(status).Render("index", fiber.Map{
			"PublishedYear": "2025",
			"BlogPages":     pageMeta,
		})
	})

	app.Get("/blog/:lang/:title", func(c *fiber.Ctx) error {
		metadata, parsedMarkdownDesktop, parsedMarkdownMobile, err := readBlogPost(c.Params("lang") + "/" + c.Params("title"))
		if err != nil {
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("failed to find '%s' post", c.Params("title")))
		}

		return c.Render("layouts/blog-page", fiber.Map{
			"Title":                 metadata.Title,
			"PublishedDate":         metadata.PublishedTime.Format("2006-01-02 15:04:05 -07:00"),
			"PublishedYear":         strconv.Itoa(metadata.PublishedTime.Year()),
			"ParsedMarkdownDesktop": template.HTML(parsedMarkdownDesktop),
			"ParsedMarkdownMobile":  template.HTML(parsedMarkdownMobile),
		})
	})

	app.Static("/", "./static")

	log.Fatal(app.Listen(":3000"))
}

func readBlogPost(sourceName string) (metadata *frontmatter.Metadata, desktopBody string, mobileBody string, err error) {
	metadata, markdown, err := b2Client.ReadFrontmatter(sourceName + ".md")
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to read a frontmatter file: %w", err)
	}

	var bufDesktop bytes.Buffer
	if err := md.Convert(markdown, &bufDesktop); err != nil {
		return nil, "", "", fmt.Errorf("convert source context from md to html (desktop version): %w", err)
	}
	var bufMobile bytes.Buffer
	if err := md.Convert(markdown, &bufMobile); err != nil {
		return nil, "", "", fmt.Errorf("convert source context from md to html (mobile version): %w", err)
	}

	return metadata, bufDesktop.String(), bufMobile.String(), nil
}
