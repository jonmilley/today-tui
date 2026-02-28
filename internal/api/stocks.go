package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

type StockQuote struct {
	Symbol    string
	Name      string
	Price     float64
	Change    float64
	ChangePct float64
}

// YahooClient holds a persistent HTTP session (cookie jar + crumb) required
// by the Yahoo Finance API since mid-2024.
type YahooClient struct {
	client *http.Client
	crumb  string
	mu     sync.Mutex
}

const browserUA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"

func NewYahooClient() *YahooClient {
	jar, _ := cookiejar.New(nil)
	return &YahooClient{
		client: &http.Client{
			Jar:     jar,
			Timeout: 15 * time.Second,
		},
	}
}

// initCrumb visits finance.yahoo.com to populate cookies then fetches the crumb.
// Must be called with mu held.
func (yc *YahooClient) initCrumb() error {
	// Seed the cookie jar by visiting Yahoo Finance.
	seed, err := http.NewRequest("GET", "https://finance.yahoo.com", nil)
	if err != nil {
		return err
	}
	seed.Header.Set("User-Agent", browserUA)
	seed.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	seed.Header.Set("Accept-Language", "en-US,en;q=0.9")
	yc.client.Do(seed) //nolint:errcheck — only need the cookies

	// Fetch the crumb token.
	cr, err := http.NewRequest("GET", "https://query1.finance.yahoo.com/v1/test/getcrumb", nil)
	if err != nil {
		return err
	}
	cr.Header.Set("User-Agent", browserUA)
	cr.Header.Set("Accept", "text/plain")

	resp, err := yc.client.Do(cr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	crumb := strings.TrimSpace(string(body))
	if resp.StatusCode != http.StatusOK || crumb == "" {
		return fmt.Errorf("could not obtain crumb (HTTP %d)", resp.StatusCode)
	}
	yc.crumb = crumb
	return nil
}

// FetchQuotes fetches quotes for the given symbols, refreshing the crumb on 401.
func (yc *YahooClient) FetchQuotes(symbols []string) ([]StockQuote, error) {
	quotes, err := yc.doFetch(symbols, false)
	if err != nil && isAuthErr(err) {
		quotes, err = yc.doFetch(symbols, true)
	}
	return quotes, err
}

func (yc *YahooClient) doFetch(symbols []string, forceRefresh bool) ([]StockQuote, error) {
	yc.mu.Lock()
	if yc.crumb == "" || forceRefresh {
		if err := yc.initCrumb(); err != nil {
			yc.mu.Unlock()
			return nil, fmt.Errorf("Yahoo Finance auth: %w", err)
		}
	}
	crumb := yc.crumb
	yc.mu.Unlock()

	apiURL := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v7/finance/quote?symbols=%s&crumb=%s",
		url.QueryEscape(strings.Join(symbols, ",")),
		url.QueryEscape(crumb),
	)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Accept", "application/json")

	resp, err := yc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		yc.mu.Lock()
		yc.crumb = ""
		yc.mu.Unlock()
		return nil, authErr("session expired")
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limited — wait a moment")
	case http.StatusOK:
		// fall through
	default:
		return nil, fmt.Errorf("Yahoo Finance: %s", resp.Status)
	}

	return parseYFQuotes(resp.Body)
}

type yahooErr struct{ msg string }

func (e yahooErr) Error() string { return e.msg }

func authErr(msg string) error       { return yahooErr{msg} }
func isAuthErr(err error) bool       { _, ok := err.(yahooErr); return ok }

type yfQuoteResponse struct {
	QuoteResponse struct {
		Result []struct {
			Symbol                     string  `json:"symbol"`
			ShortName                  string  `json:"shortName"`
			RegularMarketPrice         float64 `json:"regularMarketPrice"`
			RegularMarketChange        float64 `json:"regularMarketChange"`
			RegularMarketChangePercent float64 `json:"regularMarketChangePercent"`
		} `json:"result"`
	} `json:"quoteResponse"`
}

func parseYFQuotes(r io.Reader) ([]StockQuote, error) {
	var result yfQuoteResponse
	if err := json.NewDecoder(r).Decode(&result); err != nil {
		return nil, err
	}
	quotes := make([]StockQuote, 0, len(result.QuoteResponse.Result))
	for _, res := range result.QuoteResponse.Result {
		quotes = append(quotes, StockQuote{
			Symbol:    res.Symbol,
			Name:      res.ShortName,
			Price:     res.RegularMarketPrice,
			Change:    res.RegularMarketChange,
			ChangePct: res.RegularMarketChangePercent,
		})
	}
	return quotes, nil
}
