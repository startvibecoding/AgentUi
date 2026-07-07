package overlay

import (
	"fmt"
	"strings"

	"github.com/startvibecoding/agentui"
	"github.com/startvibecoding/agentui/ansi"
	"github.com/startvibecoding/agentui/style"
)

// Model is a fixed-height scrollable overlay for details, stats, and side
// answers.
type Model struct {
	Title        string
	Status       string
	Lines        []string
	Width        int
	Height       int
	Offset       int
	PinnedBottom bool
	Open         bool
	Style        style.Style
	TitleStyle   style.Style
	StatusStyle  style.Style
}

func New(title string, width, height int) Model {
	return Model{
		Title:        title,
		Width:        width,
		Height:       height,
		PinnedBottom: true,
		Open:         true,
		Style:        style.New().Border(style.RoundedBorder()).BorderForeground(style.Color("63")).Padding(0, 1),
		TitleStyle:   style.New().Foreground(style.Color("86")).Bold(true),
		StatusStyle:  style.New().Foreground(style.Color("240")),
	}
}

func (m Model) SetLines(lines []string) Model {
	m.Lines = append([]string(nil), lines...)
	if m.PinnedBottom {
		m.Offset = m.maxOffset()
	}
	m.clamp()
	return m
}

func (m Model) Update(msg agentui.Msg) (Model, bool) {
	if !m.Open {
		return m, false
	}
	switch msg := msg.(type) {
	case agentui.KeyMsg:
		switch {
		case msg.Type == agentui.KeyEsc || msg.Type == agentui.KeyCtrlO || msg.MatchesRunes("q"):
			m.Open = false
			return m, true
		case msg.Type == agentui.KeyUp:
			m.Scroll(-1)
			return m, true
		case msg.Type == agentui.KeyDown:
			m.Scroll(1)
			return m, true
		case msg.Type == agentui.KeyPgUp:
			m.Scroll(-m.pageSize())
			return m, true
		case msg.Type == agentui.KeyPgDown:
			m.Scroll(m.pageSize())
			return m, true
		case msg.Type == agentui.KeyHome:
			m.Offset = 0
			m.PinnedBottom = false
			return m, true
		case msg.Type == agentui.KeyEnd:
			m.Offset = m.maxOffset()
			m.PinnedBottom = true
			return m, true
		}
	case agentui.MouseMsg:
		if msg.Action != agentui.MouseActionPress {
			return m, false
		}
		switch msg.Button {
		case agentui.MouseButtonWheelUp:
			m.Scroll(-3)
			return m, true
		case agentui.MouseButtonWheelDown:
			m.Scroll(3)
			return m, true
		}
	}
	return m, false
}

func (m *Model) Scroll(delta int) {
	m.Offset += delta
	m.PinnedBottom = false
	m.clamp()
	if m.Offset == m.maxOffset() {
		m.PinnedBottom = true
	}
}

func (m Model) View() string {
	if !m.Open {
		return ""
	}
	width := max(20, m.Width)
	innerWidth := max(1, width-4)
	page := m.pageSize()
	lines := m.Lines
	if len(lines) == 0 {
		lines = []string{" "}
	}
	offset := m.Offset
	if m.PinnedBottom {
		offset = max(0, len(lines)-page)
	}
	end := min(offset+page, len(lines))
	visible := strings.Join(lines[offset:end], "\n")
	title := m.TitleStyle.Render(ansi.Truncate(m.Title, innerWidth, "..."))
	divider := strings.Repeat("─", min(innerWidth, ansi.StringWidth(ansi.Strip(title))))
	status := m.Status
	if status == "" {
		status = fmt.Sprintf("lines %d-%d/%d  Up/Down:scroll  PgUp/PgDn:page  Esc:close", offset+1, end, len(lines))
	}
	content := title + "\n" + divider + "\n" + visible + "\n" + m.StatusStyle.Render(status)
	return m.Style.Width(width).Height(page + 4).Render(content)
}

func (m Model) pageSize() int {
	if m.Height <= 4 {
		return 1
	}
	return m.Height - 4
}

func (m Model) maxOffset() int {
	return max(0, len(m.Lines)-m.pageSize())
}

func (m *Model) clamp() {
	m.Offset = min(max(0, m.Offset), m.maxOffset())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
