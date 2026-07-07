package agentui

import "strings"

// KeyType identifies a keyboard event.
type KeyType int

const (
	KeyUnknown KeyType = iota
	KeyRunes
	KeySpace
	KeyEnter
	KeyTab
	KeyEsc
	KeyBackspace
	KeyDelete
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDown
	KeyCtrlA
	KeyCtrlB
	KeyCtrlC
	KeyCtrlD
	KeyCtrlE
	KeyCtrlF
	KeyCtrlG
	KeyCtrlH
	KeyCtrlJ
	KeyCtrlK
	KeyCtrlL
	KeyCtrlM
	KeyCtrlN
	KeyCtrlO
	KeyCtrlP
	KeyCtrlR
	KeyCtrlT
	KeyCtrlU
	KeyCtrlW
	KeyCtrlLeft
	KeyCtrlRight
)

// KeyMsg describes one keyboard event. Runes is populated when Type is
// KeyRunes. Paste is true for bracketed paste payloads.
type KeyMsg struct {
	Type  KeyType
	Runes []rune
	Alt   bool
	Paste bool
}

func (m KeyMsg) String() string {
	switch m.Type {
	case KeyRunes:
		return string(m.Runes)
	case KeySpace:
		return " "
	case KeyEnter:
		if m.Alt {
			return "alt+enter"
		}
		return "enter"
	case KeyTab:
		return "tab"
	case KeyEsc:
		return "esc"
	case KeyBackspace:
		return "backspace"
	case KeyDelete:
		return "delete"
	case KeyLeft:
		if m.Alt {
			return "alt+left"
		}
		return "left"
	case KeyRight:
		if m.Alt {
			return "alt+right"
		}
		return "right"
	case KeyUp:
		return "up"
	case KeyDown:
		return "down"
	case KeyHome:
		return "home"
	case KeyEnd:
		return "end"
	case KeyPgUp:
		return "pgup"
	case KeyPgDown:
		return "pgdown"
	case KeyCtrlLeft:
		return "ctrl+left"
	case KeyCtrlRight:
		return "ctrl+right"
	default:
		if name := ctrlName(m.Type); name != "" {
			return name
		}
		return "unknown"
	}
}

func ctrlName(t KeyType) string {
	names := map[KeyType]string{
		KeyCtrlA: "ctrl+a",
		KeyCtrlB: "ctrl+b",
		KeyCtrlC: "ctrl+c",
		KeyCtrlD: "ctrl+d",
		KeyCtrlE: "ctrl+e",
		KeyCtrlF: "ctrl+f",
		KeyCtrlG: "ctrl+g",
		KeyCtrlH: "ctrl+h",
		KeyCtrlJ: "ctrl+j",
		KeyCtrlK: "ctrl+k",
		KeyCtrlL: "ctrl+l",
		KeyCtrlM: "ctrl+m",
		KeyCtrlN: "ctrl+n",
		KeyCtrlO: "ctrl+o",
		KeyCtrlP: "ctrl+p",
		KeyCtrlR: "ctrl+r",
		KeyCtrlT: "ctrl+t",
		KeyCtrlU: "ctrl+u",
		KeyCtrlW: "ctrl+w",
	}
	return names[t]
}

// MatchesRunes reports whether m is a rune key matching s after trimming
// surrounding whitespace. It is convenient for modal shortcuts such as q/y/n.
func (m KeyMsg) MatchesRunes(s string) bool {
	return m.Type == KeyRunes && strings.TrimSpace(string(m.Runes)) == s
}
