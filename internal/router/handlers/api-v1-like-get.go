package handlers

import (
	"fmt"
	"log/slog"

	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

type GetLikeHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &GetLikeHandler{})
}

func (r *GetLikeHandler) Filter() (method string, path string) {
	return "GET", "/api/v1/like"
}

func (r *GetLikeHandler) IsTemplated() bool {
	return false
}

func (r *GetLikeHandler) TemplatesToInject() []string {
	return []string{"views/partials/blog-page-like-button.html"}
}

func (r *GetLikeHandler) ToCache() router.CacheSetting {
	return router.Disabled
}

func (r *GetLikeHandler) ToValidateLang() router.LangSetting {
	return router.InReferer
}

func (r *GetLikeHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	path, pathParts, _, err := router.GetPathFromReferer(c)
	if err != nil {
		return fiber.StatusBadRequest, fmt.Errorf("error getting path from referer: %w", err)
	}

	if len(pathParts) != 3 && pathParts[1] != "blog" {
		return fiber.StatusBadRequest, fmt.Errorf("invalid path format: expected '/:lang/blog/:page', got '%s'", path)
	}

	page := pathParts[2]
	pageLink := lang + "/" + page + ".md"
	if pages, _ := supplements.B2Client.Scan(pageLink); len(pages) == 0 {
		return fiber.StatusNotFound, fmt.Errorf("server did not find '%s' article", pageLink)
	}

	ip := c.IP()
	likeStatus := supplements.ClientCache.GetLikeStatus(ip, page)

	slog.Debug("someone requested the like status!", slog.String("ip", ip), slog.String("page", page), slog.Bool("like_status", likeStatus))
	templateMap["Liked"] = likeStatus
	templateMap["LikedCount"] = supplements.ClientCache.GetLikeCount(page)

	return fiber.StatusOK, nil
}
