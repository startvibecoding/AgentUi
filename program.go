package agentui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type programConfig struct {
	input          io.Reader
	output         io.Writer
	inputTTY       bool
	reportFocus    bool
	mouse          bool
	altScreen      bool
	renderer       bool
	bracketedPaste bool
	escTimeout     time.Duration
}

// ProgramOption configures a Program.
type ProgramOption func(*programConfig)

// WithInput sets the source used by the input reader. Passing nil disables
// input reading, which is useful in tests that drive the program with Send.
func WithInput(r io.Reader) ProgramOption {
	return func(c *programConfig) { c.input = r }
}

// WithOutput sets the render destination.
func WithOutput(w io.Writer) ProgramOption {
	return func(c *programConfig) {
		if w != nil {
			c.output = w
		}
	}
}

// WithInputTTY requests raw terminal input when the input is an *os.File.
func WithInputTTY() ProgramOption {
	return func(c *programConfig) { c.inputTTY = true }
}

// WithReportFocus enables terminal focus reporting when supported.
func WithReportFocus() ProgramOption {
	return func(c *programConfig) { c.reportFocus = true }
}

// WithMouse enables SGR mouse tracking.
func WithMouse() ProgramOption {
	return func(c *programConfig) { c.mouse = true }
}

// WithAltScreen renders in the terminal alternate screen.
func WithAltScreen() ProgramOption {
	return func(c *programConfig) { c.altScreen = true }
}

// WithoutBracketedPaste disables terminal bracketed-paste mode.
func WithoutBracketedPaste() ProgramOption {
	return func(c *programConfig) { c.bracketedPaste = false }
}

// WithEscTimeout sets how long the input reader waits before treating a lone
// escape byte as KeyEsc instead of the start of a longer escape sequence.
// Passing a non-positive duration disables the timeout.
func WithEscTimeout(d time.Duration) ProgramOption {
	return func(c *programConfig) { c.escTimeout = d }
}

// WithoutRenderer disables automatic View rendering.
func WithoutRenderer() ProgramOption {
	return func(c *programConfig) { c.renderer = false }
}

// Program runs a Model and serializes messages onto one update loop.
type Program struct {
	model Model
	cfg   programConfig

	msgs chan Msg
	done chan struct{}

	mu      sync.Mutex
	running bool
	closed  bool
	last    string
}

// NewProgram creates a Program for m.
func NewProgram(m Model, opts ...ProgramOption) *Program {
	cfg := programConfig{
		input:          os.Stdin,
		output:         os.Stdout,
		inputTTY:       true,
		renderer:       true,
		bracketedPaste: true,
		escTimeout:     50 * time.Millisecond,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &Program{
		model: m,
		cfg:   cfg,
		msgs:  make(chan Msg, 256),
		done:  make(chan struct{}),
	}
}

// Send posts msg to the update loop. It is safe to call from goroutines.
func (p *Program) Send(msg Msg) {
	if msg == nil {
		return
	}
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()
	if closed {
		return
	}
	select {
	case p.msgs <- msg:
	case <-p.done:
	}
}

// Println writes a line outside the managed render tree.
func (p *Program) Println(s string) {
	if p.cfg.output == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprint(p.cfg.output, "\x1b[2K\r")
	fmt.Fprint(p.cfg.output, terminalLineEndings(s))
	fmt.Fprint(p.cfg.output, "\r\n")
	if p.cfg.renderer && p.last != "" {
		p.renderLocked(p.last)
	}
}

// Run starts the program and blocks until QuitMsg is received or input closes.
func (p *Program) Run() (Model, error) {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return p.model, fmt.Errorf("program already running")
	}
	p.running = true
	p.mu.Unlock()

	var term *terminalState
	if f, ok := p.cfg.input.(*os.File); ok && p.cfg.inputTTY {
		t, err := makeRaw(f)
		if err == nil {
			term = t
			defer term.restore()
		}
	}
	p.enterTerminalModes()
	defer p.exitTerminalModes()

	if p.cfg.input != nil {
		go p.readInput()
	}
	if f, ok := p.cfg.output.(*os.File); ok {
		if w, h, ok := terminalSize(f); ok {
			p.Send(WindowSizeMsg{Width: w, Height: h})
		}
		go p.watchResize(f)
	}

	if cmd := p.model.Init(); cmd != nil {
		p.exec(cmd)
	}
	p.render()

	for {
		select {
		case msg := <-p.msgs:
			switch msg := msg.(type) {
			case nil:
				continue
			case QuitMsg:
				p.close()
				return p.model, nil
			case batchMsg:
				for _, cmd := range msg {
					p.exec(cmd)
				}
				continue
			case repaintMsg:
				p.render()
				continue
			}

			next, cmd := p.model.Update(msg)
			if next != nil {
				p.model = next
			}
			p.render()
			if cmd != nil {
				p.exec(cmd)
			}
		case <-p.done:
			return p.model, nil
		}
	}
}

func (p *Program) exec(cmd Cmd) {
	if cmd == nil {
		return
	}
	go func() {
		msg := cmd()
		switch m := msg.(type) {
		case nil:
			return
		case batchMsg:
			for _, cmd := range m {
				p.exec(cmd)
			}
		case sequenceMsg:
			p.execSequence(m)
		default:
			p.Send(msg)
		}
	}()
}

func (p *Program) execSequence(cmds []Cmd) {
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		select {
		case <-p.done:
			return
		default:
		}
		switch msg := cmd().(type) {
		case nil:
			continue
		case batchMsg:
			for _, cmd := range msg {
				p.exec(cmd)
			}
		case sequenceMsg:
			p.execSequence(msg)
		case QuitMsg:
			p.Send(msg)
			return
		default:
			p.Send(msg)
		}
	}
}

func (p *Program) render() {
	if !p.cfg.renderer || p.cfg.output == nil || p.model == nil {
		return
	}
	view := p.model.View()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.last = view
	p.renderLocked(view)
}

func (p *Program) renderLocked(view string) {
	if p.cfg.output == nil {
		return
	}
	fmt.Fprint(p.cfg.output, "\x1b[?25l\x1b[H\x1b[2J")
	fmt.Fprint(p.cfg.output, terminalLineEndings(view))
	fmt.Fprint(p.cfg.output, "\x1b[0m")
}

func terminalLineEndings(s string) string {
	if !strings.Contains(s, "\n") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + strings.Count(s, "\n"))
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if i == 0 || s[i-1] != '\r' {
				b.WriteByte('\r')
			}
			b.WriteByte('\n')
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func (p *Program) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	close(p.done)
}

func (p *Program) enterTerminalModes() {
	if p.cfg.output == nil {
		return
	}
	if p.cfg.altScreen {
		fmt.Fprint(p.cfg.output, "\x1b[?1049h")
	}
	if p.cfg.reportFocus {
		fmt.Fprint(p.cfg.output, "\x1b[?1004h")
	}
	if p.cfg.mouse {
		fmt.Fprint(p.cfg.output, "\x1b[?1000h\x1b[?1006h")
	}
	if p.cfg.bracketedPaste {
		fmt.Fprint(p.cfg.output, "\x1b[?2004h")
	}
}

func (p *Program) exitTerminalModes() {
	if p.cfg.output == nil {
		return
	}
	if p.cfg.bracketedPaste {
		fmt.Fprint(p.cfg.output, "\x1b[?2004l")
	}
	if p.cfg.mouse {
		fmt.Fprint(p.cfg.output, "\x1b[?1006l\x1b[?1000l")
	}
	if p.cfg.reportFocus {
		fmt.Fprint(p.cfg.output, "\x1b[?1004l")
	}
	fmt.Fprint(p.cfg.output, "\x1b[?25h\x1b[0m")
	if p.cfg.altScreen {
		fmt.Fprint(p.cfg.output, "\x1b[?1049l")
	}
}
