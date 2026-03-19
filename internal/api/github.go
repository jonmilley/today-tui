package api

import (
	"encoding/json"
	"fmt"
	"net/http"
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
}

type GitHubClient struct {
	token  string
	repo   string
	client *http.Client
}

func NewGitHubClient(token, repo string) *GitHubClient {
	return &GitHubClient{
		token:  token,
		repo:   repo,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *GitHubClient) GetOpenIssues() ([]Issue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues?state=open&per_page=50", c.repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "today-tui/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API: %s", resp.Status)
	}
	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, err
	}
	return issues, nil
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
