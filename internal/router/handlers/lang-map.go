package handlers

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

type MapHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &MapHandler{})
}

func (r *MapHandler) Filter() (method string, path string) {
	return "GET", "/:lang/map"
}

func (r *MapHandler) IsTemplated() bool {
	return false
}

func (r *MapHandler) TemplatesToInject() []string {
	return []string{"views/pages/global-map.html"}
}

func (r *MapHandler) ToCache() router.CacheSetting {
	return router.Disabled
}

func (r *MapHandler) ToValidateLang() router.LangSetting {
	return router.InPath
}

func (r *MapHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	pages, err := supplements.B2Client.Scan(lang + "/")
	status := fiber.StatusOK
	if err != nil {
		slog.Error("received an error while scanning b2 pages",
			slog.String("error", err.Error()),
			slog.String("lang", lang),
		)
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

	templateMap["MapMarkers"] = mapMarkers
	templateMap["MapLocationLat"] = 45.4507
	templateMap["MapLocationLong"] = 68.8319

	return status, nil
}
