package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

type PathMatcher struct {
	app *fiber.App
}

func NewPathMatcher() *PathMatcher {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	return &PathMatcher{app: app}
}

func (pm *PathMatcher) AddRoute(method, pattern string) {
	pm.app.Add(method, pattern, func(c *fiber.Ctx) error {
		c.Locals("pattern", pattern)
		return nil
	})
}

func (pm *PathMatcher) MatchPath(method, path string) (pattern string, params map[string]string, matched bool) {
	fctx := &fasthttp.RequestCtx{}
	fctx.Request.Header.SetMethod(method)
	fctx.Request.SetRequestURI(path)

	ctx := pm.app.AcquireCtx(fctx)
	defer pm.app.ReleaseCtx(ctx)

	pm.app.Handler()(fctx)

	if ctx.Route() == nil {
		return "", nil, false
	}

	pattern = ctx.Locals("pattern").(string)

	params = make(map[string]string)
	for _, paramName := range ctx.Route().Params {
		params[paramName] = ctx.Params(paramName)
	}

	return pattern, params, true
}
