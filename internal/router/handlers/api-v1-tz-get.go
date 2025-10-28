package handlers

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

type GetTimezoneHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &GetTimezoneHandler{})
}

func (r *GetTimezoneHandler) Filter() (method string, path string) {
	return "GET", "/api/v1/tz"
}

func (r *GetTimezoneHandler) IsTemplated() bool {
	return false
}

func (r *GetTimezoneHandler) TemplatesToInject() []string {
	return []string{}
}

func (r *GetTimezoneHandler) ToCache() router.CacheSetting {
	return router.Disabled
}

func (r *GetTimezoneHandler) ToValidateLang() router.LangSetting {
	return router.NotRequired
}

func (r *GetTimezoneHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	timestampString := c.Query("timestamp")
	tz := c.Query("tz")

	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
		slog.Warn("unable to parse a client timezone, defaulting to UTC", slog.String("error", err.Error()), slog.String("tz", tz))
	}

	var format string
	var timestamp time.Time
	for _, format = range []string{"2006-01-02 15:04:05 -07:00", time.RFC3339} {
		if timestamp, err = time.Parse(format, timestampString); err == nil {
			break
		}
	}

	if err != nil {
		return fiber.StatusBadRequest, fmt.Errorf("invalid timestamp format for '%s'", timestampString)
	}

	templateMap["Output"] = []byte(timestamp.In(loc).Format("2006-01-02 15:04:05 -07:00"))

	return fiber.StatusOK, nil
}
