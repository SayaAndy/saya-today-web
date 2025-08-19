package router

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	tm.Add("global-map", "views/pages/global-map.html")
}

func Lang_Map(l map[string]*locale.LocaleConfig, langs []string, b2Client *b2.B2Client) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		lang := c.Params("lang")
		if !slices.Contains(langs, lang) {
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("server does not support '%s' language... yet??", lang))
		}

		pages, err := b2Client.Scan(lang + "/")
		status := fiber.StatusOK
		if err != nil {
			status = fiber.StatusPartialContent
			pages = []*b2.BlogPage{}
		}

		type MapMarker struct {
			Title          string  `json:"Title"`
			PageLink       string  `json:"PageLink"`
			Lat            float64 `json:"Lat"`
			Long           float64 `json:"Long"`
			AccuracyMeters int64   `json:"AccuracyMeters"`
			Thumbnail      string  `json:"Thumbnail"`
		}

		mapMarkers := make([]*MapMarker, 0, len(pages))
		for _, page := range pages {
			geolocationParts := strings.Split(page.Metadata.Geolocation, " ")
			if len(geolocationParts) < 2 {
				continue
			}

			var x, y float64
			var areaError int64
			if len(geolocationParts) >= 2 {
				x, _ = strconv.ParseFloat(geolocationParts[0], 64)
				y, _ = strconv.ParseFloat(geolocationParts[1], 64)
			}
			if len(geolocationParts) >= 3 {
				areaError, _ = strconv.ParseInt(geolocationParts[2], 10, 64)
			}

			mapMarkers = append(mapMarkers, &MapMarker{
				Title:          page.Metadata.Title,
				PageLink:       fmt.Sprintf("/%s/blog/%s", lang, page.FileName),
				Lat:            x,
				Long:           y,
				AccuracyMeters: areaError,
				Thumbnail:      page.Metadata.Thumbnail,
			})
		}

		content, err := tm.Render("global-map", fiber.Map{
			"Lang":            lang,
			"L":               l[lang],
			"MapLocationLat":  45.4507,
			"MapLocationLong": 68.8319,
			"MapMarkers":      mapMarkers,
		})
		if err != nil {
			slog.Warn("failed to generate page", slog.String("page", "/"+lang+"/map"), slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate page")
		}

		return c.Type("html").Status(status).Send(content)
	}
}
