package agentui

import (
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type inputRead struct {
	data []byte
	err  error
}

func (p *Program) readInput() {
	reads := make(chan inputRead, 1)
	go readInputChunks(p.cfg.input, reads)

	var pending []byte
	var timer *time.Timer
	var timerC <-chan time.Time

	stopTimer := func() {
		if timer == nil {
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer = nil
		timerC = nil
	}
	startTimer := func() {
		stopTimer()
		if p.cfg.escTimeout <= 0 || !shouldTimeoutPending(pending) {
			return
		}
		timer = time.NewTimer(p.cfg.escTimeout)
		timerC = timer.C
	}

	for {
		select {
		case read := <-reads:
			if len(read.data) > 0 {
				pending = append(pending, read.data...)
				pending = p.sendParsedInput(pending, false)
			}
			if read.err != nil {
				stopTimer()
				if len(pending) > 0 {
					p.sendParsedInput(pending, true)
				}
				p.Send(QuitMsg{})
				return
			}
			if len(pending) > 0 {
				startTimer()
			} else {
				stopTimer()
			}
		case <-timerC:
			timerC = nil
			if len(pending) > 0 {
				pending = p.sendParsedInput(pending, true)
			}
			if len(pending) > 0 {
				startTimer()
			}
		case <-p.done:
			stopTimer()
			return
		}
	}
}

func readInputChunks(r io.Reader, reads chan<- inputRead) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		read := inputRead{err: err}
		if n > 0 {
			read.data = append([]byte(nil), buf[:n]...)
		}
		reads <- read
		if err != nil {
			return
		}
	}
}

func (p *Program) sendParsedInput(data []byte, flush bool) []byte {
	msgs, rest := parseInput(data, flush)
	for _, msg := range msgs {
		p.Send(msg)
	}
	return rest
}

// ParseInput converts terminal bytes into messages. The returned rest contains
// incomplete trailing escape or UTF-8 sequences.
func ParseInput(data []byte) ([]Msg, []byte) {
	return parseInput(data, false)
}

func parseInput(data []byte, flush bool) ([]Msg, []byte) {
	var msgs []Msg
	for len(data) > 0 {
		if data[0] == 0x1b {
			msg, n, complete := parseEscape(data)
			if !complete {
				if flush {
					msgs = append(msgs, KeyMsg{Type: KeyEsc})
					data = data[1:]
					continue
				}
				return msgs, data
			}
			if msg != nil {
				msgs = append(msgs, msg)
			}
			data = data[n:]
			continue
		}

		b := data[0]
		switch b {
		case '\r', '\n':
			msgs = append(msgs, KeyMsg{Type: KeyEnter})
			data = data[1:]
		case '\t':
			msgs = append(msgs, KeyMsg{Type: KeyTab})
			data = data[1:]
		case 0x7f:
			msgs = append(msgs, KeyMsg{Type: KeyBackspace})
			data = data[1:]
		default:
			if b < 0x20 {
				if t := ctrlKeyType(b); t != KeyUnknown {
					msgs = append(msgs, KeyMsg{Type: t})
				}
				data = data[1:]
				continue
			}
			r, n := utf8.DecodeRune(data)
			if r == utf8.RuneError && n == 1 && !utf8.FullRune(data) {
				return msgs, data
			}
			if r == ' ' {
				msgs = append(msgs, KeyMsg{Type: KeySpace})
			} else {
				msgs = append(msgs, KeyMsg{Type: KeyRunes, Runes: []rune{r}})
			}
			data = data[n:]
		}
	}
	return msgs, nil
}

func shouldTimeoutPending(data []byte) bool {
	if len(data) == 0 || data[0] != 0x1b {
		return false
	}
	return !strings.HasPrefix(string(data), "\x1b[200~")
}

func parseEscape(data []byte) (Msg, int, bool) {
	if len(data) == 1 {
		return KeyMsg{Type: KeyEsc}, 1, false
	}
	if strings.HasPrefix(string(data), "\x1b[200~") {
		end := strings.Index(string(data), "\x1b[201~")
		if end < 0 {
			return nil, 0, false
		}
		payload := string(data[len("\x1b[200~"):end])
		return KeyMsg{Type: KeyRunes, Runes: []rune(payload), Paste: true}, end + len("\x1b[201~"), true
	}
	if strings.HasPrefix(string(data), "\x1b[<") {
		if msg, n, ok := parseSGRMouse(data); ok {
			return msg, n, true
		}
		return nil, 0, false
	}
	if len(data) >= 3 {
		switch string(data[:3]) {
		case "\x1b[A":
			return KeyMsg{Type: KeyUp}, 3, true
		case "\x1b[B":
			return KeyMsg{Type: KeyDown}, 3, true
		case "\x1b[C":
			return KeyMsg{Type: KeyRight}, 3, true
		case "\x1b[D":
			return KeyMsg{Type: KeyLeft}, 3, true
		case "\x1b[H":
			return KeyMsg{Type: KeyHome}, 3, true
		case "\x1b[F":
			return KeyMsg{Type: KeyEnd}, 3, true
		case "\x1b[I":
			return FocusMsg{}, 3, true
		case "\x1b[O":
			return BlurMsg{}, 3, true
		}
	}
	if len(data) >= 4 {
		switch string(data[:4]) {
		case "\x1b[3~":
			return KeyMsg{Type: KeyDelete}, 4, true
		case "\x1b[5~":
			return KeyMsg{Type: KeyPgUp}, 4, true
		case "\x1b[6~":
			return KeyMsg{Type: KeyPgDown}, 4, true
		case "\x1b[1~", "\x1b[7~":
			return KeyMsg{Type: KeyHome}, 4, true
		case "\x1b[4~", "\x1b[8~":
			return KeyMsg{Type: KeyEnd}, 4, true
		}
	}
	if len(data) >= 6 {
		switch string(data[:6]) {
		case "\x1b[1;5C":
			return KeyMsg{Type: KeyCtrlRight}, 6, true
		case "\x1b[1;5D":
			return KeyMsg{Type: KeyCtrlLeft}, 6, true
		}
	}
	if len(data) >= 2 {
		if data[1] == '\r' || data[1] == '\n' {
			return KeyMsg{Type: KeyEnter, Alt: true}, 2, true
		}
		if data[1] == 'b' || data[1] == 'B' {
			return KeyMsg{Type: KeyLeft, Alt: true}, 2, true
		}
		if data[1] == 'f' || data[1] == 'F' {
			return KeyMsg{Type: KeyRight, Alt: true}, 2, true
		}
		r, n := utf8.DecodeRune(data[1:])
		if r == utf8.RuneError && n == 1 && !utf8.FullRune(data[1:]) {
			return nil, 0, false
		}
		if n > 0 && r != utf8.RuneError {
			return KeyMsg{Type: KeyRunes, Runes: []rune{r}, Alt: true}, 1 + n, true
		}
	}
	return KeyMsg{Type: KeyEsc}, 1, true
}

func parseSGRMouse(data []byte) (MouseMsg, int, bool) {
	s := string(data)
	end := strings.IndexAny(s, "Mm")
	if end < 0 {
		return MouseMsg{}, 0, false
	}
	final := s[end]
	body := strings.TrimPrefix(s[:end], "\x1b[<")
	parts := strings.Split(body, ";")
	if len(parts) != 3 {
		return MouseMsg{}, end + 1, true
	}
	code, _ := strconv.Atoi(parts[0])
	x, _ := strconv.Atoi(parts[1])
	y, _ := strconv.Atoi(parts[2])
	msg := MouseMsg{X: x, Y: y}
	if final == 'm' {
		msg.Action = MouseActionRelease
	} else {
		msg.Action = MouseActionPress
	}
	switch {
	case code&64 != 0 && code&1 == 0:
		msg.Button = MouseButtonWheelUp
	case code&64 != 0 && code&1 == 1:
		msg.Button = MouseButtonWheelDown
	case code&3 == 0:
		msg.Button = MouseButtonLeft
	case code&3 == 1:
		msg.Button = MouseButtonMiddle
	case code&3 == 2:
		msg.Button = MouseButtonRight
	default:
		msg.Button = MouseButtonUnknown
	}
	return msg, end + 1, true
}

func ctrlKeyType(b byte) KeyType {
	switch b {
	case 0x01:
		return KeyCtrlA
	case 0x02:
		return KeyCtrlB
	case 0x03:
		return KeyCtrlC
	case 0x04:
		return KeyCtrlD
	case 0x05:
		return KeyCtrlE
	case 0x06:
		return KeyCtrlF
	case 0x07:
		return KeyCtrlG
	case 0x08:
		return KeyCtrlH
	case 0x0a:
		return KeyCtrlJ
	case 0x0b:
		return KeyCtrlK
	case 0x0c:
		return KeyCtrlL
	case 0x0d:
		return KeyCtrlM
	case 0x0e:
		return KeyCtrlN
	case 0x0f:
		return KeyCtrlO
	case 0x10:
		return KeyCtrlP
	case 0x12:
		return KeyCtrlR
	case 0x14:
		return KeyCtrlT
	case 0x15:
		return KeyCtrlU
	case 0x17:
		return KeyCtrlW
	default:
		return KeyUnknown
	}
}
