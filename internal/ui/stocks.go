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

type fetchStocksMsg struct{}
type gotStocksMsg struct {
	quotes []api.StockQuote
	err    error
}

type stocksPane struct {
	yc       *api.YahooClient
	symbols  []string
	quotes   []api.StockQuote
	loading  bool
	err      string
	lastSync time.Time
	viewport viewport.Model
	width    int
	height   int
	focused  bool
}

func newStocksPane(yc *api.YahooClient, symbols []string) stocksPane {
	return stocksPane{yc: yc, symbols: symbols, loading: true}
}

func (p stocksPane) Init() tea.Cmd {
	return func() tea.Msg { return fetchStocksMsg{} }
}

func doFetchStocks(yc *api.YahooClient, symbols []string) tea.Cmd {
	return func() tea.Msg {
		quotes, err := yc.FetchQuotes(symbols)
		return gotStocksMsg{quotes: quotes, err: err}
	}
}

func (p stocksPane) Update(msg tea.Msg) (stocksPane, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case fetchStocksMsg:
		p.loading = true
		cmds = append(cmds, doFetchStocks(p.yc, p.symbols))
	case gotStocksMsg:
		p.loading = false
		if msg.err != nil {
			p.err = msg.err.Error()
		} else {
			p.err = ""
			p.quotes = msg.quotes
			p.lastSync = time.Now()
		}
		p.viewport.SetContent(p.renderContent())
	case tea.KeyMsg:
		if p.focused && msg.String() == "r" {
			p.loading = true
			cmds = append(cmds, doFetchStocks(p.yc, p.symbols))
		}
	}
	var vpCmd tea.Cmd
	p.viewport, vpCmd = p.viewport.Update(msg)
	if vpCmd != nil {
		cmds = append(cmds, vpCmd)
	}
	return p, tea.Batch(cmds...)
}

func (p *stocksPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.viewport.Width = w - 4
	p.viewport.Height = h - 4
	p.viewport.SetContent(p.renderContent())
}

func (p *stocksPane) SetFocused(f bool) { p.focused = f }

func (p stocksPane) renderContent() string {
	if p.width == 0 {
		return ""
	}
	if p.loading {
		return dimStyle.Render("  Fetching quotes...")
	}
	if p.err != "" {
		return errStyle.Render("  Error: " + truncate(p.err, p.width-6))
	}

	// Fixed column widths (excluding name).
	// Layout: 2(indent) + symW + 1 + priceW + 1 + chgW + 1 + pctW = 2+6+1+8+1+8+1+7 = 34
	const symW, priceW, chgW, pctW = 6, 8, 8, 7
	const fixedW = 2 + symW + 1 + priceW + 1 + chgW + 1 + pctW

	innerW := p.width - 4 // lipgloss border removes 2 chars each side
	nameW := innerW - fixedW - 1 // -1 for space before name col
	showName := nameW >= 5

	var sb strings.Builder

	// Header
	sep := dimStyle.Render("  " + strings.Repeat("─", innerW-2))
	if showName {
		hdr := fmt.Sprintf("  %-*s %-*s %*s %*s %*s",
			symW, "SYM", nameW, "NAME", priceW, "PRICE", chgW, "CHG", pctW, "%CHG")
		sb.WriteString(dimStyle.Render(hdr) + "\n")
	} else {
		hdr := fmt.Sprintf("  %-*s %*s %*s %*s",
			symW, "SYM", priceW, "PRICE", chgW, "CHG", pctW, "%CHG")
		sb.WriteString(dimStyle.Render(hdr) + "\n")
	}
	sb.WriteString(sep + "\n")

	for _, q := range p.quotes {
		priceStr := fmt.Sprintf("%.2f", q.Price)
		chgStr := fmt.Sprintf("%+.2f", q.Change)
		pctStr := fmt.Sprintf("%+.2f%%", q.ChangePct)

		color := colorPos
		if q.Change < 0 {
			color = colorNeg
		}
		chgStyled := lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%*s", chgW, chgStr))
		pctStyled := lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("%-*s", pctW, pctStr))

		var line string
		if showName {
			name := truncate(q.Name, nameW)
			line = fmt.Sprintf("  %-*s %-*s %*s %s %s",
				symW, q.Symbol, nameW, name, priceW, priceStr, chgStyled, pctStyled)
		} else {
			line = fmt.Sprintf("  %-*s %*s %s %s",
				symW, q.Symbol, priceW, priceStr, chgStyled, pctStyled)
		}
		sb.WriteString(line + "\n")
	}

	if !p.lastSync.IsZero() {
		sb.WriteString("\n" + dimStyle.Render(fmt.Sprintf("  Updated: %s", p.lastSync.Format("3:04 PM"))))
	}
	return sb.String()
}

func (p stocksPane) View() string {
	accentStyle := lipgloss.NewStyle().Foreground(colorStocks).Bold(true)
	title := accentStyle.Render("STOCKS")
	sep := dimStyle.Render(strings.Repeat("─", p.width-4))

	inner := lipgloss.JoinVertical(lipgloss.Left, title, sep, p.viewport.View())
	return paneStyle(colorStocks, p.focused, p.width, p.height).Render(inner)
}
