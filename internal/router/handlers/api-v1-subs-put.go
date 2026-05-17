package handlers

import (
	"github.com/SayaAndy/saya-today-web/internal/mailer"
	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/SayaAndy/saya-today-web/l10n"
	"github.com/gofiber/fiber/v2"
)

type PutSubsHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &PutSubsHandler{})
}

func (r *PutSubsHandler) Filter() (method string, path string) {
	return "PUT", "/api/v1/subs"
}

func (r *PutSubsHandler) IsTemplated() bool {
	return false
}

func (r *PutSubsHandler) TemplatesToInject() []string {
	return []string{"views/partials/personal-page-status.html"}
}

func (r *PutSubsHandler) ToCache() router.CacheSetting {
	return router.Disabled
}

func (r *PutSubsHandler) ToValidateLang() router.LangSetting {
	return router.InReferer
}

func (r *PutSubsHandler) RateLimiter() *fiber.Handler {
	return &router.RateLimiterMedium
}

func (r *PutSubsHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	templateMap["StatusId"] = "subs-message"

	subscriptionType := c.FormValue("tags")
	var subscriptionTypeEnum mailer.SubscriptionType
	switch subscriptionType {
	case "all":
		subscriptionTypeEnum = mailer.All
	case "none":
		subscriptionTypeEnum = mailer.None
	case "specific":
		subscriptionTypeEnum = mailer.Specific
	default:
		templateMap["Status"] = "Failed"
		templateMap["Message"] = l10n.T.GetPath(lang, "UserProfile", "SubscribeInvalidType").(string)
		return fiber.StatusUnprocessableEntity, nil
	}

	specificTags := c.FormValue("tags_picked")
	if err = supplements.Mailer.Subscribe(supplements.Mailer.GetHash(c.IP()), subscriptionTypeEnum, specificTags); err != nil {
		templateMap["Status"] = "Failed"
		templateMap["Message"] = l10n.T.GetPath(lang, "UserProfile", "FailedToSubscribe").(string)
		return fiber.StatusUnprocessableEntity, nil
	}

	templateMap["Status"] = "OK"
	templateMap["Message"] = l10n.T.GetPath(lang, "UserProfile", "SubscribedSuccessfully").(string)
	return fiber.StatusOK, nil
}
