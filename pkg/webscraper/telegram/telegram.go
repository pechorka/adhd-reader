package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/pechorka/adhd-reader/pkg/runeslice"
	"github.com/pechorka/adhd-reader/pkg/webscraper/internal/ua"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

const LinkPattern = `https?:\/\/t\.me\/[a-zA-Z0-9_\-]+\/\d+`

var regExp = regexp.MustCompile(LinkPattern)

type Scraper struct {
	httpCli *http.Client
}

func New() *Scraper {
	return &Scraper{
		httpCli: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *Scraper) Support(link string) bool {
	return regExp.MatchString(link)
}

func (s *Scraper) Scrape(ctx context.Context, link string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("User-Agent", ua.UserAgent)

	resp, err := s.httpCli.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return "", "", errors.Wrap(err, "read body from telegraph")
		}
		return "", "", fmt.Errorf("status code: %d, body: %s", resp.StatusCode, body)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", "", errors.Wrap(err, "create goquery document from telegraph response body")
	}

	title, ok := doc.Find("meta[property='og:title']").Attr("content")
	if !ok {
		return "", "", errors.New("can't find title")
	}
	description, ok := doc.Find("meta[property='og:description']").Attr("content")
	if !ok {
		return "", "", errors.New("can't find description")
	}
	description = html.UnescapeString(description)
	firstLine, _, _ := strings.Cut(description, "\n")
	if utf8.RuneCountInString(firstLine) > 100 {
		firstLine = runeslice.NRunes(firstLine, 100)
	}

	title += ": " + firstLine

	return title, description, nil
}
