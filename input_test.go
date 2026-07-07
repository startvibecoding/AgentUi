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

func TestParseInputKeepsIncompleteEscape(t *testing.T) {
	msgs, rest := ParseInput([]byte{0x1b})
	if len(msgs) != 0 {
		t.Fatalf("msgs = %#v, want none", msgs)
	}
	if string(rest) != "\x1b" {
		t.Fatalf("rest = %q, want escape", rest)
	}
}

func TestParseInputFlushesIncompleteEscape(t *testing.T) {
	msgs, rest := parseInput([]byte{0x1b}, true)
	if len(rest) != 0 {
		t.Fatalf("rest = %q, want empty", rest)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	if got := msgs[0].(KeyMsg); got.Type != KeyEsc {
		t.Fatalf("msg = %#v, want KeyEsc", got)
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

func TestParseInputAltEnterAndAltRunes(t *testing.T) {
	msgs, rest := ParseInput([]byte("\x1b\r\x1bb"))
	if len(rest) != 0 {
		t.Fatalf("rest = %q, want empty", rest)
	}
	if len(msgs) != 2 {
		t.Fatalf("len(msgs) = %d, want 2: %#v", len(msgs), msgs)
	}
	enter := msgs[0].(KeyMsg)
	if enter.Type != KeyEnter || !enter.Alt {
		t.Fatalf("alt enter = %#v", enter)
	}
	left := msgs[1].(KeyMsg)
	if left.Type != KeyLeft || !left.Alt {
		t.Fatalf("alt+b = %#v, want alt left", left)
	}
}

func TestParseInputHomeEndVariants(t *testing.T) {
	msgs, rest := ParseInput([]byte("\x1b[H\x1b[F\x1b[1~\x1b[4~"))
	if len(rest) != 0 {
		t.Fatalf("rest = %q, want empty", rest)
	}
	want := []KeyType{KeyHome, KeyEnd, KeyHome, KeyEnd}
	if len(msgs) != len(want) {
		t.Fatalf("len(msgs) = %d, want %d: %#v", len(msgs), len(want), msgs)
	}
	for i, msg := range msgs {
		if got := msg.(KeyMsg).Type; got != want[i] {
			t.Fatalf("msg %d type = %v, want %v", i, got, want[i])
		}
	}
}

func TestParseInputKeepsIncompleteUTF8(t *testing.T) {
	msgs, rest := ParseInput([]byte{0xe4, 0xbd})
	if len(msgs) != 0 {
		t.Fatalf("msgs = %#v, want none", msgs)
	}
	if string(rest) != string([]byte{0xe4, 0xbd}) {
		t.Fatalf("rest = %q, want incomplete UTF-8 bytes", rest)
	}
	msgs, rest = parseInput(append(rest, 0xa0), false)
	if len(rest) != 0 {
		t.Fatalf("rest after complete rune = %q, want empty", rest)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	got := msgs[0].(KeyMsg)
	if got.Type != KeyRunes || string(got.Runes) != "你" {
		t.Fatalf("msg = %#v, want rune 你", got)
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
