package handlers

import (
	"log/slog"
	"time"

	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

type SitemapXmlHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &SitemapXmlHandler{})
}

func (r *SitemapXmlHandler) Filter() (method string, path string) {
	return "GET", "/sitemap.xml"
}

func (r *SitemapXmlHandler) IsTemplated() bool {
	return false
}

func (r *SitemapXmlHandler) TemplatesToInject() []string {
	return []string{"views/pages/sitemap.xml"}
}

func (r *SitemapXmlHandler) ToCache() router.CacheSetting {
	return router.ByUrlOnly
}

func (r *SitemapXmlHandler) CacheDuration() time.Duration {
	return time.Hour
}

func (r *SitemapXmlHandler) ToValidateLang() router.LangSetting {
	return router.NotRequired
}

func (r *SitemapXmlHandler) ContentType() string {
	return fiber.MIMEApplicationXMLCharsetUTF8
}

func (r *SitemapXmlHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	urls := []router.SitemapInfo{}
	for _, route := range router.Routes {
		sitemapInfo := route.SitemapInfo(supplements)
		if sitemapInfo == nil {
			continue
		}

		method, match := route.Filter()
		lastMod, err := supplements.TemplateManager.GetLastModified(method + " " + match)
		if err != nil {
			slog.Warn("failed to get last modified for a route",
				slog.String("error", err.Error()),
				slog.String("method", method),
				slog.String("match", match))
			lastMod = time.Now()
		}

		for i := range sitemapInfo {
			if sitemapInfo[i].LastModified.Before(lastMod) {
				sitemapInfo[i].LastModified = lastMod
			}
			if sitemapInfo[i].ChangeFreq == "" {
				sitemapInfo[i].ChangeFreq = "monthly"
			}
			urls = append(urls, sitemapInfo[i])
		}
	}

	templateMap["URLs"] = urls

	return fiber.StatusOK, nil
}
