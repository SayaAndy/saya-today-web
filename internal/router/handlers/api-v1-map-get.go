package handlers

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/SayaAndy/saya-today-web/internal/blog"
	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

type GetMapHandler struct {
	router.BasicHandler
}

func init() {
	router.Routes = append(router.Routes, &GetMapHandler{})
}

func (r *GetMapHandler) Filter() (method string, path string) {
	return "GET", "/api/v1/map"
}

func (r *GetMapHandler) IsTemplated() bool {
	return false
}

func (r *GetMapHandler) TemplatesToInject() []string {
	return []string{"views/partials/global-map-widget.html"}
}

func (r *GetMapHandler) ToCache() router.CacheSetting {
	return router.ByUrlAndQuery
}

func (r *GetMapHandler) ToValidateLang() router.LangSetting {
	return router.InReferer
}

func (r *GetMapHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	codename := c.Query("codename")
	zoom := c.QueryInt("zoom", 4)
	zoomPosition := c.Query("zoomPosition")

	pages, err := supplements.BlogClient.Scan(lang + "/")
	status := fiber.StatusOK
	if err != nil {
		slog.Error("received an error while scanning blog pages",
			slog.String("error", err.Error()),
			slog.String("lang", lang),
		)
		status = fiber.StatusPartialContent
		pages = []*blog.Page{}
	}
	slices.SortFunc(pages, func(a *blog.Page, b *blog.Page) int {
		return a.Metadata.PublishedTime.Compare(b.Metadata.PublishedTime)
	})

	type MapMarker struct {
		Index          int     `json:"Index"`
		Title          string  `json:"Title"`
		PageLink       string  `json:"PageLink"`
		Lat            float64 `json:"Lat"`
		Long           float64 `json:"Long"`
		AccuracyMeters int64   `json:"AccuracyMeters"`
		Thumbnail      string  `json:"Thumbnail"`
		ToHighlight    bool    `json:"ToHighlight"`
	}

	templateMap["MapLocationLat"] = 45.4507
	templateMap["MapLocationLong"] = 68.8319
	templateMap["MapLocationZoom"] = zoom
	templateMap["ZoomPosition"] = zoomPosition

	mapMarkers := make([]*MapMarker, 0, len(pages))
	for i, page := range pages {
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

		toHighlight := page.FileName == codename
		if toHighlight {
			templateMap["MapLocationLat"] = x
			templateMap["MapLocationLong"] = y
		}

		mapMarkers = append(mapMarkers, &MapMarker{
			Index:          i,
			Title:          page.Metadata.Title,
			PageLink:       fmt.Sprintf("/%s/blog/%s", lang, page.FileName),
			Lat:            x,
			Long:           y,
			AccuracyMeters: areaError,
			Thumbnail:      page.Metadata.Thumbnail,
			ToHighlight:    toHighlight,
		})
	}

	templateMap["MapMarkers"] = mapMarkers

	return status, nil
}
