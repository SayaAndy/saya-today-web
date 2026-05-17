package handlers

import (
	"log/slog"

	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/SayaAndy/saya-today-web/l10n"
	"github.com/gofiber/fiber/v2"
)

type UnsubscribeHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &UnsubscribeHandler{})
}

func (r *UnsubscribeHandler) Filter() (method string, path string) {
	return "GET", "/:lang/user/unsubscribe"
}

func (r *UnsubscribeHandler) IsTemplated() bool {
	return false
}

func (r *UnsubscribeHandler) TemplatesToInject() []string {
	return []string{"views/pages/unsubscribe-page.html"}
}

func (r *UnsubscribeHandler) ToValidateLang() router.LangSetting {
	return router.InPath
}

func (r *UnsubscribeHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	var statusEmoji, statusText, statusColor string
	var status int

	unsubscribeCode := c.FormValue("code")
	if unsubscribeCode == "" {
		statusColor = "0, 0, 255"
		statusEmoji = "(╭ರ_•́)"
		statusText = l10n.T.GetPath(lang, "UnsubscribePage", "UnsetCode").(string)
		status = fiber.ErrBadRequest.Code
	} else if clientError, serverError := supplements.Mailer.Unsubscribe(unsubscribeCode); clientError != nil {
		slog.Info("got a client error when unsubscribing", slog.String("error", clientError.Error()))
		statusColor = "255, 0, 0"
		statusEmoji = "(͠≖~≖  ͡ )"
		statusText = l10n.T.GetPath(lang, "UnsubscribePage", "InvalidCode").(string)
		status = fiber.ErrBadRequest.Code
	} else if serverError != nil {
		slog.Error("got a server error when unsubscribing", slog.String("error", serverError.Error()))
		statusColor = "255, 128, 0"
		statusEmoji = "( ˶°ㅁ°) !!"
		statusText = l10n.T.GetPath(lang, "UnsubscribePage", "OnServerError").(string)
		status = fiber.ErrInternalServerError.Code
	} else {
		statusColor = "0, 255, 0"
		statusEmoji = "♡⸜(˶˃ ᵕ ˂˶)⸝♡"
		statusText = l10n.T.GetPath(lang, "UnsubscribePage", "Success").(string)
		status = fiber.StatusOK
	}

	templateMap["StatusColor"] = statusColor
	templateMap["StatusEmoji"] = statusEmoji
	templateMap["StatusText"] = statusText

	return status, nil
}
