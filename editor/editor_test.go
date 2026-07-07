package editor

import (
	"strings"
	"testing"

	"github.com/startvibecoding/agentui"
	"github.com/startvibecoding/agentui/ansi"
	"github.com/startvibecoding/agentui/style"
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

func TestEditorHomeEndKeys(t *testing.T) {
	m := New(40).SetValue("abc")
	m, _ = m.Update(special(agentui.KeyHome))
	m, _ = m.Update(key("X"))
	if got, want := m.Value(), "Xabc"; got != want {
		t.Fatalf("after home insert Value = %q, want %q", got, want)
	}
	m, _ = m.Update(special(agentui.KeyEnd))
	m, _ = m.Update(key("Z"))
	if got, want := m.Value(), "XabcZ"; got != want {
		t.Fatalf("after end insert Value = %q, want %q", got, want)
	}
}

func TestEditorDeleteToLineBoundaries(t *testing.T) {
	m := New(40).SetValue("hello world")
	m, _ = m.Update(agentui.KeyMsg{Type: agentui.KeyCtrlLeft})
	m, _ = m.Update(special(agentui.KeyCtrlK))
	if got, want := m.Value(), "hello "; got != want {
		t.Fatalf("after ctrl+k Value = %q, want %q", got, want)
	}
	m = m.SetValue("hello world")
	m, _ = m.Update(agentui.KeyMsg{Type: agentui.KeyCtrlLeft})
	m, _ = m.Update(special(agentui.KeyCtrlU))
	if got, want := m.Value(), "world"; got != want {
		t.Fatalf("after ctrl+u Value = %q, want %q", got, want)
	}
}

func TestEditorAltArrowMovesByWord(t *testing.T) {
	m := New(40).SetValue("hello world")
	m, _ = m.Update(agentui.KeyMsg{Type: agentui.KeyLeft, Alt: true})
	if line, col := m.CursorPos(); line != 0 || col != 6 {
		t.Fatalf("after alt+left CursorPos = (%d,%d), want (0,6)", line, col)
	}
	m, _ = m.Update(agentui.KeyMsg{Type: agentui.KeyRight, Alt: true})
	if line, col := m.CursorPos(); line != 0 || col != 11 {
		t.Fatalf("after alt+right CursorPos = (%d,%d), want (0,11)", line, col)
	}
}

func TestEditorInsertStringPreservesNewlinesAtCursor(t *testing.T) {
	m := New(40).SetValue("ab")
	m, _ = m.Update(special(agentui.KeyLeft))
	m = m.InsertString("x\ny")
	if got, want := m.Value(), "ax\nyb"; got != want {
		t.Fatalf("Value = %q, want %q", got, want)
	}
}

func TestEditorBlurIgnoresInput(t *testing.T) {
	m := New(40).Blur()
	m, _ = m.Update(key("ignored"))
	if got := m.Value(); got != "" {
		t.Fatalf("blurred editor Value = %q, want empty", got)
	}
	m = m.Focus()
	m, _ = m.Update(key("accepted"))
	if got := m.Value(); got != "accepted" {
		t.Fatalf("focused editor Value = %q, want accepted", got)
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

func TestEditorCopiesDoNotShareBufferMutations(t *testing.T) {
	original := New(40).SetValue("one")
	edited := original.InsertString(" two")
	if got, want := original.Value(), "one"; got != want {
		t.Fatalf("original Value = %q, want %q", got, want)
	}
	if got, want := edited.Value(), "one two"; got != want {
		t.Fatalf("edited Value = %q, want %q", got, want)
	}

	updated, _ := original.Update(key("!"))
	if got, want := original.Value(), "one"; got != want {
		t.Fatalf("original after Update = %q, want %q", got, want)
	}
	if got, want := updated.Value(), "one!"; got != want {
		t.Fatalf("updated Value = %q, want %q", got, want)
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

func TestEditorCursorDisplayLineWithWrappedText(t *testing.T) {
	m := New(3).SetValue("abcd")
	if got := m.cursorDisplayLine(3); got != 1 {
		t.Fatalf("cursorDisplayLine() = %d, want 1", got)
	}
}

func TestEditorPlaceholderDoesNotLeakANSIFragment(t *testing.T) {
	m := New(40).SetPlaceholder("Type a message...")
	view := m.View()
	if strings.Contains(view, "[38;5;240m") {
		t.Fatalf("View() leaked ANSI fragment: %q", view)
	}
}

func TestEditorPlaceholderRespectsWidth(t *testing.T) {
	m := New(18).
		SetPlaceholder("Type here. Try 1111*22222*3333 or paste multiple lines.").
		SetPrompt("> ").
		SetStyle(style.New().Border(style.RoundedBorder()).Padding(0, 1))
	view := m.View()
	for _, line := range strings.Split(view, "\n") {
		if width := ansi.StringWidth(line); width > 18 {
			t.Fatalf("line width = %d, want <= 18: %q\nview:\n%s", width, line, view)
		}
	}
}
