package router

import (
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	tm.Add("blog-catalogue", "views/layouts/general-page.html", "views/pages/blog-catalogue.html")
}

func Lang_Blog(l map[string]*locale.LocaleConfig, langs []string, b2Client *b2.B2Client) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		lang := c.Params("lang")
		if !slices.Contains(langs, lang) {
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("server does not support '%s' language... yet??", lang))
		}

		querySort := c.Query("sort")
		if querySort == "" {
			querySort = "publicationDateDesc"
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

		tagsMap := make(map[string]int)
		for _, page := range pages {
			slog.Debug("enlist page for catalogue", slog.Any("page", page), slog.String("endpoint", "/"+lang+"/blog"))
			for _, tag := range page.Metadata.Tags {
				tagsMap[tag]++
			}
		}

		type Tag struct {
			Name  string `json:"Name" yaml:"name"`
			Count int    `json:"Count" yaml:"count"`
		}

		tagsArray := make([]Tag, 0, len(tagsMap))
		for tag, count := range tagsMap {
			tagsArray = append(tagsArray, Tag{tag, count})
		}
		slices.SortFunc(tagsArray, func(a Tag, b Tag) int {
			return strings.Compare(a.Name, b.Name)
		})

		content, err := tm.Render("blog-catalogue", fiber.Map{
			"QuerySort":     querySort,
			"QueryTags":     strings.Join(queryTags, ","),
			"Tags":          tagsArray,
			"Lang":          lang,
			"L":             l[lang],
			"PublishedYear": "2025",
			"Title":         l[lang].BlogSearch.Header,
		})
		if err != nil {
			slog.Warn("failed to generate page", slog.String("page", "/"+lang+"/blog"), slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate page")
		}

		return c.Type("html").Status(status).Send(content)
	}
}
