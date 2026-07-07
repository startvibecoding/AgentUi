package viewport

import (
	"strings"

	"github.com/startvibecoding/agentui"
	"github.com/startvibecoding/agentui/ansi"
	"github.com/startvibecoding/agentui/style"
)

const mouseWheelScrollLines = 3

// Model is a fixed-size scrollable viewport.
type Model struct {
	Width        int
	Height       int
	content      string
	items        []string
	itemMode     bool
	offset       int
	followBottom bool
	wrap         bool
	style        style.Style
}

func New(width, height int) Model {
	return Model{Width: width, Height: height, followBottom: true, wrap: true}
}

func (m Model) SetSize(width, height int) Model {
	m.Width = width
	m.Height = height
	if m.followBottom {
		m.offset = m.maxOffset()
	}
	m.clampOffset()
	return m
}

func (m *Model) SetContent(content string) {
	m.content = content
	m.items = nil
	m.itemMode = false
	if m.followBottom {
		m.offset = m.maxOffset()
	}
	m.clampOffset()
}

func (m Model) Content() string { return m.content }

// SetItems replaces the viewport content with item blocks. Items are separated
// by one blank visual line, matching transcript-style TUI layouts.
func (m Model) SetItems(items []string) Model {
	m.itemMode = true
	m.items = append([]string(nil), items...)
	m.content = joinItems(m.items)
	if m.followBottom {
		m.offset = m.maxOffset()
	}
	m.clampOffset()
	return m
}

// AppendItem appends one item block. If the viewport is following the bottom,
// it keeps the new content visible; otherwise the user's scroll position is
// preserved.
func (m Model) AppendItem(item string) Model {
	if !m.itemMode {
		m.items = nil
		if m.content != "" {
			m.items = append(m.items, m.content)
		}
		m.itemMode = true
	}
	m.items = append(m.items, item)
	m.content = joinItems(m.items)
	if m.followBottom {
		m.offset = m.maxOffset()
	}
	m.clampOffset()
	return m
}

func (m Model) ItemCount() int {
	if !m.itemMode {
		return 0
	}
	return len(m.items)
}

func (m Model) SetWrap(enabled bool) Model {
	m.wrap = enabled
	if m.followBottom {
		m.offset = m.maxOffset()
	}
	m.clampOffset()
	return m
}

func (m *Model) GotoBottom() {
	m.followBottom = true
	m.offset = m.maxOffset()
}

func (m *Model) GotoTop() {
	m.followBottom = false
	m.offset = 0
}

func (m Model) AtBottom() bool { return m.offset >= m.maxOffset() }
func (m Model) Offset() int    { return m.offset }

func (m *Model) PageUp()   { m.scroll(-max(1, m.Height)) }
func (m *Model) PageDown() { m.scroll(max(1, m.Height)) }

func (m Model) SetStyle(s style.Style) Model {
	m.style = s
	return m
}

func (m Model) Update(msg agentui.Msg) (Model, agentui.Cmd) {
	switch msg := msg.(type) {
	case agentui.KeyMsg:
		switch msg.Type {
		case agentui.KeyPgUp:
			m.PageUp()
		case agentui.KeyPgDown:
			m.PageDown()
		case agentui.KeyUp:
			m.scroll(-mouseWheelScrollLines)
		case agentui.KeyDown:
			m.scroll(mouseWheelScrollLines)
		case agentui.KeyHome:
			m.GotoTop()
		case agentui.KeyEnd:
			m.GotoBottom()
		}
	case agentui.MouseMsg:
		if msg.Action != agentui.MouseActionPress {
			break
		}
		switch msg.Button {
		case agentui.MouseButtonWheelUp:
			m.scroll(-mouseWheelScrollLines)
		case agentui.MouseButtonWheelDown:
			m.scroll(mouseWheelScrollLines)
		}
	}
	return m, nil
}

func (m *Model) scroll(delta int) {
	m.offset += delta
	m.followBottom = false
	m.clampOffset()
	if m.AtBottom() {
		m.followBottom = true
	}
}

func (m Model) View() string {
	if m.Width <= 0 || m.Height <= 0 {
		return ""
	}
	lines := m.visualLines()
	offset := m.offset
	if m.followBottom {
		offset = max(0, len(lines)-m.Height)
	}
	end := min(offset+m.Height, len(lines))
	var visible []string
	if offset < len(lines) {
		visible = lines[offset:end]
	}
	out := make([]string, m.Height)
	for i := 0; i < m.Height; i++ {
		if i < len(visible) {
			out[i] = fitLine(visible[i], m.Width)
		} else {
			out[i] = strings.Repeat(" ", m.Width)
		}
	}
	return m.style.Render(strings.Join(out, "\n"))
}

func (m Model) maxOffset() int {
	return max(0, len(m.visualLines())-m.Height)
}

func (m *Model) clampOffset() {
	m.offset = min(max(0, m.offset), m.maxOffset())
}

func (m Model) visualLines() []string {
	raw := m.rawLines()
	if len(raw) == 0 {
		return nil
	}
	if !m.wrap || m.Width <= 0 {
		return raw
	}
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		wrapped := ansi.Wrap(line, m.Width, "/")
		parts := strings.Split(wrapped, "\n")
		if len(parts) == 0 {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, parts...)
	}
	return lines
}

func (m Model) rawLines() []string {
	if m.itemMode {
		if len(m.items) == 0 {
			return nil
		}
		lines := make([]string, 0, len(m.items)*2)
		for i, item := range m.items {
			if i > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, strings.Split(item, "\n")...)
		}
		return lines
	}
	if m.content == "" {
		return nil
	}
	return strings.Split(m.content, "\n")
}

func joinItems(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, "\n\n")
}

func fitLine(line string, width int) string {
	w := ansi.StringWidth(line)
	if w > width {
		return ansi.Truncate(line, width, "")
	}
	return line + strings.Repeat(" ", width-w)
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
