package router

import (
	"log/slog"
	"strings"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	tm.Add("general-page", "views/layouts/general-page.html")
}

func Api_V1_GeneralPage(l map[string]*locale.LocaleConfig, langs []config.AvailableLanguageConfig) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		path := c.Path()

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
		cacheKey := "general-page." + lang
		if val, ok := PCache.Get(cacheKey); val != nil && ok {
			c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
			return c.Status(fiber.StatusOK).Type("html").Send(val)
		}

		content, err := tm.Render("general-page", fiber.Map{
			"L":           l[lang],
			"Lang":        lang,
			"QueryString": string(c.Request().URI().QueryString()),
		})
		if err != nil {
			slog.Warn("failed to generate div", slog.String("path", path), slog.String("error", err.Error()))
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
		}

		go PCache.Set(cacheKey, content, int64(len(content)))
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return c.Status(fiber.StatusOK).Type("html").Send(content)
	}
}
