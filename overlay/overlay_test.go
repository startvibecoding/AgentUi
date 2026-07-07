package overlay

import (
	"strings"
	"testing"

	"github.com/startvibecoding/agentui"
	"github.com/startvibecoding/agentui/ansi"
	"github.com/startvibecoding/agentui/style"
)

func TestOverlayScrollAndClose(t *testing.T) {
	m := New("Details", 50, 6).SetLines([]string{"1", "2", "3", "4", "5"})
	if !strings.Contains(m.View(), "5") {
		t.Fatalf("overlay should start pinned to bottom:\n%s", m.View())
	}
	m, handled := m.Update(agentui.KeyMsg{Type: agentui.KeyHome})
	if !handled || m.Offset != 0 {
		t.Fatalf("home offset = %d handled=%v", m.Offset, handled)
	}
	m, handled = m.Update(agentui.KeyMsg{Type: agentui.KeyEsc})
	if !handled || m.Open {
		t.Fatalf("overlay should close")
	}
}

func TestOverlayViewUsesExactHeightWithoutCroppingTop(t *testing.T) {
	m := New("Agent details", 50, 8).SetLines([]string{
		"line 1",
		"line 2",
		"line 3",
		"line 4",
		"line 5",
		"line 6",
	})
	view := m.View()
	if got := style.Height(view); got != 8 {
		t.Fatalf("View height = %d, want 8\n%s", got, view)
	}
	plain := ansi.Strip(view)
	if !strings.Contains(plain, "Agent details") {
		t.Fatalf("overlay top was cropped:\n%s", plain)
	}
	if !strings.Contains(plain, "Esc:close") {
		t.Fatalf("overlay status line was cropped:\n%s", plain)
	}
}

func TestOverlayViewRespectsWidthWithCJKTitle(t *testing.T) {
	m := New("用户查看 AGENTS 文件内容", 24, 7).SetLines([]string{"content"})
	view := m.View()
	for _, line := range strings.Split(view, "\n") {
		if width := ansi.StringWidth(line); width > 24 {
			t.Fatalf("line width = %d, want <= 24: %q\nview:\n%s", width, line, view)
		}
	}
}
