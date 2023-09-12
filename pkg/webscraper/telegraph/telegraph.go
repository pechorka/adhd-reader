package telegraph

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pechorka/adhd-reader/pkg/webscraper/internal/ua"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

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
	u, err := url.Parse(link)
	if err != nil {
		return false
	}

	return u.Hostname() == "telegra.ph"
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

	// exctract  <meta property="og:title" content="Основы здорового рациона">
	title, ok := doc.Find("meta[property='og:title']").Attr("content")
	if !ok {
		return "", "", errors.New("can't find title")
	}
	article := text(doc.Find("article"))

	return title, article, nil
}

// text is modified version of goquery.Selection.Text, that concatenates each node with new line
func text(s *goquery.Selection) string {
	var buf bytes.Buffer

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
			buf.WriteByte('\n')
		}
		if n.FirstChild != nil {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
	}
	for _, n := range s.Nodes {
		f(n)
	}

	return buf.String()
}
