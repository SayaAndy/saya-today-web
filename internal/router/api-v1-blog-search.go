package router

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
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
)

func init() {
	tm.Add("catalogue-blog-cards", "views/partials/catalogue-blog-cards.html", "views/partials/catalogue-blog-card-tags.html")
}

func Api_V1_BlogSearch(l map[string]*locale.LocaleConfig, langs []string, b2Client *b2.B2Client) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		sort := c.Query("sort")
		lang := c.Query("lang")
		tz := c.Query("tz")

		if !slices.Contains(langs, lang) {
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("server does not support '%s' language", lang))
		}

		loc, err := time.LoadLocation(tz)
		if err != nil {
			loc = time.UTC
			slog.Warn("unable to parse a client timezone, defaulting to UTC", slog.String("error", err.Error()), slog.String("tz", tz))
		}

		cacheKey := "blog-search." + lang + ".pages-list"
		var pages []*b2.BlogPage
		if pagesBytes, ok := PCache.Get(cacheKey); pagesBytes != nil || ok {
			json.Unmarshal(pagesBytes, &pages)
		} else {
			pages, err = b2Client.Scan(lang + "/")
			if err != nil {
				c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
				return c.Status(fiber.ErrInternalServerError.Code).SendString(fmt.Sprintf("failed to scan pages for '%s' lang: %v", lang, err))
			}
			pagesBytes, _ := json.Marshal(pages)
			PCache.SetWithTTL(cacheKey, pagesBytes, int64(len(pagesBytes)), 5*time.Minute)
		}

		encodedQuery := c.Request().URI().QueryString()
		re, err := regexp.Compile(`tags\[\]=([\w]+)`)
		if err != nil {
			slog.Warn("failed to generate regex for tags gathering", slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate regex for tags gathering")
		}
		decodedQuery, _ := url.QueryUnescape(string(encodedQuery))
		matches := re.FindAllStringSubmatch(decodedQuery, -1)

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
						"LikeCount":        CCache.GetLikeCount(page.FileName),
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

		content, err := tm.Render("catalogue-blog-cards", fiber.Map{
			"BlogPages": pageMeta,
			"L":         l[lang],
		})
		if err != nil {
			slog.Warn("failed to generate div", slog.String("page", "/"+lang+"/blog"), slog.String("error", err.Error()))
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
		}

		return c.Type("html").Send(content)
	}
}
