package spinner

import (
	"testing"
	"time"
)

func TestSpinnerAdvances(t *testing.T) {
	m := New([]string{"a", "b"}, time.Millisecond)
	if got := m.View(); got != "a" {
		t.Fatalf("initial frame = %q", got)
	}
	m, _ = m.Update(StartMsg{})
	m, _ = m.Update(TickMsg(time.Now()))
	if got := m.View(); got != "b" {
		t.Fatalf("next frame = %q", got)
	}
	m, _ = m.Update(StopMsg{})
	m, _ = m.Update(TickMsg(time.Now()))
	if got := m.View(); got != "b" {
		t.Fatalf("stopped frame = %q", got)
	}
}
