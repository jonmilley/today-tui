package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Issue struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
	Labels  []struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"labels"`
	CreatedAt time.Time `json:"created_at"`
	// PullRequest is non-nil when this item is actually a PR. The GitHub
	// /issues endpoint returns issues and PRs together; we filter PRs out.
	PullRequest *struct{} `json:"pull_request,omitempty"`
}

type TodoBackend interface {
	GetOpenIssues() ([]Issue, error)
	CreateIssue(title string) (*Issue, error)
	CloseIssue(number int) error
}

type GitHubClient struct {
	token  string
	repo   string
	client *http.Client
}

var _ TodoBackend = (*GitHubClient)(nil)

func NewGitHubClient(token, repo string) *GitHubClient {
	return &GitHubClient{
		token:  token,
		repo:   repo,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// maxIssuePages caps how many pages of issues we'll fetch (100/page = 500
// open issues max). The TUI list isn't useful past that, and an unbounded
// follow could stall the pane on a very busy repo.
const maxIssuePages = 5

func (c *GitHubClient) GetOpenIssues() ([]Issue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues?state=open&per_page=100", c.repo)
	var all []Issue
	for page := 0; page < maxIssuePages && url != ""; page++ {
		issues, next, err := c.fetchIssuePage(url)
		if err != nil {
			return nil, err
		}
		all = append(all, issues...)
		url = next
	}
	filtered := all[:0]
	for _, iss := range all {
		if iss.PullRequest != nil {
			continue
		}
		filtered = append(filtered, iss)
	}
	return filtered, nil
}

// fetchIssuePage fetches one page of issues and returns its contents plus the
// URL of the next page (empty when there is none).
func (c *GitHubClient) fetchIssuePage(url string) ([]Issue, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "today-tui/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("GitHub API: %s", resp.Status)
	}
	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, "", err
	}
	return issues, parseNextLink(resp.Header.Get("Link")), nil
}

// linkNextRe matches the next-page URL from a GitHub Link header, e.g.
//
//	<https://api.github.com/...?page=2>; rel="next", <...>; rel="last"
var linkNextRe = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

func parseNextLink(header string) string {
	if header == "" {
		return ""
	}
	m := linkNextRe.FindStringSubmatch(header)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func (c *GitHubClient) CreateIssue(title string) (*Issue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues", c.repo)
	body, _ := json.Marshal(map[string]string{"title": title})
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "today-tui/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("GitHub API: %s", resp.Status)
	}
	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

func (c *GitHubClient) CloseIssue(number int) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", c.repo, number)
	req, err := http.NewRequest("PATCH", url, strings.NewReader(`{"state":"closed"}`))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "today-tui/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API: %s", resp.Status)
	}
	return nil
}
