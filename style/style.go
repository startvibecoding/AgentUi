package style

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/startvibecoding/agentui/ansi"
)

// Color is an ANSI 256-color value expressed as a decimal string.
type Color string

// Border describes border glyphs.
type Border struct {
	Top         string
	Bottom      string
	Left        string
	Right       string
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
}

// RoundedBorder returns a compact rounded border.
func RoundedBorder() Border {
	return Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}
}

// NormalBorder returns a plain square border.
func NormalBorder() Border {
	return Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
	}
}

// Style renders text with a small stdlib-only subset of terminal styling.
type Style struct {
	fg       *Color
	bg       *Color
	borderFg *Color

	bold    bool
	italic  bool
	reverse bool

	paddingTop    int
	paddingRight  int
	paddingBottom int
	paddingLeft   int

	border       bool
	borderTop    bool
	borderGlyphs Border

	width    int
	height   int
	maxWidth int
}

// New creates an empty style.
func New() Style { return Style{} }

func (s Style) Foreground(c Color) Style { s.fg = &c; return s }
func (s Style) Background(c Color) Style { s.bg = &c; return s }
func (s Style) BorderForeground(c Color) Style {
	s.borderFg = &c
	return s
}
func (s Style) Bold(v bool) Style    { s.bold = v; return s }
func (s Style) Italic(v bool) Style  { s.italic = v; return s }
func (s Style) Reverse(v bool) Style { s.reverse = v; return s }

func (s Style) Padding(vertical, horizontal int) Style {
	if vertical < 0 {
		vertical = 0
	}
	if horizontal < 0 {
		horizontal = 0
	}
	s.paddingTop = vertical
	s.paddingBottom = vertical
	s.paddingLeft = horizontal
	s.paddingRight = horizontal
	return s
}

func (s Style) Padding4(top, right, bottom, left int) Style {
	s.paddingTop = max(0, top)
	s.paddingRight = max(0, right)
	s.paddingBottom = max(0, bottom)
	s.paddingLeft = max(0, left)
	return s
}

func (s Style) Border(b Border) Style {
	s.border = true
	s.borderGlyphs = b
	return s
}

func (s Style) BorderTop(v bool) Style { s.borderTop = v; return s }
func (s Style) Width(w int) Style      { s.width = max(0, w); return s }
func (s Style) Height(h int) Style     { s.height = max(0, h); return s }
func (s Style) MaxWidth(w int) Style   { s.maxWidth = max(0, w); return s }

func (s Style) GetHorizontalFrameSize() int {
	n := s.paddingLeft + s.paddingRight
	if s.border {
		n += 2
	}
	return n
}

func (s Style) GetBackground() Color {
	if s.bg == nil {
		return ""
	}
	return *s.bg
}

// Render applies the style to text.
func (s Style) Render(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" && text == "" {
		lines = []string{""}
	}
	if s.maxWidth > 0 {
		for i := range lines {
			lines[i] = ansi.Truncate(lines[i], s.maxWidth, "")
		}
	}
	contentWidth := s.contentWidth(lines)
	lines = s.padLines(lines, contentWidth)
	if s.border || s.borderTop {
		lines = s.renderBorder(lines, contentWidth)
	}
	if s.height > 0 {
		for len(lines) < s.height {
			lines = append(lines, strings.Repeat(" ", ansi.StringWidth(lines[0])))
		}
		if len(lines) > s.height {
			lines = lines[:s.height]
		}
	}
	out := strings.Join(lines, "\n")
	return s.applyANSI(out)
}

func (s Style) contentWidth(lines []string) int {
	w := 0
	for _, line := range lines {
		if lw := ansi.StringWidth(line); lw > w {
			w = lw
		}
	}
	if s.width > 0 {
		frame := s.GetHorizontalFrameSize()
		if s.borderTop && !s.border {
			frame = 0
		}
		if target := s.width - frame; target > w {
			w = target
		}
	}
	if w < 0 {
		w = 0
	}
	return w
}

func (s Style) padLines(lines []string, contentWidth int) []string {
	left := strings.Repeat(" ", s.paddingLeft)
	right := strings.Repeat(" ", s.paddingRight)
	blank := left + strings.Repeat(" ", contentWidth) + right
	out := make([]string, 0, len(lines)+s.paddingTop+s.paddingBottom)
	for i := 0; i < s.paddingTop; i++ {
		out = append(out, blank)
	}
	for _, line := range lines {
		if w := ansi.StringWidth(line); w < contentWidth {
			line += strings.Repeat(" ", contentWidth-w)
		}
		out = append(out, left+line+right)
	}
	for i := 0; i < s.paddingBottom; i++ {
		out = append(out, blank)
	}
	return out
}

func (s Style) renderBorder(lines []string, contentWidth int) []string {
	b := s.borderGlyphs
	if b.Top == "" {
		b = NormalBorder()
	}
	innerW := contentWidth + s.paddingLeft + s.paddingRight
	if innerW < 0 {
		innerW = 0
	}
	top := b.TopLeft + strings.Repeat(b.Top, innerW) + b.TopRight
	if s.borderFg != nil {
		top = ansiCode("38;5", *s.borderFg) + top + "\x1b[0m"
	}
	if s.borderTop && !s.border {
		return append([]string{strings.Repeat(b.Top, innerW)}, lines...)
	}
	out := make([]string, 0, len(lines)+2)
	out = append(out, top)
	for _, line := range lines {
		if w := ansi.StringWidth(line); w < innerW {
			line += strings.Repeat(" ", innerW-w)
		}
		left, right := b.Left, b.Right
		if s.borderFg != nil {
			left = ansiCode("38;5", *s.borderFg) + left + "\x1b[0m"
			right = ansiCode("38;5", *s.borderFg) + right + "\x1b[0m"
		}
		out = append(out, left+line+right)
	}
	bottom := b.BottomLeft + strings.Repeat(b.Bottom, innerW) + b.BottomRight
	if s.borderFg != nil {
		bottom = ansiCode("38;5", *s.borderFg) + bottom + "\x1b[0m"
	}
	out = append(out, bottom)
	return out
}

func (s Style) applyANSI(text string) string {
	var codes []string
	if s.bold {
		codes = append(codes, "1")
	}
	if s.italic {
		codes = append(codes, "3")
	}
	if s.reverse {
		codes = append(codes, "7")
	}
	if s.fg != nil {
		codes = append(codes, colorCode("38;5", *s.fg))
	}
	if s.bg != nil {
		codes = append(codes, colorCode("48;5", *s.bg))
	}
	if len(codes) == 0 {
		return text
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}

func colorCode(prefix string, c Color) string {
	n, err := strconv.Atoi(string(c))
	if err != nil || n < 0 || n > 255 {
		return "0"
	}
	return fmt.Sprintf("%s;%d", prefix, n)
}

func ansiCode(prefix string, c Color) string {
	return "\x1b[" + colorCode(prefix, c) + "m"
}

// Width returns visible terminal width.
func Width(s string) int { return ansi.StringWidth(s) }

// Height returns visible terminal height before wrapping.
func Height(s string) int { return ansi.Height(s) }

// JoinVertical joins blocks top-to-bottom.
func JoinVertical(blocks ...string) string {
	filtered := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block != "" {
			filtered = append(filtered, block)
		}
	}
	return strings.Join(filtered, "\n")
}

// JoinHorizontal joins blocks line-by-line, padding shorter blocks.
func JoinHorizontal(blocks ...string) string {
	if len(blocks) == 0 {
		return ""
	}
	split := make([][]string, len(blocks))
	maxH := 0
	widths := make([]int, len(blocks))
	for i, block := range blocks {
		split[i] = strings.Split(block, "\n")
		if len(split[i]) > maxH {
			maxH = len(split[i])
		}
		for _, line := range split[i] {
			if w := ansi.StringWidth(line); w > widths[i] {
				widths[i] = w
			}
		}
	}
	out := make([]string, maxH)
	for row := 0; row < maxH; row++ {
		var b strings.Builder
		for i := range split {
			line := ""
			if row < len(split[i]) {
				line = split[i][row]
			}
			b.WriteString(line)
			if w := ansi.StringWidth(line); w < widths[i] {
				b.WriteString(strings.Repeat(" ", widths[i]-w))
			}
		}
		out[row] = b.String()
	}
	return strings.Join(out, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
