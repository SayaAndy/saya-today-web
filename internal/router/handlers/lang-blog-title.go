package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/internal/frontmatter"
	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
	"github.com/yuin/goldmark"
)

func init() {
	router.Routes = append(router.Routes, &BlogPageHandler{})
}

type BlogPageHandler struct {
	router.BasicHandler
}

func (r *BlogPageHandler) Filter() (method string, path string) {
	return "GET", "/:lang/blog/:title"
}

func (r *BlogPageHandler) IsTemplated() bool {
	return true
}

func (r *BlogPageHandler) TemplatesToInject() []string {
	return []string{"views/pages/blog-page.html"}
}

func (r *BlogPageHandler) ToCache() router.CacheSetting {
	return router.ByUrlOnly
}

func (r *BlogPageHandler) CacheDuration() time.Duration {
	return 15 * time.Minute
}

func (r *BlogPageHandler) ToValidateLang() router.LangSetting {
	return router.InPath
}

func (r *BlogPageHandler) RenderBody(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	_, pathParts, _, err := router.GetPathFromReferer(c)
	if err != nil {
		return fiber.StatusBadRequest, fmt.Errorf("failed to get path from referer: %w", err)
	}

	metadata, parsedMarkdown, err := readBlogPost(supplements.MarkdownRenderer, supplements.B2Client, lang+"/"+pathParts[2])
	if err != nil {
		return fiber.StatusNotFound, fmt.Errorf("failed to find '%s' post: %w", pathParts[2], err)
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

	templateMap["MapLocationX"] = x
	templateMap["MapLocationY"] = y
	templateMap["MapLocationAreaMeters"] = areaError
	templateMap["Title"] = metadata.Title
	templateMap["ParsedMarkdown"] = template.HTML(parsedMarkdown)
	templateMap["PublishedDate"] = metadata.PublishedTime.Format("2006-01-02 15:04:05 -07:00")
	templateMap["ActionDate"] = metadata.ActionDate
	templateMap["ShortDescription"] = metadata.ShortDescription

	go supplements.ClientCache.View(c.IP(), pathParts[2])

	return fiber.StatusOK, nil
}

func (r *BlogPageHandler) RenderHeader(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	path, pathParts, _, err := router.GetPathFromReferer(c)
	if err != nil {
		return fiber.StatusBadRequest, fmt.Errorf("failed to get path from referer: %w", err)
	}

	metadata, _, err := supplements.B2Client.ReadFrontmatter(lang + "/" + pathParts[2] + ".md")
	if err != nil {
		return fiber.StatusNotFound, fmt.Errorf("could not read '%s' for metadata: %w", path, err)
	}

	templateMap["Title"] = metadata.Title

	return fiber.StatusOK, nil
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
