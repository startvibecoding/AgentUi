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
