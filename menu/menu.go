package menu

import (
	"fmt"
	"strings"

	"github.com/startvibecoding/agentui"
	"github.com/startvibecoding/agentui/ansi"
	"github.com/startvibecoding/agentui/style"
)

// Item is one selectable menu row.
type Item struct {
	Title       string
	Description string
	Value       string
	Marked      bool
}

// Model is a compact modal menu with optional search text managed by callers.
type Model struct {
	Title      string
	Subtitle   string
	Footer     string
	Error      string
	Message    string
	Items      []Item
	Cursor     int
	Width      int
	MaxVisible int
	Open       bool

	Style         style.Style
	MutedStyle    style.Style
	SelectedStyle style.Style
	ErrorStyle    style.Style
}

func New(title string, width int) Model {
	return Model{
		Title:         title,
		Width:         width,
		MaxVisible:    10,
		Open:          true,
		Style:         style.New().Border(style.RoundedBorder()).BorderForeground(style.Color("63")).Padding(1, 2),
		MutedStyle:    style.New().Foreground(style.Color("240")),
		SelectedStyle: style.New().Foreground(style.Color("86")).Bold(true),
		ErrorStyle:    style.New().Foreground(style.Color("196")).Bold(true),
	}
}

func (m Model) Move(delta int) Model {
	if len(m.Items) == 0 {
		return m
	}
	m.Cursor += delta
	if m.Cursor < 0 {
		m.Cursor = len(m.Items) - 1
	}
	if m.Cursor >= len(m.Items) {
		m.Cursor = 0
	}
	return m
}

func (m Model) Selected() (Item, bool) {
	if len(m.Items) == 0 || m.Cursor < 0 || m.Cursor >= len(m.Items) {
		return Item{}, false
	}
	return m.Items[m.Cursor], true
}

func (m Model) Update(msg agentui.Msg) (Model, bool) {
	key, ok := msg.(agentui.KeyMsg)
	if !ok || !m.Open {
		return m, false
	}
	switch key.Type {
	case agentui.KeyUp:
		return m.Move(-1), true
	case agentui.KeyDown:
		return m.Move(1), true
	case agentui.KeyEsc:
		m.Open = false
		return m, true
	}
	if key.MatchesRunes("q") {
		m.Open = false
		return m, true
	}
	return m, false
}

func (m Model) View() string {
	if !m.Open {
		return ""
	}
	width := m.Width
	if width < 40 {
		width = 40
	}
	var lines []string
	if m.Title != "" {
		lines = append(lines, m.Title)
	}
	if m.Subtitle != "" {
		lines = append(lines, m.MutedStyle.Render(ansi.Truncate(m.Subtitle, width-6, "...")))
	}
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	if len(m.Items) == 0 {
		lines = append(lines, m.MutedStyle.Render("No items."))
	} else {
		start, end := visibleRange(m.Cursor, len(m.Items), m.MaxVisible)
		for i := start; i < end; i++ {
			item := m.Items[i]
			cursor := "  "
			rowStyle := style.New()
			if i == m.Cursor {
				cursor = "› "
				rowStyle = m.SelectedStyle
			}
			marker := "  "
			if item.Marked {
				marker = "* "
			}
			line := ansi.Truncate(cursor+marker+item.Title, width-6, "...")
			lines = append(lines, rowStyle.Render(line))
			if item.Description != "" {
				lines = append(lines, m.MutedStyle.Render("  "+ansi.Truncate(item.Description, width-8, "...")))
			}
		}
		if len(m.Items) > m.MaxVisible {
			lines = append(lines, "", m.MutedStyle.Render(fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(m.Items))))
		}
	}
	if m.Message != "" {
		lines = append(lines, "", m.MutedStyle.Render(m.Message))
	}
	if m.Error != "" {
		lines = append(lines, "", m.ErrorStyle.Render(m.Error))
	}
	if m.Footer != "" {
		lines = append(lines, "", m.MutedStyle.Render(m.Footer))
	}
	return m.Style.Width(width).Render(strings.Join(lines, "\n"))
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
