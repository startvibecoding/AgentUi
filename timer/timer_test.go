package timer

import (
	"testing"
	"time"
)

func TestTimerStartStopReset(t *testing.T) {
	m := NewWithInterval(time.Millisecond)
	var cmd any
	m, cmd = m.Update(StartStopMsg{Running: true})
	if cmd == nil || !m.Running() {
		t.Fatalf("timer should start")
	}
	time.Sleep(2 * time.Millisecond)
	if m.Elapsed() <= 0 {
		t.Fatal("elapsed should advance while running")
	}
	m, _ = m.Update(StartStopMsg{Running: false})
	stopped := m.Elapsed()
	time.Sleep(2 * time.Millisecond)
	if m.Elapsed() != stopped {
		t.Fatal("elapsed should not advance after stop")
	}
	m, _ = m.Update(ResetMsg{})
	if m.Elapsed() != 0 {
		t.Fatalf("elapsed after reset = %s", m.Elapsed())
	}
}
