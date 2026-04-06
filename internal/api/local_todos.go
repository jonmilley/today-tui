package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type localTodoStore struct {
	NextID int     `json:"next_id"`
	Todos  []Issue `json:"todos"`
}

// LocalTodoClient implements TodoBackend using a JSON file on disk.
type LocalTodoClient struct {
	path string
}

var _ TodoBackend = (*LocalTodoClient)(nil)

func NewLocalTodoClient() *LocalTodoClient {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	return &LocalTodoClient{
		path: filepath.Join(home, ".config", "today-tui", "todos.json"),
	}
}

func (c *LocalTodoClient) load() (localTodoStore, error) {
	data, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		return localTodoStore{NextID: 1}, nil
	}
	if err != nil {
		return localTodoStore{}, err
	}
	var store localTodoStore
	if err := json.Unmarshal(data, &store); err != nil {
		return localTodoStore{}, err
	}
	if store.NextID < 1 {
		store.NextID = 1
	}
	return store, nil
}

func (c *LocalTodoClient) save(store localTodoStore) error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0o600)
}

func (c *LocalTodoClient) GetOpenIssues() ([]Issue, error) {
	store, err := c.load()
	if err != nil {
		return nil, err
	}
	return store.Todos, nil
}

func (c *LocalTodoClient) CreateIssue(title string) (*Issue, error) {
	store, err := c.load()
	if err != nil {
		return nil, err
	}
	issue := Issue{
		Number:    store.NextID,
		Title:     title,
		State:     "open",
		CreatedAt: time.Now(),
	}
	store.Todos = append([]Issue{issue}, store.Todos...)
	store.NextID++
	if err := c.save(store); err != nil {
		return nil, err
	}
	return &issue, nil
}

func (c *LocalTodoClient) CloseIssue(number int) error {
	store, err := c.load()
	if err != nil {
		return err
	}
	for i, t := range store.Todos {
		if t.Number == number {
			store.Todos = append(store.Todos[:i], store.Todos[i+1:]...)
			break
		}
	}
	return c.save(store)
}
