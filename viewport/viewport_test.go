package viewport

import (
	"strings"
	"testing"

	"github.com/startvibecoding/agentui"
	"github.com/startvibecoding/agentui/ansi"
)

func TestViewportFollowBottomAndScroll(t *testing.T) {
	m := New(5, 2)
	m.SetContent("1\n2\n3\n4")
	if !strings.Contains(m.View(), "3") || !strings.Contains(m.View(), "4") {
		t.Fatalf("view should follow bottom:\n%s", m.View())
	}
	m, _ = m.Update(agentui.KeyMsg{Type: agentui.KeyPgUp})
	if m.AtBottom() {
		t.Fatal("after PgUp viewport should not be at bottom")
	}
	if !strings.Contains(m.View(), "1") {
		t.Fatalf("view should show top after PgUp:\n%s", m.View())
	}
	m, _ = m.Update(agentui.KeyMsg{Type: agentui.KeyEnd})
	if !m.AtBottom() {
		t.Fatal("End should return to bottom")
	}
}

func TestViewportPadsHeight(t *testing.T) {
	m := New(4, 3)
	m.SetContent("x")
	lines := strings.Split(m.View(), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines = %d, want 3: %q", len(lines), m.View())
	}
}

func TestViewportWrapsLongLines(t *testing.T) {
	m := New(10, 4)
	m.SetContent("用户查看 AGENTS 文件内容")
	view := m.View()
	plain := strings.Join(strings.Fields(ansi.Strip(view)), "")
	if !strings.Contains(plain, "用户查看AGENTS文件内容") {
		t.Fatalf("viewport lost or reordered wrapped text: %q\nraw:\n%s", plain, view)
	}
	for _, line := range strings.Split(view, "\n") {
		if width := ansi.StringWidth(line); width > 10 {
			t.Fatalf("line width = %d, want <= 10: %q\nraw:\n%s", width, line, view)
		}
	}
}

func TestViewportWrapsStyledLongLines(t *testing.T) {
	m := New(10, 5)
	m.SetContent("\x1b[3m用户查看 AGENTS 文件内容\x1b[0m")
	view := m.View()
	plain := strings.Join(strings.Fields(ansi.Strip(view)), "")
	if !strings.Contains(plain, "用户查看AGENTS文件内容") {
		t.Fatalf("viewport lost or reordered styled wrapped text: %q\nraw:\n%s", plain, view)
	}
	for _, line := range strings.Split(view, "\n") {
		if width := ansi.StringWidth(line); width > 10 {
			t.Fatalf("line width = %d, want <= 10: %q\nraw:\n%s", width, line, view)
		}
	}
}

func TestViewportCanDisableWrap(t *testing.T) {
	m := New(10, 2).SetWrap(false)
	m.SetContent("用户查看 AGENTS 文件内容")
	view := m.View()
	plain := strings.Join(strings.Fields(ansi.Strip(view)), "")
	if strings.Contains(plain, "文件内容") {
		t.Fatalf("viewport should truncate when wrap is disabled:\n%s", view)
	}
}

func TestViewportSetItemsSeparatesBlocksAndCountsItems(t *testing.T) {
	m := New(12, 5).SetItems([]string{"one", "two\nthree"})
	if got := m.ItemCount(); got != 2 {
		t.Fatalf("ItemCount() = %d, want 2", got)
	}
	lines := strings.Split(m.View(), "\n")
	want := []string{"one", "", "two", "three"}
	for i, text := range want {
		if got := strings.TrimSpace(ansi.Strip(lines[i])); got != text {
			t.Fatalf("line %d = %q, want %q\nview:\n%s", i, got, text, m.View())
		}
	}
}

func TestViewportAppendItemAutoFollowsBottom(t *testing.T) {
	m := New(8, 3).SetItems([]string{"one", "two"})
	m = m.AppendItem("three")
	if !m.AtBottom() {
		t.Fatal("AppendItem should keep following bottom")
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "three") {
		t.Fatalf("view should include appended item:\n%s", view)
	}
}

func TestViewportAppendItemPreservesScrolledPosition(t *testing.T) {
	m := New(8, 3).SetItems([]string{"one", "two", "three", "four"})
	m, _ = m.Update(agentui.KeyMsg{Type: agentui.KeyPgUp})
	before := m.Offset()
	m = m.AppendItem("five")
	if m.AtBottom() {
		t.Fatal("AppendItem should not force bottom after user scrolled up")
	}
	if got := m.Offset(); got != before {
		t.Fatalf("offset = %d, want preserved %d", got, before)
	}
	view := ansi.Strip(m.View())
	if strings.Contains(view, "five") {
		t.Fatalf("scrolled-up viewport should not jump to appended item:\n%s", view)
	}
}

func TestViewportMouseWheelScrollsItemContent(t *testing.T) {
	m := New(8, 3).SetItems([]string{"one", "two", "three", "four"})
	m.GotoTop()
	m, _ = m.Update(agentui.MouseMsg{Action: agentui.MouseActionPress, Button: agentui.MouseButtonWheelDown})
	if m.Offset() == 0 {
		t.Fatal("mouse wheel down should increase offset")
	}
	m, _ = m.Update(agentui.MouseMsg{Action: agentui.MouseActionPress, Button: agentui.MouseButtonWheelUp})
	if got := m.Offset(); got != 0 {
		t.Fatalf("mouse wheel up offset = %d, want 0", got)
	}
}

func TestViewportSetSizeClampsOffset(t *testing.T) {
	m := New(8, 2).SetItems([]string{"one", "two", "three", "four"})
	m.GotoTop()
	m, _ = m.Update(agentui.KeyMsg{Type: agentui.KeyPgDown})
	if m.Offset() == 0 {
		t.Fatal("setup expected non-zero offset")
	}
	m = m.SetSize(8, 20)
	if got := m.Offset(); got != 0 {
		t.Fatalf("offset = %d, want clamped to 0", got)
	}
}

func TestViewportItemsCanUseTruncationMode(t *testing.T) {
	m := New(10, 2).SetWrap(false).SetItems([]string{"用户查看 AGENTS 文件内容"})
	view := m.View()
	plain := strings.Join(strings.Fields(ansi.Strip(view)), "")
	if strings.Contains(plain, "文件内容") {
		t.Fatalf("viewport should truncate item line when wrap is disabled:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if width := ansi.StringWidth(line); width > 10 {
			t.Fatalf("line width = %d, want <= 10: %q\nraw:\n%s", width, line, view)
		}
	}
}
