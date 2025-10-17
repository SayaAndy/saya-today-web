package router

import (
	"fmt"
	"html/template"
	"log/slog"
	"math/rand"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/internal/factgiver"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
	"github.com/yuin/goldmark"
)

var FactGiver *factgiver.FactGiver

func init() {
	tm.Add("general-page-body", "views/partials/general-page-body.html")
}

func Api_V1_GeneralPage_Body(l map[string]*locale.LocaleConfig, langs []string, b2Client *b2.B2Client, md goldmark.Markdown) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)

		referer := c.Get("Referer", "")
		if referer == "" {
			return c.Status(fiber.ErrBadRequest.Code).SendString("'Referer' header is empty")
		}
		urlStruct, err := url.ParseRequestURI(referer)
		if err != nil {
			return c.Status(fiber.ErrBadRequest.Code).SendString(fmt.Sprintf("'Referer' header is invalid: %s", err.Error()))
		}

		path := urlStruct.EscapedPath()

		cacheKey := fmt.Sprintf("body.%s", path)
		if val, ok := PCache.Get(cacheKey); val != nil && ok {
			c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
			return c.Status(fiber.StatusOK).Type("html").Send(val)
		}

		pathParts := strings.Split(strings.Trim(path, "/"), "/")
		if len(pathParts) == 0 {
			return c.Status(fiber.ErrBadRequest.Code).SendString("'Referer' header is invalid: expect format '/{lang}/...'")
		}

		lang := pathParts[0]
		if !slices.Contains(langs, lang) {
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("server does not support '%s' language... yet??", lang))
		}

		values := fiber.Map{
			"L":           l[lang],
			"Lang":        lang,
			"QueryString": string(c.Request().URI().QueryString()),
		}
		var additionalTemplates []string

		if len(pathParts) == 2 && pathParts[1] == "blog" {
			querySort := c.Query("sort")
			if querySort == "" {
				querySort = "publicationDateDesc"
			}

			encodedQuery := c.Request().URI().QueryString()
			re, err := regexp.Compile(`tags\[\]=([\w]+)`)
			if err != nil {
				slog.Warn("failed to generate regex for tags gathering", slog.String("error", err.Error()))
				return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate regex for tags gathering")
			}
			decodedQuery, _ := url.QueryUnescape(string(encodedQuery))
			matches := re.FindAllStringSubmatch(decodedQuery, -1)

			queryTags := make([]string, 0, len(matches))
			for _, match := range matches {
				queryTags = append(queryTags, string(match[1]))
			}

			pages, err := b2Client.Scan(lang + "/")
			if err != nil {
				slog.Warn("failed to scan pages via b2", slog.String("error", err.Error()))
				return c.Status(fiber.ErrInternalServerError.Code).SendString(fmt.Sprintf("failed to scan pages via b2: %s", slog.String("error", err.Error())))
			}

			tagsMap := make(map[string]int)
			for _, page := range pages {
				for _, tag := range page.Metadata.Tags {
					tagsMap[tag]++
				}
			}
			slog.Debug("enlist pages for catalogue", slog.Int("tag_count", len(tagsMap)), slog.Int("page_count", len(pages)), slog.String("path", c.Path()))

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

			values["Tags"] = tagsArray
			values["QuerySort"] = querySort
			values["QueryTags"] = strings.Join(queryTags, ",")
			values["Title"] = l[lang].BlogSearch.Header

			additionalTemplates = append(additionalTemplates, "views/pages/blog-catalogue.html")
		} else if len(pathParts) == 3 && pathParts[1] == "blog" {
			metadata, parsedMarkdown, err := readBlogPost(md, b2Client, lang+"/"+pathParts[2])
			if err != nil {
				return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("failed to find '%s' post", pathParts[2]))
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

			values["MapLocationX"] = x
			values["MapLocationY"] = y
			values["MapLocationAreaMeters"] = areaError
			values["Title"] = metadata.Title
			values["ParsedMarkdown"] = template.HTML(parsedMarkdown)
			values["PublishedDate"] = metadata.PublishedTime.Format("2006-01-02 15:04:05 -07:00")
			values["ActionDate"] = metadata.ActionDate

			additionalTemplates = append(additionalTemplates, "views/pages/blog-page.html")

			go CCache.View(c.IP(), pathParts[2])
		} else if len(pathParts) == 1 {
			values["Title"] = l[lang].HomePage.Header
			values["FilledHeartCount"] = uint(40)
			values["OutlineHeartCount"] = uint(40)
			values["GifName"] = fmt.Sprintf("otter-%d.gif", rand.Int()%3+1)
			values["FunFacts"] = FactGiver.Give(lang)
			additionalTemplates = append(additionalTemplates, "views/pages/home-page.html")
		}

		content, err := tm.Render("general-page-body", values, additionalTemplates...)
		if err != nil {
			slog.Warn("failed to generate div", slog.String("path", path), slog.String("error", err.Error()))
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
		}

		go PCache.SetWithTTL(cacheKey, content, int64(len(content)), 5*time.Minute)
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return c.Status(fiber.StatusOK).Type("html").Send(content)
	}
}
