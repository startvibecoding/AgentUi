package style

import (
	"strings"
	"testing"

	"github.com/startvibecoding/agentui/ansi"
)

func TestStyleBorderWidth(t *testing.T) {
	out := New().Border(RoundedBorder()).Padding(0, 1).Width(10).Render("你好")
	lines := strings.Split(ansi.Strip(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines = %d, want 3:\n%s", len(lines), out)
	}
	for _, line := range lines {
		if w := ansi.StringWidth(line); w != 10 {
			t.Fatalf("line width = %d, want 10: %q\n%s", w, line, out)
		}
	}
}

func TestJoinHorizontalPadsBlocks(t *testing.T) {
	got := JoinHorizontal("a\nbb", "中")
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(lines))
	}
	if ansi.StringWidth(lines[0]) != ansi.StringWidth(lines[1]) {
		t.Fatalf("widths differ: %q", got)
	}
}
