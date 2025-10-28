package handlers

import (
	"fmt"
	"html/template"
	"log/slog"
	"strings"

	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

type SendVerificationCodeHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &SendVerificationCodeHandler{})
}

func (r *SendVerificationCodeHandler) Filter() (method string, path string) {
	return "POST", "/api/v1/email/send-verification-code"
}

func (r *SendVerificationCodeHandler) IsTemplated() bool {
	return false
}

func (r *SendVerificationCodeHandler) TemplatesToInject() []string {
	return []string{"views/partials/personal-page-status.html"}
}

func (r *SendVerificationCodeHandler) ToCache() router.CacheSetting {
	return router.Disabled
}

func (r *SendVerificationCodeHandler) ToValidateLang() router.LangSetting {
	return router.InReferer
}

func (r *SendVerificationCodeHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	id := c.IP()
	templateMap["StatusId"] = "email-message"

	email := c.FormValue("email")
	if email == "" {
		templateMap["Status"] = "Failed"
		templateMap["Message"] = supplements.Localization[lang].UserProfile.EmailEmpty
		return fiber.StatusUnprocessableEntity, nil
	}

	isTaken, err := supplements.Mailer.MailIsTaken(email)
	if err != nil {
		templateMap["Status"] = "Failed"
		templateMap["Message"] = supplements.Localization[lang].UserProfile.VerificationCodeSendingError
		slog.Error("failed to check if address is already taken", slog.String("error", err.Error()))
		return fiber.StatusUnprocessableEntity, nil
	}

	if isTaken {
		templateMap["Status"] = "Failed"
		templateMap["Message"] = supplements.Localization[lang].UserProfile.EmailTaken
		return fiber.StatusUnprocessableEntity, nil
	}

	if isAllowed, whenAllowed, _ := supplements.Mailer.IsAllowedToRetryVerification(id); !isAllowed {
		templateMap["Status"] = "Failed"
		templateMap["Message"] = strings.ReplaceAll(supplements.Localization[lang].UserProfile.DelayTilVerification, "{}", whenAllowed.Format("2006-01-02 15:04:05 MST"))
		templateMap["DataAttributes"] = map[string]any{
			"striked-end-time": template.HTMLAttr(fmt.Sprintf("data-striked-end-time=\"%d\"", whenAllowed.UnixMilli())),
		}
		return fiber.StatusUnprocessableEntity, nil
	}

	if previousEmail, _, _ := supplements.Mailer.GetInfo(supplements.Mailer.GetHash(id)); previousEmail == email {
		templateMap["Status"] = "Failed"
		templateMap["Message"] = supplements.Localization[lang].UserProfile.EmailAlreadyValidated
		return fiber.StatusUnprocessableEntity, nil
	}

	if err = supplements.Mailer.SendVerificationCode(id, email, lang); err != nil {
		templateMap["Status"] = "Failed"
		templateMap["Message"] = supplements.Localization[lang].UserProfile.VerificationCodeSendingError
		slog.Error("failed to send a verification code", slog.String("error", err.Error()))
		return fiber.StatusUnprocessableEntity, nil
	}

	_, endTime, codeExpiry := supplements.Mailer.IsAllowedToRetryVerification(id)
	templateMap["Status"] = "OK"
	templateMap["Message"] = supplements.Localization[lang].UserProfile.VerificationCodeSent
	templateMap["DataAttributes"] = map[string]any{
		"striked-end-time": template.HTMLAttr(fmt.Sprintf("data-striked-end-time=\"%d\"", endTime.UnixMilli())),
		"code-expiry-time": template.HTMLAttr(fmt.Sprintf("data-code-expiry-time=\"%d\"", codeExpiry.UnixMilli())),
	}

	return fiber.StatusOK, nil
}
