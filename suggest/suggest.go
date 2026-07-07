package suggest

import (
	"strings"

	"github.com/startvibecoding/agentui/ansi"
	"github.com/startvibecoding/agentui/style"
)

// Item is one autocomplete suggestion.
type Item struct {
	Label       string
	Description string
	Value       string
}

// Model renders a small prefix-filtered suggestion dropdown.
type Model struct {
	items         []Item
	filtered      []Item
	cursor        int
	maxVisible    int
	visible       bool
	query         string
	width         int
	style         style.Style
	selectedStyle style.Style
}

func New(width int) Model {
	return Model{
		width:         width,
		maxVisible:    8,
		style:         style.New().Foreground(style.Color("240")),
		selectedStyle: style.New().Foreground(style.Color("86")).Bold(true),
	}
}

func (m Model) SetItems(items []Item) Model {
	m.items = append([]Item(nil), items...)
	return m.filter()
}

func (m Model) SetWidth(width int) Model {
	m.width = width
	return m
}

func (m Model) SetMaxVisible(n int) Model {
	if n > 0 {
		m.maxVisible = n
	}
	return m
}

func (m Model) Update(query string) Model {
	m.query = query
	return m.filter()
}

func (m Model) Visible() bool { return m.visible }

func (m Model) Selected() (Item, bool) {
	if len(m.filtered) == 0 || m.cursor < 0 || m.cursor >= len(m.filtered) {
		return Item{}, false
	}
	return m.filtered[m.cursor], true
}

func (m Model) CursorUp() Model {
	if len(m.filtered) == 0 {
		return m
	}
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.filtered) - 1
	}
	return m
}

func (m Model) CursorDown() Model {
	if len(m.filtered) == 0 {
		return m
	}
	m.cursor++
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
	}
	return m
}

func (m Model) View() string {
	if !m.visible || len(m.filtered) == 0 {
		return ""
	}
	contentWidth := m.width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}
	start, end := visibleRange(m.cursor, len(m.filtered), m.maxVisible)
	var lines []string
	for i := start; i < end; i++ {
		lines = append(lines, m.renderItem(m.filtered[i], i == m.cursor, contentWidth))
	}
	if len(m.filtered) > m.maxVisible {
		indicator := "  ↑↓ more"
		lines = append(lines, m.style.Render(padRight(indicator, contentWidth)))
	}
	return style.New().Border(style.RoundedBorder()).Width(contentWidth + 2).Render(strings.Join(lines, "\n"))
}

func (m Model) renderItem(item Item, selected bool, maxWidth int) string {
	line := item.Label
	if item.Description != "" {
		line += " " + item.Description
	}
	line = padRight(ansi.Truncate(line, maxWidth, "..."), maxWidth)
	if selected {
		return m.selectedStyle.Render(line)
	}
	return m.style.Render(line)
}

func (m Model) filter() Model {
	q := strings.ToLower(m.query)
	if q == "" {
		m.filtered = m.items
		m.visible = false
		m.cursor = clampCursor(m.cursor, len(m.filtered))
		return m
	}
	m.filtered = m.filtered[:0]
	for _, item := range m.items {
		if strings.HasPrefix(strings.ToLower(item.Label), q) || strings.HasPrefix(strings.ToLower(item.Value), q) {
			m.filtered = append(m.filtered, item)
		}
	}
	m.visible = len(m.filtered) > 0
	m.cursor = clampCursor(m.cursor, len(m.filtered))
	return m
}

func visibleRange(cursor, total, limit int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if limit <= 0 || total <= limit {
		return 0, total
	}
	start := cursor - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > total {
		start = total - limit
	}
	return start, start + limit
}

func clampCursor(cursor, total int) int {
	if total <= 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= total {
		return total - 1
	}
	return cursor
}

func padRight(s string, width int) string {
	w := ansi.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
