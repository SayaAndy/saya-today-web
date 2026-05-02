package router

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

var (
	RateLimiterStrict = limiter.New(limiter.Config{
		Max:          5,
		Expiration:   time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string { return c.IP() },
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).SendString("rate limit")
		},
	})

	RateLimiterMedium = limiter.New(limiter.Config{
		Max:          30,
		Expiration:   time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string { return c.IP() },
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).SendString("rate limit")
		},
	})

	RateLimiterLoose = limiter.New(limiter.Config{
		Max:          60,
		Expiration:   time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string { return c.IP() },
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).SendString("rate limit")
		},
	})
)
