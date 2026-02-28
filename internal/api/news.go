package api

import (
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
)

type NewsItem struct {
	Title       string
	Link        string
	Description string
	Published   time.Time
	Source      string
}

func FetchNews(feedURL string) ([]NewsItem, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return nil, err
	}

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
		}
		if item.PublishedParsed != nil {
			ni.Published = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			ni.Published = *item.UpdatedParsed
		} else {
			ni.Published = time.Now()
		}
		items = append(items, ni)
		if len(items) >= 30 {
			break
		}
	}
	return items, nil
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
