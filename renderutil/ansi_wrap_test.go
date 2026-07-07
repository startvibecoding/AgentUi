package renderutil

import (
	"fmt"
	"strings"
	"testing"
)

func TestWrapANSIPreservesEscapeSequences(t *testing.T) {
	input := "\x1b[31mabcdefghij\x1b[0m"
	got := WrapANSI(input, 4)
	plain := strings.ReplaceAll(StripANSI(got), "\n", "")
	if plain != "abcdefghij" {
		t.Fatalf("wrapped visible text = %q, want %q\nraw: %q", plain, "abcdefghij", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if width := VisibleWidth(line); width > 4 {
			t.Fatalf("line width = %d, want <= 4: %q", width, line)
		}
	}
}

func TestWrapANSIPreservesStyledVisibleText(t *testing.T) {
	got := WrapANSI("\x1b[31mred green blue\x1b[0m", 5)
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected wrapped output, got %q", got)
	}
	for _, line := range lines {
		if width := VisibleWidth(line); width > 5 {
			t.Fatalf("line width = %d, want <= 5: %q\nall: %q", width, line, got)
		}
	}
	if plain := strings.ReplaceAll(StripANSI(got), "\n", " "); !strings.Contains(plain, "red green blue") {
		t.Fatalf("wrapped visible text changed: %q", plain)
	}
}

func TestWrapANSIPreservesOSC8HyperlinkVisibleText(t *testing.T) {
	open := "\x1b]8;;https://example.com\x1b\\"
	close := "\x1b]8;;\x1b\\"
	got := WrapANSI(open+"abcdefghijkl"+close, 4)
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected wrapped hyperlink output, got %q", got)
	}
	for _, line := range lines {
		if width := VisibleWidth(line); width > 4 {
			t.Fatalf("line width = %d, want <= 4: %q\nall: %q", width, line, got)
		}
	}
	plain := strings.ReplaceAll(StripANSI(got), "\n", "")
	if plain != "abcdefghijkl" {
		t.Fatalf("wrapped visible text = %q, want %q\nraw: %q", plain, "abcdefghijkl", got)
	}
}

func TestWrapANSIHandlesCJKWidth(t *testing.T) {
	got := WrapANSI("修复终端渲染宽度", 6)
	for _, line := range strings.Split(got, "\n") {
		if width := VisibleWidth(line); width > 6 {
			t.Fatalf("line width = %d, want <= 6: %q", width, line)
		}
	}
	if plain := strings.ReplaceAll(got, "\n", ""); plain != "修复终端渲染宽度" {
		t.Fatalf("wrapped visible text changed: %q", plain)
	}
}

func TestWrapPlainTextPreservesMixedCJKASCIIOrder(t *testing.T) {
	tests := []string{
		"用户查看想 AGENTS 文件内容",
		"用户查看想 AGENTS内容",
	}
	for _, input := range tests {
		for _, width := range []int{7, 10, 13, 16, 20} {
			t.Run(fmt.Sprintf("%s/%d", input, width), func(t *testing.T) {
				got := WrapPlainText(input, width)
				flattened := strings.Join(strings.Fields(got), "")
				want := strings.Join(strings.Fields(input), "")
				if flattened != want {
					t.Fatalf("wrapped text order changed:\n got %q\nwant %q\nraw: %q", flattened, want, got)
				}
				if strings.Contains(flattened, "AG文件ENTS") {
					t.Fatalf("wrapped text interleaved CJK and ASCII token: %q\nraw: %q", flattened, got)
				}
				for _, line := range strings.Split(got, "\n") {
					if lineWidth := VisibleWidth(line); lineWidth > width {
						t.Fatalf("line width = %d, want <= %d: %q\nraw: %q", lineWidth, width, line, got)
					}
				}
			})
		}
	}
}

func TestWrapANSIDropsOversizedTrailingWhitespace(t *testing.T) {
	input := "\x1b[38;5;252m用户读取要求 AGENTS.md 文件\x1b[0m" + strings.Repeat(" ", 1000)
	got := WrapANSI(input, 20)
	lines := strings.Split(got, "\n")
	for i, line := range lines {
		if isANSIBlankLine(line) {
			t.Fatalf("line %d is visually blank after wrapping trailing padding:\n%q", i+1, got)
		}
		if width := VisibleWidth(line); width > 20 {
			t.Fatalf("line %d width = %d, want <= 20: %q", i+1, width, line)
		}
	}
	plain := strings.ReplaceAll(StripANSI(got), "\n", "")
	if strings.Join(strings.Fields(plain), "") != "用户读取要求AGENTS.md文件" {
		t.Fatalf("wrapped visible text = %q, want original content", plain)
	}
}

func TestWrapANSIBreaksPathTokensAtSlashBoundary(t *testing.T) {
	got := WrapANSI("\x1b[31minternal/agent/\x1b[0m", 9)
	plainLines := strings.Split(StripANSI(got), "\n")
	for _, line := range strings.Split(got, "\n") {
		if width := VisibleWidth(line); width > 9 {
			t.Fatalf("line width = %d, want <= 9: %q\nraw: %q", width, line, got)
		}
	}
	if flat := strings.Join(plainLines, ""); flat != "internal/agent/" {
		t.Fatalf("wrapped path order = %q, want internal/agent/", flat)
	}
}
