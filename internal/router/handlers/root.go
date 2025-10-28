package handlers

import (
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

func (r *RootHandler) RenderBody(c *fiber.Ctx, supplements *router.Supplements, _ string, templateMap fiber.Map) (statusCode int, err error) {
	templateMap["Title"] = "Choose Your Language"
	templateMap["AvailableLanguages"] = supplements.AvailableLanguages
	return fiber.StatusOK, nil
}

func (r *RootHandler) RenderHeader(c *fiber.Ctx, supplements *router.Supplements, _ string, templateMap fiber.Map) (statusCode int, err error) {
	templateMap["Title"] = "Choose Your Language"
	return fiber.StatusOK, nil
}
