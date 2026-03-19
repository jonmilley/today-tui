package ui

import (
	"fmt"
	"strings"
	"time"

	"today-tui/internal/api"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fetchNewsMsg struct{}
type gotNewsMsg struct {
	items []api.NewsItem
	err   error
}

type newsPane struct {
	feedURL   string
	items     []api.NewsItem
	selected  int
	loading   bool
	err       string
	lastSync  time.Time
	viewport  viewport.Model // list
	previewVP viewport.Model // article preview
	previewing bool
	width     int
	height    int
	focused   bool
}

func newNewsPane(feedURL string) newsPane {
	return newsPane{feedURL: feedURL, loading: true}
}

func (p newsPane) Init() tea.Cmd {
	if p.feedURL == "" {
		return nil
	}
	return func() tea.Msg { return fetchNewsMsg{} }
}

func doFetchNews(feedURL string) tea.Cmd {
	return func() tea.Msg {
		items, err := api.FetchNews(feedURL)
		return gotNewsMsg{items: items, err: err}
	}
}

func (p newsPane) Update(msg tea.Msg) (newsPane, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case fetchNewsMsg:
		if p.feedURL == "" {
			break
		}
		p.loading = true
		cmds = append(cmds, doFetchNews(p.feedURL))

	case gotNewsMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err.Error()
		} else {
			p.err = ""
			p.items = msg.items
			p.lastSync = time.Now()
			if p.selected >= len(p.items) {
				p.selected = 0
			}
		}
		p.viewport.SetContent(p.renderList())

	case tea.KeyMsg:
		if !p.focused {
			break
		}
		switch msg.String() {
		case "j", "down":
			if p.selected < len(p.items)-1 {
				p.selected++
				if p.previewing {
					p.openPreview()
				} else {
					p.viewport.SetContent(p.renderList())
					p.ensureVisible()
				}
			}
		case "k", "up":
			if p.selected > 0 {
				p.selected--
				if p.previewing {
					p.openPreview()
				} else {
					p.viewport.SetContent(p.renderList())
					p.ensureVisible()
				}
			}
		case "enter", " ":
			if p.selected < len(p.items) {
				if p.previewing {
					p.closePreview()
				} else {
					p.openPreview()
				}
			}
		case "esc":
			if p.previewing {
				p.closePreview()
			}
		case "o":
			if p.selected < len(p.items) {
				openBrowser(p.items[p.selected].Link)
			}
		case "r":
			if p.feedURL != "" {
				p.loading = true
				cmds = append(cmds, doFetchNews(p.feedURL))
			}
		}
	}

	// Route scroll events to the right viewport.
	var vpCmd tea.Cmd
	if p.previewing {
		p.previewVP, vpCmd = p.previewVP.Update(msg)
	} else {
		p.viewport, vpCmd = p.viewport.Update(msg)
	}
	if vpCmd != nil {
		cmds = append(cmds, vpCmd)
	}
	return p, tea.Batch(cmds...)
}

func (p *newsPane) openPreview() {
	p.previewing = true
	p.updateSizes()
	p.viewport.SetContent(p.renderCompactList())
	p.ensureVisibleCompact()
	if p.selected < len(p.items) {
		p.previewVP.SetContent(p.renderPreviewContent())
		p.previewVP.GotoTop()
	}
}

func (p *newsPane) closePreview() {
	p.previewing = false
	p.updateSizes()
	p.viewport.SetContent(p.renderList())
	p.ensureVisible()
}

func (p *newsPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.viewport.Width = w - 4
	p.previewVP.Width = w - 4
	p.updateSizes()
	if p.previewing {
		p.viewport.SetContent(p.renderCompactList())
		p.previewVP.SetContent(p.renderPreviewContent())
	} else {
		p.viewport.SetContent(p.renderList())
	}
}

func (p *newsPane) updateSizes() {
	if p.height == 0 {
		return
	}
	if p.previewing {
		listH, prevH := p.splitHeights()
		p.viewport.Height = listH
		p.previewVP.Height = prevH
	} else {
		p.viewport.Height = p.height - 5 // border(2)+title(1)+sep(1)+hint(1)
		if p.viewport.Height < 1 {
			p.viewport.Height = 1
		}
	}
}

// splitHeights returns (listLines, previewLines) for split-preview mode.
// overhead: border(2) + title(1) + sep(1) + divider(1) + previewTitle(1) + hint(1) = 7
func (p newsPane) splitHeights() (listH, prevH int) {
	available := p.height - 7
	if available < 4 {
		available = 4
	}
	listH = 5
	if listH > available-2 {
		listH = available - 2
	}
	prevH = available - listH
	if prevH < 2 {
		prevH = 2
	}
	return
}

func (p *newsPane) SetFocused(f bool) {
	p.focused = f
	if !f && p.previewing {
		p.closePreview()
	}
}

// formatNewsTitle renders a news item title with selection/focus highlighting.
func formatNewsTitle(title string, selected, focused bool) string {
	if selected && focused {
		return lipgloss.NewStyle().Foreground(colorNews).Bold(true).Render("▶ " + title)
	}
	if selected {
		return lipgloss.NewStyle().Foreground(colorNews).Render("  " + title)
	}
	return "  " + title
}

// renderList renders the full item list (title + age + blank line per item).
func (p newsPane) renderList() string {
	if p.width == 0 {
		return ""
	}
	if p.feedURL == "" {
		return dimStyle.Render("  No RSS feed configured.\n  Edit ~/.config/today-tui/config.json\n  to set rss_feed_url.")
	}
	if p.loading {
		return dimStyle.Render("  Fetching feed...")
	}
	if p.err != "" {
		return errStyle.Render("  Error: " + truncate(p.err, p.width-6))
	}
	if len(p.items) == 0 {
		return dimStyle.Render("  No items in feed")
	}

	contentWidth := p.width - 6
	var sb strings.Builder
	for i, item := range p.items {
		title := truncate(item.Title, contentWidth-2)
		ageStr := formatAge(time.Since(item.Published))
		sb.WriteString(formatNewsTitle(title, i == p.selected, p.focused) + "\n")
		sb.WriteString(dimStyle.Render(fmt.Sprintf("    %s", ageStr)) + "\n")
		sb.WriteString("\n")
	}
	return sb.String()
}

// renderCompactList renders one line per item (used in split-preview mode).
func (p newsPane) renderCompactList() string {
	if len(p.items) == 0 {
		return ""
	}
	contentWidth := p.width - 6
	var sb strings.Builder
	for i, item := range p.items {
		title := truncate(item.Title, contentWidth-2)
		sb.WriteString(formatNewsTitle(title, i == p.selected, p.focused) + "\n")
	}
	return sb.String()
}

// renderPreviewContent formats the selected item's description for the preview viewport.
func (p newsPane) renderPreviewContent() string {
	if p.selected >= len(p.items) {
		return ""
	}
	item := p.items[p.selected]

	titleStyle := lipgloss.NewStyle().Foreground(colorNews).Bold(true).Width(p.width - 6)
	metaStyle := dimStyle.Width(p.width - 6)
	bodyStyle := lipgloss.NewStyle().Width(p.width - 6)

	heading := titleStyle.Render(item.Title)
	meta := metaStyle.Render(fmt.Sprintf("%s  ·  %s", item.Source, formatAge(time.Since(item.Published))))

	body := item.Description
	if body == "" {
		body = dimStyle.Render("  (no description available — press 'o' to open in browser)")
	} else {
		body = bodyStyle.Render(body)
	}

	return lipgloss.JoinVertical(lipgloss.Left, heading, meta, "", body)
}

func (p *newsPane) ensureVisible() {
	lineH := 3 // title + age + blank
	targetY := p.selected * lineH
	if targetY < p.viewport.YOffset {
		p.viewport.SetYOffset(targetY)
	} else if targetY >= p.viewport.YOffset+p.viewport.Height {
		p.viewport.SetYOffset(targetY - p.viewport.Height + lineH)
	}
}

func (p *newsPane) ensureVisibleCompact() {
	targetY := p.selected
	if targetY < p.viewport.YOffset {
		p.viewport.SetYOffset(targetY)
	} else if targetY >= p.viewport.YOffset+p.viewport.Height {
		p.viewport.SetYOffset(targetY - p.viewport.Height + 1)
	}
}

func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func (p newsPane) View() string {
	accentStyle := lipgloss.NewStyle().Foreground(colorNews).Bold(true)
	paneTitle := accentStyle.Render("NEWS")
	topSep := dimStyle.Render(strings.Repeat("─", p.width-4))

	var hint string
	if p.focused {
		if p.previewing {
			hint = dimStyle.Render("  j/k: nav  Enter/Esc: close  o: browser  r: refresh")
		} else {
			hint = dimStyle.Render("  j/k: nav  Enter: preview  o: browser  r: refresh")
		}
	}

	var inner string
	if p.previewing {
		divider := dimStyle.Render(strings.Repeat("─", p.width-4))
		inner = lipgloss.JoinVertical(lipgloss.Left,
			paneTitle,
			topSep,
			p.viewport.View(),
			divider,
			p.previewVP.View(),
			hint,
		)
	} else {
		inner = lipgloss.JoinVertical(lipgloss.Left,
			paneTitle,
			topSep,
			p.viewport.View(),
			hint,
		)
	}

	return paneStyle(colorNews, p.focused, p.width, p.height).Render(inner)
}
