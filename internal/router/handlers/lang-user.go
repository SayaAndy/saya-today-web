package handlers

import (
	"fmt"
	"log/slog"

	"github.com/SayaAndy/saya-today-web/internal/mailer"
	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

func init() {
	router.Routes = append(router.Routes, &UserHandler{})
}

type UserHandler struct {
	router.BasicHandler
}

func (r *UserHandler) Filter() (method string, path string) {
	return "GET", "/:lang/user"
}

func (r *UserHandler) IsTemplated() bool {
	return true
}

func (r *UserHandler) TemplatesToInject() []string {
	return []string{"views/pages/user-page.html"}
}

func (r *UserHandler) ToCache() router.CacheSetting {
	return router.Disabled
}

func (r *UserHandler) ToValidateLang() router.LangSetting {
	return router.InPath
}

func (r *UserHandler) RenderBody(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	templateMap["Title"] = supplements.Localization[lang].UserProfile.Header

	email, _, err := supplements.Mailer.GetInfo(supplements.Mailer.GetHash(c.IP()))
	if err != nil {
		slog.Error("get info from mailer about a client", slog.String("error", err.Error()))
	}

	tagsArray, err := getTags(supplements.B2Client, lang)
	if err != nil {
		return fiber.ErrInternalServerError.Code, fmt.Errorf("failed to get the available tags")
	}

	subscriptionType, tags, err := supplements.Mailer.GetSubscriptions(c.IP())
	if err != nil {
		return fiber.ErrInternalServerError.Code, fmt.Errorf("failed to get the user subscriptions")
	}

	switch subscriptionType {
	case mailer.None:
		templateMap["TagsPicked"] = "none"
	case mailer.All:
		templateMap["TagsPicked"] = "all"
	case mailer.Specific:
		templateMap["TagsPicked"] = "specific"
	}

	templateMap["TagsPickedList"] = tags

	templateMap["Email"] = email
	templateMap["EmailCode"] = c.Query("email_code")
	templateMap["ExistingTags"] = tagsArray
	return fiber.StatusOK, nil
}

func (r *UserHandler) RenderHeader(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	templateMap["Title"] = supplements.Localization[lang].UserProfile.Header
	return fiber.StatusOK, nil
}
