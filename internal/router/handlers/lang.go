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

func (r *HomeHandler) RenderBody(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	templateMap["Title"] = supplements.Localization[lang].HomePage.Header
	templateMap["FilledHeartCount"] = uint(40)
	templateMap["OutlineHeartCount"] = uint(40)
	templateMap["GifName"] = fmt.Sprintf("otter-%d.gif", rand.Int()%3+1)
	templateMap["FunFacts"] = supplements.FactGiver.Give(lang)
	return fiber.StatusOK, nil
}

func (r *HomeHandler) RenderHeader(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	templateMap["Title"] = supplements.Localization[lang].HomePage.Header
	return fiber.StatusOK, nil
}
