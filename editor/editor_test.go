package editor

import (
	"strings"
	"testing"

	"github.com/startvibecoding/agentui"
	"github.com/startvibecoding/agentui/ansi"
)

func key(s string) agentui.KeyMsg {
	return agentui.KeyMsg{Type: agentui.KeyRunes, Runes: []rune(s)}
}

func special(t agentui.KeyType) agentui.KeyMsg {
	return agentui.KeyMsg{Type: t}
}

func TestEditorInputAndNewlines(t *testing.T) {
	m := New(40)
	m, _ = m.Update(key("one"))
	m, _ = m.Update(agentui.KeyMsg{Type: agentui.KeyEnter, Alt: true})
	m, _ = m.Update(key("two"))
	m, _ = m.Update(special(agentui.KeyCtrlJ))
	m, _ = m.Update(key("three"))
	if got, want := m.Value(), "one\ntwo\nthree"; got != want {
		t.Fatalf("Value = %q, want %q", got, want)
	}
}

func TestEditorWordMovementAndDelete(t *testing.T) {
	m := New(40).SetValue("one two three")
	m, _ = m.Update(special(agentui.KeyCtrlLeft))
	m, _ = m.Update(special(agentui.KeyCtrlW))
	if got, want := m.Value(), "one three"; got != want {
		t.Fatalf("Value = %q, want %q", got, want)
	}
}

func TestEditorSubmit(t *testing.T) {
	m := New(40)
	_, cmd := m.Update(special(agentui.KeyEnter))
	if cmd == nil {
		t.Fatal("Enter should return submit command")
	}
	if _, ok := cmd().(SubmitMsg); !ok {
		t.Fatalf("cmd msg = %#v, want SubmitMsg", cmd())
	}
}

func TestEditorViewRespectsWidth(t *testing.T) {
	m := New(12).SetValue("你好abcdef")
	view := m.View()
	for _, line := range strings.Split(view, "\n") {
		if w := ansi.StringWidth(line); w > 12 {
			t.Fatalf("line width = %d, want <= 12: %q\n%s", w, line, view)
		}
	}
}
