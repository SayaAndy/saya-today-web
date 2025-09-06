package router

import (
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	tm.Add("general-page-header", "views/partials/general-page-header.html")
}

func Api_V1_GeneralPage_Header(l map[string]*locale.LocaleConfig, langs []string, b2Client *b2.B2Client) func(c *fiber.Ctx) error {
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

		cacheKey := fmt.Sprintf("header.%s", path)
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
			values["Title"] = l[lang].BlogSearch.Header
			additionalTemplates = append(additionalTemplates, "views/pages/blog-catalogue.html")
		} else if len(pathParts) == 3 && pathParts[1] == "blog" {
			metadata, _, err := b2Client.ReadFrontmatter(lang + "/" + pathParts[2] + ".md")
			if err != nil {
				return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("could not read '%s' for content", path))
			}
			values["Title"] = metadata.Title
			values["PublishedDate"] = metadata.PublishedTime.Format("2006-01-02 15:04:05 -07:00")
			values["ActionDate"] = metadata.ActionDate
			additionalTemplates = append(additionalTemplates, "views/pages/blog-page.html")
		} else if len(pathParts) == 1 {
			values["Title"] = l[lang].HomePage.Header
			additionalTemplates = append(additionalTemplates, "views/pages/home-page.html")
		}

		content, err := tm.Render("general-page-header", values, additionalTemplates...)
		if err != nil {
			slog.Warn("failed to generate div", slog.String("path", path), slog.String("error", err.Error()))
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
		}

		go PCache.SetWithTTL(cacheKey, content, int64(len(content)), 5*time.Minute)
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return c.Status(fiber.StatusOK).Type("html").Send(content)
	}
}
