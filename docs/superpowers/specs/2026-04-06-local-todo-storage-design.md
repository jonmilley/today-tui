# Local Todo Storage Design

**Date:** 2026-04-06  
**Status:** Approved

## Summary

Add a local JSON file as an alternative Todo backend to GitHub Issues. Users choose the backend during the setup wizard. The GitHub Issues backend remains unchanged.

## Architecture

### Interface rename

`api.GitHub` is renamed to `api.TodoBackend`. All three methods stay the same:

```go
type TodoBackend interface {
    GetOpenIssues() ([]Issue, error)
    CreateIssue(title string) (*Issue, error)
    CloseIssue(number int) error
}
```

`GitHubClient` continues to implement `TodoBackend`. A new `LocalTodoClient` also implements it.

### Config changes

`config.Config` gets a new field:

```go
TodoBackend string `json:"todo_backend"` // "github" or "local"
```

`Deps.GitHub` is renamed to `Deps.Todo` (type `api.TodoBackend`).

### LocalTodoClient

New file: `internal/api/local_todos.go`

Stores todos at `~/.config/today-tui/todos.json`:

```json
{
  "next_id": 4,
  "todos": [
    { "number": 1, "title": "Buy milk", "created_at": "2026-04-06T00:00:00Z" }
  ]
}
```

- `GetOpenIssues`: reads file, returns all todos. Missing file treated as empty list.
- `CreateIssue`: appends entry with `next_id`, increments `next_id`, writes file.
- `CloseIssue`: removes entry by number. Unknown number is a no-op.
- `HTMLURL` is always empty; "open in browser" key does nothing for local todos.
- Corrupt file returns the parse error to the pane (displayed as "Error: ...").

## Wizard changes

A new first step is added before the existing GitHub repo step:

> **Todo Backend** — "Choose `github` to use GitHub Issues, or `local` for a local file."

- Choosing `local` skips the GitHub repo and token steps entirely.
- Choosing `github` proceeds through all existing steps unchanged.
- `newWizardFrom` (reconfigure flow) pre-selects the current backend value.

## app.go / todo.go changes

- `Deps.GitHub` renamed to `Deps.Todo`.
- `newTodoPane(deps.GitHub)` becomes `newTodoPane(deps.Todo)`.
- `todoPane.gh` field type changes from `api.GitHub` to `api.TodoBackend`.
- `Deps.Refresh` instantiates either `GitHubClient` or `LocalTodoClient` based on `cfg.TodoBackend`.

## Error handling and edge cases

- Missing todos JSON file on first load: treated as empty list, no error.
- Corrupt JSON file: error surfaced to the pane as "Error: ...".
- `CloseIssue` with unknown number: no-op, no error.
- Local todos have no URL: `Enter` key to open browser does nothing.

## Testing

No new test files. Existing `GitHubClient` tests are unaffected by the interface rename. `LocalTodoClient` logic is simple enough (read/write JSON) that unit tests are not required.
