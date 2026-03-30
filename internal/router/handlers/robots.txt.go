package handlers

import (
	"time"

	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

type RobotsTxtHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &RobotsTxtHandler{})
}

func (r *RobotsTxtHandler) Filter() (method string, path string) {
	return "GET", "/robots.txt"
}

func (r *RobotsTxtHandler) IsTemplated() bool {
	return false
}

func (r *RobotsTxtHandler) TemplatesToInject() []string {
	return []string{"views/pages/robots.txt"}
}

func (r *RobotsTxtHandler) ToCache() router.CacheSetting {
	return router.ByUrlOnly
}

func (r *RobotsTxtHandler) CacheDuration() time.Duration {
	return time.Hour
}

func (r *RobotsTxtHandler) ToValidateLang() router.LangSetting {
	return router.NotRequired
}

func (r *RobotsTxtHandler) ContentType() string {
	return fiber.MIMETextPlainCharsetUTF8
}

func (r *RobotsTxtHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	return fiber.StatusOK, nil
}
