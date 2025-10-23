package router

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"
	"math/rand"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/SayaAndy/saya-today-web/internal/blogtrigger"
	"github.com/SayaAndy/saya-today-web/internal/factgiver"
	"github.com/SayaAndy/saya-today-web/internal/frontmatter"
	"github.com/SayaAndy/saya-today-web/internal/mailer"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/gofiber/fiber/v2"
	"github.com/yuin/goldmark"
)

var FactGiver *factgiver.FactGiver
var Mailer *mailer.Mailer
var BlogTrigger *blogtrigger.BlogTriggerScheduler

func init() {
	assert(0, tm.Add("general-page-body", "views/partials/general-page-body.html"))
}

func Api_V1_GeneralPage_Body(l map[string]*locale.LocaleConfig, langs []config.AvailableLanguageConfig, b2Client *b2.B2Client, md goldmark.Markdown) func(c *fiber.Ctx) error {
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
		trimmedPath := strings.Trim(path, "/")

		cacheKey := fmt.Sprintf("body.%s", trimmedPath)
		if val, ok := PCache.Get(cacheKey); !strings.HasSuffix(trimmedPath, "user") && val != nil && ok {
			c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
			return c.Status(fiber.StatusOK).Type("html").Send(val)
		}

		lang := ""
		pathParts := strings.Split(trimmedPath, "/")
		if len(pathParts) == 1 && pathParts[0] == "" {
			pathParts = []string{}
		}
		if len(pathParts) > 0 {
			lang = pathParts[0]
			for _, availableLang := range langs {
				if availableLang.Name == lang {
					goto langIsAvailable
				}
			}
			return c.Status(fiber.ErrBadRequest.Code).SendString("'Referer' header is invalid: expect format '/{lang}/...'")
		}

	langIsAvailable:
		values := fiber.Map{
			"L":           l[lang],
			"Lang":        lang,
			"Path":        strings.Trim(path, "/"),
			"QueryString": string(c.Request().URI().QueryString()),
		}
		var additionalTemplates []string

		if len(pathParts) == 2 && pathParts[1] == "blog" {
			querySort := c.Query("sort")
			if querySort == "" {
				querySort = "publicationDateDesc"
			}

			encodedQuery := c.Request().URI().QueryString()
			re, err := regexp.Compile(`tags\[\]=([\w]+)`)
			if err != nil {
				slog.Warn("failed to generate regex for tags gathering", slog.String("error", err.Error()))
				return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate regex for tags gathering")
			}
			decodedQuery, _ := url.QueryUnescape(string(encodedQuery))
			matches := re.FindAllStringSubmatch(decodedQuery, -1)

			queryTags := make([]string, 0, len(matches))
			for _, match := range matches {
				queryTags = append(queryTags, string(match[1]))
			}

			tagsArray, err := getTags(b2Client, lang)
			if err != nil {
				slog.Warn("failed to get the available tags", slog.String("error", err.Error()))
				return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to gather available tags")
			}

			values["Tags"] = tagsArray
			values["QuerySort"] = querySort
			values["QueryTags"] = strings.Join(queryTags, ",")
			values["Title"] = l[lang].BlogSearch.Header

			additionalTemplates = append(additionalTemplates, "views/pages/blog-catalogue.html")
		} else if len(pathParts) == 2 && pathParts[1] == "user" {
			values["Title"] = l[lang].UserProfile.Header

			email, _, err := Mailer.GetInfo(Mailer.GetHash(c.IP()))
			if err != nil {
				slog.Error("get info from mailer about a client", slog.String("error", err.Error()))
			}

			tagsArray, err := getTags(b2Client, lang)
			if err != nil {
				slog.Warn("failed to get the available tags", slog.String("error", err.Error()))
				return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to gather available tags")
			}

			subscriptionType, tags, err := Mailer.GetSubscriptions(c.IP())
			if err != nil {
				slog.Warn("failed to get the user subscriptions", slog.String("error", err.Error()))
				return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to get the user subscriptions")
			}

			switch subscriptionType {
			case mailer.None:
				values["TagsPicked"] = "none"
			case mailer.All:
				values["TagsPicked"] = "all"
			case mailer.Specific:
				values["TagsPicked"] = "specific"
			}

			values["TagsPickedList"] = tags

			values["Email"] = email
			values["EmailCode"] = c.Query("email_code")
			values["ExistingTags"] = tagsArray

			additionalTemplates = append(additionalTemplates, "views/pages/user-page.html")
		} else if len(pathParts) == 3 && pathParts[1] == "blog" {
			metadata, parsedMarkdown, err := readBlogPost(md, b2Client, lang+"/"+pathParts[2])
			if err != nil {
				return c.Status(fiber.ErrNotFound.Code).SendString(fmt.Sprintf("failed to find '%s' post", pathParts[2]))
			}

			geolocationParts := strings.Split(metadata.Geolocation, " ")
			var x, y, areaError string
			if len(geolocationParts) >= 2 {
				x = geolocationParts[0]
				y = geolocationParts[1]
			}
			if len(geolocationParts) >= 3 {
				areaError = geolocationParts[2]
			}

			values["MapLocationX"] = x
			values["MapLocationY"] = y
			values["MapLocationAreaMeters"] = areaError
			values["Title"] = metadata.Title
			values["ParsedMarkdown"] = template.HTML(parsedMarkdown)
			values["PublishedDate"] = metadata.PublishedTime.Format("2006-01-02 15:04:05 -07:00")
			values["ActionDate"] = metadata.ActionDate

			additionalTemplates = append(additionalTemplates, "views/pages/blog-page.html")

			go CCache.View(c.IP(), pathParts[2])
		} else if len(pathParts) == 1 {
			values["Title"] = l[lang].HomePage.Header
			values["FilledHeartCount"] = uint(40)
			values["OutlineHeartCount"] = uint(40)
			values["GifName"] = fmt.Sprintf("otter-%d.gif", rand.Int()%3+1)
			values["FunFacts"] = FactGiver.Give(lang)
			additionalTemplates = append(additionalTemplates, "views/pages/home-page.html")
		} else if len(pathParts) == 0 {
			values["Title"] = "Choose Your Language"
			values["AvailableLanguages"] = langs
			additionalTemplates = append(additionalTemplates, "views/pages/language-pick.html")
		}

		content, err := tm.Render("general-page-body", values, additionalTemplates...)
		if err != nil {
			slog.Warn("failed to generate div", slog.String("path", path), slog.String("error", err.Error()))
			return c.Status(fiber.ErrInternalServerError.Code).SendString("failed to generate div")
		}

		go PCache.SetWithTTL(cacheKey, content, int64(len(content)), 5*time.Minute)
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return c.Status(fiber.StatusOK).Type("html").Send(content)
	}
}

func readBlogPost(md goldmark.Markdown, b2Client *b2.B2Client, sourceName string) (metadata *frontmatter.Metadata, html string, err error) {
	metadata, markdown, err := b2Client.ReadFrontmatter(sourceName + ".md")
	if err != nil {
		return nil, "", fmt.Errorf("failed to read a frontmatter file: %w", err)
	}

	var buf bytes.Buffer
	if err := md.Convert(markdown, &buf); err != nil {
		return nil, "", fmt.Errorf("convert source context from md to html: %w", err)
	}

	return metadata, buf.String(), nil
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
