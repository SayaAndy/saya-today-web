package router

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	assert(0, tm.Add("personal-page-status", "views/partials/personal-page-status.html"))
}

func Api_V1_Email_IsInVerification(l map[string]*locale.LocaleConfig) func(c *fiber.Ctx) error {
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
		isAllowed, whenAllowed, codeExpiry := Mailer.IsAllowedToRetryVerification(id)
		if !isAllowed {
			return api_v1_email_sendStatusHtml(c, "email-message", l, "Neutral", fiber.StatusOK, lang, strings.ReplaceAll(l[lang].UserProfile.DelayTilVerification, "{}", whenAllowed.Format("2006-01-02 15:04:05 MST")), map[string]string{
				"striked-end-time": fmt.Sprint(whenAllowed.UnixMilli()),
				"code-expiry-time": fmt.Sprint(codeExpiry.UnixMilli()),
			})
		}
		return api_v1_email_sendStatusHtml(c, "email-message", l, "OK", fiber.StatusOK, lang, "", map[string]string{
			"code-expiry-time": fmt.Sprint(codeExpiry.UnixMilli()),
		})
	}
}
