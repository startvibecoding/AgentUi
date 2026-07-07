package paste

import (
	"strings"
	"testing"

	"github.com/startvibecoding/agentui"
)

func TestManagerInsertExpandLargePaste(t *testing.T) {
	m := New(2, 20)
	text := "one\ntwo\nthree"
	marker := m.Insert(text)
	if !strings.Contains(marker, "[paste #1 +3 lines]") {
		t.Fatalf("marker = %q", marker)
	}
	if m.Count() != 1 {
		t.Fatalf("count = %d, want 1", m.Count())
	}
	if got := m.Expand("prefix " + marker); got != "prefix "+text {
		t.Fatalf("expanded = %q", got)
	}
	if m.Count() != 0 {
		t.Fatalf("count after expand = %d, want 0", m.Count())
	}
}

func TestManagerInsertSmallPaste(t *testing.T) {
	m := New(5, 500)
	if got := m.Insert("one\r\ntwo"); got != "one\ntwo" {
		t.Fatalf("small paste = %q", got)
	}
}

func TestCoalesceSplitPaste(t *testing.T) {
	keys := []agentui.KeyMsg{
		{Type: agentui.KeyRunes, Runes: []rune("one")},
		{Type: agentui.KeyEnter},
		{Type: agentui.KeyRunes, Runes: []rune("two")},
	}
	got, ok := CoalesceSplit(keys)
	if !ok || got != "one\ntwo" {
		t.Fatalf("CoalesceSplit = %q, %v", got, ok)
	}
}

func TestCoalesceSplitRejectsOrdinaryTyping(t *testing.T) {
	_, ok := CoalesceSplit([]agentui.KeyMsg{{Type: agentui.KeyRunes, Runes: []rune("abc")}})
	if ok {
		t.Fatal("ordinary typing should not coalesce")
	}
}
