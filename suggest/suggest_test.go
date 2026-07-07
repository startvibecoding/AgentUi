package suggest

import (
	"strings"
	"testing"
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
