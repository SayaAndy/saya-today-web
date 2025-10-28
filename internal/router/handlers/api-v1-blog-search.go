package handlers

import (
	"encoding/json"
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

type BlogSearchHandler struct {
	router.BasicHandler
	getTagsQuery *regexp.Regexp
}

func init() {
	router.Routes = append(router.Routes, &BlogSearchHandler{getTagsQuery: regexp.MustCompile(`tags\[\]=([\w]+)`)})
}

func (r *BlogSearchHandler) Filter() (method string, path string) {
	return "GET", "/api/v1/blog-search"
}

func (r *BlogSearchHandler) IsTemplated() bool {
	return false
}

func (r *BlogSearchHandler) TemplatesToInject() []string {
	return []string{"views/partials/catalogue-blog-cards.html", "views/partials/catalogue-blog-card-tags.html"}
}

func (r *BlogSearchHandler) ToCache() router.CacheSetting {
	return router.Disabled
}

func (r *BlogSearchHandler) ToValidateLang() router.LangSetting {
	return router.InForm
}

func (r *BlogSearchHandler) Render(c *fiber.Ctx, supplements *router.Supplements, lang string, templateMap fiber.Map) (statusCode int, err error) {
	sort := c.Query("sort")
	tz := c.Query("tz")

	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
		slog.Warn("unable to parse a client timezone, defaulting to UTC", slog.String("error", err.Error()), slog.String("tz", tz))
	}

	cacheKey := "blog-search." + lang + ".pages-list"

	var pages []*b2.BlogPage
	if pagesBytes, ok := supplements.PageCache.Get(cacheKey); pagesBytes != nil || ok {
		json.Unmarshal(pagesBytes, &pages)
	} else {
		pages, err = supplements.B2Client.Scan(lang + "/")
		if err != nil {
			return fiber.StatusInternalServerError, fmt.Errorf("failed to scan pages for '%s' lang: %w", lang, err)
		}
		pagesBytes, _ := json.Marshal(pages)
		supplements.PageCache.SetWithTTL(cacheKey, pagesBytes, int64(len(pagesBytes)), r.CacheDuration())
	}

	encodedQuery := c.Request().URI().QueryString()
	decodedQuery, _ := url.QueryUnescape(string(encodedQuery))
	matches := r.getTagsQuery.FindAllStringSubmatch(decodedQuery, -1)

	tags := make([]string, 0, len(matches))
	for _, match := range matches {
		tags = append(tags, string(match[1]))
	}

	pageMeta := make([]fiber.Map, 0, len(pages))
	for _, page := range pages {
		for _, tag := range page.Metadata.Tags {
			if len(tags) == 0 || slices.Contains(tags, tag) {
				pageMeta = append(pageMeta, fiber.Map{
					"Link":             page.Link,
					"ArticleLink":      "/" + lang + "/blog/" + page.FileName,
					"Title":            page.Metadata.Title,
					"PublishedTime":    page.Metadata.PublishedTime.In(loc).Format("2006-01-02 15:04:05 -07:00"),
					"ActionDate":       page.Metadata.ActionDate,
					"ShortDescription": page.Metadata.ShortDescription,
					"Thumbnail":        page.Metadata.Thumbnail,
					"Tags":             page.Metadata.Tags,
					"LikeCount":        supplements.ClientCache.GetLikeCount(page.FileName),
					"ViewCount":        supplements.ClientCache.GetViewCount(page.FileName),
				})
				break
			}
		}
	}

	slices.SortFunc(pageMeta, func(a, b fiber.Map) int {
		switch sort {
		case "titleAsc":
			return strings.Compare(a["Title"].(string), b["Title"].(string))
		case "titleDesc":
			return strings.Compare(b["Title"].(string), a["Title"].(string))
		case "actionDateAsc":
			return strings.Compare(a["ActionDate"].(string), b["ActionDate"].(string))
		case "actionDateDesc":
			return strings.Compare(b["ActionDate"].(string), a["ActionDate"].(string))
		case "publicationDateAsc":
			publishedTimeA, _ := time.Parse("2006-01-02 15:04:05 -07:00", a["PublishedTime"].(string))
			publishedTimeB, _ := time.Parse("2006-01-02 15:04:05 -07:00", b["PublishedTime"].(string))
			return publishedTimeA.Compare(publishedTimeB)
		case "publicationDateDesc":
			publishedTimeA, _ := time.Parse("2006-01-02 15:04:05 -07:00", a["PublishedTime"].(string))
			publishedTimeB, _ := time.Parse("2006-01-02 15:04:05 -07:00", b["PublishedTime"].(string))
			return publishedTimeB.Compare(publishedTimeA)
		}
		return 0
	})

	templateMap["BlogPages"] = pageMeta

	return fiber.StatusOK, nil
}
