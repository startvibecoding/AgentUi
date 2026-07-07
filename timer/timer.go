package timer

import (
	"time"

	"github.com/startvibecoding/agentui"
)

// TickMsg is emitted while a timer is running.
type TickMsg time.Time

// StartStopMsg changes running state.
type StartStopMsg struct{ Running bool }

// ResetMsg clears elapsed time.
type ResetMsg struct{}

// Model is a tiny stopwatch-style timer.
type Model struct {
	interval time.Duration
	running  bool
	start    time.Time
	elapsed  time.Duration
}

func NewWithInterval(interval time.Duration) Model {
	if interval <= 0 {
		interval = time.Second
	}
	return Model{interval: interval}
}

func New() Model { return NewWithInterval(time.Second) }

func (m Model) Elapsed() time.Duration {
	if m.running && !m.start.IsZero() {
		return m.elapsed + time.Since(m.start)
	}
	return m.elapsed
}

func (m Model) Running() bool { return m.running }

func (m Model) Start() agentui.Cmd {
	return func() agentui.Msg { return StartStopMsg{Running: true} }
}

func (m Model) Stop() agentui.Cmd {
	return func() agentui.Msg { return StartStopMsg{Running: false} }
}

func (m Model) Reset() agentui.Cmd {
	return func() agentui.Msg { return ResetMsg{} }
}

func (m Model) Update(msg agentui.Msg) (Model, agentui.Cmd) {
	switch msg.(type) {
	case TickMsg:
		if m.running {
			return m, m.tick()
		}
	case StartStopMsg:
		v := msg.(StartStopMsg)
		if v.Running && !m.running {
			m.running = true
			m.start = time.Now()
			return m, m.tick()
		}
		if !v.Running && m.running {
			m.elapsed += time.Since(m.start)
			m.start = time.Time{}
			m.running = false
		}
	case ResetMsg:
		m.elapsed = 0
		if m.running {
			m.start = time.Now()
		} else {
			m.start = time.Time{}
		}
	}
	return m, nil
}

func (m Model) tick() agentui.Cmd {
	return agentui.Tick(m.interval, func(t time.Time) agentui.Msg { return TickMsg(t) })
}
