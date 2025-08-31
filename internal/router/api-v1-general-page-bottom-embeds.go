package router

import (
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"

	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	tm.Add("general-page-bottom-embeds", "views/partials/general-page-bottom-embeds.html")
}

func Api_V1_GeneralPage_BottomEmbeds(l map[string]*locale.LocaleConfig, langs []string, b2Client *b2.B2Client) func(c *fiber.Ctx) error {
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
		pathParts := strings.Split(strings.Trim(path, "/"), "/")
		if len(pathParts) < 2 {
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
		}

		content, err := tm.Render("general-page-bottom-embeds", values, additionalTemplates...)
		if err != nil {
			slog.Warn("failed to generate div", slog.String("path", path), slog.String("error", err.Error()))
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
		}

		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return c.Status(fiber.StatusOK).Type("html").Send(content)
	}
}
