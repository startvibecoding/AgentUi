package editor

import (
	"strings"

	"github.com/startvibecoding/agentui"
	"github.com/startvibecoding/agentui/ansi"
	"github.com/startvibecoding/agentui/style"
)

// SubmitMsg is returned when Enter submits the current input.
type SubmitMsg struct{}

// Model is a multi-line text editor. Enter submits; Alt+Enter and Ctrl+J
// insert newlines.
type Model struct {
	buf         *buffer
	focus       bool
	cursorOn    bool
	width       int
	maxLines    int
	placeholder string
	prompt      string
	style       style.Style
	cursorStyle style.Style
}

func New(width int) Model {
	return Model{
		buf:         newBuffer(),
		focus:       true,
		cursorOn:    true,
		width:       width,
		maxLines:    5,
		placeholder: "Type a message...",
		style:       style.New().Background(style.Color("236")).Padding(0, 1),
		cursorStyle: style.New().Background(style.Color("236")).Reverse(true),
	}
}

func (m Model) Init() agentui.Cmd { return nil }
func (m Model) Focus() Model      { m.focus = true; m.cursorOn = true; return m }
func (m Model) Blur() Model       { m.focus = false; return m }
func (m Model) Focused() bool     { return m.focus }
func (m Model) Value() string     { return m.buf.Value() }

func (m Model) SetValue(text string) Model {
	m.buf.SetValue(text)
	return m
}

func (m Model) Reset() Model {
	m.buf.Reset()
	return m
}

func (m Model) SetWidth(w int) Model               { m.width = w; return m }
func (m Model) SetMaxLines(n int) Model            { m.maxLines = n; return m }
func (m Model) SetPlaceholder(s string) Model      { m.placeholder = s; return m }
func (m Model) Placeholder() string                { return m.placeholder }
func (m Model) SetPrompt(s string) Model           { m.prompt = s; return m }
func (m Model) SetStyle(s style.Style) Model       { m.style = s; return m }
func (m Model) SetCursorStyle(s style.Style) Model { m.cursorStyle = s; return m }
func (m Model) LineCount() int                     { return m.buf.LineCount() }
func (m Model) CursorPos() (int, int)              { return m.buf.CursorPos() }

func (m Model) CursorEnd() Model {
	m.buf.MoveEndAll()
	return m
}

func (m Model) InsertString(s string) Model {
	m.buf.InsertString(s)
	return m
}

func (m Model) AtFirstLine() bool {
	line, _ := m.buf.CursorPos()
	return line == 0
}

func (m Model) AtLastLine() bool {
	line, _ := m.buf.CursorPos()
	return line >= m.buf.LineCount()-1
}

func (m Model) Update(msg agentui.Msg) (Model, agentui.Cmd) {
	if !m.focus {
		return m, nil
	}
	key, ok := msg.(agentui.KeyMsg)
	if !ok {
		return m, nil
	}
	return m.handleKey(key)
}

func (m Model) handleKey(msg agentui.KeyMsg) (Model, agentui.Cmd) {
	if msg.Type == agentui.KeyEnter && msg.Alt {
		m.buf.InsertNewline()
		return m, nil
	}
	switch msg.Type {
	case agentui.KeyEnter:
		return m, func() agentui.Msg { return SubmitMsg{} }
	case agentui.KeyCtrlJ:
		m.buf.InsertNewline()
	case agentui.KeyBackspace, agentui.KeyCtrlH:
		m.buf.DeleteBack()
	case agentui.KeyDelete:
		m.buf.DeleteForward()
	case agentui.KeyLeft:
		if msg.Alt {
			m.buf.MoveWordLeft()
		} else {
			m.buf.MoveLeft()
		}
	case agentui.KeyRight:
		if msg.Alt {
			m.buf.MoveWordRight()
		} else {
			m.buf.MoveRight()
		}
	case agentui.KeyCtrlLeft:
		m.buf.MoveWordLeft()
	case agentui.KeyCtrlRight:
		m.buf.MoveWordRight()
	case agentui.KeyUp:
		m.buf.MoveUp()
	case agentui.KeyDown:
		m.buf.MoveDown()
	case agentui.KeyHome, agentui.KeyCtrlA:
		m.buf.MoveHome()
	case agentui.KeyEnd, agentui.KeyCtrlE:
		m.buf.MoveEnd()
	case agentui.KeyCtrlK:
		m.buf.DeleteToLineEnd()
	case agentui.KeyCtrlU:
		m.buf.DeleteToLineStart()
	case agentui.KeyCtrlW:
		m.buf.DeleteWordBack()
	case agentui.KeyRunes:
		for _, r := range msg.Runes {
			if r == '\n' {
				m.buf.InsertNewline()
			} else {
				m.buf.InsertRune(r)
			}
		}
	case agentui.KeySpace:
		m.buf.InsertRune(' ')
	case agentui.KeyTab:
		m.buf.InsertString("  ")
	}
	return m, nil
}

func (m Model) View() string {
	promptW := ansi.StringWidth(m.prompt)
	contentW := m.width - m.style.GetHorizontalFrameSize()
	if contentW < 1 {
		contentW = 1
	}
	availW := contentW - promptW
	if availW < 1 {
		availW = 1
	}

	text := m.buf.Value()
	isEmpty := text == ""
	var displayLines []displayLine
	switch {
	case isEmpty && !m.focus:
		displayLines = []displayLine{{text: ""}}
	case isEmpty:
		displayLines = []displayLine{{text: m.renderEmptyLine()}}
	default:
		displayLines = m.buildDisplayLines(availW)
	}

	maxVis := m.maxLines
	if maxVis < 1 {
		maxVis = 1
	}
	cursorDisplayLine := m.cursorDisplayLine(availW)
	start := 0
	if len(displayLines) > maxVis {
		start = cursorDisplayLine - maxVis/2
		if start < 0 {
			start = 0
		}
		if start+maxVis > len(displayLines) {
			start = len(displayLines) - maxVis
		}
	}
	end := min(start+maxVis, len(displayLines))

	cursorLine, cursorCol := m.buf.CursorPos()
	rendered := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		line := displayLines[i].text
		if !isEmpty && m.focus && m.cursorOn && i == cursorDisplayLine {
			line = m.insertCursor(displayLines[i], cursorLine, cursorCol, availW)
		}
		rendered = append(rendered, m.prompt+line)
	}
	return m.style.Width(m.width).Render(strings.Join(rendered, "\n"))
}

func (m Model) renderEmptyLine() string {
	if m.placeholder == "" {
		return m.cursorStyle.Render(" ")
	}
	dim := style.New().Foreground(style.Color("240")).Background(style.Color("236"))
	runes := []rune(m.placeholder)
	if len(runes) == 0 {
		return m.cursorStyle.Render(" ")
	}
	return m.cursorStyle.Render(string(runes[0])) + dim.Render(string(runes[1:]))
}

func (m Model) cursorDisplayLine(availW int) int {
	cursorLine, cursorCol := m.buf.CursorPos()
	lines := m.buildDisplayLines(availW)
	for i, line := range lines {
		if line.bufLine == cursorLine && cursorCol >= line.startCol && cursorCol <= line.endCol {
			return i
		}
	}
	if len(lines) == 0 {
		return 0
	}
	return len(lines) - 1
}

func (m Model) insertCursor(line displayLine, bufLine, bufCol int, maxWidth int) string {
	if line.bufLine != bufLine {
		return line.text
	}
	runes := []rune(line.text)
	pos := bufCol - line.startCol
	pos = min(max(0, pos), len(runes))
	if pos < len(runes) {
		return string(runes[:pos]) + m.cursorStyle.Render(string(runes[pos])) + string(runes[pos+1:])
	}
	if ansi.StringWidth(line.text)+1 > maxWidth {
		return string(runes)
	}
	return string(runes) + m.cursorStyle.Render(" ")
}

type displayLine struct {
	text     string
	bufLine  int
	startCol int
	endCol   int
}

func (m Model) buildDisplayLines(availW int) []displayLine {
	rawLines := strings.Split(m.buf.Value(), "\n")
	lines := make([]displayLine, 0, len(rawLines))
	for i, line := range rawLines {
		lines = append(lines, wrapLineSegments(line, availW, i, 0)...)
	}
	if len(lines) == 0 {
		return []displayLine{{text: ""}}
	}
	return lines
}

func wrapLineSegments(line string, width int, bufLine, startCol int) []displayLine {
	if width <= 0 || ansi.StringWidth(line) <= width {
		return []displayLine{{text: line, bufLine: bufLine, startCol: startCol, endCol: startCol + len([]rune(line))}}
	}
	runes := []rune(line)
	var out []displayLine
	var current []rune
	currentW := 0
	segmentStart := startCol
	for i, r := range runes {
		rw := ansi.RuneWidth(r)
		if currentW > 0 && currentW+rw > width {
			out = append(out, displayLine{text: string(current), bufLine: bufLine, startCol: segmentStart, endCol: startCol + i})
			segmentStart = startCol + i
			current = []rune{r}
			currentW = rw
			continue
		}
		current = append(current, r)
		currentW += rw
	}
	if len(current) > 0 {
		out = append(out, displayLine{text: string(current), bufLine: bufLine, startCol: segmentStart, endCol: startCol + len(runes)})
	}
	return out
}
