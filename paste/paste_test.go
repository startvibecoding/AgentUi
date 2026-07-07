package paste

import (
	"strings"
	"testing"
	"time"

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

func TestManagerInsertLargeCharPaste(t *testing.T) {
	m := New(20, 10)
	text := strings.Repeat("x", 11)
	marker := m.Insert(text)
	if marker != "[paste #1 11 chars]" {
		t.Fatalf("marker = %q, want char marker", marker)
	}
	if got := m.Expand(marker); got != text {
		t.Fatalf("expanded = %q, want original text", got)
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

func TestCoalesceSplitPasteCtrlJ(t *testing.T) {
	keys := []agentui.KeyMsg{
		{Type: agentui.KeyRunes, Runes: []rune("one")},
		{Type: agentui.KeyCtrlJ},
		{Type: agentui.KeyRunes, Runes: []rune("two")},
	}
	got, ok := CoalesceSplit(keys)
	if !ok || got != "one\ntwo" {
		t.Fatalf("CoalesceSplit = %q, %v", got, ok)
	}
}

func TestCoalesceSplitPasteNormalizesRunesLineEndings(t *testing.T) {
	got, ok := CoalesceSplit([]agentui.KeyMsg{{Type: agentui.KeyRunes, Runes: []rune("one\r\ntwo\rthree")}})
	if !ok || got != "one\ntwo\nthree" {
		t.Fatalf("CoalesceSplit = %q, %v", got, ok)
	}
}

func TestCoalesceSplitPasteIncludesSpacesAfterNewline(t *testing.T) {
	keys := []agentui.KeyMsg{
		{Type: agentui.KeyRunes, Runes: []rune("one")},
		{Type: agentui.KeyEnter},
		{Type: agentui.KeySpace},
		{Type: agentui.KeyRunes, Runes: []rune("two")},
	}
	got, ok := CoalesceSplit(keys)
	if !ok || got != "one\n two" {
		t.Fatalf("CoalesceSplit = %q, %v", got, ok)
	}
}

func TestCoalesceSplitRejectsOrdinaryTyping(t *testing.T) {
	_, ok := CoalesceSplit([]agentui.KeyMsg{{Type: agentui.KeyRunes, Runes: []rune("abc")}})
	if ok {
		t.Fatal("ordinary typing should not coalesce")
	}
}

func TestCoalesceSplitRejectsAltEnterAndNavigation(t *testing.T) {
	tests := [][]agentui.KeyMsg{
		{
			{Type: agentui.KeyRunes, Runes: []rune("one")},
			{Type: agentui.KeyEnter, Alt: true},
			{Type: agentui.KeyRunes, Runes: []rune("two")},
		},
		{
			{Type: agentui.KeyRunes, Runes: []rune("one")},
			{Type: agentui.KeyEnter},
			{Type: agentui.KeyUp},
		},
	}
	for _, keys := range tests {
		if got, ok := CoalesceSplit(keys); ok {
			t.Fatalf("CoalesceSplit(%#v) = %q, true; want rejected", keys, got)
		}
	}
}

func TestHasLineBreakMatchesPasteSemantics(t *testing.T) {
	if !HasLineBreak([]agentui.KeyMsg{{Type: agentui.KeyRunes, Runes: []rune("one\ntwo")}}) {
		t.Fatal("KeyRunes newline should count as line break")
	}
	if !HasLineBreak([]agentui.KeyMsg{{Type: agentui.KeyCtrlJ}}) {
		t.Fatal("Ctrl+J should count as line break")
	}
	if HasLineBreak([]agentui.KeyMsg{{Type: agentui.KeyEnter, Alt: true}}) {
		t.Fatal("Alt+Enter should not count as split paste line break")
	}
}

func TestSplitPasteIdleDelayExtendsMultilineCandidates(t *testing.T) {
	keys := []agentui.KeyMsg{
		{Type: agentui.KeyRunes, Runes: []rune("one")},
		{Type: agentui.KeyEnter},
	}
	got := SplitPasteIdleDelay(keys, time.Millisecond, 120*time.Millisecond)
	if got != 120*time.Millisecond {
		t.Fatalf("SplitPasteIdleDelay = %s, want 120ms", got)
	}
	ordinary := SplitPasteIdleDelay([]agentui.KeyMsg{{Type: agentui.KeyRunes, Runes: []rune("one")}}, time.Millisecond, 120*time.Millisecond)
	if ordinary != time.Millisecond {
		t.Fatalf("ordinary idle delay = %s, want 1ms", ordinary)
	}
}

func TestSplitPasteIdleWaitsPastNormalIdleWindow(t *testing.T) {
	keys := []agentui.KeyMsg{
		{Type: agentui.KeyRunes, Runes: []rune("one")},
		{Type: agentui.KeyEnter},
	}
	last := time.Now()
	if SplitPasteIdle(keys, last, last.Add(2*time.Millisecond), time.Millisecond, 120*time.Millisecond) {
		t.Fatal("split paste candidate became idle after normal input delay")
	}
	if !SplitPasteIdle(keys, last, last.Add(121*time.Millisecond), time.Millisecond, 120*time.Millisecond) {
		t.Fatal("split paste candidate did not become idle after paste delay")
	}
}
