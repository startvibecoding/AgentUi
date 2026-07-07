package editor

import (
	"strings"
	"unicode"

	"github.com/startvibecoding/agentui/ansi"
)

type buffer struct {
	lines        []string
	cursorLine   int
	cursorCol    int
	preferredCol int
}

func newBuffer() *buffer {
	return &buffer{lines: []string{""}}
}

func (b *buffer) Clone() *buffer {
	if b == nil {
		return newBuffer()
	}
	return &buffer{
		lines:        append([]string(nil), b.lines...),
		cursorLine:   b.cursorLine,
		cursorCol:    b.cursorCol,
		preferredCol: b.preferredCol,
	}
}

func (b *buffer) Value() string {
	return strings.Join(b.lines, "\n")
}

func (b *buffer) SetValue(text string) {
	text = normalizeNewlines(text)
	if text == "" {
		b.lines = []string{""}
	} else {
		b.lines = strings.Split(text, "\n")
	}
	b.MoveEndAll()
}

func (b *buffer) Reset() {
	b.lines = []string{""}
	b.cursorLine = 0
	b.cursorCol = 0
	b.preferredCol = 0
}

func (b *buffer) LineCount() int { return len(b.lines) }

func (b *buffer) RuneCount() int {
	n := 0
	for i, line := range b.lines {
		n += len([]rune(line))
		if i < len(b.lines)-1 {
			n++
		}
	}
	return n
}

func (b *buffer) CursorPos() (int, int) {
	b.clampCursor()
	return b.cursorLine, b.cursorCol
}

func (b *buffer) InsertRune(r rune) {
	b.clampCursor()
	line := []rune(b.lines[b.cursorLine])
	line = append(line[:b.cursorCol], append([]rune{r}, line[b.cursorCol:]...)...)
	b.lines[b.cursorLine] = string(line)
	b.cursorCol++
	b.preferredCol = b.cursorCol
}

func (b *buffer) InsertString(s string) {
	s = normalizeNewlines(s)
	if !strings.Contains(s, "\n") {
		for _, r := range s {
			b.InsertRune(r)
		}
		return
	}
	b.clampCursor()
	line := []rune(b.lines[b.cursorLine])
	before := string(line[:b.cursorCol])
	after := string(line[b.cursorCol:])
	parts := strings.Split(s, "\n")
	newLines := make([]string, 0, len(parts))
	newLines = append(newLines, before+parts[0])
	for i := 1; i < len(parts)-1; i++ {
		newLines = append(newLines, parts[i])
	}
	newLines = append(newLines, parts[len(parts)-1]+after)
	result := make([]string, 0, len(b.lines)+len(newLines)-1)
	result = append(result, b.lines[:b.cursorLine]...)
	result = append(result, newLines...)
	result = append(result, b.lines[b.cursorLine+1:]...)
	b.lines = result
	b.cursorLine += len(newLines) - 1
	b.cursorCol = len([]rune(parts[len(parts)-1]))
	b.preferredCol = b.cursorCol
}

func (b *buffer) InsertNewline() {
	b.clampCursor()
	line := []rune(b.lines[b.cursorLine])
	before := string(line[:b.cursorCol])
	after := string(line[b.cursorCol:])
	result := make([]string, 0, len(b.lines)+1)
	result = append(result, b.lines[:b.cursorLine]...)
	result = append(result, before, after)
	result = append(result, b.lines[b.cursorLine+1:]...)
	b.lines = result
	b.cursorLine++
	b.cursorCol = 0
	b.preferredCol = 0
}

func (b *buffer) DeleteBack() {
	b.clampCursor()
	if b.cursorCol > 0 {
		line := []rune(b.lines[b.cursorLine])
		b.lines[b.cursorLine] = string(append(line[:b.cursorCol-1], line[b.cursorCol:]...))
		b.cursorCol--
		b.preferredCol = b.cursorCol
		return
	}
	if b.cursorLine > 0 {
		prev := b.lines[b.cursorLine-1]
		curr := b.lines[b.cursorLine]
		b.cursorCol = len([]rune(prev))
		b.lines[b.cursorLine-1] = prev + curr
		b.lines = append(b.lines[:b.cursorLine], b.lines[b.cursorLine+1:]...)
		b.cursorLine--
		b.preferredCol = b.cursorCol
	}
}

func (b *buffer) DeleteForward() {
	b.clampCursor()
	line := []rune(b.lines[b.cursorLine])
	if b.cursorCol < len(line) {
		b.lines[b.cursorLine] = string(append(line[:b.cursorCol], line[b.cursorCol+1:]...))
		return
	}
	if b.cursorLine < len(b.lines)-1 {
		b.lines[b.cursorLine] += b.lines[b.cursorLine+1]
		b.lines = append(b.lines[:b.cursorLine+1], b.lines[b.cursorLine+2:]...)
	}
}

func (b *buffer) DeleteToLineEnd() {
	b.clampCursor()
	line := []rune(b.lines[b.cursorLine])
	if b.cursorCol < len(line) {
		b.lines[b.cursorLine] = string(line[:b.cursorCol])
	}
}

func (b *buffer) DeleteToLineStart() {
	b.clampCursor()
	line := []rune(b.lines[b.cursorLine])
	if b.cursorCol > 0 {
		b.lines[b.cursorLine] = string(line[b.cursorCol:])
		b.cursorCol = 0
		b.preferredCol = 0
	}
}

func (b *buffer) DeleteWordBack() {
	b.clampCursor()
	if b.cursorCol == 0 {
		b.DeleteBack()
		return
	}
	line := []rune(b.lines[b.cursorLine])
	end := b.cursorCol
	start := end
	for start > 0 && unicode.IsSpace(line[start-1]) {
		start--
	}
	for start > 0 && !unicode.IsSpace(line[start-1]) {
		start--
	}
	next := make([]rune, 0, len(line)-(end-start))
	next = append(next, line[:start]...)
	next = append(next, line[end:]...)
	b.lines[b.cursorLine] = string(next)
	b.cursorCol = start
	b.preferredCol = start
}

func (b *buffer) MoveLeft() {
	b.clampCursor()
	if b.cursorCol > 0 {
		b.cursorCol--
		b.preferredCol = b.cursorCol
		return
	}
	if b.cursorLine > 0 {
		b.cursorLine--
		b.cursorCol = len([]rune(b.lines[b.cursorLine]))
		b.preferredCol = b.cursorCol
	}
}

func (b *buffer) MoveRight() {
	b.clampCursor()
	lineLen := len([]rune(b.lines[b.cursorLine]))
	if b.cursorCol < lineLen {
		b.cursorCol++
		b.preferredCol = b.cursorCol
		return
	}
	if b.cursorLine < len(b.lines)-1 {
		b.cursorLine++
		b.cursorCol = 0
		b.preferredCol = 0
	}
}

func (b *buffer) MoveWordLeft() {
	runes := []rune(b.Value())
	pos := b.absoluteCursor()
	for pos > 0 && unicode.IsSpace(runes[pos-1]) {
		pos--
	}
	for pos > 0 && !unicode.IsSpace(runes[pos-1]) {
		pos--
	}
	b.setAbsoluteCursor(pos)
}

func (b *buffer) MoveWordRight() {
	runes := []rune(b.Value())
	pos := b.absoluteCursor()
	for pos < len(runes) && unicode.IsSpace(runes[pos]) {
		pos++
	}
	for pos < len(runes) && !unicode.IsSpace(runes[pos]) {
		pos++
	}
	b.setAbsoluteCursor(pos)
}

func (b *buffer) MoveUp() bool {
	b.clampCursor()
	if b.cursorLine == 0 {
		return false
	}
	b.cursorLine--
	lineLen := len([]rune(b.lines[b.cursorLine]))
	b.cursorCol = min(b.preferredCol, lineLen)
	return true
}

func (b *buffer) MoveDown() bool {
	b.clampCursor()
	if b.cursorLine >= len(b.lines)-1 {
		return false
	}
	b.cursorLine++
	lineLen := len([]rune(b.lines[b.cursorLine]))
	b.cursorCol = min(b.preferredCol, lineLen)
	return true
}

func (b *buffer) MoveHome() {
	b.cursorCol = 0
	b.preferredCol = 0
}

func (b *buffer) MoveEnd() {
	b.clampCursor()
	b.cursorCol = len([]rune(b.lines[b.cursorLine]))
	b.preferredCol = b.cursorCol
}

func (b *buffer) MoveEndAll() {
	if len(b.lines) == 0 {
		b.lines = []string{""}
	}
	b.cursorLine = len(b.lines) - 1
	b.cursorCol = len([]rune(b.lines[b.cursorLine]))
	b.preferredCol = b.cursorCol
}

func (b *buffer) absoluteCursor() int {
	b.clampCursor()
	pos := 0
	for i := 0; i < b.cursorLine; i++ {
		pos += len([]rune(b.lines[i])) + 1
	}
	return pos + b.cursorCol
}

func (b *buffer) setAbsoluteCursor(pos int) {
	total := b.RuneCount()
	pos = min(max(0, pos), total)
	for i, line := range b.lines {
		lineLen := len([]rune(line))
		if pos <= lineLen {
			b.cursorLine = i
			b.cursorCol = pos
			b.preferredCol = pos
			return
		}
		pos -= lineLen + 1
	}
	b.MoveEndAll()
}

func (b *buffer) cursorDisplayCol() int {
	b.clampCursor()
	line := []rune(b.lines[b.cursorLine])
	return ansi.StringWidth(string(line[:b.cursorCol]))
}

func (b *buffer) clampCursor() {
	if len(b.lines) == 0 {
		b.lines = []string{""}
	}
	b.cursorLine = min(max(0, b.cursorLine), len(b.lines)-1)
	lineLen := len([]rune(b.lines[b.cursorLine]))
	b.cursorCol = min(max(0, b.cursorCol), lineLen)
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
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
