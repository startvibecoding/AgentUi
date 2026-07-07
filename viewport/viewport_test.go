package viewport

import (
	"strings"
	"testing"

	"github.com/startvibecoding/agentui"
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
