package router

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/url"
	"strings"

	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	assert(0, tm.Add("personal-page-status", "views/partials/personal-page-status.html"))
}

func Api_V1_Email_SendVerificationCode(l map[string]*locale.LocaleConfig) func(c *fiber.Ctx) error {
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
		id := c.IP()

		email := c.FormValue("email")
		if email == "" {
			return api_v1_email_sendStatusHtml(c, "email-message", l, "Failed", fiber.ErrUnprocessableEntity.Code, lang, l[lang].UserProfile.EmailEmpty, map[string]string{})
		}

		isTaken, err := Mailer.MailIsTaken(email)
		if err != nil {
			slog.Error("failed to check if address is already taken", slog.String("error", err.Error()))
			return api_v1_email_sendStatusHtml(c, "email-message", l, "Failed", fiber.ErrUnprocessableEntity.Code, lang, l[lang].UserProfile.VerificationCodeSendingError, map[string]string{})
		}
		if isTaken {
			return api_v1_email_sendStatusHtml(c, "email-message", l, "Failed", fiber.ErrUnprocessableEntity.Code, lang, l[lang].UserProfile.EmailTaken, map[string]string{})
		}

		if isAllowed, whenAllowed, _ := Mailer.IsAllowedToRetryVerification(id); !isAllowed {
			return api_v1_email_sendStatusHtml(c, "email-message", l, "Failed", fiber.ErrUnprocessableEntity.Code, lang, strings.ReplaceAll(l[lang].UserProfile.DelayTilVerification, "{}", whenAllowed.Format("2006-01-02 15:04:05 MST")), map[string]string{
				"striked-end-time": fmt.Sprint(whenAllowed.UnixMilli()),
			})
		}

		if previousEmail, _, _ := Mailer.GetInfo(id); previousEmail == email {
			return api_v1_email_sendStatusHtml(c, "email-message", l, "Failed", fiber.ErrUnprocessableEntity.Code, lang, l[lang].UserProfile.EmailAlreadyValidated, map[string]string{})
		}

		if err = Mailer.SendVerificationCode(id, email, lang); err != nil {
			slog.Error("failed to send a verification code", slog.String("error", err.Error()))
			return api_v1_email_sendStatusHtml(c, "email-message", l, "Failed", fiber.ErrUnprocessableEntity.Code, lang, l[lang].UserProfile.VerificationCodeSendingError, map[string]string{})
		}

		_, endTime, codeExpiry := Mailer.IsAllowedToRetryVerification(id)
		return api_v1_email_sendStatusHtml(c, "email-message", l, "OK", fiber.StatusOK, lang, strings.ReplaceAll(l[lang].UserProfile.VerificationCodeSent, "{}", endTime.Format("2006-01-02 15:04:05 MST")), map[string]string{
			"striked-end-time": fmt.Sprint(endTime.UnixMilli()),
			"code-expiry-time": fmt.Sprint(codeExpiry.UnixMilli()),
		})
	}
}

func api_v1_email_sendStatusHtml(c *fiber.Ctx, divId string, l map[string]*locale.LocaleConfig, status string, code int, lang string, message string, dataAttributes map[string]string) error {
	sterileDataset := make(map[string]interface{})
	for k, v := range dataAttributes {
		sterileDataset[k] = template.HTMLAttr(fmt.Sprintf("data-%s=\"%s\"", k, template.HTMLEscapeString(v)))
	}

	content, err := tm.Render("personal-page-status", fiber.Map{
		"L":              l[lang],
		"Status":         status,
		"Message":        message,
		"StatusId":       divId,
		"DataAttributes": sterileDataset,
	})
	if err != nil {
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		slog.Error("failed to render the email status message", slog.String("error", err.Error()), slog.String("div_id", divId), slog.String("message", message))
		return c.Status(fiber.ErrInternalServerError.Code).SendString("Failed to render the email status message, please ask administrator for a more detailed cause")
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.Status(code).Send(content)
}
