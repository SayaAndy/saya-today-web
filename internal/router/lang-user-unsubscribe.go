package router

import (
	"fmt"
	"html/template"
	"log/slog"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	assert(0, tm.Add("unsubscribe-page", "views/pages/unsubscribe-page.html"))
}

func Lang_User_Unsubscribe(l map[string]*locale.LocaleConfig, langs []config.AvailableLanguageConfig) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		lang := c.Params("lang")
		for _, availableLang := range langs {
			if availableLang.Name == lang {
				goto langIsAvailable
			}
		}
		return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("server does not support '%s' language", lang))

	langIsAvailable:
		var statusEmoji, statusText, statusColor string
		var status int

		unsubscribeCode := c.FormValue("code")
		if unsubscribeCode == "" {
			statusColor = "0, 0, 255"
			statusEmoji = "(╭ರ_•́)"
			statusText = l[lang].UnsubscribePage.UnsetCode
			status = fiber.ErrBadRequest.Code
		} else if clientError, serverError := Mailer.Unsubscribe(unsubscribeCode); clientError != nil {
			statusColor = "255, 0, 0"
			statusEmoji = "(͠≖~≖  ͡ )"
			statusText = l[lang].UnsubscribePage.InvalidCode
			status = fiber.ErrBadRequest.Code
		} else if serverError != nil {
			statusColor = "255, 128, 0"
			statusEmoji = "( ˶°ㅁ°) !!"
			statusText = l[lang].UnsubscribePage.OnServerError
			status = fiber.ErrInternalServerError.Code
		} else {
			statusColor = "0, 255, 0"
			statusEmoji = "♡⸜(˶˃ ᵕ ˂˶)⸝♡"
			statusText = l[lang].UnsubscribePage.Success
			status = fiber.StatusOK
		}

		content, err := tm.Render("unsubscribe-page", fiber.Map{
			"Lang":        lang,
			"L":           l[lang],
			"StatusEmoji": statusEmoji,
			"StatusText":  statusText,
			"StatusColor": template.HTML(statusColor),
		})
		if err != nil {
			slog.Warn("failed to generate page", slog.String("page", "/"+lang+"/user/unsubscribe"), slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate page")
		}

		return c.Type("html").Status(status).Send(content)
	}
}
