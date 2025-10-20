package router

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	tm.Add("general-page-footer", "views/partials/general-page-footer.html")
}

func Api_V1_GeneralPage_Footer(l map[string]*locale.LocaleConfig, langs []config.AvailableLanguageConfig) func(c *fiber.Ctx) error {
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

		lang := ""
		pathParts := strings.Split(strings.Trim(path, "/"), "/")
		if len(pathParts) == 1 && pathParts[0] == "" {
			pathParts = []string{}
		}
		if len(pathParts) > 0 {
			lang = pathParts[0]
			for _, availableLang := range langs {
				if availableLang.Name == lang {
					goto langIsAvailable
				}
			}
			return c.Status(fiber.ErrBadRequest.Code).SendString("'Referer' header is invalid: expect format '/{lang}/...'")
		}

	langIsAvailable:
		values := fiber.Map{
			"L":           l[lang],
			"Lang":        lang,
			"Path":        strings.Trim(path, "/"),
			"QueryString": string(c.Request().URI().QueryString()),
		}
		var additionalTemplates []string

		if len(pathParts) == 2 && pathParts[1] == "blog" {
			additionalTemplates = append(additionalTemplates, "views/pages/blog-catalogue.html")
		} else if len(pathParts) == 3 && pathParts[1] == "blog" {
			additionalTemplates = append(additionalTemplates, "views/pages/blog-page.html")
		} else if len(pathParts) == 1 {
			additionalTemplates = append(additionalTemplates, "views/pages/home-page.html")
		} else if len(pathParts) == 0 {
			additionalTemplates = append(additionalTemplates, "views/pages/language-pick.html")
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
