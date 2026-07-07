package ansi

import (
	"strings"
	"testing"
)

func TestStringWidthCJKAndANSI(t *testing.T) {
	s := "\x1b[31m你a\x1b[0m🙂"
	if got := StringWidth(s); got != 5 {
		t.Fatalf("StringWidth(%q) = %d, want 5", s, got)
	}
	if got := Strip(s); got != "你a🙂" {
		t.Fatalf("Strip = %q", got)
	}
}

func TestTruncatePreservesWidth(t *testing.T) {
	got := Truncate("\x1b[31m你好world\x1b[0m", 7, "...")
	if width := StringWidth(got); width > 7 {
		t.Fatalf("width = %d, want <= 7, got %q", width, got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("got %q, want suffix", got)
	}
}

func TestHardwrap(t *testing.T) {
	got := Hardwrap("你好abc", 4, false)
	want := "你好\nabc"
	if got != want {
		t.Fatalf("Hardwrap = %q, want %q", got, want)
	}
}

func TestWrapPreservesMixedCJKASCIIOrder(t *testing.T) {
	tests := []string{
		"用户查看 AGENTS 文件内容",
		"用户查看AGENTS内容",
		"\x1b[3m用户查看 AGENTS 文件内容\x1b[0m",
	}
	for _, input := range tests {
		for _, width := range []int{7, 10, 13, 16, 20} {
			got := Wrap(input, width, "/")
			flattened := strings.Join(strings.Fields(Strip(got)), "")
			want := strings.Join(strings.Fields(Strip(input)), "")
			if flattened != want {
				t.Fatalf("Wrap(%q, %d) order = %q, want %q\nraw: %q", input, width, flattened, want, got)
			}
			for _, line := range strings.Split(got, "\n") {
				if lineWidth := StringWidth(line); lineWidth > width {
					t.Fatalf("line width = %d, want <= %d: %q\nraw: %q", lineWidth, width, line, got)
				}
			}
		}
	}
}

func TestTruncatePreservesMixedCJKASCIIOrder(t *testing.T) {
	input := "\x1b[3m用户查看 AGENTS 文件内容\x1b[0m"
	got := Truncate(input, 15, "")
	plain := Strip(got)
	if !strings.HasPrefix(strings.Join(strings.Fields(plain), ""), "用户查看AGENTS") {
		t.Fatalf("Truncate order = %q from raw %q", plain, got)
	}
	if width := StringWidth(got); width > 15 {
		t.Fatalf("width = %d, want <= 15: %q", width, got)
	}
}
