# Contributing to today-tui

Thank you for your interest in contributing to `today-tui`! This document provides guidelines and instructions for setting up your development environment and making contributions.

## Development Setup

### Requirements
- **Go 1.24+**
- **golangci-lint** (optional, but highly recommended)
- **Make** (optional, for using the provided Makefile)

### Getting Started
1. Fork and clone the repository.
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Create a local configuration (if you haven't run the app before):
   ```bash
   go run main.go
   ```
   Follow the setup wizard to generate `~/.config/today-tui/config.json`.

## Development Workflow

### Building and Running
To build the project:
```bash
make build
# or
go build -o today .
```

To run during development:
```bash
go run main.go
```

### Running Tests
We aim for high test coverage for logic-heavy components. Always run tests before submitting a PR:
```bash
go test ./...
```

### Linting
We use `golangci-lint` to maintain code quality. Please ensure your changes pass the linter:
```bash
golangci-lint run
```
The configuration is located in `.golangci.yml`.

## Project Architecture

- `main.go`: Entry point, handles dependency injection.
- `internal/api/`: Contains interfaces and clients for external services (GitHub, Weather, Yahoo Finance, etc.).
- `internal/config/`: Configuration loading, saving, and defaults.
- `internal/ui/`: Bubble Tea models and views for the dashboard panes and setup wizard.

### Dependency Injection
All API clients are defined as interfaces in the `internal/api` package. This allows for easy mocking in tests. When adding a new feature or pane:
1. Define a clear interface in `internal/api`.
2. Implement the real client.
3. Inject the interface into the UI components via `ui.Deps` in `main.go`.

## Submitting a Pull Request

1. Create a new branch for your feature or bugfix: `git checkout -b feat/my-new-feature`.
2. Commit your changes with descriptive messages.
3. Ensure tests and linting pass.
4. Push to your fork and submit a Pull Request.
5. Provide a clear description of the changes and link any related issues.

## Code Style
- Follow standard Go idioms and formatting (`go fmt`).
- Ensure exported functions and types are documented with comments.
- Keep UI logic (view) separate from business logic (API clients) where possible.
