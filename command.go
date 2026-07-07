package agentui

import "time"

// Cmd performs side work and returns a message for the update loop.
type Cmd func() Msg

type batchMsg []Cmd

// Quit is a command that stops a Program.
var Quit Cmd = func() Msg { return QuitMsg{} }

// Batch runs commands concurrently and forwards each returned message.
func Batch(cmds ...Cmd) Cmd {
	filtered := make([]Cmd, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd != nil {
			filtered = append(filtered, cmd)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return func() Msg { return batchMsg(filtered) }
}

// Tick waits for d and then maps the current time to a message.
func Tick(d time.Duration, fn func(time.Time) Msg) Cmd {
	if fn == nil {
		return nil
	}
	return func() Msg {
		t := time.Now()
		if d > 0 {
			timer := time.NewTimer(d)
			t = <-timer.C
		}
		return fn(t)
	}
}
