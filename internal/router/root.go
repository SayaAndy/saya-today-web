package router

import (
	"log/slog"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/gofiber/fiber/v2"
)

func init() {
	tm.Add("index", "views/index.html")
}

func Root(localeCfg []config.AvailableLanguageConfig) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		content, err := tm.Render("index", fiber.Map{
			"AvailableLanguages": localeCfg,
		})
		if err != nil {
			slog.Warn("failed to generate page", slog.String("page", "/"), slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate page")
		}

		return c.Type("html").Send(content)
	}
}
