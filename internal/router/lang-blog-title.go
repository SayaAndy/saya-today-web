package router

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/internal/frontmatter"
	"github.com/SayaAndy/saya-today-web/internal/templatemanager"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
	"github.com/yuin/goldmark"
)

func Lang_Blog_Title(tm *templatemanager.TemplateManager, l map[string]*locale.LocaleConfig, langs []string, b2Client *b2.B2Client, md goldmark.Markdown) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		lang := c.Params("lang")
		if !slices.Contains(langs, lang) {
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("server does not support '%s' language... yet??", lang))
		}

		metadata, parsedMarkdown, err := readBlogPost(md, b2Client, lang+"/"+c.Params("title"))
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
			slog.Warn("failed to generate page", slog.String("page", "/"+lang+"/blog/"+c.Params("title")), slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate page")
		}

		return c.Type("html").Send(content)
	}
}

func readBlogPost(md goldmark.Markdown, b2Client *b2.B2Client, sourceName string) (metadata *frontmatter.Metadata, html string, err error) {
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
