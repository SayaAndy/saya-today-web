package router

import (
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/gofiber/fiber/v2"
)

func init() {
	tm.Add("blog-page-like-button", "views/partials/blog-page-like-button.html")
}

func Api_V1_Like_Put(b2 *b2.B2Client) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)

		referer := c.Get("Referer", "")
		if referer == "" {
			return c.Status(fiber.ErrBadRequest.Code).SendString("'Referer' header is empty")
		}
		urlStruct, err := url.ParseRequestURI(referer)
		if err != nil {
			return c.Status(fiber.ErrBadRequest.Code).SendString(fmt.Sprintf("'Referer' header is invalid: %s", err.Error()))
		}

		path := urlStruct.EscapedPath()
		pathParts := strings.Split(strings.Trim(path, "/"), "/")
		if len(pathParts) != 3 {
			return c.Status(fiber.ErrBadRequest.Code).SendString("'Referer' header is invalid: expect format '/{lang}/blog/{page}'")
		}

		lang, page := pathParts[0], pathParts[2]

		pageLink := lang + "/" + page + ".md"
		if pages, _ := b2.Scan(pageLink); len(pages) == 0 {
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("server did not find '%s' article", pageLink))
		}

		ip := c.IP()
		newLikeStatus, err := strconv.ParseBool(c.FormValue("like", "true"))
		if err != nil {
			return c.Status(fiber.ErrBadRequest.Code).SendString("invalid 'like' value")
		}

		if newLikeStatus {
			CCache.LikeOn(ip, page)
		} else {
			CCache.LikeOff(ip, page)
		}

		slog.Debug("someone pressed the like button!", slog.String("ip", ip), slog.String("page", page), slog.String("new_like_status", fmt.Sprint(newLikeStatus)))
		if c.Get("HX-Request", "false") == "true" {
			content, err := tm.Render("blog-page-like-button", fiber.Map{
				"Liked": newLikeStatus,
			})
			if err != nil {
				slog.Warn("failed to generate div", slog.String("path", path), slog.String("error", err.Error()))
				return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
			}
			c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
			return c.Status(fiber.StatusOK).Send(content)
		}

		return c.Status(fiber.StatusOK).SendString(fmt.Sprint(newLikeStatus))
	}
}

func Api_V1_Like_Get(b2 *b2.B2Client) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)

		referer := c.Get("Referer", "")
		if referer == "" {
			return c.Status(fiber.ErrBadRequest.Code).SendString("'Referer' header is empty")
		}
		urlStruct, err := url.ParseRequestURI(referer)
		if err != nil {
			return c.Status(fiber.ErrBadRequest.Code).SendString(fmt.Sprintf("'Referer' header is invalid: %s", err.Error()))
		}

		path := urlStruct.EscapedPath()
		pathParts := strings.Split(strings.Trim(path, "/"), "/")
		if len(pathParts) != 3 {
			return c.Status(fiber.ErrBadRequest.Code).SendString("'Referer' header is invalid: expect format '/{lang}/blog/{page}'")
		}

		lang, page := pathParts[0], pathParts[2]

		pageLink := lang + "/" + page + ".md"
		if pages, _ := b2.Scan(pageLink); len(pages) == 0 {
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("server did not find '%s' article", pageLink))
		}

		ip := c.IP()
		likeStatus := CCache.GetLikeStatus(ip, page)

		slog.Debug("someone requested the like status!", slog.String("ip", ip), slog.String("page", page), slog.Bool("like_status", likeStatus))
		if c.Get("HX-Request", "false") == "true" {
			content, err := tm.Render("blog-page-like-button", fiber.Map{
				"Liked": likeStatus,
			})
			if err != nil {
				slog.Warn("failed to generate div", slog.String("path", path), slog.String("error", err.Error()))
				return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
			}
			c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
			return c.Status(fiber.StatusOK).Send(content)
		}

		return c.Status(fiber.StatusOK).SendString(fmt.Sprint(likeStatus))
	}
}
