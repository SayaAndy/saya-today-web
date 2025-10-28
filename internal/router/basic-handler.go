package router

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

type BasicHandler struct{}

var _ Route = &BasicHandler{}

func (r *BasicHandler) Filter() (method string, path string) {
	panic("handler did not implement Filter method")
}

func (r *BasicHandler) IsTemplated() bool {
	return false
}

func (r *BasicHandler) ToCache() CacheSetting {
	return Disabled
}

func (r *BasicHandler) CacheDuration() time.Duration {
	return 5 * time.Minute
}

func (r *BasicHandler) ToValidateLang() LangSetting {
	return NotRequired
}

func (r *BasicHandler) TemplatesToInject() []string {
	return []string{}
}

func (r *BasicHandler) Render(c *fiber.Ctx, supplements *Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	if !r.IsTemplated() {
		panic("handler did not implement Render method (while being non-templated)")
	}
	return fiber.StatusNoContent, nil
}

func (r *BasicHandler) RenderHeader(c *fiber.Ctx, supplements *Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	return fiber.StatusNoContent, nil
}

func (r *BasicHandler) RenderBody(c *fiber.Ctx, supplements *Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	return fiber.StatusNoContent, nil
}

func (r *BasicHandler) RenderFooter(c *fiber.Ctx, supplements *Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	return fiber.StatusNoContent, nil
}

func (r *BasicHandler) RenderTopEmbeds(c *fiber.Ctx, supplements *Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	return fiber.StatusNoContent, nil
}

func (r *BasicHandler) RenderBottomEmbeds(c *fiber.Ctx, supplements *Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	return fiber.StatusNoContent, nil
}
