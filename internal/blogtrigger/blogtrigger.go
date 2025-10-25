package blogtrigger

import (
	"fmt"
	"log/slog"

	"github.com/SayaAndy/saya-today-web/config"
	"github.com/SayaAndy/saya-today-web/internal/b2"
	"github.com/go-co-op/gocron/v2"
)

type BlogTriggerScheduler struct {
	s              gocron.Scheduler
	knownBlogPages map[string]map[string]*b2.BlogPage
	b2Client       *b2.B2Client
	onTrigger      func([]*b2.BlogPage) error
}

func NewBlogTriggerScheduler(b2Client *b2.B2Client, availableLanguages []config.AvailableLanguageConfig, cron string, onTrigger func([]*b2.BlogPage) error) (*BlogTriggerScheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create new scheduler: %w", err)
	}

	knownBlogPages := make(map[string]map[string]*b2.BlogPage, len(availableLanguages))
	for _, lang := range availableLanguages {
		knownBlogPages[lang.Name] = make(map[string]*b2.BlogPage)
	}

	bts := &BlogTriggerScheduler{s, knownBlogPages, b2Client, onTrigger}
	defer bts.s.Start()

	bts.s.NewJob(gocron.CronJob(cron, false), gocron.NewTask(func(bts *BlogTriggerScheduler) {
		posts, err := bts.scan()
		if err != nil {
			slog.Error("failed to execute scanning new blog pages cron job", slog.String("error", err.Error()))
			return
		}
		if err = onTrigger(posts); err != nil {
			slog.Error("error happened on callback function after scanning new blog pages", slog.String("error", err.Error()))
			return
		}
	}, bts))

	if _, err = bts.scan(); err != nil {
		return nil, fmt.Errorf("failed to scan existing blog pages in b2: %w", err)
	}

	return bts, nil
}

func (bts *BlogTriggerScheduler) scan() (newPages []*b2.BlogPage, err error) {
	newPages = make([]*b2.BlogPage, 0)
	for lang := range bts.knownBlogPages {
		posts, err := bts.b2Client.Scan(lang + "/")
		if err != nil {
			return nil, fmt.Errorf("failed to scan blog pages in b2 on '%s': %w", lang, err)
		}
		for _, post := range posts {
			if _, ok := bts.knownBlogPages[lang][post.FileName]; !ok {
				post.Lang = lang
				newPages = append(newPages, post)
				bts.knownBlogPages[lang][post.FileName] = post
			}
		}
	}
	return newPages, nil
}
