package router

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

func Api_V1_TZ() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
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

		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)

		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("invalid timestamp format")
		}

		return c.Status(fiber.StatusOK).SendString(timestamp.In(loc).Format(format))
	}
}
