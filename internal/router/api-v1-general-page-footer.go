package router

import (
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	tm.Add("general-page-footer", "views/partials/general-page-footer.html")
}

func Api_V1_GeneralPage_Footer(l map[string]*locale.LocaleConfig, langs []string) func(c *fiber.Ctx) error {
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

		cacheKey := fmt.Sprintf("footer.%s", path)
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
			additionalTemplates = append(additionalTemplates, "views/pages/blog-catalogue.html")
		} else if len(pathParts) == 3 && pathParts[1] == "blog" {
			additionalTemplates = append(additionalTemplates, "views/pages/blog-page.html")
		} else if len(pathParts) == 1 {
			additionalTemplates = append(additionalTemplates, "views/pages/home-page.html")
		}

		content, err := tm.Render("general-page-footer", values, additionalTemplates...)
		if err != nil {
			slog.Warn("failed to generate div", slog.String("path", path), slog.String("error", err.Error()))
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
		}

		go PCache.SetWithTTL(cacheKey, content, int64(len(content)), 5*time.Minute)
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return c.Status(fiber.StatusOK).Type("html").Send(content)
	}
}
