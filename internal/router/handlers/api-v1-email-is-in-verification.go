package handlers

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

type OngoingVerificationHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &OngoingVerificationHandler{})
}

func (r *OngoingVerificationHandler) Filter() (method string, path string) {
	return "GET", "/api/v1/email/is-in-verification"
}

func (r *OngoingVerificationHandler) IsTemplated() bool {
	return false
}

func (r *OngoingVerificationHandler) TemplatesToInject() []string {
	return []string{"views/partials/personal-page-status.html"}
}

func (r *OngoingVerificationHandler) ToCache() router.CacheSetting {
	return router.Disabled
}

func (r *OngoingVerificationHandler) ToValidateLang() router.LangSetting {
	return router.InReferer
}

func (r *OngoingVerificationHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	isAllowed, whenAllowed, codeExpiry := supplements.Mailer.IsAllowedToRetryVerification(c.IP())

	sterileDataset := make(map[string]any)
	sterileDataset["code-expiry-time"] = template.HTMLAttr(fmt.Sprintf("data-code-expiry-time=\"%d\"", codeExpiry.UnixMilli()))

	templateMap["StatusId"] = "email-message"
	templateMap["DataAttributes"] = sterileDataset

	if !isAllowed {
		sterileDataset["striked-end-time"] = template.HTMLAttr(fmt.Sprintf("data-striked-end-time=\"%d\"", whenAllowed.UnixMilli()))
		templateMap["Status"] = "Neutral"
		templateMap["Message"] = strings.ReplaceAll(supplements.Localization[lang].UserProfile.DelayTilVerification, "{}", whenAllowed.Format("2006-01-02 15:04:05 MST"))
	} else {
		templateMap["Status"] = "OK"
		templateMap["Message"] = ""
	}

	return fiber.StatusOK, nil
}
