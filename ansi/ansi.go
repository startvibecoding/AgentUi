package ansi

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Strip removes ANSI escape sequences from s.
func Strip(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			if n := escapeLen(s[i:]); n > 0 {
				i += n
				continue
			}
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && n == 1 {
			i++
			continue
		}
		b.WriteRune(r)
		i += n
	}
	return b.String()
}

// StringWidth returns the terminal display width of s in cells. ANSI escape
// sequences are zero-width.
func StringWidth(s string) int {
	w := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			if n := escapeLen(s[i:]); n > 0 {
				i += n
				continue
			}
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && n == 1 {
			i++
			continue
		}
		w += RuneWidth(r)
		i += n
	}
	return w
}

// Height returns the number of terminal rows occupied by s before wrapping.
func Height(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// RuneWidth returns the display width of one rune in terminal cells.
func RuneWidth(r rune) int {
	switch {
	case r == 0:
		return 0
	case r == '\t':
		return 4
	case r == '\n' || r == '\r':
		return 0
	case r < 0x20 || (r >= 0x7f && r < 0xa0):
		return 0
	case unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || unicode.Is(unicode.Mc, r):
		return 0
	case r == 0x200d || (r >= 0xfe00 && r <= 0xfe0f):
		return 0
	case isWideRune(r):
		return 2
	default:
		return 1
	}
}

// Truncate shortens s to maxWidth cells and appends suffix when truncation
// occurs. ANSI escapes are preserved as zero-width bytes.
func Truncate(s string, maxWidth int, suffix string) string {
	if maxWidth <= 0 {
		return ""
	}
	if StringWidth(s) <= maxWidth {
		return s
	}
	suffixW := StringWidth(suffix)
	if suffixW >= maxWidth {
		return truncateCells(suffix, maxWidth)
	}
	return truncateCells(s, maxWidth-suffixW) + suffix
}

func truncateCells(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	var b strings.Builder
	w := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			if n := escapeLen(s[i:]); n > 0 {
				b.WriteString(s[i : i+n])
				i += n
				continue
			}
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && n == 1 {
			i++
			continue
		}
		rw := RuneWidth(r)
		if w+rw > maxWidth {
			break
		}
		b.WriteRune(r)
		w += rw
		i += n
	}
	return b.String()
}

// Hardwrap wraps each input line to width cells. ANSI escapes are preserved.
func Hardwrap(s string, width int, preserveSpace bool) string {
	return wrap(s, width, "", preserveSpace)
}

// Wrap wraps each input line to width cells, preferring breakpoints when
// possible. The breakpoints string contains runes that are allowed break sites.
func Wrap(s string, width int, breakpoints string) string {
	return wrap(s, width, breakpoints, false)
}

func wrap(s string, width int, breakpoints string, preserveSpace bool) string {
	if width <= 0 || s == "" {
		return s
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		out = append(out, wrapLine(line, width, breakpoints, preserveSpace)...)
	}
	return strings.Join(out, "\n")
}

func wrapLine(s string, width int, breakpoints string, preserveSpace bool) []string {
	if StringWidth(s) <= width {
		if preserveSpace {
			return []string{s}
		}
		return []string{trimRightVisibleSpace(s)}
	}

	var lines []string
	var b strings.Builder
	lineW := 0
	lastBreakByte := -1
	lastBreakOut := -1
	raw := s

	for i := 0; i < len(raw); {
		if raw[i] == 0x1b {
			if n := escapeLen(raw[i:]); n > 0 {
				b.WriteString(raw[i : i+n])
				i += n
				continue
			}
		}
		r, n := utf8.DecodeRuneInString(raw[i:])
		if r == utf8.RuneError && n == 1 {
			i++
			continue
		}
		rw := RuneWidth(r)
		if lineW+rw > width && lineW > 0 {
			if lastBreakByte >= 0 && lastBreakOut > 0 {
				line := b.String()[:lastBreakOut]
				if !preserveSpace {
					line = trimRightVisibleSpace(line)
				}
				lines = append(lines, line)
				rest := raw[lastBreakByte:]
				if strings.HasPrefix(rest, string(r)) && strings.ContainsRune(breakpoints, r) {
					rest = raw[lastBreakByte+n:]
				}
				return append(lines, wrapLine(strings.TrimLeft(rest, " \t"), width, breakpoints, preserveSpace)...)
			}
			line := b.String()
			if !preserveSpace {
				line = trimRightVisibleSpace(line)
			}
			lines = append(lines, line)
			b.Reset()
			lineW = 0
			lastBreakByte = -1
			lastBreakOut = -1
			continue
		}
		if strings.ContainsRune(breakpoints, r) || unicode.IsSpace(r) {
			lastBreakByte = i + n
			lastBreakOut = b.Len() + n
		}
		b.WriteRune(r)
		lineW += rw
		i += n
	}
	if b.Len() > 0 {
		line := b.String()
		if !preserveSpace {
			line = trimRightVisibleSpace(line)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func trimRightVisibleSpace(s string) string {
	plain := Strip(s)
	trimmed := strings.TrimRight(plain, " \t")
	if len(trimmed) == len(plain) {
		return s
	}
	return Truncate(s, StringWidth(trimmed), "")
}

func escapeLen(s string) int {
	if len(s) < 2 || s[0] != 0x1b {
		return 0
	}
	switch s[1] {
	case '[':
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}
	case ']':
		for i := 2; i < len(s); i++ {
			if s[i] == 0x07 {
				return i + 1
			}
			if i+1 < len(s) && s[i] == 0x1b && s[i+1] == '\\' {
				return i + 2
			}
		}
	case '(', ')', '*', '+', '-', '.', '/':
		if len(s) >= 3 {
			return 3
		}
	default:
		return 2
	}
	return 0
}

func isWideRune(r rune) bool {
	return (r >= 0x1100 && r <= 0x115f) ||
		(r >= 0x2329 && r <= 0x232a) ||
		(r >= 0x2e80 && r <= 0xa4cf) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f000 && r <= 0x1faff) ||
		(r >= 0x20000 && r <= 0x3fffd)
}
