package overlay

import (
	"strings"
	"testing"

	"github.com/startvibecoding/agentui"
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
