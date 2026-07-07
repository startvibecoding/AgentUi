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
	liveHeight     int
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

// WithLiveHeight sets the maximum number of rows managed by the normal-screen
// live renderer. When unset, Program uses the terminal height when it can read
// one. Overflowing top rows are printed to native terminal scrollback.
func WithLiveHeight(h int) ProgramOption {
	return func(c *programConfig) {
		if h < 0 {
			h = 0
		}
		c.liveHeight = h
	}
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
	liveH   int
	termW   int
	termH   int
	autoTop int
	autoLog []string
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
	p.Print(s, true)
}

// Print writes text outside the managed render tree. If newline is true, one
// terminal newline is appended after s.
func (p *Program) Print(s string, newline bool) {
	if p.cfg.output == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.printLocked(s, newline)
}

func (p *Program) printLocked(s string, newline bool) {
	p.printBatchLocked([]printMsg{{text: s, newline: newline}}, true)
}

func (p *Program) printBatchLocked(msgs []printMsg, repaint bool) {
	p.clearLiveLocked()
	for _, msg := range msgs {
		fmt.Fprint(p.cfg.output, terminalLineEndings(msg.text))
		if msg.newline {
			fmt.Fprint(p.cfg.output, "\r\n")
		}
	}
	p.liveH = 0
	if repaint && p.cfg.renderer && p.last != "" {
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
	var initialSize *WindowSizeMsg
	if f, ok := p.cfg.output.(*os.File); ok {
		if w, h, ok := terminalSize(f); ok {
			p.setTerminalSize(w, h)
			initialSize = &WindowSizeMsg{Width: w, Height: h}
		}
		go p.watchResize(f)
	}

	if cmd := p.model.Init(); cmd != nil {
		p.exec(cmd)
	}
	if initialSize != nil {
		next, cmd := p.model.Update(*initialSize)
		if next != nil {
			p.model = next
		}
		if cmd != nil {
			p.exec(cmd)
		}
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
			case printMsg:
				p.Print(msg.text, msg.newline)
				continue
			case printBatchMsg:
				p.printBatch(msg)
				continue
			}
			if msg, ok := msg.(WindowSizeMsg); ok {
				p.setTerminalSize(msg.Width, msg.Height)
			}

			next, cmd := p.model.Update(msg)
			if next != nil {
				p.model = next
			}
			var pending Msg
			if cmd != nil {
				if msg, ok := p.startCmdAndWait(cmd, time.Millisecond); ok {
					if p.handlePrintBeforeRender(msg) {
						p.render()
						continue
					}
					pending = msg
				}
			}
			p.render()
			if pending != nil {
				p.forwardCmdMsg(pending)
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
		p.forwardCmdMsg(msg)
	}()
}

func (p *Program) startCmdAndWait(cmd Cmd, wait time.Duration) (Msg, bool) {
	if cmd == nil {
		return nil, true
	}
	ch := make(chan Msg, 1)
	go func() { ch <- cmd() }()
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case msg := <-ch:
		return msg, true
	case <-timer.C:
		go func() { p.forwardCmdMsg(<-ch) }()
		return nil, false
	}
}

func (p *Program) forwardCmdMsg(msg Msg) {
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
}

func (p *Program) handlePrintBeforeRender(msg Msg) bool {
	switch m := msg.(type) {
	case nil:
		return false
	case printMsg:
		p.printWithoutRepaint([]printMsg{m})
		return true
	case printBatchMsg:
		p.printWithoutRepaint(m)
		return true
	default:
		return false
	}
}

func (p *Program) printWithoutRepaint(msgs []printMsg) {
	if p.cfg.output == nil || len(msgs) == 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.printBatchLocked(msgs, false)
}

func (p *Program) execSequence(cmds []Cmd) {
	var prints []printMsg
	flushPrints := func() bool {
		if len(prints) == 0 {
			return true
		}
		batch := make(printBatchMsg, len(prints))
		copy(batch, prints)
		prints = prints[:0]
		select {
		case p.msgs <- batch:
			return true
		case <-p.done:
			return false
		}
	}

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
			if !flushPrints() {
				return
			}
			for _, cmd := range msg {
				p.exec(cmd)
			}
		case sequenceMsg:
			if !flushPrints() {
				return
			}
			p.execSequence(msg)
		case QuitMsg:
			if !flushPrints() {
				return
			}
			p.Send(msg)
			return
		case printMsg:
			prints = append(prints, msg)
		default:
			if !flushPrints() {
				return
			}
			p.Send(msg)
		}
	}
	flushPrints()
}

func (p *Program) printBatch(msgs []printMsg) {
	if p.cfg.output == nil || len(msgs) == 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.printBatchLocked(msgs, true)
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
	if p.cfg.altScreen {
		fmt.Fprint(p.cfg.output, "\x1b[?25l\x1b[H\x1b[2J")
		p.resetAutoScrollback()
	} else {
		fmt.Fprint(p.cfg.output, "\x1b[?25l\x1b[?7l")
		p.clearLiveLocked()
		commit, live := p.splitNormalView(view)
		if commit != "" {
			fmt.Fprint(p.cfg.output, terminalLineEndings(commit))
			fmt.Fprint(p.cfg.output, "\r\n")
		}
		view = live
	}
	fmt.Fprint(p.cfg.output, terminalLineEndings(view))
	fmt.Fprint(p.cfg.output, "\x1b[0m")
	if !p.cfg.altScreen {
		fmt.Fprint(p.cfg.output, "\x1b[?7h")
	}
	p.liveH = viewHeight(view)
}

func (p *Program) splitNormalView(view string) (string, string) {
	maxLive := p.normalLiveHeight()
	if maxLive <= 0 || view == "" {
		p.resetAutoScrollback()
		return "", view
	}
	lines := strings.Split(view, "\n")
	if p.autoTop > len(lines) || !sameStringPrefix(lines, p.autoLog) {
		p.resetAutoScrollback()
	}
	targetTop := len(lines) - maxLive
	if targetTop < 0 {
		targetTop = 0
	}
	if targetTop < p.autoTop {
		targetTop = p.autoTop
	}
	var commit string
	if targetTop > p.autoTop {
		commitLines := append([]string(nil), lines[p.autoTop:targetTop]...)
		commit = strings.Join(commitLines, "\n")
		p.autoLog = append(p.autoLog, commitLines...)
		p.autoTop = targetTop
	}
	live := strings.Join(lines[p.autoTop:], "\n")
	return commit, live
}

func (p *Program) resetAutoScrollback() {
	p.autoTop = 0
	p.autoLog = nil
}

func sameStringPrefix(lines, prefix []string) bool {
	if len(prefix) > len(lines) {
		return false
	}
	for i := range prefix {
		if lines[i] != prefix[i] {
			return false
		}
	}
	return true
}

func (p *Program) normalLiveHeight() int {
	if p.cfg.liveHeight > 0 {
		return p.cfg.liveHeight
	}
	if p.termH > 0 {
		return p.termH
	}
	return 0
}

func (p *Program) setTerminalSize(w, h int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.termW = w
	p.termH = h
}

func (p *Program) clearLiveLocked() {
	if p.liveH <= 0 || p.cfg.output == nil {
		return
	}
	if p.liveH > 1 {
		fmt.Fprintf(p.cfg.output, "\x1b[%dA", p.liveH-1)
	}
	for i := 0; i < p.liveH; i++ {
		fmt.Fprint(p.cfg.output, "\r\x1b[2K")
		if i < p.liveH-1 {
			fmt.Fprint(p.cfg.output, "\x1b[1B")
		}
	}
	if p.liveH > 1 {
		fmt.Fprintf(p.cfg.output, "\x1b[%dA", p.liveH-1)
	}
	fmt.Fprint(p.cfg.output, "\r")
	p.liveH = 0
}

func viewHeight(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
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
	fmt.Fprint(p.cfg.output, "\x1b[?7h\x1b[?25h\x1b[0m")
	if p.cfg.altScreen {
		fmt.Fprint(p.cfg.output, "\x1b[?1049l")
	}
}
