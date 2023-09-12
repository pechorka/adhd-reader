package webscraper

import (
	"context"
	"errors"

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
		},
	}
}

func (ws *WebScrapper) Scrape(ctx context.Context, link string) (string, string, error) {
	for _, s := range ws.scrapers {
		if s.Support(link) {
			return s.Scrape(ctx, link)
		}
	}

	return "", "", ErrUnsupportedLink
}
