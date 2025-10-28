package handlers

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

type PutLikeHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &PutLikeHandler{})
}

func (r *PutLikeHandler) Filter() (method string, path string) {
	return "PUT", "/api/v1/like"
}

func (r *PutLikeHandler) IsTemplated() bool {
	return false
}

func (r *PutLikeHandler) TemplatesToInject() []string {
	return []string{"views/partials/blog-page-like-button.html"}
}

func (r *PutLikeHandler) ToCache() router.CacheSetting {
	return router.Disabled
}

func (r *PutLikeHandler) ToValidateLang() router.LangSetting {
	return router.InReferer
}

func (r *PutLikeHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
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

	newLikeStatus, err := strconv.ParseBool(c.FormValue("like", "true"))
	if err != nil {
		return fiber.StatusBadRequest, fmt.Errorf("invalid like value '%s'", c.FormValue("like"))
	}

	ip := c.IP()
	if newLikeStatus {
		supplements.ClientCache.LikeOn(ip, page)
	} else {
		supplements.ClientCache.LikeOff(ip, page)
	}

	slog.Debug("someone pressed the like button!", slog.String("ip", ip), slog.String("page", page), slog.String("new_like_status", fmt.Sprint(newLikeStatus)))
	templateMap["Liked"] = newLikeStatus
	templateMap["LikedCount"] = supplements.ClientCache.GetLikeCount(page)
	templateMap["StatusId"] = "email-message"

	return fiber.StatusOK, nil
}
