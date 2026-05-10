package api

import (
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

type NewsItem struct {
	Title       string
	Link        string
	Description string
	Published   time.Time
	Source      string // feed title (e.g. "VOCM News")
	FeedHost    string // feed URL host, "www." stripped (e.g. "vocm.com")
}

type News interface {
	FetchNews(feedURLs []string) ([]NewsItem, error)
}

type NewsClient struct{}

var _ News = (*NewsClient)(nil)

func NewNewsClient() *NewsClient {
	return &NewsClient{}
}

const maxNewsItems = 30

// FetchNews fetches all given feeds concurrently, merges them, sorts by
// publish time descending, and returns up to maxNewsItems entries. Per-feed
// errors are tolerated when at least one feed succeeds; if all feeds fail,
// the first error is returned.
func (c *NewsClient) FetchNews(feedURLs []string) ([]NewsItem, error) {
	if len(feedURLs) == 0 {
		return nil, nil
	}

	type result struct {
		items []NewsItem
		err   error
	}
	results := make([]result, len(feedURLs))
	var wg sync.WaitGroup
	for i, u := range feedURLs {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			items, err := fetchSingleFeed(u)
			results[i] = result{items: items, err: err}
		}(i, u)
	}
	wg.Wait()

	var merged []NewsItem
	var firstErr error
	for _, r := range results {
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		merged = append(merged, r.items...)
	}
	if len(merged) == 0 && firstErr != nil {
		return nil, firstErr
	}

	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].Published.After(merged[j].Published)
	})
	if len(merged) > maxNewsItems {
		merged = merged[:maxNewsItems]
	}
	return merged, nil
}

func fetchSingleFeed(feedURL string) ([]NewsItem, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return nil, err
	}

	host := feedHost(feedURL)
	items := make([]NewsItem, 0, len(feed.Items))
	for _, item := range feed.Items {
		// Prefer Content over Description (usually richer)
		raw := item.Content
		if raw == "" {
			raw = item.Description
		}

		ni := NewsItem{
			Title:       item.Title,
			Link:        item.Link,
			Description: normalizeText(stripHTML(raw)),
			Source:      feed.Title,
			FeedHost:    host,
		}
		if item.PublishedParsed != nil {
			ni.Published = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			ni.Published = *item.UpdatedParsed
		} else {
			ni.Published = time.Now()
		}
		items = append(items, ni)
		if len(items) >= maxNewsItems {
			break
		}
	}
	return items, nil
}

// feedHost extracts the hostname from a feed URL, stripping a leading "www."
// so display labels read more naturally. Returns the original string on parse
// failure so callers always have something usable.
func feedHost(feedURL string) string {
	u, err := url.Parse(feedURL)
	if err != nil || u.Host == "" {
		return feedURL
	}
	return strings.TrimPrefix(u.Hostname(), "www.")
}

// stripHTML walks the HTML parse tree and extracts plain text,
// inserting newlines at block-level element boundaries.
func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		// Fallback: strip angle-bracket tags manually.
		var b strings.Builder
		inTag := false
		for _, r := range s {
			switch {
			case r == '<':
				inTag = true
			case r == '>':
				inTag = false
			case !inTag:
				b.WriteRune(r)
			}
		}
		return b.String()
	}

	var buf strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "p", "div", "li", "br",
				"h1", "h2", "h3", "h4", "h5", "h6",
				"blockquote", "pre", "tr":
				buf.WriteRune('\n')
			}
		}
	}
	walk(doc)
	return buf.String()
}

// normalizeText collapses excessive whitespace and blank lines.
func normalizeText(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	prevBlank := false
	for _, l := range lines {
		l = strings.TrimRight(l, " \t")
		blank := strings.TrimSpace(l) == ""
		if blank && prevBlank {
			continue // collapse consecutive blank lines
		}
		out = append(out, l)
		prevBlank = blank
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
