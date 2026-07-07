package renderutil

import (
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestLooksLikeMarkdownDetectsVibecodingCases(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "plain url prose", in: "see https://example.com/path/for/details", want: false},
		{name: "heading and list", in: "# Summary\n\n- item", want: true},
		{name: "inline code", in: "use `code` here", want: true},
		{name: "single emphasis", in: "1111*22222*3333", want: true},
		{name: "single underscore emphasis", in: "1111_22222_3333", want: true},
		{name: "plain prose", in: "ordinary assistant text", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LooksLikeMarkdown(tt.in); got != tt.want {
				t.Fatalf("LooksLikeMarkdown(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestMarkdownStyleWrapWidthIsBounded(t *testing.T) {
	tests := []struct {
		contentWidth int
		want         int
	}{
		{contentWidth: 1, want: 80},
		{contentWidth: 80, want: 80},
		{contentWidth: 160, want: 160},
		{contentWidth: 300, want: 160},
	}
	for _, tt := range tests {
		if got := MarkdownStyleWrapWidth(tt.contentWidth); got != tt.want {
			t.Fatalf("MarkdownStyleWrapWidth(%d) = %d, want %d", tt.contentWidth, got, tt.want)
		}
	}
}

func TestTrimANSIBlankLinesRemovesStyledEmptyEdges(t *testing.T) {
	input := "\x1b[38;5;252m\x1b[0m\n  \x1b[31mcontent\x1b[0m\n\x1b[38;5;252m   \x1b[0m"
	got := TrimANSIBlankLines(input)
	want := "  \x1b[31mcontent\x1b[0m"
	if got != want {
		t.Fatalf("TrimANSIBlankLines() = %q, want %q", got, want)
	}
}

func TestRenderMarkdownEmphasisPreservesTextOrder(t *testing.T) {
	rendered := RenderMarkdown("用户查看 *AGENTS* 文件内容", 24)
	got := strings.Join(strings.Fields(StripANSI(rendered)), "")
	if got != "用户查看AGENTS文件内容" {
		t.Fatalf("rendered text order = %q, want 用户查看AGENTS文件内容\nraw:\n%s", got, rendered)
	}
}

func TestRenderMarkdownInlineEmphasisStyleScope(t *testing.T) {
	rendered := RenderMarkdown("1111*22222*3333", 80)
	cells := ansiStyleCells(rendered)
	assertItalicSpan(t, cells, "1111", false)
	assertItalicSpan(t, cells, "22222", true)
	assertItalicSpan(t, cells, "3333", false)
}

func TestRenderMarkdownWrapsWithinViewportAfterANSIWrap(t *testing.T) {
	raw := "用户查看 *AGENTS* 文件内容，并打开 `internal/tui/renderutil/ansi_wrap.go` 继续检查折行。"
	rendered := WrapANSI(RenderMarkdown(raw, 24), 24)
	for _, line := range strings.Split(rendered, "\n") {
		if width := VisibleWidth(line); width > 24 {
			t.Fatalf("line width = %d, want <= 24: %q\nraw:\n%s", width, line, rendered)
		}
	}
	plain := strings.Join(strings.Fields(StripANSI(rendered)), "")
	for _, want := range []string{"用户查看AGENTS文件内容", "internal/tui/renderutil/ansi_wrap.go"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("rendered text missing %q in %q\nraw:\n%s", want, plain, rendered)
		}
	}
}

type styledCell struct {
	r      rune
	italic bool
}

func ansiStyleCells(s string) []styledCell {
	var cells []styledCell
	italic := false
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			if n, ok := parseSGR(s[i:], &italic); ok {
				i += n
				continue
			}
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && n == 1 {
			i++
			continue
		}
		cells = append(cells, styledCell{r: r, italic: italic})
		i += n
	}
	return cells
}

func parseSGR(s string, italic *bool) (int, bool) {
	if len(s) < 3 || s[0] != 0x1b || s[1] != '[' {
		return 0, false
	}
	end := strings.IndexByte(s, 'm')
	if end < 0 {
		return 0, false
	}
	body := s[2:end]
	if body == "" {
		*italic = false
		return end + 1, true
	}
	for _, part := range strings.Split(body, ";") {
		code, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		switch code {
		case 0:
			*italic = false
		case 3:
			*italic = true
		case 23:
			*italic = false
		}
	}
	return end + 1, true
}

func assertItalicSpan(t *testing.T, cells []styledCell, text string, want bool) {
	t.Helper()
	runes := []rune(text)
	start := indexStyledRunes(cells, runes)
	if start < 0 {
		t.Fatalf("visible text %q not found in %q", text, styledCellsString(cells))
	}
	for i := range runes {
		if got := cells[start+i].italic; got != want {
			t.Fatalf("italic(%q char %d) = %v, want %v in visible %q", text, i, got, want, styledCellsString(cells))
		}
	}
}

func indexStyledRunes(cells []styledCell, runes []rune) int {
	if len(runes) == 0 || len(runes) > len(cells) {
		return -1
	}
	for i := 0; i+len(runes) <= len(cells); i++ {
		ok := true
		for j := range runes {
			if cells[i+j].r != runes[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func styledCellsString(cells []styledCell) string {
	var b strings.Builder
	for _, cell := range cells {
		b.WriteRune(cell.r)
	}
	return b.String()
}
