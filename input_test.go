package agentui

import "testing"

func TestParseInputKeys(t *testing.T) {
	msgs, rest := ParseInput([]byte("a \x1b[A\x1b[1;5C\x03"))
	if len(rest) != 0 {
		t.Fatalf("rest = %q, want empty", rest)
	}
	if len(msgs) != 5 {
		t.Fatalf("len(msgs) = %d, want 5: %#v", len(msgs), msgs)
	}
	if got := msgs[0].(KeyMsg); got.Type != KeyRunes || string(got.Runes) != "a" {
		t.Fatalf("msg0 = %#v", got)
	}
	if got := msgs[1].(KeyMsg); got.Type != KeySpace {
		t.Fatalf("msg1 = %#v", got)
	}
	if got := msgs[2].(KeyMsg); got.Type != KeyUp {
		t.Fatalf("msg2 = %#v", got)
	}
	if got := msgs[3].(KeyMsg); got.Type != KeyCtrlRight {
		t.Fatalf("msg3 = %#v", got)
	}
	if got := msgs[4].(KeyMsg); got.Type != KeyCtrlC {
		t.Fatalf("msg4 = %#v", got)
	}
}

func TestParseInputPaste(t *testing.T) {
	msgs, rest := ParseInput([]byte("\x1b[200~one\ntwo\x1b[201~"))
	if len(rest) != 0 {
		t.Fatalf("rest = %q, want empty", rest)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	msg := msgs[0].(KeyMsg)
	if msg.Type != KeyRunes || !msg.Paste || string(msg.Runes) != "one\ntwo" {
		t.Fatalf("paste msg = %#v", msg)
	}
}

func TestParseInputMouseWheel(t *testing.T) {
	msgs, rest := ParseInput([]byte("\x1b[<64;10;5M\x1b[<65;10;5M"))
	if len(rest) != 0 {
		t.Fatalf("rest = %q, want empty", rest)
	}
	if len(msgs) != 2 {
		t.Fatalf("len(msgs) = %d, want 2", len(msgs))
	}
	up := msgs[0].(MouseMsg)
	if up.Button != MouseButtonWheelUp || up.Action != MouseActionPress || up.X != 10 || up.Y != 5 {
		t.Fatalf("wheel up = %#v", up)
	}
	down := msgs[1].(MouseMsg)
	if down.Button != MouseButtonWheelDown || down.Action != MouseActionPress {
		t.Fatalf("wheel down = %#v", down)
	}
}
