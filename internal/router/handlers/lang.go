package handlers

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

func init() {
	router.Routes = append(router.Routes, &HomeHandler{})
}

type HomeHandler struct {
	router.BasicHandler
}

func (r *HomeHandler) Filter() (method string, path string) {
	return "GET", "/:lang<len(2)>"
}

func (r *HomeHandler) IsTemplated() bool {
	return true
}

func (r *HomeHandler) TemplatesToInject() []string {
	return []string{"views/pages/home-page.html"}
}

func (r *HomeHandler) ToCache() router.CacheSetting {
	return router.ByUrlOnly
}

func (r *HomeHandler) CacheDuration() time.Duration {
	return 10 * time.Minute
}

func (r *HomeHandler) ToValidateLang() router.LangSetting {
	return router.InPath
}

func (r *HomeHandler) SitemapInfo(supplements *router.Supplements) []router.SitemapInfo {
	sitemapInfo := []router.SitemapInfo{}
	for _, lang := range supplements.AvailableLanguages {
		sitemapInfo = append(sitemapInfo, router.SitemapInfo{Loc: "/" + lang.Name, Priority: 0.7})
	}
	return sitemapInfo
}

func (r *HomeHandler) AddMeta(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (meta []router.MetaField, err error) {
	return []router.MetaField{
		{Property: "og:title", Content: supplements.Localization[lang].HomePage.Header},
		{Property: "og:description", Content: supplements.Localization[lang].HomePage.HomePageDescription},
		{Property: "og:image", Content: fmt.Sprintf(
			supplements.PhotoStorage.HomePageGifs.BaseUrl,
			supplements.PhotoStorage.HomePageGifs.Indexes[rand.Int()%len(supplements.PhotoStorage.HomePageGifs.Indexes)],
		)},
		{Property: "og:url", Content: fmt.Sprintf("%s/%s", templateMap["CanonicalEndpoint"], lang)},
		{Property: "og:type", Content: "website"},
	}, nil
}

func (r *HomeHandler) RenderBody(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	templateMap["Title"] = supplements.Localization[lang].HomePage.Header
	templateMap["FilledHeartCount"] = uint(40)
	templateMap["OutlineHeartCount"] = uint(40)
	templateMap["GifUrl"] = fmt.Sprintf(
		supplements.PhotoStorage.HomePageGifs.BaseUrl,
		supplements.PhotoStorage.HomePageGifs.Indexes[rand.Int()%len(supplements.PhotoStorage.HomePageGifs.Indexes)],
	)
	templateMap["FunFacts"] = supplements.FactGiver.Give(lang)
	return fiber.StatusOK, nil
}

func (r *HomeHandler) RenderHeader(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	templateMap["Title"] = supplements.Localization[lang].HomePage.Header
	return fiber.StatusOK, nil
}
