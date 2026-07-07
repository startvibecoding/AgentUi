package renderutil

import (
	"strings"

	"github.com/startvibecoding/GoStreamingMarkdown/gsm"
)

const (
	minMarkdownStyleWrap = 80
	maxMarkdownStyleWrap = 160
)

// MarkdownStyleWrapWidth keeps the Markdown renderer from wrapping at tiny
// viewport widths while avoiding huge padded intermediate strings. Final
// viewport wrapping should still be handled by WrapANSI.
func MarkdownStyleWrapWidth(contentWidth int) int {
	if contentWidth < minMarkdownStyleWrap {
		return minMarkdownStyleWrap
	}
	if contentWidth > maxMarkdownStyleWrap {
		return maxMarkdownStyleWrap
	}
	return contentWidth
}

// MarkdownRenderer renders Markdown to terminal ANSI output using
// GoStreamingMarkdown, the same streaming renderer used by vibecoding.
type MarkdownRenderer struct {
	contentWidth int
	stream       *gsm.Stream
}

// NewMarkdownRenderer returns a Markdown renderer configured for the given
// terminal content width.
func NewMarkdownRenderer(contentWidth int) *MarkdownRenderer {
	r := &MarkdownRenderer{}
	r.SetWidth(contentWidth)
	return r
}

// SetWidth updates the terminal content width and recreates the underlying
// stream when the bounded Markdown style width changes.
func (r *MarkdownRenderer) SetWidth(contentWidth int) {
	styleWidth := MarkdownStyleWrapWidth(contentWidth)
	if r.stream != nil && r.contentWidth == contentWidth {
		return
	}
	r.contentWidth = contentWidth
	r.stream = gsm.NewStream(styleWidth, nil)
}

// Render converts raw Markdown to ANSI terminal output and trims visually blank
// leading/trailing lines. Call WrapANSI on the result for the final viewport
// width when contentWidth is smaller than MarkdownStyleWrapWidth.
func (r *MarkdownRenderer) Render(raw string) string {
	if r == nil {
		return ""
	}
	if r.stream == nil {
		r.SetWidth(r.contentWidth)
	}
	r.stream.Update(raw)
	return TrimANSIBlankLines(r.stream.Output())
}

// RenderMarkdown renders raw Markdown once using a renderer sized for
// contentWidth.
func RenderMarkdown(raw string, contentWidth int) string {
	return NewMarkdownRenderer(contentWidth).Render(raw)
}

// TrimANSIBlankLines removes leading and trailing lines that are visually
// blank after ANSI escape sequences are ignored.
func TrimANSIBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	start := 0
	end := len(lines)
	for start < end && isANSIBlankLine(lines[start]) {
		start++
	}
	for end > start && isANSIBlankLine(lines[end-1]) {
		end--
	}
	return strings.Join(lines[start:end], "\n")
}

// LooksLikeMarkdown reports whether s contains Markdown that benefits from a
// terminal renderer. Plain prose stays on the text wrapping path so long URLs
// and ordinary sentences are not reformatted unnecessarily.
func LooksLikeMarkdown(s string) bool {
	if strings.Contains(s, "```") || strings.Contains(s, "~~~") || strings.Contains(s, "`") {
		return true
	}
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case isMarkdownHeading(trimmed):
			return true
		case strings.HasPrefix(trimmed, "> "):
			return true
		case strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ "):
			return true
		case isOrderedMarkdownList(trimmed):
			return true
		case strings.HasPrefix(trimmed, "|") && strings.Count(trimmed, "|") >= 2:
			return true
		case strings.Contains(trimmed, "**") || strings.Contains(trimmed, "__"):
			return true
		case hasMarkdownEmphasis(trimmed):
			return true
		}
	}
	return false
}

func hasMarkdownEmphasis(s string) bool {
	return hasDelimitedMarkdownSpan(s, '*') || hasDelimitedMarkdownSpan(s, '_')
}

func hasDelimitedMarkdownSpan(s string, delim rune) bool {
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] != delim {
			continue
		}
		if i+1 < len(runes) && runes[i+1] == delim {
			i++
			continue
		}
		for j := i + 1; j < len(runes); j++ {
			if runes[j] == delim {
				return j > i+1
			}
		}
	}
	return false
}

func isMarkdownHeading(s string) bool {
	i := 0
	for i < len(s) && s[i] == '#' {
		i++
	}
	return i > 0 && i <= 6 && i < len(s) && isMarkdownSpace(s[i])
}

func isOrderedMarkdownList(s string) bool {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i > 0 && i+1 < len(s) && s[i] == '.' && isMarkdownSpace(s[i+1])
}

func isMarkdownSpace(b byte) bool {
	return b == ' ' || b == '\t'
}
