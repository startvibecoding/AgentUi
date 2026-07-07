package menu

import (
	"strings"
	"testing"

	"github.com/startvibecoding/agentui"
)

func TestMenuMoveAndClose(t *testing.T) {
	m := New("Models", 50)
	m.Items = []Item{{Title: "a"}, {Title: "b"}, {Title: "c"}}
	m, handled := m.Update(agentui.KeyMsg{Type: agentui.KeyDown})
	if !handled || m.Cursor != 1 {
		t.Fatalf("cursor = %d handled=%v", m.Cursor, handled)
	}
	selected, ok := m.Selected()
	if !ok || selected.Title != "b" {
		t.Fatalf("selected = %#v ok=%v", selected, ok)
	}
	m, handled = m.Update(agentui.KeyMsg{Type: agentui.KeyEsc})
	if !handled || m.Open {
		t.Fatalf("menu should close")
	}
}

func TestMenuView(t *testing.T) {
	m := New("Sessions", 60)
	m.Items = []Item{{Title: "abc", Description: "3 msgs", Marked: true}}
	view := m.View()
	if !strings.Contains(view, "Sessions") || !strings.Contains(view, "abc") {
		t.Fatalf("unexpected view:\n%s", view)
	}
}
