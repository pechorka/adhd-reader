package webscraper

import (
	"context"
	"errors"
	"strings"

	"github.com/pechorka/adhd-reader/pkg/webscraper/telegram"
	"github.com/pechorka/adhd-reader/pkg/webscraper/telegraph"
)

var ErrUnsupportedLink = errors.New("unsupported link")

type scraper interface {
	Support(link string) bool
	Scrape(ctx context.Context, link string) (title string, body string, err error)
}

type WebScrapper struct {
	scrapers []scraper
}

func New() *WebScrapper {
	return &WebScrapper{
		scrapers: []scraper{
			telegraph.New(),
			telegram.New(),
		},
	}
}

func (ws *WebScrapper) Scrape(ctx context.Context, link string) (string, string, error) {
	link = strings.TrimSpace(link)
	for _, s := range ws.scrapers {
		if s.Support(link) {
			return s.Scrape(ctx, link)
		}
	}

	return "", "", ErrUnsupportedLink
}
