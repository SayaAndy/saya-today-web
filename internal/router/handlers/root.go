package handlers

import (
	"fmt"
	"time"

	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

func init() {
	router.Routes = append(router.Routes, &RootHandler{})
}

type RootHandler struct {
	router.BasicHandler
}

func (r *RootHandler) Filter() (method string, path string) {
	return "GET", "/"
}

func (r *RootHandler) IsTemplated() bool {
	return true
}

func (r *RootHandler) TemplatesToInject() []string {
	return []string{"views/pages/language-pick.html"}
}

func (r *RootHandler) ToCache() router.CacheSetting {
	return router.ByUrlOnly
}

func (r *RootHandler) CacheDuration() time.Duration {
	return 24 * time.Hour
}

func (r *RootHandler) SitemapInfo(_ *router.Supplements) []router.SitemapInfo {
	return []router.SitemapInfo{{Loc: "/", Priority: 0.7}}
}

func (r *RootHandler) AddMeta(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (meta []router.MetaField, err error) {
	return []router.MetaField{
		{Property: "og:title", Content: "Saya Blog"},
		{Property: "og:description", Content: "The blog of a 25 years old, travelling God knows where, proud to be not mainstream // Личный блог 25-летки, странствующего по ебеням, зато не мейнстрим"},
		{Property: "og:url", Content: fmt.Sprint(templateMap["CanonicalEndpoint"]) + "/"},
		{Property: "og:type", Content: "website"},
	}, nil
}

func (r *RootHandler) RenderBody(c *fiber.Ctx, supplements *router.Supplements, _ string, templateMap fiber.Map) (statusCode int, err error) {
	templateMap["Title"] = "Choose Your Language"
	templateMap["AvailableLanguages"] = supplements.AvailableLanguages
	return fiber.StatusOK, nil
}

func (r *RootHandler) RenderHeader(c *fiber.Ctx, supplements *router.Supplements, _ string, templateMap fiber.Map) (statusCode int, err error) {
	templateMap["Title"] = "Choose Your Language"
	return fiber.StatusOK, nil
}
