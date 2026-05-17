package handlers

import (
	"html/template"
	"log/slog"

	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/SayaAndy/saya-today-web/l10n"
	"github.com/gofiber/fiber/v2"
)

type VerifyCodeHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &VerifyCodeHandler{})
}

func (r *VerifyCodeHandler) Filter() (method string, path string) {
	return "POST", "/api/v1/email/verify"
}

func (r *VerifyCodeHandler) IsTemplated() bool {
	return false
}

func (r *VerifyCodeHandler) TemplatesToInject() []string {
	return []string{"views/partials/personal-page-status.html"}
}

func (r *VerifyCodeHandler) ToCache() router.CacheSetting {
	return router.Disabled
}

func (r *VerifyCodeHandler) ToValidateLang() router.LangSetting {
	return router.InReferer
}

func (r *VerifyCodeHandler) RateLimiter() *fiber.Handler {
	return &router.RateLimiterStrict
}

func (r *VerifyCodeHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	verificationCode := c.FormValue("email_code")
	templateMap["StatusId"] = "verification-message"

	if verificationCode == "" {
		templateMap["Status"] = "Failed"
		templateMap["Message"] = l10n.T.GetPath(lang, "UserProfile", "VerificationEmpty").(string)
		return fiber.StatusUnprocessableEntity, nil
	}

	if err = supplements.Mailer.Verify(verificationCode, lang); err != nil {
		slog.Warn("verification code is invalid", slog.String("verification_code", verificationCode), slog.String("error", err.Error()))
		templateMap["Status"] = "Failed"
		templateMap["Message"] = l10n.T.GetPath(lang, "UserProfile", "VerificationFailed").(string)
		return fiber.StatusUnprocessableEntity, nil
	}

	templateMap["Status"] = "OK"
	templateMap["Message"] = l10n.T.GetPath(lang, "UserProfile", "VerificationSuccess").(string)
	templateMap["DataAttributes"] = map[string]any{
		"hide-verification-panel": template.HTMLAttr("data-code-expiry-time=\"true\""),
	}
	return fiber.StatusOK, nil
}
