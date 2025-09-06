package router

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	tm.Add("general-page", "views/layouts/general-page.html")
}

func Api_V1_GeneralPage(l map[string]*locale.LocaleConfig, langs []string) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		path := c.Path()

		pathParts := strings.Split(strings.Trim(path, "/"), "/")
		if len(pathParts) == 0 {
			return c.Status(fiber.ErrBadRequest.Code).SendString("url path is invalid: expect format '/{lang}/...'")
		}

		lang := pathParts[0]
		if !slices.Contains(langs, lang) {
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("server does not support '%s' language... yet??", lang))
		}

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
