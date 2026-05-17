package handlers

import (
	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

type MapHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &MapHandler{})
}

func (r *MapHandler) Filter() (method string, path string) {
	return "GET", "/:lang/map"
}

func (r *MapHandler) IsTemplated() bool {
	return false
}

func (r *MapHandler) TemplatesToInject() []string {
	return []string{"views/pages/global-map.html"}
}

func (r *MapHandler) ToCache() router.CacheSetting {
	return router.Disabled
}

func (r *MapHandler) ToValidateLang() router.LangSetting {
	return router.InPath
}

func (r *MapHandler) SitemapInfo(supplements *router.Supplements) []router.SitemapInfo {
	sitemapInfo := []router.SitemapInfo{}
	for _, lang := range supplements.AvailableLanguages {
		sitemapInfo = append(sitemapInfo, router.SitemapInfo{Loc: "/" + lang.Name + "/map", Priority: 0.3})
	}
	return sitemapInfo
}

func (r *MapHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	return fiber.StatusOK, nil
}
