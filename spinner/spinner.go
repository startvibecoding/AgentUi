package spinner

import (
	"time"

	"github.com/startvibecoding/agentui"
)

// TickMsg advances a spinner frame.
type TickMsg time.Time

type StartMsg struct{}
type StopMsg struct{}

// Model is a small frame-based spinner.
type Model struct {
	frames   []string
	interval time.Duration
	index    int
	running  bool
}

func New(frames []string, interval time.Duration) Model {
	if len(frames) == 0 {
		frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	}
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	return Model{frames: append([]string(nil), frames...), interval: interval}
}

func Default() Model { return New(nil, 100*time.Millisecond) }

func (m Model) View() string {
	if len(m.frames) == 0 {
		return ""
	}
	return m.frames[m.index%len(m.frames)]
}

func (m Model) Running() bool { return m.running }

func (m Model) Start() agentui.Cmd {
	return func() agentui.Msg { return StartMsg{} }
}

func (m Model) Stop() agentui.Cmd {
	return func() agentui.Msg { return StopMsg{} }
}

func (m Model) Update(msg agentui.Msg) (Model, agentui.Cmd) {
	switch msg.(type) {
	case StartMsg:
		m.running = true
		return m, m.tick()
	case StopMsg:
		m.running = false
		return m, nil
	case TickMsg:
		if m.running {
			m.index = (m.index + 1) % len(m.frames)
			return m, m.tick()
		}
	}
	return m, nil
}

func (m Model) tick() agentui.Cmd {
	return agentui.Tick(m.interval, func(t time.Time) agentui.Msg { return TickMsg(t) })
}
