package router

import (
	"net/url"
	"strings"

	"github.com/SayaAndy/saya-today-web/internal/mailer"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	assert(0, tm.Add("personal-page-status", "views/partials/personal-page-status.html"))
}

func Api_V1_Subs_Put(l map[string]*locale.LocaleConfig) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)

		id := c.IP()
		lang := "en"

		var referer, path string
		var pathParts []string
		var urlStruct *url.URL
		var err error

		referer = c.Get("Referer", "")
		if referer == "" {
			goto skipFetchingLang
		}

		urlStruct, err = url.ParseRequestURI(referer)
		if err != nil {
			goto skipFetchingLang
		}

		path = urlStruct.EscapedPath()
		pathParts = strings.Split(strings.Trim(path, "/"), "/")
		if len(pathParts) == 0 {
			goto skipFetchingLang
		}

		lang = pathParts[0]

	skipFetchingLang:
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
			return api_v1_email_sendStatusHtml(c, "subs-message", l, "Failed", fiber.StatusUnprocessableEntity, lang, l[lang].UserProfile.SubscribeInvalidType, map[string]string{})
		}

		specificTags := c.FormValue("tags_picked")
		if err = Mailer.Subscribe(id, subscriptionTypeEnum, specificTags); err != nil {
			return api_v1_email_sendStatusHtml(c, "subs-message", l, "Failed", fiber.StatusUnprocessableEntity, lang, l[lang].UserProfile.FailedToSubscribe, map[string]string{})
		}

		return api_v1_email_sendStatusHtml(c, "subs-message", l, "OK", fiber.StatusOK, lang, l[lang].UserProfile.SubscribedSuccessfully, map[string]string{})
	}
}
