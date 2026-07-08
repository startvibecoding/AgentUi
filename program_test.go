package agentui

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

type recordModel struct {
	init    Cmd
	view    string
	updates *[]Msg
}

func (m recordModel) Init() Cmd { return m.init }

func (m recordModel) Update(msg Msg) (Model, Cmd) {
	if m.updates != nil {
		*m.updates = append(*m.updates, msg)
	}
	return m, nil
}

func (m recordModel) View() string { return m.view }

type printBeforeRenderModel struct {
	view string
}

func (m printBeforeRenderModel) Init() Cmd {
	return Sequence(
		func() Msg { return "submit" },
		Tick(10*time.Millisecond, func(time.Time) Msg { return QuitMsg{} }),
	)
}

func (m printBeforeRenderModel) Update(msg Msg) (Model, Cmd) {
	if msg == "submit" {
		m.view = "new live"
		return m, Println("completed before render")
	}
	return m, nil
}

func (m printBeforeRenderModel) View() string {
	if m.view == "" {
		return "old live"
	}
	return m.view
}

func TestProgramStopsWhenInputCloses(t *testing.T) {
	p := NewProgram(
		recordModel{},
		WithInput(strings.NewReader("")),
		WithOutput(io.Discard),
		WithoutRenderer(),
	)

	done := make(chan error, 1)
	go func() {
		_, err := p.Run()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after input closed")
	}
}

func TestProgramRunsSequenceInOrder(t *testing.T) {
	var calls []string
	var updates []Msg
	init := Sequence(
		func() Msg {
			calls = append(calls, "one")
			return "one"
		},
		nil,
		func() Msg {
			calls = append(calls, "two")
			return "two"
		},
		Quit,
	)
	p := NewProgram(
		recordModel{init: init, updates: &updates},
		WithInput(nil),
		WithOutput(io.Discard),
		WithoutRenderer(),
	)

	if _, err := p.Run(); err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if got, want := strings.Join(calls, ","), "one,two"; got != want {
		t.Fatalf("command calls = %q, want %q", got, want)
	}
	if len(updates) != 2 || updates[0] != "one" || updates[1] != "two" {
		t.Fatalf("updates = %#v, want one/two", updates)
	}
}

func TestProgramPrintCommandBypassesUpdate(t *testing.T) {
	var out bytes.Buffer
	var updates []Msg
	init := Sequence(
		Println("completed block"),
		Quit,
	)
	p := NewProgram(
		recordModel{init: init, view: "live view", updates: &updates},
		WithInput(nil),
		WithOutput(&out),
	)

	if _, err := p.Run(); err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if len(updates) != 0 {
		t.Fatalf("updates = %#v, want no model updates for print command", updates)
	}
	got := out.String()
	if !strings.Contains(got, "completed block\r\n") {
		t.Fatalf("output missing printed block with CRLF: %q", got)
	}
	if !strings.Contains(got, "live view") {
		t.Fatalf("output missing managed live view after print: %q", got)
	}
	if strings.Contains(got, "\x1b[2J") {
		t.Fatalf("normal-screen print should not clear the full screen: %q", got)
	}
	if strings.Contains(got, "\x1b[J") {
		t.Fatalf("normal-screen print should not use erase-to-end-of-screen: %q", got)
	}
}

func TestProgramSequencePrintsAdjacentLinesBeforeRepaint(t *testing.T) {
	var out bytes.Buffer
	init := Sequence(
		Println("completed one"),
		Println("completed two"),
		Quit,
	)
	p := NewProgram(
		recordModel{init: init, view: "live view"},
		WithInput(nil),
		WithOutput(&out),
	)

	if _, err := p.Run(); err != nil {
		t.Fatalf("Run error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "completed one\r\ncompleted two\r\n\x1b[?25l\x1b[?7llive view") {
		t.Fatalf("completed lines should print as one block before repaint: %q", got)
	}
	if strings.Contains(got, "completed one\r\n\x1b[?25l\x1b[?7llive view") {
		t.Fatalf("live view was repainted between adjacent print commands: %q", got)
	}
}

func TestProgramPrintClearsOnlyManagedLiveRegion(t *testing.T) {
	var out bytes.Buffer
	p := NewProgram(recordModel{view: "live one\nlive two"}, WithInput(nil), WithOutput(&out))

	p.render()
	p.Print("completed block", true)

	got := out.String()
	if !strings.Contains(got, "completed block\r\n") {
		t.Fatalf("output missing printed block: %q", got)
	}
	if !strings.Contains(got, "\x1b[1A\r\x1b[2K\x1b[1B\r\x1b[2K\x1b[1A\rcompleted block\r\n") {
		t.Fatalf("print should clear only the previous two-line live region before writing: %q", got)
	}
	if !strings.Contains(got, "completed block\r\n\x1b[?25l\x1b[?7llive one\r\nlive two") {
		t.Fatalf("live view was not redrawn after printed block: %q", got)
	}
	if strings.Contains(got, "\x1b[2J") {
		t.Fatalf("normal-screen print should not clear the full screen: %q", got)
	}
}

func TestUpdateReturnedPrintRunsBeforeNextRender(t *testing.T) {
	var out bytes.Buffer
	p := NewProgram(printBeforeRenderModel{}, WithInput(nil), WithOutput(&out))

	if _, err := p.Run(); err != nil {
		t.Fatalf("Run error = %v", err)
	}
	got := out.String()
	completed := strings.Index(got, "completed before render\r\n")
	newLive := strings.LastIndex(got, "new live")
	if completed < 0 || newLive < 0 {
		t.Fatalf("output missing completed print or new live view: %q", got)
	}
	if completed > newLive {
		t.Fatalf("new live view should be rendered after completed print: %q", got)
	}
	if strings.Contains(got[:completed], "new live") {
		t.Fatalf("new live view was rendered before completed print: %q", got)
	}
}

func TestNormalRendererDisablesAutowrapAroundLiveView(t *testing.T) {
	var out bytes.Buffer
	p := NewProgram(recordModel{view: "full width live view"}, WithInput(nil), WithOutput(&out))

	p.render()

	got := out.String()
	if !strings.Contains(got, "\x1b[?25l\x1b[?7lfull width live view\x1b[0m\x1b[?7h") {
		t.Fatalf("normal renderer should disable autowrap only around live view: %q", got)
	}
}

func TestNormalRendererRepaintsLiveViewInPlace(t *testing.T) {
	var out bytes.Buffer
	p := NewProgram(recordModel{view: "static\ninput: a"}, WithInput(nil), WithOutput(&out))

	p.render()
	out.Reset()
	p.model = recordModel{view: "static\ninput: ab"}
	p.render()

	got := out.String()
	if strings.Contains(got, "\x1b[2K") {
		t.Fatalf("stable live repaint should not blank whole lines before redraw: %q", got)
	}
	if !strings.Contains(got, "\x1b[1A\rstatic\x1b[K\x1b[1B\rinput: ab\x1b[K") {
		t.Fatalf("stable live repaint should overwrite in place: %q", got)
	}
}

func TestNormalRendererClearsStaleLinesAfterShrink(t *testing.T) {
	var out bytes.Buffer
	p := NewProgram(recordModel{view: "one\ntwo\nthree"}, WithInput(nil), WithOutput(&out))

	p.render()
	out.Reset()
	p.model = recordModel{view: "one\ntwo"}
	p.render()

	got := out.String()
	if !strings.Contains(got, "one\x1b[K\x1b[1B\rtwo\x1b[K\x1b[s\x1b[1B\r\x1b[2K\x1b[u") {
		t.Fatalf("shrinking live repaint should clear stale bottom rows and restore cursor: %q", got)
	}
	if p.liveH != 2 {
		t.Fatalf("liveH = %d, want 2", p.liveH)
	}
}

func TestNormalRendererAutoCommitsOverflowToScrollback(t *testing.T) {
	var out bytes.Buffer
	p := NewProgram(
		recordModel{view: "one\ntwo\nthree"},
		WithInput(nil),
		WithOutput(&out),
		WithLiveHeight(2),
	)

	p.render()

	got := out.String()
	if !strings.Contains(got, "one\r\ntwo\r\nthree") {
		t.Fatalf("overflow top row should be printed before managed live rows: %q", got)
	}
	if p.liveH != 2 {
		t.Fatalf("liveH = %d, want 2", p.liveH)
	}
	if p.autoTop != 1 {
		t.Fatalf("autoTop = %d, want 1 committed row", p.autoTop)
	}
}

func TestNormalRendererAutoCommitsOnlyNewOverflowRows(t *testing.T) {
	var out bytes.Buffer
	p := NewProgram(
		recordModel{view: "one\ntwo\nthree"},
		WithInput(nil),
		WithOutput(&out),
		WithLiveHeight(2),
	)
	p.render()
	out.Reset()
	p.model = recordModel{view: "one\ntwo\nthree\nfour"}

	p.render()

	got := out.String()
	if strings.Contains(got, "one\r\n") {
		t.Fatalf("previously committed row was printed again: %q", got)
	}
	if !strings.Contains(got, "two\r\nthree\r\nfour") {
		t.Fatalf("new overflow row should stitch before remaining live rows: %q", got)
	}
	if p.liveH != 2 {
		t.Fatalf("liveH = %d, want 2", p.liveH)
	}
	if p.autoTop != 2 {
		t.Fatalf("autoTop = %d, want 2 committed rows", p.autoTop)
	}
}

func TestNormalRendererResetsAutoCommitWhenPrefixChanges(t *testing.T) {
	var out bytes.Buffer
	p := NewProgram(
		recordModel{view: "one\ntwo\nthree"},
		WithInput(nil),
		WithOutput(&out),
		WithLiveHeight(2),
	)
	p.render()
	out.Reset()
	p.model = recordModel{view: "new-one\nnew-two\nnew-three"}

	p.render()

	got := out.String()
	if !strings.Contains(got, "new-one\r\nnew-two\r\nnew-three") {
		t.Fatalf("changed prefix should reset auto scrollback state: %q", got)
	}
	if p.autoTop != 1 {
		t.Fatalf("autoTop = %d, want 1 after recommitting changed prefix", p.autoTop)
	}
}

func TestTerminalModesEnableBracketedPasteByDefault(t *testing.T) {
	var out bytes.Buffer
	p := NewProgram(recordModel{}, WithInput(nil), WithOutput(&out), WithoutRenderer())

	p.enterTerminalModes()
	p.exitTerminalModes()

	got := out.String()
	if !strings.Contains(got, "\x1b[?2004h") {
		t.Fatalf("enter modes did not enable bracketed paste: %q", got)
	}
	if !strings.Contains(got, "\x1b[?2004l") {
		t.Fatalf("exit modes did not disable bracketed paste: %q", got)
	}
}

func TestProgramRenderUsesCRLFLineEndings(t *testing.T) {
	var out bytes.Buffer
	p := NewProgram(recordModel{view: "one\ntwo\nthree"}, WithInput(nil), WithOutput(&out))

	p.render()

	got := out.String()
	if !strings.Contains(got, "one\r\ntwo\r\nthree") {
		t.Fatalf("render output did not normalize line endings: %q", got)
	}
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Fatalf("render output contains bare LF: %q", got)
	}
}

func TestTerminalLineEndingsPreservesExistingCRLF(t *testing.T) {
	got := terminalLineEndings("one\r\ntwo\nthree")
	want := "one\r\ntwo\r\nthree"
	if got != want {
		t.Fatalf("terminalLineEndings = %q, want %q", got, want)
	}
}
