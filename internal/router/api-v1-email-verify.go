package router

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	assert(0, tm.Add("personal-page-status", "views/partials/personal-page-status.html"))
}

func Api_V1_Email_Verify(l map[string]*locale.LocaleConfig) func(c *fiber.Ctx) error {
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
		if len(pathParts) != 2 {
			return c.Status(fiber.ErrBadRequest.Code).SendString("'Referer' header is invalid: expect format '/{lang}/user'")
		}

		lang := pathParts[0]

		verificationCode := c.FormValue("email_code")
		if verificationCode == "" {
			return api_v1_email_sendStatusHtml(c, "verification-message", l, "Failed", fiber.ErrUnprocessableEntity.Code, lang, l[lang].UserProfile.VerificationEmpty, map[string]string{})
		}

		if err = Mailer.Verify(verificationCode, lang); err != nil {
			slog.Error("verification code is invalid", slog.String("verification_code", verificationCode), slog.String("error", err.Error()))
			return api_v1_email_sendStatusHtml(c, "verification-message", l, "Failed", fiber.ErrUnprocessableEntity.Code, lang, l[lang].UserProfile.VerificationFailed, map[string]string{})
		}
		return api_v1_email_sendStatusHtml(c, "verification-message", l, "OK", fiber.StatusOK, lang, l[lang].UserProfile.VerificationSuccess+"\n\n"+l[lang].UserProfile.RefreshPage, map[string]string{
			"hide-verification-panel": "true",
		})
	}
}
