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
