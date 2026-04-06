package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/jonmilley/today-tui/internal/api"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fetchTodosMsg struct{}
type gotTodosMsg struct {
	issues []api.Issue
	err    error
}
type closedIssueMsg struct {
	number int
	err    error
}
type createdIssueMsg struct {
	issue api.Issue
	err   error
}

type todoPane struct {
	gh         api.GitHub
	issues     []api.Issue
	selected   int
	loading    bool
	err        string
	lastSync   time.Time
	viewport   viewport.Model
	width      int
	height     int
	focused    bool
	status     string
	creating   bool
	confirming bool
	confirmNum int
	titleInput textinput.Model
}

func newTodoPane(gh api.GitHub) todoPane {
	ti := textinput.New()
	ti.Placeholder = "Issue title…"
	ti.CharLimit = 256
	return todoPane{gh: gh, loading: true, titleInput: ti}
}

// IsCapturing returns true when the pane owns all keyboard input (create/confirm mode).
func (p todoPane) IsCapturing() bool { return p.creating || p.confirming }

func (p todoPane) Init() tea.Cmd {
	return func() tea.Msg { return fetchTodosMsg{} }
}

func fetchIssues(gh api.GitHub) tea.Cmd {
	return func() tea.Msg {
		issues, err := gh.GetOpenIssues()
		return gotTodosMsg{issues: issues, err: err}
	}
}

func closeIssue(gh api.GitHub, number int) tea.Cmd {
	return func() tea.Msg {
		err := gh.CloseIssue(number)
		return closedIssueMsg{number: number, err: err}
	}
}

func submitCreateIssue(gh api.GitHub, title string) tea.Cmd {
	return func() tea.Msg {
		issue, err := gh.CreateIssue(title)
		if err != nil {
			return createdIssueMsg{err: err}
		}
		return createdIssueMsg{issue: *issue}
	}
}

func (p todoPane) Update(msg tea.Msg) (todoPane, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case fetchTodosMsg:
		p.loading = true
		cmds = append(cmds, fetchIssues(p.gh))

	case gotTodosMsg:
		p, cmd := p.handleGotTodos(msg)
		return p, cmd

	case closedIssueMsg:
		p, cmd := p.handleClosedIssue(msg)
		return p, cmd

	case createdIssueMsg:
		p, cmd := p.handleCreatedIssue(msg)
		return p, cmd

	case tea.KeyMsg:
		if !p.focused {
			break
		}
		p, cmd := p.handleKeyMsg(msg)
		return p, cmd
	}

	var vpCmd tea.Cmd
	p.viewport, vpCmd = p.viewport.Update(msg)
	if vpCmd != nil {
		cmds = append(cmds, vpCmd)
	}
	return p, tea.Batch(cmds...)
}

func (p todoPane) handleGotTodos(msg gotTodosMsg) (todoPane, tea.Cmd) {
	p.loading = false
	if msg.err != nil {
		p.err = msg.err.Error()
	} else {
		p.err = ""
		p.issues = msg.issues
		p.lastSync = time.Now()
		if p.selected >= len(p.issues) {
			p.selected = max(0, len(p.issues)-1)
		}
	}
	p.viewport.SetContent(p.renderContent())
	return p, nil
}

func (p todoPane) handleClosedIssue(msg closedIssueMsg) (todoPane, tea.Cmd) {
	if msg.err != nil {
		p.status = fmt.Sprintf("Error: %s", msg.err)
	} else {
		p.status = fmt.Sprintf("Closed #%d", msg.number)
		for i, iss := range p.issues {
			if iss.Number == msg.number {
				p.issues = append(p.issues[:i], p.issues[i+1:]...)
				break
			}
		}
		if p.selected >= len(p.issues) {
			p.selected = max(0, len(p.issues)-1)
		}
	}
	p.viewport.SetContent(p.renderContent())
	return p, nil
}

func (p todoPane) handleCreatedIssue(msg createdIssueMsg) (todoPane, tea.Cmd) {
	p.creating = false
	p.titleInput.Reset()
	p.titleInput.Blur()
	p.updateViewportHeight()
	if msg.err != nil {
		p.status = "Error: " + msg.err.Error()
	} else {
		p.status = fmt.Sprintf("Created #%d", msg.issue.Number)
		// Optimistically prepend the new issue
		p.issues = append([]api.Issue{msg.issue}, p.issues...)
		p.selected = 0
	}
	p.viewport.SetContent(p.renderContent())
	return p, nil
}

func (p todoPane) handleKeyMsg(msg tea.KeyMsg) (todoPane, tea.Cmd) {
	if p.confirming {
		return p.handleConfirmCloseKey(msg)
	}
	if p.creating {
		return p.handleCreateModeKey(msg)
	}
	return p.handleNavigationKey(msg)
}

func (p todoPane) handleConfirmCloseKey(msg tea.KeyMsg) (todoPane, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg.String() {
	case "y", "Y":
		p.confirming = false
		p.status = "Closing…"
		cmds = append(cmds, closeIssue(p.gh, p.confirmNum))
	case "n", "N", "esc":
		p.confirming = false
		p.status = ""
		p.viewport.SetContent(p.renderContent())
	}
	return p, tea.Batch(cmds...)
}

func (p todoPane) handleCreateModeKey(msg tea.KeyMsg) (todoPane, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg.Type {
	case tea.KeyEnter:
		title := strings.TrimSpace(p.titleInput.Value())
		if title != "" {
			p.status = "Creating…"
			cmds = append(cmds, submitCreateIssue(p.gh, title))
		}
	case tea.KeyEsc:
		p.creating = false
		p.titleInput.Reset()
		p.titleInput.Blur()
		p.status = ""
		p.updateViewportHeight()
		p.viewport.SetContent(p.renderContent())
	default:
		var tiCmd tea.Cmd
		p.titleInput, tiCmd = p.titleInput.Update(msg)
		cmds = append(cmds, tiCmd)
	}
	return p, tea.Batch(cmds...)
}

func (p todoPane) handleNavigationKey(msg tea.KeyMsg) (todoPane, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg.String() {
	case "j", keyDown:
		if p.selected < len(p.issues)-1 {
			p.selected++
			p.viewport.SetContent(p.renderContent())
			p.ensureVisible()
		}
	case "k", keyUp:
		if p.selected > 0 {
			p.selected--
			p.viewport.SetContent(p.renderContent())
			p.ensureVisible()
		}
	case keyEnter:
		if p.selected < len(p.issues) {
			openBrowser(p.issues[p.selected].HTMLURL)
		}
	case "c":
		if p.selected < len(p.issues) {
			p.confirming = true
			p.confirmNum = p.issues[p.selected].Number
			p.status = fmt.Sprintf("Close #%d?", p.confirmNum)
			p.viewport.SetContent(p.renderContent())
		}
	case "n":
		p.creating = true
		p.status = ""
		p.titleInput.Focus()
		p.updateViewportHeight()
		p.viewport.SetContent(p.renderContent())
		cmds = append(cmds, textinput.Blink)
	case "r":
		cmds = append(cmds, func() tea.Msg { return fetchTodosMsg{} })
	}

	var vpCmd tea.Cmd
	p.viewport, vpCmd = p.viewport.Update(msg)
	if vpCmd != nil {
		cmds = append(cmds, vpCmd)
	}
	return p, tea.Batch(cmds...)
}

func (p *todoPane) updateViewportHeight() {
	if p.height == 0 {
		return
	}
	// fixed: border(2) + title(1) + sep(1) = 4
	// normal footer: status(1) + hint(1) = 2
	// create footer: label(1) + input(1) + hint(1) = 3
	bottomLines := 2
	if p.creating {
		bottomLines = 3
	}
	h := p.height - 4 - bottomLines
	if h < 1 {
		h = 1
	}
	p.viewport.Height = h
}

func (p *todoPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.viewport.Width = w - 4
	p.updateViewportHeight()
	p.viewport.SetContent(p.renderContent())
}

func (p *todoPane) SetFocused(f bool) {
	p.focused = f
	if !f {
		if p.creating {
			p.creating = false
			p.titleInput.Reset()
			p.titleInput.Blur()
			p.updateViewportHeight()
		}
		if p.confirming {
			p.confirming = false
			p.status = ""
		}
	}
}

func (p todoPane) renderContent() string {
	if p.loading {
		return dimStyle.Render("  Loading issues…")
	}
	if p.err != "" {
		return errStyle.Render("  Error: " + p.err)
	}
	if len(p.issues) == 0 {
		return dimStyle.Render("  No open issues — press 'n' to create one")
	}

	var sb strings.Builder
	contentWidth := p.width - 6
	for i, iss := range p.issues {
		labels := ""
		for _, l := range iss.Labels {
			labels += " [" + l.Name + "]"
		}
		title := truncate(iss.Title, contentWidth-len(labels)-6)
		line := fmt.Sprintf("  #%-4d %s%s", iss.Number, title, labels)
		line = padRight(line, contentWidth)

		if i == p.selected && p.focused {
			sb.WriteString(lipgloss.NewStyle().
				Foreground(colorTodo).Bold(true).
				Render("▶ "+line[2:]) + "\n")
		} else if i == p.selected {
			sb.WriteString(lipgloss.NewStyle().
				Foreground(colorTodo).
				Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}
	return sb.String()
}

func (p *todoPane) ensureVisible() {
	targetY := p.selected
	if targetY < p.viewport.YOffset {
		p.viewport.SetYOffset(targetY)
	} else if targetY >= p.viewport.YOffset+p.viewport.Height {
		p.viewport.SetYOffset(targetY - p.viewport.Height + 1)
	}
}

func (p todoPane) View() string {
	accentStyle := lipgloss.NewStyle().Foreground(colorTodo).Bold(true)
	title := accentStyle.Render("TODO")
	count := ""
	if !p.loading && p.err == "" {
		count = dimStyle.Render(fmt.Sprintf("  %d open", len(p.issues)))
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top, title, count)
	sep := dimStyle.Render(strings.Repeat("─", p.width-4))

	parts := []string{header, sep, p.viewport.View()}

	if p.creating {
		inputWidth := p.width - 6
		if inputWidth < 10 {
			inputWidth = 10
		}
		p.titleInput.Width = inputWidth
		formLabel := lipgloss.NewStyle().Foreground(colorTodo).Bold(true).Render("  New todo:")
		formInput := "  " + p.titleInput.View()
		formHint := dimStyle.Render("  Enter: create  Esc: cancel")
		parts = append(parts, formLabel, formInput, formHint)
	} else {
		// Status line: show message when present, blank line otherwise (keeps hint stable)
		statusLine := ""
		if p.status != "" {
			statusLine = dimStyle.Render("  " + p.status)
		}
		hint := dimStyle.Render("  n: new  j/k: nav  c: close  Enter: open  r: refresh")
		if p.confirming {
			hint = dimStyle.Render("  y: yes  n/Esc: cancel")
		}
		parts = append(parts, statusLine, hint)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return paneStyle(colorTodo, p.focused, p.width, p.height).Render(inner)
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "linux":
		cmd, args = "xdg-open", []string{url}
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "cmd", []string{"/c", "start", url}
	default:
		return
	}
	exec.Command(cmd, args...).Start() //nolint:errcheck
}
