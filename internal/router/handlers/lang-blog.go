package handlers

import (
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/internal/router"
	"github.com/gofiber/fiber/v2"
)

func init() {
	router.Routes = append(router.Routes, &CatalogueHandler{getTagsQuery: regexp.MustCompile(`tags\[\]=([\w]+)`)})
}

type CatalogueHandler struct {
	router.BasicHandler
	getTagsQuery *regexp.Regexp
}

func (r *CatalogueHandler) Filter() (method string, path string) {
	return "GET", "/:lang/blog"
}

func (r *CatalogueHandler) IsTemplated() bool {
	return true
}

func (r *CatalogueHandler) TemplatesToInject() []string {
	return []string{"views/pages/blog-catalogue.html"}
}

func (r *CatalogueHandler) ToCache() router.CacheSetting {
	return router.ByUrlAndQuery
}

func (r *CatalogueHandler) CacheDuration() time.Duration {
	return 5 * time.Minute
}

func (r *CatalogueHandler) ToValidateLang() router.LangSetting {
	return router.InPath
}

func (r *CatalogueHandler) RenderBody(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	querySort := c.Query("sort")
	if querySort == "" {
		querySort = "publicationDateDesc"
	}

	encodedQuery := c.Request().URI().QueryString()
	decodedQuery, _ := url.QueryUnescape(string(encodedQuery))
	matches := r.getTagsQuery.FindAllStringSubmatch(decodedQuery, -1)

	queryTags := make([]string, 0, len(matches))
	for _, match := range matches {
		queryTags = append(queryTags, string(match[1]))
	}

	tagsArray, err := getTags(supplements.B2Client, lang)
	if err != nil {
		return fiber.StatusInternalServerError, fmt.Errorf("failed to get the available tags")
	}

	templateMap["Tags"] = tagsArray
	templateMap["QuerySort"] = querySort
	templateMap["QueryTags"] = strings.Join(queryTags, ",")
	templateMap["Title"] = supplements.Localization[lang].BlogSearch.Header

	return fiber.StatusOK, nil
}

func (r *CatalogueHandler) RenderHeader(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	templateMap["Title"] = supplements.Localization[lang].BlogSearch.Header
	return fiber.StatusOK, nil
}

type Tag struct {
	Name  string `json:"Name" yaml:"name"`
	Count int    `json:"Count" yaml:"count"`
}

func getTags(b2Client *b2.B2Client, lang string) (tags []Tag, err error) {
	pages, err := b2Client.Scan(lang + "/")
	if err != nil {
		slog.Warn("failed to scan pages via b2", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to scan pages via b2: %w", err)
	}

	tagsMap := make(map[string]int)
	for _, page := range pages {
		for _, tag := range page.Metadata.Tags {
			tagsMap[tag]++
		}
	}
	slog.Debug("enlist pages for catalogue", slog.Int("tag_count", len(tagsMap)), slog.Int("page_count", len(pages)), slog.String("lang", lang))

	tagsArray := make([]Tag, 0, len(tagsMap))
	for tag, count := range tagsMap {
		tagsArray = append(tagsArray, Tag{tag, count})
	}
	slices.SortFunc(tagsArray, func(a Tag, b Tag) int {
		return strings.Compare(a.Name, b.Name)
	})

	return tagsArray, nil
}
