package suggest

import (
	"strings"
	"testing"

	"github.com/startvibecoding/agentui/ansi"
)

func TestSuggestFilterAndSelection(t *testing.T) {
	m := New(30).SetItems([]Item{
		{Label: "/mode", Value: "/mode "},
		{Label: "/model", Value: "/model "},
		{Label: "/help", Value: "/help"},
	})
	m = m.Update("/mo")
	if !m.Visible() {
		t.Fatal("suggestions should be visible")
	}
	item, ok := m.Selected()
	if !ok || item.Label != "/mode" {
		t.Fatalf("selected = %#v, %v", item, ok)
	}
	m = m.CursorDown()
	item, _ = m.Selected()
	if item.Label != "/model" {
		t.Fatalf("selected after down = %#v", item)
	}
	if !strings.Contains(m.View(), "/model") {
		t.Fatalf("view missing /model:\n%s", m.View())
	}
}

func TestSuggestBareSlashShowsCommands(t *testing.T) {
	m := New(30).SetItems([]Item{
		{Label: "/mode", Value: "/mode "},
		{Label: "/model", Value: "/model "},
	})
	m = m.Update("/")
	if !m.Visible() {
		t.Fatal("suggestions should be visible for bare slash")
	}
	item, ok := m.Selected()
	if !ok || !strings.HasPrefix(item.Label, "/") {
		t.Fatalf("selected = %#v, %v; want slash command", item, ok)
	}
}

func TestSuggestMatchesValuePrefix(t *testing.T) {
	m := New(30).SetItems([]Item{
		{Label: "Plan mode", Value: "/mode plan"},
		{Label: "Agent mode", Value: "/mode agent"},
	})
	m = m.Update("/mode a")
	item, ok := m.Selected()
	if !ok || item.Value != "/mode agent" {
		t.Fatalf("selected = %#v, %v; want /mode agent", item, ok)
	}
}

func TestSuggestCursorClampsAfterRefilter(t *testing.T) {
	m := New(30).SetItems([]Item{
		{Label: "/mode", Value: "/mode "},
		{Label: "/model", Value: "/model "},
		{Label: "/help", Value: "/help"},
	})
	m = m.Update("/m").CursorDown().CursorDown()
	m = m.Update("/h")
	item, ok := m.Selected()
	if !ok || item.Label != "/help" {
		t.Fatalf("selected after refilter = %#v, %v; want /help", item, ok)
	}
}

func TestSuggestViewRespectsWidthWithCJKDescription(t *testing.T) {
	m := New(18).SetItems([]Item{
		{Label: "/agents", Description: "用户查看 AGENTS 文件内容", Value: "/agents"},
	})
	m = m.Update("/a")
	view := m.View()
	for _, line := range strings.Split(view, "\n") {
		if width := ansi.StringWidth(line); width > 18 {
			t.Fatalf("line width = %d, want <= 18: %q\nview:\n%s", width, line, view)
		}
	}
	if !strings.Contains(view, "...") {
		t.Fatalf("view should show truncation suffix:\n%s", view)
	}
}

func TestSuggestViewShowsMoreIndicatorForLongLists(t *testing.T) {
	items := []Item{
		{Label: "/one", Value: "/one"},
		{Label: "/two", Value: "/two"},
		{Label: "/three", Value: "/three"},
		{Label: "/four", Value: "/four"},
	}
	m := New(24).SetMaxVisible(2).SetItems(items)
	m = m.Update("/")
	view := m.View()
	if !strings.Contains(view, "more") {
		t.Fatalf("long suggestion list missing more indicator:\n%s", view)
	}
	if strings.Contains(view, "/four") {
		t.Fatalf("initial visible window should be limited:\n%s", view)
	}
}
